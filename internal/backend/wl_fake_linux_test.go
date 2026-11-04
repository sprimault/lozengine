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
	"net"
	"sync"
	"syscall"
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

	// seat est le siège lié, keyboard et pointer les objets d'entrée que le client en
	// a tirés, et caps ce que le siège annonce au moment du `bind`.
	//
	// Les capacités partent là et non plus tard parce que c'est ce qu'un compositeur
	// fait : l'événement suit immédiatement la liaison, et c'est précisément ce que le
	// client doit écouter assez tôt pour l'entendre.
	seat     uint32
	keyboard uint32
	pointer  uint32
	caps     uint32

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

	// pool est l'objet `wl_shm_pool` en cours, mem la projection que le compositeur a
	// faite du descripteur reçu, et buffers les tampons qu'on y a créés.
	//
	// **Projeter la mémoire est ce qui donne sa valeur au compositeur de pacotille sur
	// ce chemin** : un cas peut alors relire les octets exactement comme un vrai
	// compositeur les lirait, au lieu de croire une requête bien formée.
	pool    uint32
	mem     []byte
	buffers map[uint32]wlFakeBuffer

	// fds est la file des descripteurs reçus, dans l'ordre d'arrivée, comme du côté
	// client : rien ne lie un descripteur au message qu'on lit.
	fds []int

	// attached est le dernier tampon commis, et releaseOnCommit fait rendre le tampon
	// aussitôt — ce qui est le régime courant d'un compositeur qui recopie la surface,
	// et celui que la description appelle « typically ».
	attached        uint32
	releaseOnCommit bool

	// sock est la socket du côté serveur, par laquelle un cas pousse un événement que
	// nulle requête ne provoque — une touche enfoncée, un cran de molette.
	sock *net.UnixConn

	// seen porte les requêtes reconnues dans l'ordre, et refused le refus prononcé.
	// Il n'y en a qu'un : une erreur de protocole est fatale, donc rien ne suit.
	seen    []string
	refused string
}

// wlFakeBuffer est ce qu'un `wl_buffer` porte, tel que le compositeur l'a reçu.
type wlFakeBuffer struct {
	at                    int
	width, height, stride int
	format                uint32
}

// wlNewFake monte un compositeur de pacotille dans son régime ordinaire.
func wlNewFake(globals []wlFakeGlobal, formats []uint32) *wlFake {
	return &wlFake{
		globals:         globals,
		formats:         formats,
		issued:          make(map[uint32]bool),
		buffers:         make(map[uint32]wlFakeBuffer),
		ping:            true,
		sequences:       1,
		releaseOnCommit: true,
		caps:            wlSeatPointer | wlSeatKeyboard,
	}
}

// wlDefaultGlobals est ce qu'un compositeur ordinaire annonce de ce qui nous intéresse.
func wlDefaultGlobals() []wlFakeGlobal {
	return []wlFakeGlobal{
		{1, "wl_compositor", 6},
		{2, "wl_shm", 2},
		{3, "wl_output", 4},
		{4, "xdg_wm_base", 7},
		{5, "wl_seat", 9},
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
		case "wl_seat":
			f.seat = id
			out.begin(id, wlSeatCapabilities)
			out.uint(f.caps)
			out.end()
		case "xdg_wm_base":
			f.wmBase = id
		}
		f.note("bind %s", iface)

	case sender == f.seat && opcode == wlSeatGetPointer:
		f.pointer = r.uint()
		f.note("get_pointer")

	case sender == f.seat && opcode == wlSeatGetKeyboard:
		f.keyboard = r.uint()
		f.note("get_keyboard")

	// **Le routage par émetteur n'est pas une élégance ici, il est obligatoire** :
	// `wl_pointer.set_cursor` et `wl_keyboard.release` portent tous deux l'opcode zéro,
	// et `wl_pointer.release` le même un que `wl_seat.get_keyboard`.
	case sender == f.pointer && opcode == wlPointerSetCursor:
		serial, surface := r.uint(), r.uint()
		if !f.issued[serial] {
			// **Le protocole ne refuse pas cela, et c'est pourquoi ce compositeur
			// le refuse.** Il dit qu'une requête au numéro périmé est ignorée, ce
			// qui fait reparaître le curseur sans la moindre erreur — le seul mode
			// de panne du lot que rien d'autre n'attraperait.
			f.refuse(out, f.pointer, wlErrInvalidObject,
				fmt.Sprintf("set_cursor carries serial %d, which no enter event sent", serial))
			return
		}
		f.note("set_cursor %d surface %d", serial, surface)

	case sender == f.pointer && opcode == wlPointerRelease:
		f.pointer = 0
		f.note("release_pointer")

	case sender == f.keyboard && opcode == wlKeyboardRelease:
		f.keyboard = 0
		f.note("release_keyboard")

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

	case sender == f.shm && opcode == wlShmCreatePool:
		id, size := r.uint(), r.uint()
		if len(f.fds) == 0 {
			f.refuse(out, id, wlErrInvalidObject, "create_pool came without a file descriptor")
			return
		}
		fd := f.fds[0]
		f.fds = f.fds[1:]

		mem, err := syscall.Mmap(fd, 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
		syscall.Close(fd)
		if err != nil {
			f.refuse(out, id, wlErrInvalidObject,
				fmt.Sprintf("mapping the %d bytes the client shared: %v", size, err))
			return
		}
		f.unmap()
		f.pool, f.mem = id, mem
		f.note("create_pool %d", size)

	case sender == f.pool && opcode == wlShmPoolCreateBuffer:
		id, at := r.uint(), r.uint()
		width, height, stride, format := r.int(), r.int(), r.int(), r.uint()
		if !f.offers(format) {
			f.refuse(out, id, wlErrInvalidObject,
				fmt.Sprintf("format %#x was never advertised", format))
			return
		}
		f.buffers[id] = wlFakeBuffer{
			at: int(at), width: int(width), height: int(height),
			stride: int(stride), format: format,
		}
		f.note("create_buffer %d×%d at %d", width, height, at)

	case sender == f.pool && opcode == wlShmPoolDestroy:
		f.note("destroy_pool")

	case opcode == wlBufferDestroy && f.isBuffer(sender):
		delete(f.buffers, sender)
		f.note("destroy_buffer")

	case sender == f.surface && opcode == wlSurfaceAttach:
		id := r.uint()
		switch {
		case !f.configured:
			f.refuse(out, f.xdg, wlXDGErrUnconfiguredBuffer,
				"a buffer was attached before the first configure")
		case id != 0 && !f.isBuffer(id):
			f.refuse(out, id, wlErrInvalidObject, "attach names an object that is not a buffer")
		default:
			f.attached = id
			f.note("attach %d", id)
		}

	case sender == f.surface && opcode == wlSurfaceDamage:
		x, y, width, height := r.int(), r.int(), r.int(), r.int()
		f.note("damage %d,%d %d×%d", x, y, width, height)

	case sender == f.surface && opcode == wlSurfaceCommit:
		f.note("commit")
		f.commit(out)
	}
}

// offers dit si le format a bien été annoncé, ce que `create_buffer` exige.
func (f *wlFake) offers(format uint32) bool {
	for _, announced := range f.formats {
		if announced == format {
			return true
		}
	}
	return false
}

// isBuffer dit si l'identifiant est celui d'un tampon vivant.
func (f *wlFake) isBuffer(id uint32) bool {
	_, ok := f.buffers[id]
	return ok
}

// unmap rend la projection de la mémoire partagée, s'il y en a une.
func (f *wlFake) unmap() {
	if f.mem != nil {
		syscall.Munmap(f.mem)
		f.mem = nil
	}
}

// shared relit la mémoire partagée comme un vrai compositeur la lirait.
//
// La copie est voulue : la tranche vit sous le verrou, et la rendre telle quelle laisserait
// un cas lire pendant qu'une requête écrit.
func (f *wlFake) shared(at, n int) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	if at+n > len(f.mem) {
		return nil
	}
	return append([]byte(nil), f.mem[at:at+n]...)
}

// bufferOf rend ce que le compositeur a retenu d'un tampon.
func (f *wlFake) bufferOf(id uint32) (wlFakeBuffer, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	b, ok := f.buffers[id]
	return b, ok
}

// commit applique ce qu'un commit rend effectif, et répond ce que le protocole impose.
//
// **Le premier commit est le seul qui déclenche une configuration**, et la description le
// veut ainsi : « the client must perform an initial commit without any buffer attached.
// The compositor will reply with initial wl_surface state […] followed by an
// xdg_surface.configure event ». Un commit avant que le rôle soit donné ne déclenche rien,
// l'état du rôle faisant partie de ce que la configuration transporte.
func (f *wlFake) commit(out *wlWriter) {
	// **Le renvoi du tampon vient avant la sortie anticipée, et l'ordre inverse a été
	// écrit d'abord** : la configuration ne part qu'au premier commit, donc un renvoi
	// posé après cette sortie ne partait jamais. Les deux tampons du client restaient
	// alors détenus pour toujours, et le troisième tour sautait sa présentation — ce que
	// le cas de la rotation a attrapé.
	if f.attached != 0 && f.releaseOnCommit {
		out.begin(f.attached, wlBufferRelease)
		out.end()
		f.attached = 0
	}

	// Le reste n'a lieu qu'une fois : la description veut que la configuration suive le
	// premier commit, celui qui n'attache rien.
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

// collect range les descripteurs qu'une lecture a rapportés.
func (f *wlFake) collect(oob []byte) {
	if len(oob) == 0 {
		return
	}
	msgs, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range msgs {
		if fds, err := syscall.ParseUnixRights(&m); err == nil {
			f.fds = append(f.fds, fds...)
		}
	}
}

// serve met le compositeur de pacotille au bout d'une socket.
//
// **La lecture emporte les données auxiliaires**, pour la raison qui vaut du côté client :
// un descripteur accompagne un octet quelconque du flux, et `wl_shm.create_pool` en envoie
// un. Le lire autrement le perdrait sans rien signaler, et le pool serait refusé pour une
// cause qui n'a rien à voir.
//
// **Elle passe par la connexion et jamais par le descripteur entier**, et c'est ce qui fait
// qu'elle se termine : la fermeture que le cas enregistre interrompt une lecture faite par
// `net`, là qu'un `recvmsg` bloquant ne rendrait jamais la main. Une goroutine survivante
// resterait sur un numéro de descripteur que le noyau réattribue, et volerait ses requêtes
// au cas suivant — mesuré en intégration continue, un montage sur six expirait au bout de
// la borne d'ouverture, et le cas qui tombait n'était pas celui qui avait le défaut.
func (f *wlFake) serve(tb testing.TB, sock *net.UnixConn) {
	tb.Helper()

	oob := make([]byte, syscall.CmsgSpace(4*wlFDsPerRead))
	tb.Cleanup(f.unmap)
	f.sock = sock

	go func() {
		in := make([]byte, 4096)
		have := 0

		for {
			n, oobn, _, _, err := sock.ReadMsgUnix(in[have:], oob)
			if err != nil || n == 0 {
				return
			}
			have += n
			f.collect(oob[:oobn])

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

// emit pousse vers le client des événements que nulle requête ne provoque.
//
// **C'est le seul chemin par lequel une entrée arrive**, et il n'a pas d'équivalent dans
// le reste du compositeur de pacotille : une configuration ou un format répondent à une
// requête, un appui sur une touche ne répond à rien. Les messages partent d'un bloc pour
// qu'une trame de défilement arrive entière, ce qui est ce que le client doit savoir
// traiter.
//
// Elle s'appelle entre deux pas et jamais pendant : le compositeur n'écrit que lorsqu'il
// reçoit, donc rien n'entrelace cette écriture avec les siennes tant qu'un cas ne lit pas
// en même temps qu'il émet.
func (f *wlFake) emit(t *testing.T, messages ...[]byte) {
	t.Helper()

	var out []byte
	for _, m := range messages {
		out = append(out, m...)
	}
	if _, err := f.sock.Write(out); err != nil {
		t.Fatalf("émission de %d octets vers le client : %v", len(out), err)
	}
}

// emitFD pousse un événement accompagné d'un descripteur de fichier.
//
// C'est la forme de `wl_keyboard.keymap`, le seul événement du protocole de base à en
// porter un. Le noyau en installe un duplicata chez le client, donc celui du cas reste
// ouvert et à lui.
func (f *wlFake) emitFD(t *testing.T, fd int, message []byte) {
	t.Helper()

	rights := syscall.UnixRights(fd)
	if _, _, err := f.sock.WriteMsgUnix(message, rights, nil); err != nil {
		t.Fatalf("émission d'un événement avec le descripteur %d : %v", fd, err)
	}
}

// entered enregistre le numéro de série d'un `enter` que le cas va émettre.
//
// Sans lui, le compositeur refuserait la requête de curseur qui le cite : il ne tient pour
// émis que ce qu'il a émis lui-même, et les entrées viennent du cas.
func (f *wlFake) entered(serial uint32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issued[serial] = true
}

// wlFakeConn monte une connexion servie par le compositeur donné.
//
// Elle prend un [testing.TB] et non un `*testing.T` parce qu'un banc monte le même
// compositeur : ce que la fonction lui demande — une aide de trace et un nettoyage — est
// dans l'interface commune, et restreindre au cas de test obligerait à écrire un second
// montage qui dériverait du premier.
func wlFakeConn(tb testing.TB, f *wlFake) *wlConn {
	tb.Helper()

	c, server := wlConnPair(tb)
	f.serve(tb, server)
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
	f.seat, f.keyboard, f.pointer = 30, 31, 32
	return f
}
