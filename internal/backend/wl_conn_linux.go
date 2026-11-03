// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La connexion au compositeur : où le joindre, comment lire son flux sans perdre un
// descripteur, et comment nommer ce qui l'a fait tomber.
//
// La différence de fond avec X11 n'est pas le protocole, c'est l'ancillaire. Un message
// Wayland peut porter des descripteurs de fichier dans les données auxiliaires de la
// socket, et la spécification avertit que « any byte of the stream, even the message
// header, may carry the ancillary data with file descriptors » : il n'y a donc aucune
// lecture qu'on puisse faire sans tampon de contrôle, pas même celle d'un en-tête. Une
// lecture ordinaire ferait fermer les descripteurs par le noyau, sans erreur et sans
// trace.

package backend

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// wlFDsPerRead est le nombre de descripteurs qu'une lecture peut recevoir d'un coup.
//
// Le protocole de base n'en passe qu'un à la fois — la carte du clavier —, mais rien n'y
// oblige le compositeur, et un tampon trop court est le seul mode de panne irréparable de
// ce fichier : le noyau ferme les descripteurs en excès et pose `MSG_CTRUNC`, donc on sait
// qu'on a perdu quelque chose sans pouvoir le récupérer. Huit est large et coûte quelques
// dizaines d'octets une fois pour toutes.
const wlFDsPerRead = 8

// wlDefaultDisplay est le nom de socket qu'un client emploie quand rien ne le nomme.
const wlDefaultDisplay = "wayland-0"

// wlHandshake borne l'ouverture, de la liaison du registre à la configuration de la
// fenêtre.
//
// **Elle borne la phase entière et non chaque lecture**, ce qui est la seule façon de
// borner quelque chose : un compositeur qui livrerait un octet toutes les secondes ne
// déclencherait jamais une échéance par lecture, et l'ouverture n'avancerait pas pour
// autant. Toute l'ouverture est de la communication entre deux processus de la même
// machine et prend ordinairement moins d'une milliseconde ; dix secondes sont donc quatre
// ordres de grandeur de marge, et bornent quand même un compositeur en panne.
const wlHandshake = 10 * time.Second

// wlConn est une connexion ouverte à un compositeur.
//
// Elle porte les deux chemins de lecture que les deux régimes du backend demandent, et
// c'est la structure d'X11 : une lecture **bloquante** pendant l'ouverture, où il n'y a
// rien d'autre à faire qu'attendre la réponse, et une lecture **qui rend la main** dans la
// boucle de jeu, où un `Input` qui attendrait figerait le jeu. Les deux passent par le même
// tampon et la même file de descripteurs, donc un descripteur reçu pendant l'ouverture
// reste disponible ensuite.
type wlConn struct {
	sock *net.UnixConn

	// raw et read servent la lecture qui rend la main. La valeur de fonction est
	// montée une fois : la passer à `raw.Read` sur une méthode produirait une valeur
	// de méthode, donc une allocation par appel sur le chemin d'une image.
	raw  syscall.RawConn
	read func(uintptr) bool

	readN, readOOB, readFlags int
	readErrno                 error

	// oob est le tampon de contrôle, réutilisé d'une lecture à l'autre.
	oob []byte

	out wlWriter

	// in porte les octets reçus et pas encore découpés, et have leur nombre. Un
	// message peut arriver en plusieurs morceaux, donc ce qui reste survit à la
	// lecture suivante.
	in   []byte
	have int

	// fds est la file des descripteurs reçus, dans l'ordre d'arrivée.
	//
	// **Une file et non un champ, parce que rien ne lie un descripteur au message
	// qu'on lit** : il arrive avec un octet quelconque du flux, souvent avant le
	// message qui s'en sert. L'appelant le retire quand la description lui dit qu'un
	// argument de type `fd` est là, et l'ordre fait le reste.
	fds []int

	// handler reçoit les messages qui ne sont pas de `wl_display`. Il est posé par le
	// niveau qui connaît les objets ; la connexion, elle, ne sait router que ce qui la
	// concerne.
	handler func(sender uint32, opcode uint16, r *wlReader) error

	// nextID est le prochain identifiant jamais employé, et freedIDs ceux que
	// `delete_id` a rendus. La spécification exige une allocation dense, donc les
	// neufs se prennent en croissant et les rendus se réemploient en premier.
	nextID   uint32
	freedIDs []uint32

	err error
}

// wlDisplayPath rend le chemin de la socket à joindre, d'après l'environnement.
//
// Trois régimes, dans l'ordre que la documentation de référence donne. `WAYLAND_DISPLAY`
// absolu est employé tel quel — c'est permis depuis Wayland 1.15 —, un nom relatif se
// résout dans `XDG_RUNTIME_DIR`, et son absence vaut `wayland-0`, qui est relatif comme le
// reste.
//
// **Le repli sur `/dev/shm` puis le répertoire temporaire ne vaut pas ici**, contrairement
// à la mémoire partagée : une socket est posée par le compositeur, pas par nous, donc la
// chercher ailleurs ne la trouverait pas. L'erreur nomme donc la variable manquante, qui
// est ce qu'un utilisateur peut corriger.
func wlDisplayPath() (string, error) {
	name := os.Getenv("WAYLAND_DISPLAY")
	if name == "" {
		name = wlDefaultDisplay
	}
	if filepath.IsAbs(name) {
		return name, nil
	}

	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return "", errors.New("XDG_RUNTIME_DIR is not set, and WAYLAND_DISPLAY is not an absolute path: there is no directory to look for the compositor's socket in")
	}
	return filepath.Join(dir, name), nil
}

// wlDial ouvre la socket du compositeur, ou adopte celle qu'on nous a passée.
//
// `WAYLAND_SOCKET` prime et porte un descripteur déjà connecté : c'est ainsi qu'un
// compositeur lance un client sans lui faire chercher sa socket. La variable est retirée
// de l'environnement après lecture, comme la documentation de référence le prescrit, pour
// qu'un processus enfant n'hérite pas d'un descripteur qui n'est pas le sien.
func wlDial() (*net.UnixConn, error) {
	if s := os.Getenv("WAYLAND_SOCKET"); s != "" {
		os.Unsetenv("WAYLAND_SOCKET")

		fd, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("WAYLAND_SOCKET is %q, which is not a file descriptor number: %w", s, err)
		}
		return wlAdopt(fd)
	}

	path, err := wlDisplayPath()
	if err != nil {
		return nil, err
	}
	sock, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("connecting to the compositor on %s: %w", path, err)
	}
	return sock, nil
}

// wlAdopt prend un descripteur déjà connecté et en fait une socket du paquet `net`.
//
// `net.FileConn` duplique le descripteur, donc l'original se rend aussitôt : le garder
// ouvert laisserait la connexion en vie après un `Close`, et le compositeur ne verrait
// jamais le client partir.
func wlAdopt(fd int) (*net.UnixConn, error) {
	f := os.NewFile(uintptr(fd), "wayland")
	if f == nil {
		return nil, fmt.Errorf("WAYLAND_SOCKET names descriptor %d, which is not open", fd)
	}
	defer f.Close()

	conn, err := net.FileConn(f)
	if err != nil {
		return nil, fmt.Errorf("adopting the compositor socket on descriptor %d: %w", fd, err)
	}
	sock, ok := conn.(*net.UnixConn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("descriptor %d is a %T and not a Unix socket", fd, conn)
	}
	return sock, nil
}

// wlConnect ouvre une connexion au compositeur et la prépare à parler.
//
// Elle ne fait aucun aller-retour : ce que le client apprend du compositeur commence au
// registre, que le lot suivant lie. Ce qui est monté ici est ce qu'un message demande —
// l'accès brut pour la lecture qui rend la main, les tampons, et le premier identifiant
// libre.
func wlConnect() (*wlConn, error) {
	sock, err := wlDial()
	if err != nil {
		return nil, err
	}

	raw, err := sock.SyscallConn()
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("reaching the compositor socket directly: %w", err)
	}

	return &wlConn{
		sock:   sock,
		raw:    raw,
		oob:    make([]byte, syscall.CmsgSpace(4*wlFDsPerRead)),
		in:     make([]byte, 0, 4096),
		nextID: wlFirstID,
	}, nil
}

// Close rend au système ce que la connexion a pris. Appeler deux fois est sans effet.
//
// Les descripteurs que le compositeur a envoyés et que personne n'a consommés se ferment
// ici : ils sont à nous depuis que le noyau les a installés, et les laisser fuirait un
// descripteur par carte de clavier reçue.
func (c *wlConn) Close() {
	for _, fd := range c.fds {
		syscall.Close(fd)
	}
	c.fds = nil

	if c.sock != nil {
		c.sock.Close()
		c.sock = nil
	}
}

// newID prend un identifiant pour un objet que le client crée.
//
// Les identifiants rendus par `wl_display.delete_id` passent avant les neufs. Ce n'est pas
// une économie : c'est ce qui empêche la table que le **compositeur** tient pour nous de
// croître sans borne, puisqu'il garde une entrée par identifiant jamais libéré. Un jeu qui
// demande une image d'animation par tour en consommerait soixante par seconde.
//
// L'épuisement est un refus et non un débordement : la plage haute appartient au serveur,
// donc y entrer ferait parler le client au nom d'objets qui ne sont pas les siens.
func (c *wlConn) newID() (uint32, error) {
	if n := len(c.freedIDs); n > 0 {
		id := c.freedIDs[n-1]
		c.freedIDs = c.freedIDs[:n-1]
		return id, nil
	}
	if c.nextID > wlLastID {
		return 0, fmt.Errorf("the client has used all %d object IDs the protocol reserves for it", wlLastID-wlFirstID+1)
	}
	id := c.nextID
	c.nextID++
	return id, nil
}

// bound arme une échéance de lecture, et la retire quand d vaut zéro.
//
// **Le retrait est la moitié qui compte, et c'est un piège plutôt qu'une politesse.** Une
// échéance laissée en place ne gêne pas [wlConn.receive] aujourd'hui, qui ne se met jamais
// en attente — elle lit ce qui est là et rend `EAGAIN` sinon. Mais c'est une propriété de
// l'implémentation et non du contrat : le jour où cette lecture attendrait la lisibilité,
// l'échéance ferait tomber la fenêtre dix secondes après son ouverture, sur un chemin qui
// n'a rien à voir avec elle.
func (c *wlConn) bound(d time.Duration) error {
	if d == 0 {
		return c.sock.SetReadDeadline(time.Time{})
	}
	return c.sock.SetReadDeadline(time.Now().Add(d))
}

// takeFD retire le plus ancien descripteur reçu.
//
// L'appelant le réclame quand la description de l'événement qu'il lit annonce un argument
// de type `fd`. Son absence n'est pas un cas à rattraper : elle veut dire que le flux et la
// file ne sont plus en correspondance, donc que quelque chose a été lu de travers.
func (c *wlConn) takeFD() (int, bool) {
	if len(c.fds) == 0 {
		return -1, false
	}
	fd := c.fds[0]
	c.fds = c.fds[1:]
	return fd, true
}

// flush envoie ce que le tampon de sortie porte, et le vide.
//
// Un message incomplet est refusé plutôt qu'envoyé : sa taille vaudrait zéro, et le
// compositeur lirait tout ce qui suit comme ses arguments.
func (c *wlConn) flush() error {
	if c.out.writing {
		return errors.New("flushing the compositor socket while a message is still being written")
	}
	if len(c.out.buf) == 0 {
		return nil
	}

	if _, err := c.sock.Write(c.out.buf); err != nil {
		return fmt.Errorf("writing to the compositor: %w", err)
	}
	c.out.reset()
	return nil
}

// room garantit que le tampon d'entrée peut accueillir n octets de plus.
func (c *wlConn) room(n int) {
	if cap(c.in)-c.have >= n {
		c.in = c.in[:c.have+n]
		return
	}
	grown := make([]byte, c.have+n, 2*(c.have+n))
	copy(grown, c.in[:c.have])
	c.in = grown
}

// fill lit en bloquant, et c'est le régime de l'ouverture.
//
// Elle passe par `ReadMsgUnix` et non par une lecture ordinaire, pour la raison que la
// tête du fichier donne : un descripteur peut accompagner n'importe quel octet, y compris
// pendant l'ouverture.
func (c *wlConn) fill() error {
	c.room(4096)

	n, oobn, flags, _, err := c.sock.ReadMsgUnix(c.in[c.have:], c.oob)
	if err != nil {
		return fmt.Errorf("reading from the compositor: %w", err)
	}
	if n == 0 {
		return errors.New("the compositor closed the connection")
	}
	c.have += n
	return c.collect(oobn, flags)
}

// receive lit ce qui est disponible sans attendre, et dit si elle a lu quelque chose.
//
// **Elle rend la main plutôt que d'attendre**, parce que le contrat de fenêtre l'exige de
// la boucle de jeu : `Paced` rend faux pour cette famille, donc un `Input` qui bloquerait
// figerait le jeu sur un compositeur qui ne dit rien.
func (c *wlConn) receive() (bool, error) {
	c.room(4096)

	if c.read == nil {
		c.read = func(fd uintptr) bool {
			for {
				c.readN, c.readOOB, c.readFlags, _, c.readErrno =
					syscall.Recvmsg(int(fd), c.in[c.have:], c.oob, 0)
				// Un signal interrompt la lecture sans que rien n'ait été lu ni
				// perdu. Elle se reprend, ce que la couche réseau ferait aussi.
				if c.readErrno != syscall.EINTR {
					return true
				}
			}
		}
	}
	if err := c.raw.Read(c.read); err != nil {
		return false, fmt.Errorf("reading from the compositor: %w", err)
	}

	switch n, errno := c.readN, c.readErrno; {
	case errno == syscall.EAGAIN:
		return false, nil
	case errno != nil:
		return false, fmt.Errorf("reading from the compositor: %w", errno)
	case n == 0:
		// Une lecture qui rend zéro sans erreur est la fermeture du pair, et c'est le
		// régime que le contrat de fenêtre a prévu : un compositeur qui meurt n'est
		// pas un joueur qui ferme la fenêtre.
		return false, errors.New("the compositor closed the connection")
	default:
		c.have += n
		return true, c.collect(c.readOOB, c.readFlags)
	}
}

// collect range les descripteurs qu'une lecture a rapportés.
//
// **`MSG_CTRUNC` est le seul mode de panne irréparable de la connexion**, et c'est pour
// cela qu'il est traité comme une erreur fatale et non comme un avertissement : le noyau a
// déjà fermé les descripteurs qui ne tenaient pas dans le tampon de contrôle. Continuer
// voudrait dire attendre indéfiniment une carte de clavier ou un tampon que plus personne
// ne peut livrer.
func (c *wlConn) collect(oobn, flags int) error {
	if flags&syscall.MSG_CTRUNC != 0 {
		return fmt.Errorf("the compositor sent more than %d file descriptors at once, and the kernel closed the ones that did not fit", wlFDsPerRead)
	}
	if oobn == 0 {
		return nil
	}

	msgs, err := syscall.ParseSocketControlMessage(c.oob[:oobn])
	if err != nil {
		return fmt.Errorf("reading the ancillary data of a compositor message: %w", err)
	}
	for _, m := range msgs {
		fds, err := syscall.ParseUnixRights(&m)
		if err != nil {
			// Un message de contrôle qui n'est pas un passage de descripteurs
			// n'est pas une erreur : le noyau en pose d'autres, et ils ne nous
			// concernent pas.
			continue
		}
		c.fds = append(c.fds, fds...)
	}
	return nil
}

// dispatch découpe ce que le tampon porte et route chaque message complet.
//
// Un message est gardé entier : sa taille vit dans son en-tête, donc un message à cheval
// sur deux lectures attend la suivante sans qu'on ait à connaître ses arguments. C'est ce
// qui permet de router sans rien savoir des interfaces.
func (c *wlConn) dispatch() error {
	for {
		if c.have < wlHeaderSize {
			return nil
		}
		second := wlOrder.Uint32(c.in[4:])
		size := int(second >> 16)

		switch {
		case size < wlHeaderSize:
			return fmt.Errorf("the compositor sent a message of %d bytes, below the %d-byte header", size, wlHeaderSize)
		case c.have < size:
			return nil
		}

		sender, opcode := wlOrder.Uint32(c.in), uint16(second&0xffff)
		r := wlReader{buf: c.in[wlHeaderSize:size]}

		if err := c.route(sender, opcode, &r); err != nil {
			return err
		}
		if r.err != nil {
			return r.err
		}

		copy(c.in, c.in[size:c.have])
		c.have -= size
	}
}

// route remet un message à qui le traite.
//
// `wl_display` est traité ici, et c'est le seul objet dans ce cas : ses deux événements
// portent sur la connexion elle-même et non sur ce qu'un jeu voit. Le reste va au
// gestionnaire que le niveau au-dessus a posé, et son absence est une erreur de montage
// plutôt qu'un message à jeter en silence.
func (c *wlConn) route(sender uint32, opcode uint16, r *wlReader) error {
	if sender == wlDisplayID {
		return c.display(opcode, r)
	}
	if c.handler == nil {
		return fmt.Errorf("object %d sent event %d, and no handler is set to receive it", sender, opcode)
	}
	return c.handler(sender, opcode, r)
}

// display traite les deux événements de `wl_display`.
//
// L'erreur est fatale par définition — la description la nomme « fatal error event » —,
// donc elle est retenue dans la connexion et rendue à la boucle, qui s'arrêtera dessus.
// Le message du compositeur est repris tel quel : c'est lui qui dit ce qui n'a pas plu, et
// le reformuler perdrait la seule information utile.
func (c *wlConn) display(opcode uint16, r *wlReader) error {
	switch opcode {
	case wlDisplayError:
		object, code, message := r.uint(), r.uint(), r.string()
		if r.err != nil {
			return r.err
		}
		c.err = fmt.Errorf("the compositor refused object %d with %s: %s",
			object, wlErrorName(code), message)
		return c.err

	case wlDisplayDeleteID:
		id := r.uint()
		if r.err != nil {
			return r.err
		}
		c.freedIDs = append(c.freedIDs, id)
		return nil

	default:
		// Une version ultérieure de l'interface pourrait en ajouter un. On ne l'a pas
		// demandée, donc le recevoir est une erreur du compositeur et non un cas à
		// ignorer.
		return fmt.Errorf("wl_display sent event %d, which version 1 does not define", opcode)
	}
}

// roundtrip demande un aller-retour au compositeur et attend sa réponse.
//
// C'est la seule façon de savoir que tout ce qu'on a envoyé a été traité : le protocole
// est asynchrone, donc une requête qui n'a pas de réponse n'en donne aucune nouvelle.
// `wl_display.sync` crée un rappel que le compositeur détruit en répondant.
//
// **Elle bloque, et c'est le régime de l'ouverture uniquement.** La boucle de jeu n'appelle
// jamais ceci : elle lit ce qui est là et avance. Un compositeur muet y bloquerait donc le
// montage de la fenêtre et non le jeu — ce que X11 fait aussi pour sa poignée de main, et
// ce qui reste à borner pour les deux.
func (c *wlConn) roundtrip() error {
	id, err := c.newID()
	if err != nil {
		return err
	}

	c.out.begin(wlDisplayID, wlDisplaySync)
	c.out.uint(id)
	c.out.end()
	if err := c.flush(); err != nil {
		return err
	}

	done := false
	previous := c.handler
	c.handler = func(sender uint32, opcode uint16, r *wlReader) error {
		if sender == id && opcode == wlCallbackDone {
			r.uint()
			done = true
			return nil
		}
		if previous == nil {
			return nil
		}
		return previous(sender, opcode, r)
	}
	defer func() { c.handler = previous }()

	for !done {
		if err := c.fill(); err != nil {
			return fmt.Errorf("waiting for the compositor to answer a sync: %w", err)
		}
		if err := c.dispatch(); err != nil {
			return err
		}
	}
	return nil
}
