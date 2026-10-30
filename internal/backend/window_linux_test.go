// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0
// Ce qui ferme une fenêtre et ce qui ne la ferme pas : la demande du gestionnaire avec
// son bit d'envoi, la destruction, une erreur de protocole qui emporte sa cause — et le
// retrait de l'affichage, qui n'est pas une fermeture.

package backend

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
)

// windowPair monte une fenêtre sur une connexion de test et rend le bout par lequel un
// cas joue le serveur.
//
// Les atomes sont posés à la main : les demander exigerait un serveur qui réponde, et
// ce ne sont pas eux que ces cas éprouvent — seulement ce que la fenêtre fait des
// événements qui les portent.
func windowPair(t *testing.T) (*window, *net.UnixConn) {
	t.Helper()

	c, server := connPair(t)
	return &window{
		c:            c,
		id:           0x04000001,
		protocols:    0x120,
		deleteWindow: 0x121,
		width:        480,
		height:       270,
	}, server
}

// sendEvent écrit un événement côté serveur.
func sendEvent(t *testing.T, server *net.UnixConn, m []byte) {
	t.Helper()
	if _, err := server.Write(m); err != nil {
		t.Fatalf("écriture de l'événement : %v", err)
	}
}

// TestConfigureNotifyUpdatesTheSize vérifie que la taille suivie est celle que le
// serveur annonce.
//
// C'est elle que la boucle lit pour monter son tampon et calculer l'agrandissement.
// Garder la taille demandée alors que le gestionnaire de fenêtres a imposé la sienne
// ferait peindre dans un tampon qui ne correspond plus à la fenêtre.
func TestConfigureNotifyUpdatesTheSize(t *testing.T) {
	w, server := windowPair(t)

	m := eventOf(eventConfigureNotify, 1)
	order.PutUint16(m[20:], 1024)
	order.PutUint16(m[22:], 576)
	sendEvent(t, server, m)

	w.Input()

	width, height := w.Size()
	if width != 1024 || height != 576 {
		t.Errorf("taille %d×%d, attendu 1024×576", width, height)
	}
	if w.Closed() {
		t.Error("un redimensionnement a fermé la fenêtre")
	}
}

// TestTheCloseRequestArrivesWithTheSentBit verrouille le piège du bit d'envoi.
//
// La demande de fermeture est **toujours** fabriquée par le gestionnaire de fenêtres,
// donc son premier octet porte le bit qui dit « cet événement vient d'un autre
// client ». Comparer le numéro sans masquer ce bit ne reconnaît jamais le message : la
// croix ne ferme pas la fenêtre, et rien n'indique pourquoi — ni erreur, ni événement
// manquant.
func TestTheCloseRequestArrivesWithTheSentBit(t *testing.T) {
	w, server := windowPair(t)

	m := eventOf(eventClientMessage|eventSentBit, 4)
	order.PutUint32(m[8:], w.protocols)
	order.PutUint32(m[12:], w.deleteWindow)
	sendEvent(t, server, m)

	w.Input()

	if !w.Closed() {
		t.Error("la demande de fermeture n'a pas fermé la fenêtre")
	}
	if err := w.Err(); err != nil {
		t.Errorf("une fermeture demandée par le joueur rapporte une erreur : %v", err)
	}
}

// TestAMessageOfAnotherProtocolDoesNotClose vérifie que seule la demande de fermeture
// ferme.
//
// Un gestionnaire de fenêtres envoie d'autres messages sous le même type — une
// vérification de vie, par exemple. Ne regarder que le type fermerait la fenêtre sur
// le premier d'entre eux, et le jeu s'arrêterait de lui-même au bout de quelques
// secondes.
func TestAMessageOfAnotherProtocolDoesNotClose(t *testing.T) {
	w, server := windowPair(t)

	m := eventOf(eventClientMessage|eventSentBit, 4)
	order.PutUint32(m[8:], w.protocols)
	order.PutUint32(m[12:], 0x999) // un protocole que nous n'avons pas listé
	sendEvent(t, server, m)

	w.Input()

	if w.Closed() {
		t.Error("un message d'un autre protocole a fermé la fenêtre")
	}
}

// TestUnmappingIsNotClosing vérifie qu'une fenêtre masquée n'est pas une fenêtre
// fermée.
//
// C'est ce qui arrive quand le joueur réduit la fenêtre. La prendre pour une fermeture
// arrêterait le jeu à la première réduction — et la taille doit rester celle qu'elle
// avait, sans quoi la boucle n'aurait plus de tampon valide au retour.
func TestUnmappingIsNotClosing(t *testing.T) {
	w, server := windowPair(t)

	sendEvent(t, server, eventOf(eventMapNotify, 2))
	w.Input()
	if !w.mapped {
		t.Fatal("la fenêtre n'est pas tenue pour affichée après son apparition")
	}

	sendEvent(t, server, eventOf(eventUnmapNotify, 3))
	w.Input()

	if w.Closed() {
		t.Error("une fenêtre masquée a été tenue pour fermée")
	}
	if w.mapped {
		t.Error("la fenêtre est encore tenue pour affichée")
	}
	if width, height := w.Size(); width != 480 || height != 270 {
		t.Errorf("taille %d×%d après le masquage, attendu celle d'avant", width, height)
	}
}

// TestDestroyNotifyCloses vérifie qu'une fenêtre détruite par le serveur est tenue
// pour fermée.
//
// Cela arrive sans qu'on l'ait demandé, un gestionnaire de fenêtres pouvant fermer de
// force. Continuer à présenter dessus produirait une erreur de protocole par image.
func TestDestroyNotifyCloses(t *testing.T) {
	w, server := windowPair(t)

	sendEvent(t, server, eventOf(eventDestroyNotify, 5))
	w.Input()

	if !w.Closed() {
		t.Error("la destruction de la fenêtre ne l'a pas fermée")
	}
}

// TestAProtocolErrorEndsTheWindowWithItsCause vérifie qu'une erreur du serveur met fin
// à la fenêtre **et** laisse sa cause.
//
// C'est exactement ce que le membre Err du contrat a été ajouté pour porter : sans
// lui, une requête refusée se rapporterait comme un arrêt propre, ce qui est le régime
// courant du développement d'un backend.
func TestAProtocolErrorEndsTheWindowWithItsCause(t *testing.T) {
	w, server := windowPair(t)

	m := eventOf(msgError, 6)
	m[1] = 8 // Match
	sendEvent(t, server, m)

	w.Input()

	if !w.Closed() {
		t.Error("une erreur de protocole n'a pas fermé la fenêtre")
	}
	if w.Err() == nil {
		t.Error("une erreur de protocole n'a laissé aucune cause")
	}
}

// TestLosingTheServerLeavesItsCause vérifie que la perte de la connexion se distingue
// d'une fermeture par le joueur.
//
// Une session qui se termine, un serveur qui s'arrête : la fenêtre disparaît sans que
// personne n'ait cliqué la croix. Un jeu qui recevrait nil dans les deux cas ne
// pourrait ni le consigner ni le rapporter.
func TestLosingTheServerLeavesItsCause(t *testing.T) {
	w, server := windowPair(t)
	server.Close()

	w.Input()

	if !w.Closed() {
		t.Error("la perte du serveur n'a pas fermé la fenêtre")
	}
	if w.Err() == nil {
		t.Error("la perte du serveur n'a laissé aucune cause")
	}
}

// TestTheFirstCauseIsKeptNotTheLast vérifie que c'est la première cause qui est
// gardée.
//
// Ce qui suit une connexion perdue n'en est que la conséquence, et rapporter la
// dernière erreur ferait remonter le symptôme plutôt que l'origine.
func TestTheFirstCauseIsKeptNotTheLast(t *testing.T) {
	w, _ := windowPair(t)

	first := errors.New("la première")
	w.fail(first)
	w.fail(errors.New("la seconde"))

	if w.Err() != first {
		t.Errorf("cause gardée %v, attendu la première", w.Err())
	}
}

// TestEventsFromOpeningAreNotLost vérifie que ce qui est arrivé pendant l'ouverture est
// traité au premier pas.
//
// La première taille réelle arrive là : le serveur l'annonce pendant que l'ouverture
// attend encore ses atomes. La perdre ferait démarrer la boucle sur la taille demandée
// au lieu de celle accordée.
func TestEventsFromOpeningAreNotLost(t *testing.T) {
	w, _ := windowPair(t)

	kept := eventOf(eventConfigureNotify, 1)
	order.PutUint16(kept[20:], 800)
	order.PutUint16(kept[22:], 450)
	w.c.pending = append(w.c.pending, kept)

	w.Input()

	if width, height := w.Size(); width != 800 || height != 450 {
		t.Errorf("taille %d×%d, attendu 800×450", width, height)
	}
	if len(w.c.pending) != 0 {
		t.Errorf("%d événement(s) restent en attente après le pas", len(w.c.pending))
	}
}

// TestFocusOutForgetsHeldKeys vérifie que les touches tenues sont oubliées quand la
// fenêtre perd le focus.
//
// Le relâchement est perdu, pas différé : la fenêtre ne reçoit plus rien après, et le
// serveur dirait encore « enfoncée » si on le lui demandait. Sans cet oubli, le
// personnage marcherait tout seul au retour dans le jeu.
func TestFocusOutForgetsHeldKeys(t *testing.T) {
	w, server := windowPair(t)
	w.state.Press(input.KeyA)

	sendEvent(t, server, eventOf(eventFocusOut, 7))
	in := w.Input()

	if in.Held(input.KeyA) {
		t.Error("une touche tenue l'est restée après la perte du focus")
	}
}

// keyEventOf fabrique un événement de touche portant son code et son instant.
//
// L'instant compte autant que le code : c'est lui qui sépare une auto-répétition d'un
// vrai relâchement.
func keyEventOf(kind, keycode byte, at uint32) []byte {
	m := eventOf(kind, 1)
	m[1] = keycode
	order.PutUint32(m[4:], at)
	return m
}

// codeA est le code de touche que le serveur envoie pour la lettre A : le code 30 du
// noyau, plus les huit que le protocole réserve.
const codeA = 30 + keycodeShift

// TestAutoRepeatDoesNotFlickerTheKey vérifie qu'une auto-répétition ne fait ni
// clignoter la touche ni rearmer son front montant.
//
// Doigt posé, le serveur produit appui, relâchement, appui, relâchement — le brief du
// jalon l'annonçait comme un piège cher, parade comprise. **La mesure a montré que le
// backend n'a rien à faire pour cela** : le front montant du moteur se calcule par
// comparaison entre deux pas et non sur une transition, et l'état retient ce qui a été
// enfoncé pendant le pas même si le doigt s'est relevé. Les deux événements tombent à
// quelques millisecondes dans un pas de seize, donc ils s'annulent sans rien laisser.
//
// **Ce que ce cas garde est donc la liaison et non une parade** : que le décodage
// traduise bien les deux événements vers la même place, et que l'état s'y comporte
// comme on le croit. Il rougit si l'un des deux décodages disparaît, ce qui a été
// vérifié ; il ne rougirait pas si l'on retirait une parade, puisqu'il n'y en a pas.
//
// Le cas résiduel qu'il ne couvre pas : les deux événements séparés par une frontière
// de pas, qui coûte un pas d'extinction. C'est l'extension du clavier, demandée à
// l'ouverture, qui le ferme en supprimant ces relâchements à la source.
func TestAutoRepeatDoesNotFlickerTheKey(t *testing.T) {
	w, server := windowPair(t)

	sendEvent(t, server, keyEventOf(eventKeyPress, codeA, 100))
	in := w.Input()
	if !in.Held(input.KeyA) {
		t.Fatal("la touche n'est pas tenue après son appui")
	}
	if !in.JustPressed(input.KeyA) {
		t.Fatal("le front montant manque au premier appui")
	}

	// Ce que le serveur envoie pour un doigt qui reste posé : les deux événements
	// portent le même instant, et c'est tout ce qui les distingue d'un relâchement
	// suivi d'un nouvel appui.
	repeat := keyEventOf(eventKeyRelease, codeA, 200)
	repeat = append(repeat, keyEventOf(eventKeyPress, codeA, 200)...)
	sendEvent(t, server, repeat)

	in = w.Input()
	if !in.Held(input.KeyA) {
		t.Error("une auto-répétition a relâché la touche")
	}
	if in.JustPressed(input.KeyA) {
		t.Error("une auto-répétition a rearmé le front montant")
	}
}

// TestAReleaseStopsTheKeyInTheSameStep vérifie qu'un relâchement compte dans le pas
// où il arrive.
//
// C'est le pendant du cas précédent et il porte la limite de l'autre : un relâchement
// seul, arrivé en dernier, doit éteindre la touche avant que la fenêtre rende la main.
// S'il était différé — ce qu'une parade contre l'auto-répétition serait tentée de
// faire —, le jeu verrait la touche tenue un pas de trop, soit un personnage qui
// glisse après qu'on a lâché la flèche.
func TestAReleaseStopsTheKeyInTheSameStep(t *testing.T) {
	w, server := windowPair(t)

	sendEvent(t, server, keyEventOf(eventKeyPress, codeA, 100))
	w.Input()

	sendEvent(t, server, keyEventOf(eventKeyRelease, codeA, 200))

	if w.Input().Held(input.KeyA) {
		t.Error("la touche est encore tenue après son relâchement")
	}
}

// TestKeymapNotifyRestoresTheKeyboard vérifie la reprise de l'état sur l'événement que
// le serveur envoie après chaque prise de focus.
//
// Il porte un piège de forme que le brief a nommé : **c'est le seul événement du
// protocole sans numéro de séquence**, ses deux octets de séquence portant déjà de
// l'état de clavier. Lire un numéro là déplacerait le vecteur d'autant, et les touches
// tenues seraient les voisines de celles qu'on croit.
func TestKeymapNotifyRestoresTheKeyboard(t *testing.T) {
	w, server := windowPair(t)
	w.state.Press(input.KeyLeft)

	// Construit à la main, et pas par le fabricant d'événements ordinaire : celui-ci
	// écrirait un numéro de séquence là où cet événement porte des touches.
	m := make([]byte, eventSize)
	m[0] = eventKeymapNotify
	m[1+3] |= 1 << 6 // la lettre A, code 38 pour le serveur
	sendEvent(t, server, m)

	in := w.Input()
	if !in.Held(input.KeyA) {
		t.Error("la lettre A n'est pas tenue alors que le serveur la nomme")
	}
	if in.Held(input.KeyLeft) {
		t.Error("une touche absente du vecteur est restée tenue")
	}
}

// TestPresentSplitsTheImageIntoBands vérifie qu'une image trop grande pour une requête
// part en bandes, chacune disant où elle va.
//
// Le découpage n'est pas une précaution : une requête est bornée par le protocole, et
// une image de la taille d'un écran dépasse la borne du handshake. Une bande mal
// placée ne produit aucune erreur — elle recopie une partie de l'image au mauvais
// endroit, ce qui ressemble à un défaut de projection.
//
// **L'image reste petite exprès.** Personne ne lit la socket pendant que la
// présentation écrit, donc tout ce qu'elle envoie doit tenir dans le tampon du
// système : au-delà, l'envoi attend un lecteur qui ne viendra qu'après. Mesuré en
// régime rouge, où une faute supprimant le découpage a fait envoyer dix kilooctets
// d'un bloc et bloqué l'écriture jusqu'au délai du harnais — le cas a bien rougi, mais
// pour le blocage et non pour le compte de bandes.
func TestPresentSplitsTheImageIntoBands(t *testing.T) {
	w, server := windowPair(t)
	w.gc = 0x04000002

	// Un plafond qui ne laisse passer que trois lignes de soixante-quatre pixels,
	// posé par un champ plutôt que par un vrai petit serveur : c'est le régime d'un
	// serveur sans `BIG-REQUESTS` sur une grande fenêtre.
	w.c.setup.maxRequest = 1024

	w.Present(render.NewBuffer(64, 10))

	if err := w.Err(); err != nil {
		t.Fatalf("présentation en échec : %v", err)
	}

	// Trois bandes de trois lignes et une de une : 24 octets d'en-tête et 256 par
	// ligne.
	sent := make([]byte, 3*(putImageHeader+3*256)+(putImageHeader+256))
	if _, err := io.ReadFull(server, sent); err != nil {
		t.Fatalf("lecture côté serveur : %v", err)
	}

	var bands [][2]uint16
	for at := 0; at < len(sent); {
		if sent[at] != opPutImage {
			t.Fatalf("octet %d : opcode %d, attendu un transfert d'image", at, sent[at])
		}
		bands = append(bands, [2]uint16{order.Uint16(sent[at+18:]), order.Uint16(sent[at+14:])})
		at += 4 * int(order.Uint16(sent[at+2:]))
	}

	want := [][2]uint16{{0, 3}, {3, 3}, {6, 3}, {9, 1}}
	if len(bands) != len(want) {
		t.Fatalf("%d bande(s) envoyée(s), %d attendues : %v", len(bands), len(want), bands)
	}
	for i := range want {
		if bands[i] != want[i] {
			t.Errorf("bande %d à l'ordonnée %d sur %d ligne(s), attendu %d sur %d",
				i, bands[i][0], bands[i][1], want[i][0], want[i][1])
		}
	}
}

// TestPresentSendsOneRequestWhenTheImageFits vérifie qu'une image qui tient part d'un
// seul bloc.
//
// C'est ce que l'extension de taille achète, et c'est le régime courant : découper une
// image qui tient coûterait un en-tête et un aller-retour de protocole par bande, pour
// rien.
func TestPresentSendsOneRequestWhenTheImageFits(t *testing.T) {
	w, server := windowPair(t)
	w.gc = 0x04000002

	w.Present(render.NewBuffer(64, 10))

	sent := make([]byte, putImageHeader+10*256)
	if _, err := io.ReadFull(server, sent); err != nil {
		t.Fatalf("lecture côté serveur : %v", err)
	}
	if got := order.Uint16(sent[14:]); got != 10 {
		t.Errorf("hauteur de la première bande %d, attendu les dix lignes", got)
	}
	if got := 4 * int(order.Uint16(sent[2:])); got != len(sent) {
		t.Errorf("la requête annonce %d octets, l'image entière en fait %d", got, len(sent))
	}
}

// TestPresentDoesNothingOnAClosedWindow vérifie qu'une fenêtre finie n'envoie plus
// rien.
//
// Présenter sur une fenêtre que le serveur a détruite produirait une erreur de
// protocole par image, et la boucle du jeu continue d'appeler jusqu'à ce qu'elle lise
// la fermeture.
//
// **Le constat se fait côté serveur, et la première version de ce cas était fausse
// pour l'avoir fait ailleurs** : elle regardait le tampon d'émission, que [Present]
// vide en envoyant. Le tampon était donc vide dans les deux cas, et le cas restait
// vert même en retirant la garde qu'il prétendait éprouver — c'est le défaut que la
// doctrine de test appelle le plus facile à ne pas voir, et il a fallu le régime rouge
// pour le trouver.
func TestPresentDoesNothingOnAClosedWindow(t *testing.T) {
	w, server := windowPair(t)
	w.gc = 0x04000002
	w.closed = true

	w.Present(render.NewBuffer(64, 10))

	// Rien ne doit arriver, donc la lecture doit expirer plutôt que rendre des
	// octets. Le délai est court et remplace celui du harnais : attendre cinq
	// secondes pour constater un silence serait du temps perdu à chaque passage.
	if err := server.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("délai de lecture : %v", err)
	}
	if n, err := server.Read(make([]byte, 64)); err == nil {
		t.Errorf("%d octets envoyés pour une fenêtre fermée", n)
	}
}

// TestCloseTwiceIsHarmless vérifie que la seconde fermeture ne fait rien et que la
// fenêtre reste fermée.
//
// L'ordre compte dans le corps : poser l'état avant de court-circuiter. Un
// court-circuit en tête laisserait Closed mentir au second appel, ce qu'un cas de test
// a déjà montré sur l'autre backend.
func TestCloseTwiceIsHarmless(t *testing.T) {
	w, _ := windowPair(t)

	w.Close()
	if !w.Closed() {
		t.Error("la fenêtre n'est pas fermée après Close")
	}

	w.Close()
	if !w.Closed() {
		t.Error("la fenêtre ne se dit plus fermée après un second Close")
	}
}
