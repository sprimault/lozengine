// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Un compositeur de pacotille qui juge au lieu d'enregistrer.
//
// **C'est la différence qui lui donne sa valeur, et elle a été demandée pour une raison
// précise** : un faux serveur qui accepte tout prouve seulement qu'on sait composer des
// octets. Celui-ci refuse ce que le protocole interdit — un tampon attaché avant la
// première configuration, un numéro de série acquitté sans avoir été émis, une requête
// émise avant que la surface ait un rôle, un `ping` resté sans réponse —, et chaque refus
// est un `wl_display.error`, donc fatal, exactement comme en face.
//
// Le traitement des requêtes est séparé de la socket, et c'est ce qui rend ses jugements
// éprouvables : un cas qui veut montrer qu'un refus mord remet la requête fautive
// directement, sans écrire un client de travers.

package backend

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

// wlFakeGlobal est un global que le compositeur de pacotille annonce.
type wlFakeGlobal struct {
	name    uint32
	iface   string
	version uint32
}

// wlFake est l'état qu'un compositeur tient d'un client, réduit à ce que l'ouverture
// d'une fenêtre touche.
type wlFake struct {
	// mu garde l'état, qu'une goroutine de lecture écrit et qu'un cas relit une fois
	// le montage rendu.
	mu sync.Mutex

	globals []wlFakeGlobal
	formats []uint32

	// Les objets que le client a créés, dans l'ordre où l'ouverture les crée. Zéro
	// veut dire « pas encore », et c'est ce qui permet de juger un ordre.
	registry   uint32
	compositor uint32
	shm        uint32
	wmBase     uint32
	surface    uint32
	xdg        uint32
	toplevel   uint32

	// configured dit qu'une séquence de configuration a été émise, et issued porte
	// les numéros de série émis — c'est contre elle que s'apprécie un acquittement.
	configured bool
	issued     map[uint32]bool
	nextSerial uint32

	// pinged porte le numéro d'un `ping` resté sans réponse, et zéro quand il n'y en
	// a pas.
	pinged uint32

	// ping demande au compositeur de sonder le client avant de le configurer, ce qui
	// est son régime par défaut : un `pong` manquant ferme alors le montage, donc
	// l'oubli ne peut pas passer inaperçu.
	ping bool

	// sequences est le nombre de séquences de configuration émises au premier commit.
	// Au-delà d'une, le client ne doit acquitter que la dernière.
	sequences int

	// proposeW et proposeH sont la taille que le `configure` du rôle annonce. Zéro
	// laisse le client décider, ce que la description prévoit.
	proposeW, proposeH int32

	// closing fait suivre la configuration d'une demande de fermeture.
	closing bool

	// seen porte les requêtes reconnues dans l'ordre, et refused le refus prononcé.
	// Il n'y en a qu'un : une erreur de protocole est fatale, donc rien ne suit.
	seen    []string
	refused string
}

// wlNewFake monte un compositeur de pacotille dans son régime ordinaire.
func wlNewFake(globals []wlFakeGlobal, formats []uint32) *wlFake {
	return &wlFake{
		globals:   globals,
		formats:   formats,
		issued:    make(map[uint32]bool),
		ping:      true,
		sequences: 1,
	}
}

// wlDefaultGlobals est ce qu'un compositeur ordinaire annonce de ce qui nous intéresse.
func wlDefaultGlobals() []wlFakeGlobal {
	return []wlFakeGlobal{
		{1, "wl_compositor", 6},
		{2, "wl_shm", 2},
		{3, "wl_output", 4},
		{4, "xdg_wm_base", 7},
	}
}

// note retient une requête reconnue, pour qu'un cas lise ce que le compositeur a vu.
func (f *wlFake) note(format string, a ...any) {
	f.seen = append(f.seen, fmt.Sprintf(format, a...))
}

// refuse prononce une erreur de protocole et la retient.
//
// L'objet nommé est celui que le protocole désigne pour ce code, et non celui qui a émis
// la requête : la description attache `unconfigured_buffer` à `xdg_surface` alors que
// c'est `wl_surface.attach` qui le lève.
func (f *wlFake) refuse(out *wlWriter, object, code uint32, reason string) {
	f.refused = reason
	out.begin(wlDisplayID, wlDisplayError)
	out.uint(object)
	out.uint(code)
	out.string(reason)
	out.end()
}

// serial délivre un numéro de série, comme un compositeur le fait pour tout ce qui
// demande un accusé.
func (f *wlFake) serial() uint32 {
	f.nextSerial++
	return f.nextSerial
}

// saw rend ce que le compositeur a reconnu, dans l'ordre.
func (f *wlFake) saw() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.seen...)
}

// refusal rend le refus prononcé, ou la chaîne vide quand rien n'a été refusé.
func (f *wlFake) refusal() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refused
}

// handle traite une requête et compose ce qu'elle appelle en réponse.
//
// Une requête qu'un refus précède est ignorée : la connexion est morte du point de vue du
// protocole, et répondre à ce qui suit masquerait la cause.
func (f *wlFake) handle(sender uint32, opcode uint16, r *wlReader, out *wlWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.refused != "" {
		return
	}

	switch {
	case sender == wlDisplayID && opcode == wlDisplayGetRegistry:
		f.registry = r.uint()
		for _, g := range f.globals {
			out.begin(f.registry, wlRegistryGlobal)
			out.uint(g.name)
			out.string(g.iface)
			out.uint(g.version)
			out.end()
		}

	case sender == wlDisplayID && opcode == wlDisplaySync:
		out.begin(r.uint(), wlCallbackDone)
		out.uint(0)
		out.end()

	case sender == f.registry && opcode == wlRegistryBind:
		r.uint()
		iface, _, id := r.string(), r.uint(), r.uint()
		switch iface {
		case "wl_compositor":
			f.compositor = id
		case "wl_shm":
			f.shm = id
			for _, format := range f.formats {
				out.begin(id, wlShmFormat)
				out.uint(format)
				out.end()
			}
		case "xdg_wm_base":
			f.wmBase = id
		}
		f.note("bind %s", iface)

	case sender == f.compositor && opcode == wlCompositorCreateSurface:
		f.surface = r.uint()
		f.note("create_surface")

	case sender == f.wmBase && opcode == wlXDGWMBaseGetXDGSurface:
		id, surface := r.uint(), r.uint()
		if surface != f.surface {
			f.refuse(out, surface, wlErrInvalidObject, "get_xdg_surface names a surface that does not exist")
			return
		}
		f.xdg = id
		f.note("get_xdg_surface")

	case sender == f.wmBase && opcode == wlXDGWMBasePong:
		serial := r.uint()
		if serial != f.pinged {
			f.refuse(out, f.wmBase, wlXDGErrUnresponsive,
				fmt.Sprintf("pong carries serial %d, and the ping sent %d", serial, f.pinged))
			return
		}
		f.pinged = 0
		f.note("pong %d", serial)

	case sender == f.xdg && opcode == wlXDGSurfaceGetToplevel:
		f.toplevel = r.uint()
		f.note("get_toplevel")

	case sender == f.xdg && opcode == wlXDGSurfaceAckConfigure:
		serial := r.uint()
		switch {
		case f.toplevel == 0:
			f.refuse(out, f.xdg, wlXDGErrNotConstructed,
				"ack_configure came before the surface was given a role")
		case !f.issued[serial]:
			f.refuse(out, f.xdg, wlXDGErrInvalidSerial,
				fmt.Sprintf("serial %d was never sent", serial))
		case f.pinged != 0:
			f.refuse(out, f.wmBase, wlXDGErrUnresponsive,
				fmt.Sprintf("ping %d went unanswered", f.pinged))
		default:
			f.note("ack_configure %d", serial)
		}

	case sender == f.toplevel && opcode == wlXDGToplevelSetTitle:
		f.note("set_title %s", r.string())

	case sender == f.toplevel && opcode == wlXDGToplevelSetAppID:
		f.note("set_app_id %s", r.string())

	case sender == f.surface && opcode == wlSurfaceAttach:
		if !f.configured {
			f.refuse(out, f.xdg, wlXDGErrUnconfiguredBuffer,
				"a buffer was attached before the first configure")
			return
		}
		f.note("attach %d", r.uint())

	case sender == f.surface && opcode == wlSurfaceCommit:
		f.note("commit")
		f.commit(out)
	}
}

// commit applique ce qu'un commit rend effectif, et répond ce que le protocole impose.
//
// **Le premier commit est le seul qui déclenche une configuration**, et la description le
// veut ainsi : « the client must perform an initial commit without any buffer attached.
// The compositor will reply with initial wl_surface state […] followed by an
// xdg_surface.configure event ». Un commit avant que le rôle soit donné ne déclenche rien,
// l'état du rôle faisant partie de ce que la configuration transporte.
func (f *wlFake) commit(out *wlWriter) {
	if f.toplevel == 0 || f.configured {
		return
	}

	if f.ping {
		f.pinged = f.serial()
		out.begin(f.wmBase, wlXDGWMBasePing)
		out.uint(f.pinged)
		out.end()
	}

	for i := 0; i < f.sequences; i++ {
		f.configure(out)
	}
	f.configured = true

	if f.closing {
		out.begin(f.toplevel, wlXDGToplevelClose)
		out.end()
	}
}

// configure émet une séquence de configuration : l'état du rôle, puis sa clôture.
//
// Les deux événements portent l'opcode zéro et ne viennent pas du même objet, ce qui est
// le piège que le routage du client doit éviter. Le tableau des états est émis vide — une
// longueur nulle et rien derrière —, ce qui est le cas d'une fenêtre ordinaire.
func (f *wlFake) configure(out *wlWriter) {
	out.begin(f.toplevel, wlXDGToplevelConfigure)
	out.uint(uint32(f.proposeW))
	out.uint(uint32(f.proposeH))
	out.uint(0)
	out.end()

	serial := f.serial()
	f.issued[serial] = true

	out.begin(f.xdg, wlXDGSurfaceConfigure)
	out.uint(serial)
	out.end()
}

// serve met le compositeur de pacotille au bout d'une socket.
func (f *wlFake) serve(t *testing.T, sock *os.File) {
	t.Helper()

	go func() {
		in := make([]byte, 4096)
		have := 0

		for {
			n, err := sock.Read(in[have:])
			if err != nil || n == 0 {
				return
			}
			have += n

			for have >= wlHeaderSize {
				second := wlOrder.Uint32(in[4:])
				size := int(second >> 16)
				if size < wlHeaderSize || have < size {
					break
				}
				sender, opcode := wlOrder.Uint32(in), uint16(second&0xffff)
				r := wlReader{buf: in[wlHeaderSize:size]}

				var out wlWriter
				f.handle(sender, opcode, &r, &out)
				if len(out.buf) > 0 {
					if _, err := sock.Write(out.buf); err != nil {
						return
					}
				}

				copy(in, in[size:have])
				have -= size
			}
		}
	}()
}

// wlFakeConn monte une connexion servie par le compositeur donné.
func wlFakeConn(t *testing.T, f *wlFake) *wlConn {
	t.Helper()

	c, server := wlConnPair(t)
	f.serve(t, server)
	return c
}

// wlOpenPair monte une connexion servie par un compositeur de pacotille ordinaire.
//
// Le sondage par `ping` y est retiré : les cas du registre s'arrêtent avant qu'une surface
// existe, donc avant le commit qui le déclenche, et le laisser n'y changerait rien que le
// lecteur aurait à écarter.
func wlOpenPair(t *testing.T, globals []wlFakeGlobal, formats []uint32) *wlConn {
	t.Helper()

	f := wlNewFake(globals, formats)
	f.ping = false
	return wlFakeConn(t, f)
}

// wlFeed remet une requête au compositeur sans passer par une socket.
//
// C'est ce qui permet d'éprouver un refus sans écrire un client de travers : la requête
// fautive s'exprime en trois valeurs, là qu'un client qui omet un geste serait une seconde
// écriture du montage et dériverait de la première.
func wlFeed(f *wlFake, sender uint32, opcode uint16, args ...uint32) {
	buf := make([]byte, 4*len(args))
	for i, a := range args {
		wlOrder.PutUint32(buf[4*i:], a)
	}

	r := wlReader{buf: buf}
	var out wlWriter
	f.handle(sender, opcode, &r, &out)
}

// wlFakeMounted rend un compositeur dont les objets d'ouverture sont déjà nommés, le rôle
// excepté.
//
// Les identifiants sont arbitraires et distincts, ce qui est tout ce que le routage leur
// demande. Rejouer la liaison pour éprouver un jugement de surface n'apprendrait rien de
// plus et rendrait chaque cas dix lignes plus long.
func wlFakeMounted() *wlFake {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	f.registry, f.compositor, f.shm, f.wmBase = 10, 11, 12, 13
	f.surface, f.xdg = 20, 21
	return f
}
