// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"errors"
	"net"
	"testing"

	"github.com/sprimault/lozengine/input"
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
