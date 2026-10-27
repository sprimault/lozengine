// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"io"
	"math/bits"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// connPair monte une connexion sur une paire de sockets et rend l'autre bout, par
// lequel un cas de test joue le serveur.
//
// Une vraie socket et non une fausse, et c'est ce qui donne sa valeur au harnais : la
// lecture sans attente repose sur un descripteur non bloquant qui rend `EAGAIN`, et
// aucune imitation en mémoire ne reproduirait ce régime — c'est précisément celui dont
// dépend une boucle de jeu qui ne doit jamais s'arrêter sur un événement absent.
func connPair(t *testing.T) (*conn, *net.UnixConn) {
	t.Helper()

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("paire de sockets : %v", err)
	}

	client := unixConnOf(t, fds[0])
	server := unixConnOf(t, fds[1])
	t.Cleanup(func() { server.Close() })

	raw, err := client.SyscallConn()
	if err != nil {
		t.Fatalf("prise du descripteur : %v", err)
	}

	// **Tout blocage est borné**, et c'est le harnais qui l'a appris : un cas qui
	// attend un octet qui ne viendra jamais doit échouer, pas durer. Sans cette
	// borne, une seule faute — un bourrage omis, donc un octet de moins qu'annoncé —
	// a figé un cas dix minutes puis emporté le binaire de test avant que sept
	// autres aient tourné, et le rapport ne portait qu'un dépassement de délai.
	//
	// Cinq secondes sont très au-delà de ce qu'une socket locale demande, et sans
	// effet sur la lecture qui n'attend pas : celle-ci passe par le descripteur et
	// non par la couche réseau, donc aucun délai ne la concerne.
	deadline := time.Now().Add(5 * time.Second)
	if err := client.SetDeadline(deadline); err != nil {
		t.Fatalf("délai sur le client : %v", err)
	}
	if err := server.SetDeadline(deadline); err != nil {
		t.Fatalf("délai sur le serveur : %v", err)
	}

	c := &conn{
		sock: client,
		raw:  raw,
		in:   make([]byte, 4096),
		setup: setup{
			root:         0x0000012C,
			visual:       0x00000021,
			depth:        24,
			bitsPerPixel: 32,
			imageOrder:   lsbFirst,
			redMask:      0xFF0000,
			greenMask:    0x00FF00,
			blueMask:     0x0000FF,
			idBase:       0x04000000,
			idMask:       0x001FFFFF,
			maxRequest:   0x40000,
		},
	}
	c.shift = bits.TrailingZeros32(c.setup.idMask)

	// Le visuel est celui d'un serveur courant, donc la conversion est l'échange des
	// octets extrêmes. Les cas qui éprouvent les autres dispositions posent la leur.
	pixels, err := pixelOrderOf(c.setup)
	if err != nil {
		t.Fatalf("visuel de test refusé : %v", err)
	}
	c.pixels = pixels
	t.Cleanup(c.Close)

	return c, server
}

// unixConnOf reprend un descripteur de socket en connexion Go.
func unixConnOf(t *testing.T, fd int) *net.UnixConn {
	t.Helper()

	// FileConn duplique le descripteur, donc celui-ci se rend aussitôt.
	f := os.NewFile(uintptr(fd), "socketpair")
	defer f.Close()

	nc, err := net.FileConn(f)
	if err != nil {
		t.Fatalf("reprise du descripteur : %v", err)
	}
	uc, ok := nc.(*net.UnixConn)
	if !ok {
		t.Fatalf("la paire de sockets a rendu un %T", nc)
	}
	return uc
}

// eventOf fabrique un message de trente-deux octets portant un code et un numéro de
// séquence.
func eventOf(code byte, seq uint16) []byte {
	m := make([]byte, eventSize)
	m[0] = code
	order.PutUint16(m[2:], seq)
	return m
}

// TestPollReturnsNothingWhenNothingWaits vérifie que la lecture d'une image ne
// s'arrête pas quand aucun événement n'est arrivé.
//
// C'est la propriété dont toute la cadence dépend : un joueur qui ne touche à rien
// n'envoie aucun événement, et une lecture qui attendrait figerait le jeu jusqu'au
// prochain appui. Rien ne le signalerait d'ailleurs comme un défaut de lecture — le
// jeu paraîtrait simplement ne pas tourner.
func TestPollReturnsNothingWhenNothingWaits(t *testing.T) {
	c, _ := connPair(t)

	m, err := c.poll()
	if err != nil {
		t.Fatalf("lecture en échec : %v", err)
	}
	if m != nil {
		t.Errorf("un message de %d octets rendu alors que rien n'a été envoyé", len(m))
	}
}

// TestPollReadsAnEventThatArrivesInPieces vérifie qu'un message coupé en route est
// rendu entier.
//
// Une socket ne promet pas de livrer d'un bloc ce qui a été écrit d'un bloc, et c'est
// la raison d'être du tampon de réception. Sans lui, un événement arrivé en deux
// morceaux serait lu comme deux moitiés : la première décodée avec les octets de la
// seconde, donc une taille ou un code pris au hasard, et aucune erreur pour le dire.
func TestPollReadsAnEventThatArrivesInPieces(t *testing.T) {
	c, server := connPair(t)

	m := eventOf(eventConfigureNotify, 7)
	order.PutUint16(m[20:], 640)
	order.PutUint16(m[22:], 360)

	if _, err := server.Write(m[:10]); err != nil {
		t.Fatalf("écriture du premier morceau : %v", err)
	}
	if got, err := c.poll(); err != nil || got != nil {
		t.Fatalf("un message incomplet a été rendu : %v, %v", got, err)
	}
	if _, err := server.Write(m[10:]); err != nil {
		t.Fatalf("écriture du reste : %v", err)
	}

	got, err := c.poll()
	if err != nil {
		t.Fatalf("lecture en échec : %v", err)
	}
	if got == nil {
		t.Fatal("aucun message rendu après l'arrivée du reste")
	}
	if len(got) != eventSize {
		t.Fatalf("message de %d octets, attendu %d", len(got), eventSize)
	}
	if width := order.Uint16(got[20:]); width != 640 {
		t.Errorf("largeur %d, attendu 640 — les deux morceaux n'ont pas été recollés", width)
	}
}

// TestPollReportsTheServerClosingTheConnection vérifie que la perte du serveur est
// rapportée comme une erreur.
//
// C'est le régime que le contrat de fenêtre a prévu en séparant la cause de la
// fermeture : un serveur d'affichage qui s'arrête n'est pas un joueur qui ferme la
// fenêtre, et les confondre ferait rapporter un arrêt propre là où le lien a été
// coupé.
func TestPollReportsTheServerClosingTheConnection(t *testing.T) {
	c, server := connPair(t)
	server.Close()

	if _, err := c.poll(); err == nil {
		t.Fatal("la fermeture du serveur n'a rien rapporté")
	}
}

// TestReplyKeepsTheEventsThatPrecedeIt vérifie que les événements rencontrés en
// attendant une réponse sont gardés.
//
// Ils arrivent pour de bon : à l'ouverture, les atomes se demandent pendant que le
// serveur annonce déjà l'apparition de la fenêtre et sa taille réelle. Les jeter
// ferait démarrer la boucle sur la taille demandée au lieu de celle accordée, et
// l'écart ne se verrait que sur un gestionnaire de fenêtres qui impose la sienne.
func TestReplyKeepsTheEventsThatPrecedeIt(t *testing.T) {
	c, server := connPair(t)

	before := eventOf(eventMapNotify, 3)
	answer := eventOf(msgReply, 4)
	order.PutUint32(answer[8:], 0x2A)

	if _, err := server.Write(append(before, answer...)); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	m, err := c.reply(4)
	if err != nil {
		t.Fatalf("réponse en échec : %v", err)
	}
	if got := order.Uint32(m[8:]); got != 0x2A {
		t.Errorf("la réponse porte %#x, attendu 0x2a", got)
	}

	if len(c.pending) != 1 {
		t.Fatalf("%d événement(s) mis de côté, un seul attendu", len(c.pending))
	}
	if c.pending[0][0] != eventMapNotify {
		t.Errorf("l'événement gardé porte le code %d, attendu %d", c.pending[0][0], eventMapNotify)
	}
}

// TestReplyReportsAProtocolError vérifie qu'une erreur du serveur est rendue, et que
// son message nomme la requête et la sorte d'erreur.
//
// Le numéro de séquence est ce qui sert : une requête sans réponse ne se signale que
// par là, et sans lui il n'y a aucun moyen de savoir laquelle des cinq requêtes d'une
// ouverture était fausse.
func TestReplyReportsAProtocolError(t *testing.T) {
	c, server := connPair(t)

	m := eventOf(msgError, 9)
	m[1] = 8 // Match
	m[10] = opCreateWindow
	if _, err := server.Write(m); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	_, err := c.reply(9)
	if err == nil {
		t.Fatal("erreur de protocole acceptée")
	}
	if !strings.Contains(err.Error(), "Match") {
		t.Errorf("le message ne nomme pas la sorte d'erreur : %v", err)
	}
	if !strings.Contains(err.Error(), "9") {
		t.Errorf("le message ne nomme pas la requête fautive : %v", err)
	}
}

// TestReplyReportsAnErrorFromAnEarlierRequest vérifie qu'une erreur accusant une autre
// requête est rendue au lieu d'être mise de côté.
//
// C'est ce qui fait rendre les erreurs d'ouverture à l'ouverture : la création, le
// titre et les bornes n'ont pas de réponse à elles, donc leur échec n'arrive qu'en
// attendant autre chose. La poursuivre ferait échouer plus loin, sur un objet sans
// rapport avec la requête fausse.
func TestReplyReportsAnErrorFromAnEarlierRequest(t *testing.T) {
	c, server := connPair(t)

	failed := eventOf(msgError, 2)
	failed[1] = 3 // Window
	if _, err := server.Write(failed); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	_, err := c.reply(6)
	if err == nil {
		t.Fatal("une erreur accusant une requête antérieure a été ignorée")
	}
	if !strings.Contains(err.Error(), "Window") {
		t.Errorf("le message ne nomme pas la sorte d'erreur : %v", err)
	}
}

// TestSequenceStartsAtOne vérifie que la première requête après le handshake porte le
// numéro un.
//
// Tout le rattachement d'une erreur à sa cause en dépend, et un décalage d'un seul
// rang ferait accuser la requête voisine — ce qui est pire qu'un message absent,
// puisque le message serait alors précis et faux.
func TestSequenceStartsAtOne(t *testing.T) {
	c, _ := connPair(t)

	if got := c.next(); got != 1 {
		t.Errorf("première requête numérotée %d, attendu 1", got)
	}
	if got := c.next(); got != 2 {
		t.Errorf("deuxième requête numérotée %d, attendu 2", got)
	}
}

// TestIdentifiersStayInsideTheRangeTheServerGranted vérifie que les identifiants
// créés tiennent dans la plage concédée.
//
// Le serveur refuse par une erreur de choix d'identifiant ce qui en sort, et le masque
// n'est pas promis commencer au bit zéro : un décalage en dur passerait les contrôles
// sur un serveur courant pour échouer ailleurs.
func TestIdentifiersStayInsideTheRangeTheServerGranted(t *testing.T) {
	c, _ := connPair(t)

	seen := make(map[uint32]bool)
	for i := 0; i < 8; i++ {
		id := c.id()
		if id&^c.setup.idMask != c.setup.idBase {
			t.Errorf("identifiant %#x hors de la plage %#x/%#x", id, c.setup.idBase, c.setup.idMask)
		}
		if seen[id] {
			t.Errorf("identifiant %#x rendu deux fois", id)
		}
		seen[id] = true
	}
}

// TestFlushSendsNothingTwice vérifie que l'envoi vide le tampon.
//
// Sans cela, chaque envoi réexpédierait tout l'historique : le serveur recréerait la
// fenêtre à chaque image, et le numéro de séquence que nous comptons n'aurait plus
// aucun rapport avec celui qu'il compte.
func TestFlushSendsNothingTwice(t *testing.T) {
	c, server := connPair(t)

	c.mapWindow(0x04000001)
	if err := c.flush(); err != nil {
		t.Fatalf("envoi : %v", err)
	}
	if len(c.out) != 0 {
		t.Errorf("%d octets restent dans le tampon après l'envoi", len(c.out))
	}

	sent := make([]byte, 8)
	if _, err := io.ReadFull(server, sent); err != nil {
		t.Fatalf("lecture côté serveur : %v", err)
	}
	if sent[0] != opMapWindow {
		t.Errorf("opcode reçu %d, attendu %d", sent[0], opMapWindow)
	}

	if err := c.flush(); err != nil {
		t.Fatalf("second envoi : %v", err)
	}
	if err := server.SetReadDeadline(time.Now()); err != nil {
		t.Fatalf("délai de lecture : %v", err)
	}
	if n, err := server.Read(sent); err == nil {
		t.Errorf("%d octets réexpédiés par un envoi à vide", n)
	}
}
