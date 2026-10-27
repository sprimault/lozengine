// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"io"
	"testing"
)

// TestCreateWindowWritesValuesInBitOrder verrouille le piège qui coûte le plus cher à
// chercher : les valeurs à masque s'écrivent dans l'ordre croissant de leurs bits.
//
// Les inverser ne lève aucune erreur. Le serveur lirait le masque d'événements comme
// une couleur de fond et la couleur comme un masque d'événements : la fenêtre
// s'ouvrirait, colorée, et ne recevrait plus rien. Le symptôme est « le clavier ne
// marche pas », qu'on va chercher dans le décodage des touches, c'est-à-dire à l'autre
// bout du backend.
func TestCreateWindowWritesValuesInBitOrder(t *testing.T) {
	c, _ := connPair(t)

	// Une couleur de fond reconnaissable, et non le noir que l'écran annonce : avec
	// zéro des deux côtés, deux valeurs inversées se liraient encore comme justes.
	c.setup.blackPixel = 0x00ABCDEF

	const events = maskKeyPress | maskKeyRelease | maskStructureNotify
	c.createWindow(0x04000001, 480, 270, events)

	out := c.out
	if got := order.Uint32(out[28:]); got != cwBackPixel|cwEventMask {
		t.Fatalf("masque de valeurs %#x, attendu %#x", got, cwBackPixel|cwEventMask)
	}

	// Le bit de la couleur de fond est plus bas que celui du masque d'événements,
	// donc la couleur vient d'abord.
	if got := order.Uint32(out[32:]); got != c.setup.blackPixel {
		t.Errorf("première valeur %#x, attendu la couleur de fond %#x", got, c.setup.blackPixel)
	}
	if got := order.Uint32(out[36:]); got != events {
		t.Errorf("seconde valeur %#x, attendu le masque d'événements %#x", got, events)
	}
}

// TestCreateWindowAsksTheUsefulSizeWithoutAdjusting vérifie que la taille demandée part
// telle quelle.
//
// C'est l'inverse de Win32, qui reçoit la fenêtre entière et oblige à retrancher les
// bordures. Ici la bordure et la barre de titre appartiennent à une fenêtre que le
// gestionnaire met autour de la nôtre : ajuster par symétrie avec l'autre backend
// donnerait une image plus petite que demandée.
func TestCreateWindowAsksTheUsefulSizeWithoutAdjusting(t *testing.T) {
	c, _ := connPair(t)

	c.createWindow(0x04000001, 480, 270, 0)

	out := c.out
	if got := order.Uint16(out[16:]); got != 480 {
		t.Errorf("largeur demandée %d, attendu 480", got)
	}
	if got := order.Uint16(out[18:]); got != 270 {
		t.Errorf("hauteur demandée %d, attendu 270", got)
	}
	if got := order.Uint32(out[8:]); got != c.setup.root {
		t.Errorf("parent %#x, attendu la racine %#x", got, c.setup.root)
	}
	if got := order.Uint16(out[22:]); got != classInputOutput {
		t.Errorf("classe %d, attendu %d", got, classInputOutput)
	}
}

// TestRequestsDeclareTheLengthTheyWrote vérifie que chaque requête annonce exactement
// ce qu'elle a écrit.
//
// La longueur, en mots de quatre octets, est ce que le serveur suit pour trouver la
// requête suivante. Une erreur d'un seul mot ne produit pas une erreur sur la requête
// fautive : elle décale tout le flux, et le serveur lit la suivante au milieu de
// celle-ci. L'erreur qu'il finit par rendre parle alors d'autre chose.
func TestRequestsDeclareTheLengthTheyWrote(t *testing.T) {
	cases := []struct {
		name string
		send func(c *conn)
	}{
		{"la création de fenêtre", func(c *conn) { c.createWindow(0x04000001, 480, 270, maskKeyPress) }},
		{"l'apparition", func(c *conn) { c.mapWindow(0x04000001) }},
		{"la destruction", func(c *conn) { c.destroyWindow(0x04000001) }},
		{"un titre, dont la longueur n'est pas un multiple de quatre", func(c *conn) {
			c.changeProperty(0x04000001, atomWMName, atomString, 8, []byte("essai"))
		}},
		{"un titre vide", func(c *conn) {
			c.changeProperty(0x04000001, atomWMName, atomString, 8, nil)
		}},
		{"les bornes de taille", func(c *conn) { c.setFloor(0x04000001, 480, 270) }},
		{"la liste des protocoles", func(c *conn) { c.setProtocols(0x04000001, 100, 101) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := connPair(t)
			tc.send(c)

			if len(c.out)%4 != 0 {
				t.Fatalf("requête de %d octets, qui n'est pas un multiple de quatre", len(c.out))
			}
			if got := 4 * int(order.Uint16(c.out[2:])); got != len(c.out) {
				t.Errorf("longueur annoncée %d octets, %d écrits", got, len(c.out))
			}
		})
	}
}

// TestChangePropertyCountsElementsNotBytes vérifie que la longueur d'une propriété se
// compte en éléments de son format.
//
// Sur une propriété de trente-deux bits, compter les octets en annoncerait quatre fois
// trop : le serveur lirait au-delà de la requête, et son erreur de longueur
// n'indiquerait ni la propriété ni le format en cause.
func TestChangePropertyCountsElementsNotBytes(t *testing.T) {
	c, _ := connPair(t)

	c.setProtocols(0x04000001, 100, 101)

	if got := order.Uint32(c.out[20:]); got != 1 {
		t.Errorf("longueur annoncée %d élément(s), attendu 1 pour un atome de trente-deux bits", got)
	}
	if got := c.out[16]; got != 32 {
		t.Errorf("format %d, attendu 32", got)
	}
	if got := order.Uint32(c.out[24:]); got != 101 {
		t.Errorf("la liste porte %d, attendu l'atome de fermeture 101", got)
	}
}

// TestSetFloorPutsTheMinimumWhereTheHintsExpectIt vérifie la place de la taille
// minimale dans la structure des bornes.
//
// Les dix-huit mots sont de taille fixe et quatre d'entre eux, en tête, sont des
// champs devenus obsolètes qu'il faut quand même traverser. Écrire la taille à la
// place des premiers ferait lire par le gestionnaire de fenêtres une position au lieu
// d'une borne : la fenêtre serait placée en haut à gauche et pourrait toujours se
// réduire.
func TestSetFloorPutsTheMinimumWhereTheHintsExpectIt(t *testing.T) {
	c, _ := connPair(t)

	c.setFloor(0x04000001, 480, 270)

	hints := c.out[24:]
	if got := order.Uint32(hints[0:]); got != sizeHintsMinSize {
		t.Errorf("drapeaux %#x, attendu %#x — seule la taille minimale est renseignée", got, sizeHintsMinSize)
	}
	if got := order.Uint32(hints[4*sizeHintsMinIndex:]); got != 480 {
		t.Errorf("largeur minimale %d, attendu 480", got)
	}
	if got := order.Uint32(hints[4*(sizeHintsMinIndex+1):]); got != 270 {
		t.Errorf("hauteur minimale %d, attendu 270", got)
	}
	if got := order.Uint32(c.out[20:]); got != sizeHintsWords {
		t.Errorf("longueur annoncée %d éléments, attendu %d", got, sizeHintsWords)
	}
}

// TestInternAtomAsksEveryNameInOneRound vérifie que les atomes se demandent ensemble
// et que chaque nom part avec son bourrage.
//
// Les demander un par un coûterait un aller-retour par nom à chaque ouverture. Et le
// bourrage n'est pas une politesse : le serveur compte la longueur déclarée **plus**
// le bourrage pour trouver la requête suivante, donc l'omettre sur un nom de onze
// octets décale tout ce qui suit.
func TestInternAtomAsksEveryNameInOneRound(t *testing.T) {
	c, server := connPair(t)

	// Les réponses sont posées d'avance : le serveur traite les requêtes d'un client
	// dans l'ordre reçu, donc les réponses arrivent dans celui où on les attend.
	for i, atom := range []uint32{0x120, 0x121} {
		answer := eventOf(msgReply, uint16(i+1))
		order.PutUint32(answer[8:], atom)
		if _, err := server.Write(answer); err != nil {
			t.Fatalf("écriture de la réponse : %v", err)
		}
	}

	atoms, err := c.internAtom("WM_PROTOCOLS", "UTF8_STRING")
	if err != nil {
		t.Fatalf("demande d'atomes en échec : %v", err)
	}
	if len(atoms) != 2 || atoms[0] != 0x120 || atoms[1] != 0x121 {
		t.Fatalf("atomes %v, attendu [288 289]", atoms)
	}

	// Douze octets de nom pour le premier, donc aucun bourrage : vingt octets. Onze
	// pour le second, donc un octet de bourrage, et vingt aussi.
	sent := make([]byte, 40)
	if _, err := io.ReadFull(server, sent); err != nil {
		t.Fatalf("lecture côté serveur : %v", err)
	}

	if sent[0] != opInternAtom {
		t.Errorf("opcode %d, attendu %d", sent[0], opInternAtom)
	}
	if got := int(order.Uint16(sent[4:])); got != len("WM_PROTOCOLS") {
		t.Errorf("longueur du premier nom %d, attendu %d", got, len("WM_PROTOCOLS"))
	}
	if got := string(sent[8:20]); got != "WM_PROTOCOLS" {
		t.Errorf("premier nom %q", got)
	}
	if got := int(order.Uint16(sent[24:])); got != len("UTF8_STRING") {
		t.Errorf("longueur du second nom %d, attendu %d", got, len("UTF8_STRING"))
	}
	if got := string(sent[28:39]); got != "UTF8_STRING" {
		t.Errorf("second nom %q", got)
	}
	if sent[39] != 0 {
		t.Errorf("l'octet de bourrage du second nom vaut %d", sent[39])
	}
}

// TestSyncReportsAnErrorFromARequestWithoutReply vérifie que la synchronisation est ce
// qui fait rendre les erreurs d'ouverture à l'ouverture.
//
// Cinq des six requêtes d'une ouverture n'ont pas de réponse. Sans ce rendez-vous,
// leur échec n'arriverait qu'au premier pas de la boucle, sur une fenêtre qu'Open
// aurait déclarée bonne.
func TestSyncReportsAnErrorFromARequestWithoutReply(t *testing.T) {
	c, server := connPair(t)

	failed := eventOf(msgError, 1)
	failed[1] = 2 // Value
	failed[10] = opCreateWindow
	if _, err := server.Write(failed); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	c.createWindow(0x04000001, 0, 0, 0)
	if err := c.sync(); err == nil {
		t.Fatal("la synchronisation n'a pas rapporté l'erreur de la création")
	}
}
