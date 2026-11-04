// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La fenêtre Wayland : une surface, le rôle qui en fait une fenêtre de bureau, et la
// négociation de taille qui en découle.
//
// **Rien ici ne décide de la taille de la fenêtre, et c'est la différence la plus grande
// avec les deux autres familles.** Sous Windows comme sous X11, le client demande une
// taille et l'obtient ; ici il propose la sienne au montage, le compositeur répond par un
// `configure` qui suggère, et le client acquitte avant de peindre. Un tampon attaché avant
// ce premier `configure` est une erreur de protocole nommée, pas une image perdue.

package backend

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
)

// wlAppIDFallback est l'identifiant d'application employé quand l'exécutable n'en donne
// pas.
const wlAppIDFallback = "lozengine"

// wlWindow est une fenêtre portée par un compositeur Wayland.
//
// Les trois identifiants sont gardés séparément parce que les trois objets reçoivent des
// événements et qu'un opcode seul ne dit pas de qui il vient : `xdg_surface.configure` et
// `xdg_toplevel.configure` valent tous deux zéro, et ils ne portent ni les mêmes arguments
// ni le même sens.
type wlWindow struct {
	c *wlConn
	g wlGlobals

	// surface porte les pixels, xdg lui donne le protocole de bureau, et toplevel le
	// rôle de fenêtre. Les trois désignent la même chose à l'écran.
	surface  uint32
	xdg      uint32
	toplevel uint32

	// seat porte le clavier et le pointeur, et state ce qu'ils ont dit depuis le pas
	// précédent.
	//
	// L'état vit sur la fenêtre et non sur le siège parce que c'est la fenêtre qui en
	// tire une image d'entrées, et parce qu'il survit à un périphérique débranché : un
	// siège qui perd sa capacité de clavier relâche les touches, il ne jette pas le
	// pointeur avec.
	seat  wlSeat
	state input.State

	// hideCursor retient ce que l'ouverture a demandé, parce que la requête qui
	// l'applique se renvoie à chaque prise de focus du pointeur et non une fois.
	hideCursor bool

	// configured dit que le premier `configure` est arrivé, donc qu'un tampon peut
	// s'attacher. Il ne se relève jamais : la surface reste configurée jusqu'à sa
	// destruction.
	configured bool

	// serial est le numéro du dernier `configure` reçu, et pending dit qu'il attend
	// son acquittement.
	//
	// **Un seul numéro suffit, et la description l'autorise explicitement** : « if the
	// client receives multiple configure events before it can respond to one, it is
	// free to discard all but the last event it received ». Les garder tous
	// obligerait à les acquitter tous, ce qui est permis mais sans objet.
	serial  uint32
	pending bool

	// width et height sont la taille retenue : celle de l'ouverture tant que le
	// compositeur n'en propose pas d'autre.
	width  uint16
	height uint16

	// pool est la mémoire partagée de la géométrie en cours, et nil avant la première
	// présentation. Une fenêtre peut vivre sans — un jeu qui ne présente rien reste une
	// fenêtre —, et la créer au montage demanderait de deviner la taille que le
	// compositeur n'a pas encore proposée.
	pool *wlPool

	// closed retient que la fenêtre ne présentera plus rien. Il ne se relève jamais.
	closed bool

	// err porte ce qui a mis fin à la fenêtre autrement que par la volonté du joueur.
	// Seule la première cause est gardée : ce qui suit une connexion perdue n'en est
	// que la conséquence.
	err error
}

// wlOpen ouvre une fenêtre sur le compositeur que l'environnement désigne.
func wlOpen(o Options) (*wlWindow, error) {
	c, err := wlConnect()
	if err != nil {
		return nil, err
	}

	w := &wlWindow{c: c, width: o.Width, height: o.Height}
	if err := w.mount(o); err != nil {
		c.Close()
		return nil, err
	}
	return w, nil
}

// mount monte la surface, lui donne son rôle, et attend la configuration initiale.
//
// **L'ordre des six requêtes n'est pas un ordre de commodité, il est prescrit.** La
// description de `xdg_surface` est explicite sur trois points : « a role must be assigned
// before any other requests are made to the xdg_surface object » ; « creating an
// xdg_surface from a wl_surface which has a buffer attached or committed is a client
// error » ; et surtout « the client must perform an initial commit without any buffer
// attached », après quoi « the compositor will reply with initial wl_surface state […]
// followed by an xdg_surface.configure event ». Ce commit à vide n'est donc pas une astuce
// pour provoquer une réponse, c'est la marche que le protocole impose — et sans lui le
// compositeur n'envoie jamais de `configure`, donc la fenêtre n'existe pas.
//
// L'attente est bornée, et c'est le seul endroit du backend qui le soit : le reste de la
// vie de la fenêtre passe par une lecture qui rend la main. Un compositeur qui n'a pas
// répondu dans ce délai est en panne, et l'alternative serait un jeu figé sans message.
func (w *wlWindow) mount(o Options) error {
	c := w.c
	if err := c.bound(wlHandshake); err != nil {
		return err
	}

	g, err := c.wlBindGlobals()
	if err != nil {
		return err
	}
	w.g = g
	w.hideCursor = o.HideCursor
	c.handler = w.handle

	// **Le siège se prend avant la surface, et son masque de capacités est déjà
	// connu** : il est arrivé dans l'aller-retour qui a suivi la liaison, ce que le
	// registre collecte pour cette raison. Les deux objets d'entrée ne dépendent pas de
	// la surface — un `enter` ne peut venir qu'après qu'elle existe —, donc les créer
	// d'abord garde les six requêtes de la fenêtre d'un seul bloc.
	w.seat.id, w.seat.version = g.seat, g.seatVersion
	if err := w.wlApplyCaps(g.seatCaps); err != nil {
		return err
	}

	surface, err := c.newID()
	if err != nil {
		return err
	}
	xdg, err := c.newID()
	if err != nil {
		return err
	}
	toplevel, err := c.newID()
	if err != nil {
		return err
	}
	w.surface, w.xdg, w.toplevel = surface, xdg, toplevel

	c.out.begin(g.compositor, wlCompositorCreateSurface)
	c.out.uint(surface)
	c.out.end()

	c.out.begin(g.wmBase, wlXDGWMBaseGetXDGSurface)
	c.out.uint(xdg)
	c.out.uint(surface)
	c.out.end()

	c.out.begin(xdg, wlXDGSurfaceGetToplevel)
	c.out.uint(toplevel)
	c.out.end()

	c.out.begin(toplevel, wlXDGToplevelSetTitle)
	c.out.string(o.Title)
	c.out.end()

	c.out.begin(toplevel, wlXDGToplevelSetAppID)
	c.out.string(wlAppID())
	c.out.end()

	c.out.begin(surface, wlSurfaceCommit)
	c.out.end()

	if err := c.flush(); err != nil {
		return err
	}

	// La fermeture est une sortie de boucle et non une erreur : un compositeur a le
	// droit de retirer une fenêtre avant de l'avoir configurée, et le jeu le verra par
	// `Closed` au premier pas.
	for !w.configured && !w.closed {
		if err := c.fill(); err != nil {
			return fmt.Errorf("waiting for the compositor to configure the window: %w", err)
		}
		if err := c.dispatch(); err != nil {
			return err
		}
	}

	w.ack()
	if err := c.flush(); err != nil {
		return err
	}
	return c.bound(0)
}

// handle reçoit les événements des objets de la fenêtre.
//
// **Le routage se fait par émetteur avant l'opcode, et c'est obligatoire ici** : les deux
// `configure` du protocole de bureau portent le même opcode zéro, et celui de
// `xdg_toplevel` suggère une taille là que celui d'`xdg_surface` clôt la séquence. Router
// sur l'opcode seul confondrait une largeur avec un numéro de série.
func (w *wlWindow) handle(sender uint32, opcode uint16, r *wlReader) error {
	// **`wl_buffer.release` se reconnaît avant les autres, et il ne peut pas entrer
	// dans le choix qui suit** : les tampons sont plusieurs et changent d'identifiant à
	// chaque géométrie. C'est l'émetteur qui tranche et non l'opcode, qui vaut zéro ici
	// comme pour la moitié des événements du protocole.
	if opcode == wlBufferRelease && w.pool != nil && w.pool.returned(sender) {
		return nil
	}

	switch sender {
	case w.xdg:
		if opcode == wlXDGSurfaceConfigure {
			serial := r.uint()
			if r.err != nil {
				return r.err
			}
			w.serial, w.pending, w.configured = serial, true, true
		}
		return nil

	case w.toplevel:
		switch opcode {
		case wlXDGToplevelConfigure:
			width, height := r.int(), r.int()
			// Les états — maximisée, plein écran, en cours de
			// redimensionnement — ne changent rien à ce que le moteur dessine :
			// l'échelle entière et les bandes autour règlent le cadrage quelle
			// que soit la raison de la taille. Le tableau est lu pour que le
			// curseur arrive au bout du message.
			r.array()
			if r.err != nil {
				return r.err
			}
			w.propose(width, height)

		case wlXDGToplevelClose:
			w.closed = true
		}
		return nil

	case w.seat.id:
		return w.wlSeatEvent(opcode, r)

	case w.seat.keyboard:
		return w.wlKeyboardEvent(opcode, r)

	case w.seat.pointer:
		return w.wlPointerEvent(opcode, r)

	case w.g.wmBase:
		if opcode == wlXDGWMBasePing {
			serial := r.uint()
			if r.err != nil {
				return r.err
			}
			// La réponse est composée et non envoyée : l'envoi appartient au
			// chemin qui a provoqué cette lecture, et écrire sur la socket au
			// milieu d'une lecture n'achèterait rien.
			w.c.out.begin(w.g.wmBase, wlXDGWMBasePong)
			w.c.out.uint(serial)
			w.c.out.end()
		}
		return nil

	case w.surface:
		// `enter`, `leave`, `preferred_buffer_scale` et `preferred_buffer_transform`
		// arrivent ici. Les deux premiers nomment un `wl_output` qu'on n'a pas lié ;
		// les deux autres n'existent qu'à partir de la version 6 de `wl_compositor`
		// et ne suggèrent qu'une échelle, que la règle d'échelle entière du moteur
		// ne prend pas d'un compositeur.
		return nil
	}
	return nil
}

// propose retient la taille que le compositeur suggère.
//
// **Zéro veut dire « décide toi-même »**, et la description le dit : « if the width or
// height arguments are zero, it means the client should decide its own window dimension ».
// La taille d'ouverture est donc gardée telle quelle, et c'est le cas d'un compositeur qui
// n'a pas d'avis au premier `configure`.
//
// Le protocole déclare ces deux arguments signés, donc une valeur négative est sur le fil
// même si elle n'a pas de sens pour une fenêtre : elle est écartée comme zéro plutôt que
// convertie, ce qui en ferait une fenêtre de soixante-cinq mille pixels de côté.
func (w *wlWindow) propose(width, height int32) {
	if width > 0 {
		w.width = uint16(min(width, math.MaxUint16))
	}
	if height > 0 {
		w.height = uint16(min(height, math.MaxUint16))
	}
}

// ack acquitte le dernier `configure` reçu, et ne fait rien quand aucun n'attend.
//
// **L'absence n'est pas un cas à forcer** : acquitter un numéro jamais émis lève
// `invalid_serial`, et le compositeur coupe la connexion. Et un seul acquittement suffit
// pour plusieurs `configure` — « it only has to ack the last configure event ».
func (w *wlWindow) ack() {
	if !w.pending {
		return
	}
	w.c.out.begin(w.xdg, wlXDGSurfaceAckConfigure)
	w.c.out.uint(w.serial)
	w.c.out.end()
	w.pending = false
}

// wlAppID rend l'identifiant d'application que la fenêtre déclare.
//
// **C'est tout ce qu'un compositeur Wayland sait faire d'une icône, et c'est pourquoi
// [Options.Icons] reste sans effet ici** : le protocole ne transporte pas d'image de
// fenêtre, il transporte un nom par lequel le bureau retrouve une entrée `.desktop` et
// l'icône qu'elle déclare. Ne rien déclarer laisse le compositeur sans rien à chercher,
// donc une icône générique et des fenêtres regroupées de travers.
//
// Le nom de l'exécutable est ce dont le backend dispose de plus proche de l'identité du
// jeu, et c'est aussi sous ce nom qu'une entrée `.desktop` s'installe. Son absence vaut le
// nom du moteur plutôt qu'une chaîne vide, qu'un compositeur traite comme une absence de
// déclaration.
func wlAppID() string {
	exe, err := os.Executable()
	if err != nil {
		return wlAppIDFallback
	}
	if base := filepath.Base(exe); base != "." && base != string(filepath.Separator) {
		return base
	}
	return wlAppIDFallback
}

// drain lit tout ce que le compositeur a envoyé, sans rien attendre.
//
// C'est le seul régime de lecture après l'ouverture : `Paced` rend faux, donc la boucle
// dort entre deux pas et un chemin qui bloquerait figerait le jeu sur un compositeur qui
// n'a rien à dire.
func (w *wlWindow) drain() error {
	for {
		more, err := w.c.receive()
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
		if err := w.c.dispatch(); err != nil {
			return err
		}
	}
}

// Input lit ce que le compositeur a envoyé, puis rend les entrées du pas qui commence.
//
// C'est le premier appel d'un pas, et c'est ce que le contrat de [Window] impose : la
// boucle lit les entrées, met à jour, puis présente. Lire ailleurs ferait répondre la
// fenêtre en retard d'une image, et la taille servie serait celle du pas précédent.
//
// **Elle ne se met jamais en attente**, parce que [wlWindow.Paced] rend faux : la boucle
// dort entre deux pas pour tenir sa cadence, donc une attente ici l'ajouterait à la
// sienne. Un compositeur qui n'a rien dit laisse simplement l'état inchangé.
//
// **C'est aussi d'ici que part ce que la lecture a composé**, et c'est la seule raison
// pour laquelle elle écrit sur la socket : un `pong` qu'aucun chemin n'émet fait couper la
// connexion, et une requête de curseur datée d'une prise de focus est ignorée en silence
// si elle arrive trop tard. Les deux naissent pendant la lecture, et celle-ci est le
// chemin que chaque pas traverse.
func (w *wlWindow) Input() input.Input {
	if !w.closed {
		if err := w.drain(); err != nil {
			w.fail(err)
		} else if err := w.c.flush(); err != nil {
			w.fail(err)
		}
	}
	return w.state.Frame()
}

// fit garantit un pool à la géométrie demandée.
//
// **Une fenêtre qui change de taille change de pool, et c'est la seule voie** : un
// `wl_buffer` porte ses dimensions, donc les siens ne servent plus. [wlPool] dit pourquoi
// l'agrandissement en place a été écarté.
func (w *wlWindow) fit(width, height int) error {
	if w.pool != nil && w.pool.width == width && w.pool.height == height {
		return nil
	}
	if w.pool != nil {
		w.pool.drop(w.c)
		w.pool = nil
	}

	pool, err := wlNewPool(w.c, w.g, width, height)
	if err != nil {
		return err
	}
	w.pool = pool
	return nil
}

// Present pose le tampon dans la mémoire que le compositeur lit, et le lui commet.
//
// **Les pixels ne traversent rien : ils sont écrits là où le compositeur les lira.** C'est
// la différence avec X11, où la même image passe par une requête sur une socket. Ce que
// cette famille demande en échange est de savoir quel tampon est libre, donc de **lire**
// avant d'écrire — une présentation qui ne lirait pas trouverait tout détenu au second
// tour et ne montrerait plus rien.
//
// **Ce que tout ce qui n'est pas l'écriture coûte est mesuré, et c'est deux
// microsecondes** : le drain, le choix du tampon, les trois requêtes et l'envoi tiennent
// ensemble dans un centième de pour cent du budget d'une image. Cette présentation **est**
// donc son écriture, dont le prix se lit dans [wlPool.write]. Voir
// [BenchmarkWaylandPresent].
//
// Trois raisons de ne rien faire, et aucune n'est un échec, ce que le contrat prévoit : la
// fenêtre est fermée, elle n'a pas encore reçu sa configuration — un tampon attaché avant
// serait une erreur de protocole —, ou le compositeur détient les deux tampons. Ce qui est
// terminal, en revanche, passe par [wlWindow.fail] : une mémoire partagée qu'on ne peut
// plus créer n'est pas un tour perdu.
//
// L'acquittement vient juste avant le commit et non à la réception du `configure`, parce
// que c'est l'ordre que la description impose : « if a client commits the surface in
// response to the configure event, then the client must make an ack_configure request
// sometime before the commit request ».
func (w *wlWindow) Present(t *render.Buffer) {
	if w.closed || !w.configured || t == nil || len(t.Pixels) == 0 {
		return
	}

	if err := w.drain(); err != nil {
		w.fail(err)
		return
	}
	if err := w.fit(int(t.Width), int(t.Height)); err != nil {
		w.fail(err)
		return
	}

	i := w.pool.free()
	if i < 0 {
		return
	}
	b := &w.pool.buffers[i]
	w.pool.write(b.at, w.g.pixels, t.Pixels)
	b.held, w.pool.last = true, i

	c := w.c
	c.out.begin(w.surface, wlSurfaceAttach)
	c.out.uint(b.id)
	c.out.uint(0)
	c.out.uint(0)
	c.out.end()

	// **`damage` et non `damage_buffer`, parce que les deux désignent ici la même
	// région.** Le second est la requête moderne et travaille en coordonnées de tampon ;
	// le premier travaille en coordonnées de surface, et les deux coïncident tant que la
	// surface n'a ni échelle ni rotation. Le moteur ne pose jamais `set_buffer_scale` —
	// sa règle d'échelle entière lui interdit de prendre celle d'un compositeur —, donc
	// la coïncidence tient par construction et un seul chemin suffit à toutes les
	// versions.
	c.out.begin(w.surface, wlSurfaceDamage)
	c.out.uint(0)
	c.out.uint(0)
	c.out.uint(uint32(t.Width))
	c.out.uint(uint32(t.Height))
	c.out.end()

	w.ack()

	c.out.begin(w.surface, wlSurfaceCommit)
	c.out.end()

	if err := c.flush(); err != nil {
		w.fail(err)
	}
}

// Paced rend faux, et c'est la réponse que le jalon 6 a renversée.
//
// **Le jeton de cadence du protocole existe et ne peut pas servir ici.** `wl_surface.frame`
// demande un rappel pour l'image suivante, mais il « ne prend effet qu'au commit suivant » :
// il s'arme donc dans la présentation, et pas ailleurs. Or la boucle, devant une fenêtre
// qui se dit cadencée, **saute la présentation** d'un tour arrivé en avance — donc le
// jeton ne se réarmerait pas, et le tour suivant attendrait un rappel que personne n'a
// demandé. La fenêtre se figerait au premier tour en avance.
//
// C'est ce que le contrat de [Window.Paced] exige en toutes lettres : que le jeton se
// réarme **sans passer par Present**. Une page le tient, parce que son attente vit dans
// `Input` ; un compositeur Wayland ne le tient pas.
func (w *wlWindow) Paced() bool {
	return false
}

// fail retient ce qui a mis fin à la fenêtre autrement que par la volonté du joueur.
//
// Seule la première cause est gardée, comme X11 le fait : ce qui suit une connexion perdue
// n'en est que la conséquence.
func (w *wlWindow) fail(err error) {
	if w.err == nil {
		w.err = err
	}
	w.closed = true
}

// Size rend la taille que le compositeur a proposée en dernier.
func (w *wlWindow) Size() (width, height uint16) {
	return w.width, w.height
}

// Closed dit que la fenêtre ne présentera plus rien.
func (w *wlWindow) Closed() bool {
	return w.closed
}

// Err rend ce qui a mis fin à la fenêtre autrement que par la volonté du joueur.
func (w *wlWindow) Err() error {
	return w.err
}

// Close rend la connexion au compositeur.
//
// Les objets n'y sont pas détruits un par un : fermer la socket détruit tout ce que la
// connexion portait, et le compositeur n'a rien à reprendre ensuite. Détruire d'abord
// n'économiserait qu'un instant de table chez lui, au prix d'un ordre à tenir — un
// `xdg_surface` ne se détruit qu'après son rôle.
func (w *wlWindow) Close() {
	if w.pool != nil {
		w.pool.unmap()
		w.pool = nil
	}
	w.c.Close()
}
