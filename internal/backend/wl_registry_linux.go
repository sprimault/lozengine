// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le compositeur offre, ce qu'on en prend, et la disposition de pixel qui en
// découle.
//
// Un client Wayland ne demande pas ce qu'il veut : il écoute ce qu'on lui annonce, puis
// lie ce qu'il reconnaît. Deux allers-retours suffisent et il en faut deux — le premier
// parce que rien ne dit quand la liste des offres se termine, le second parce que les
// formats de la mémoire partagée n'arrivent qu'après qu'on l'a liée.

package backend

import (
	"errors"
	"fmt"
)

// wlOffer est un global que le compositeur a annoncé : son nom dans le registre et la
// version qu'il porte.
type wlOffer struct {
	name    uint32
	version uint32
}

// wlWanted décrit une interface dont le backend a besoin.
//
// **La version voulue se justifie par ce qu'elle apporte, jamais par le plafond que la
// description porte.** Lier haut n'achète rien et ferme la porte à un compositeur plus
// ancien ; lier bas prive de requêtes qu'on emploie. Le champ `gives` dit donc ce que la
// version demandée rend possible, et c'est lui qu'on relit quand on se demande pourquoi
// ce chiffre.
type wlWanted struct {
	iface   string
	version uint32
	gives   string
}

// wlNeeded énumère ce que le backend lie, avec la raison de chaque version.
//
// `wl_seat` n'y est pas : le lot qui lit le clavier et le pointeur l'ajoutera, et lier un
// objet dont personne ne se sert ne ferait qu'occuper une entrée de la table que le
// compositeur tient.
var wlNeeded = []wlWanted{
	{"wl_compositor", 6, "set_buffer_scale en 3, damage_buffer en 4, preferred_buffer_scale en 6"},
	{"wl_shm", 1, "create_pool et les formats ; release arrive en 2 et ne rend que l'objet lui-même, ce qu'une fenêtre ouverte n'a pas à faire"},
	{"xdg_wm_base", 1, "get_xdg_surface et pong, qui sont tout ce qu'une fenêtre demande"},
}

// wlGlobals porte ce que le client a lié et ce qu'il en a appris.
//
// Les versions retenues sont gardées à côté des identifiants parce que c'est d'elles que
// dépendent les requêtes qu'on a le droit d'émettre : un `damage_buffer` envoyé à un
// `wl_surface` de version 3 est une erreur de protocole, et le compositeur coupe la
// connexion.
type wlGlobals struct {
	compositor        uint32
	compositorVersion uint32

	shm uint32

	wmBase        uint32
	wmBaseVersion uint32

	// pixels est la disposition retenue parmi ce que `wl_shm` a annoncé, et format la
	// valeur sous laquelle le compositeur l'a annoncée.
	//
	// **Les deux, parce qu'elles ne servent pas au même endroit** : la disposition dit
	// comment écrire les octets, la valeur est ce que `create_buffer` exige. Et la
	// seconde ne se déduit pas de la première, les deux formats garantis portant des
	// valeurs historiques plutôt que leur code à quatre caractères.
	pixels pixelOrder
	format uint32
}

// wlRegistry collecte ce que le compositeur annonce, le temps de l'ouverture.
//
// Il ne survit pas au montage : une fois les globals liés et les formats connus, plus rien
// ne le consulte. `global_remove` est écouté quand même, parce qu'un global peut
// disparaître entre son annonce et notre `bind` — un écran débranché retire son
// `wl_output` —, et lier un nom retiré est une erreur de protocole.
type wlRegistry struct {
	id     uint32
	offers map[string]wlOffer

	// shm est l'identifiant de la mémoire partagée une fois liée, et formats ce
	// qu'elle a annoncé. L'identifiant est gardé plutôt que déduit : un opcode
	// d'événement vaut souvent zéro, donc router sur lui seul enverrait au bon endroit
	// pour la mauvaise raison, et le premier objet ajouté ici le ferait de travers.
	shm     uint32
	formats []uint32
}

// wlBindGlobals lie ce dont le backend a besoin et choisit la disposition des pixels.
//
// **Les deux allers-retours ne sont pas une précaution, ils sont la seule façon de
// savoir.** Le protocole est asynchrone et n'a pas de message « c'est tout » : après
// `get_registry`, le compositeur envoie une annonce par global, et rien ne distingue la
// fin de la liste d'un silence. Un `sync` répond une fois tout le reste traité, donc son
// retour est la preuve que la liste est complète. Le second vaut pour les formats, que
// `wl_shm` n'envoie qu'après avoir été lié.
func (c *wlConn) wlBindGlobals() (wlGlobals, error) {
	var g wlGlobals

	id, err := c.newID()
	if err != nil {
		return g, err
	}
	reg := &wlRegistry{id: id, offers: make(map[string]wlOffer, 32)}

	c.out.begin(wlDisplayID, wlDisplayGetRegistry)
	c.out.uint(reg.id)
	c.out.end()

	previous := c.handler
	c.handler = reg.handle
	defer func() { c.handler = previous }()

	if err := c.roundtrip(); err != nil {
		return g, err
	}

	if err := c.bind(reg, &g); err != nil {
		return g, err
	}
	if err := c.roundtrip(); err != nil {
		return g, err
	}

	g.pixels, g.format, err = wlPixelOrder(reg.formats)
	if err != nil {
		return g, err
	}
	return g, nil
}

// bind prend ce que la table des besoins demande, à la version que les deux acceptent.
//
// La version liée est le plus petit de ce qu'on veut et de ce qui est offert. Demander plus
// que l'offre est une erreur de protocole ; demander moins est permis et c'est ce qui rend
// un client compatible avec un compositeur plus récent.
func (c *wlConn) bind(reg *wlRegistry, g *wlGlobals) error {
	for _, want := range wlNeeded {
		offer, ok := reg.offers[want.iface]
		if !ok {
			return fmt.Errorf("the compositor does not offer %s, which this backend needs for %s",
				want.iface, want.gives)
		}

		version := min(want.version, offer.version)
		id, err := c.newID()
		if err != nil {
			return err
		}

		c.out.begin(reg.id, wlRegistryBind)
		c.out.uint(offer.name)
		c.out.newID(want.iface, version, id)
		c.out.end()

		switch want.iface {
		case "wl_compositor":
			g.compositor, g.compositorVersion = id, version
		case "wl_shm":
			g.shm, reg.shm = id, id
		case "xdg_wm_base":
			g.wmBase, g.wmBaseVersion = id, version
		}
	}
	return c.flush()
}

// handle reçoit les événements du registre et ceux de la mémoire partagée.
//
// Les deux arrivent pendant l'ouverture, et ce sont les deux seuls objets qui parlent
// avant qu'une surface existe — les rappels d'un aller-retour étant interceptés plus bas.
// Un événement d'un autre objet est donc ignoré plutôt que refusé : un compositeur a le
// droit d'annoncer ce qu'il veut sur des globals qu'on n'a pas liés.
func (r *wlRegistry) handle(sender uint32, opcode uint16, rd *wlReader) error {
	if sender == r.shm && r.shm != 0 {
		if opcode == wlShmFormat {
			r.formats = append(r.formats, rd.uint())
		}
		return rd.err
	}
	if sender != r.id {
		return nil
	}

	switch opcode {
	case wlRegistryGlobal:
		name, iface, version := rd.uint(), rd.string(), rd.uint()
		if rd.err != nil {
			return rd.err
		}
		r.offers[iface] = wlOffer{name: name, version: version}
		return nil

	case wlRegistryGlobalRemove:
		name := rd.uint()
		if rd.err != nil {
			return rd.err
		}
		for iface, offer := range r.offers {
			if offer.name == name {
				delete(r.offers, iface)
			}
		}
		return nil

	default:
		return fmt.Errorf("wl_registry sent event %d, which version 1 does not define", opcode)
	}
}

// wlPixelOrder choisit la disposition parmi les formats que le compositeur accepte, et
// rend la valeur sous laquelle il l'a annoncée.
//
// **La préférence est l'ordre du moteur, et elle lui épargne toute conversion** : `XB24`
// place le rouge, le vert et le bleu dans l'ordre où [render.Color] les porte, donc le
// tampon part tel quel. Le repli est `XR24`, qui échange le rouge et le bleu — le prix
// qu'X11 paie déjà — et que le protocole garantit : sa description le dit obligatoire pour
// tout compositeur.
//
// Les formats à alpha ne sont pas candidats, et [pixelOrderOfFourCC] dit pourquoi : les
// dispositions rapides écrivent zéro dans l'octet restant, ce qui rendrait la surface
// transparente. C'est aussi pour cela qu'`argb8888`, le second format garanti, ne sert pas
// de repli.
func wlPixelOrder(formats []uint32) (pixelOrder, uint32, error) {
	if len(formats) == 0 {
		return pixelOrder{}, 0, errors.New("the compositor announced no shared-memory pixel format, and the protocol requires at least two")
	}

	// **La valeur annoncée est gardée à côté du code, et c'est elle qu'il faut rendre** :
	// `wl_shm_pool.create_buffer` veut un format « which must be one of those
	// advertised through the wl_shm.format event », et la traduction vers un code à
	// quatre caractères est à sens unique. La refaire à l'envers rendrait `0x34325258`
	// là où le protocole attend `1` pour `xrgb8888`.
	//
	// Un compositeur qui annoncerait le même format sous ses deux valeurs — la
	// historique et son code — ferait gagner la dernière, et les deux désignent la même
	// disposition.
	offered := make(map[uint32]uint32, len(formats))
	for _, f := range formats {
		offered[wlFourCCOf(f)] = f
	}

	for _, code := range [2]uint32{formatXBGR8888, formatXRGB8888} {
		announced, ok := offered[code]
		if !ok {
			continue
		}
		order, err := pixelOrderOfFourCC(code)
		return order, announced, err
	}

	names := make([]string, 0, len(formats))
	for _, f := range formats {
		names = append(names, fourccName(wlFourCCOf(f)))
	}
	return pixelOrder{}, 0, fmt.Errorf("the compositor accepts none of the formats this backend writes; it offers %v", names)
}

// wlFourCCOf traduit une valeur de `wl_shm.format` vers un code à quatre caractères.
//
// Deux valeurs sur toutes celles du protocole ne sont pas des codes, et ce sont les deux
// garanties : la traduction porte donc sur le cas qui compte le plus. Tout le reste passe
// tel quel, la description donnant alors le code lui-même.
func wlFourCCOf(format uint32) uint32 {
	switch format {
	case wlShmARGB8888:
		return fourcc('A', 'R', '2', '4')
	case wlShmXRGB8888:
		return formatXRGB8888
	default:
		return format
	}
}
