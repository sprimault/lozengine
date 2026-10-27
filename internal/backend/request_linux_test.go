// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"io"
	"testing"

	"github.com/sprimault/lozengine/render"
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
		{"le contexte graphique", func(c *conn) { c.createGC(0x04000002, 0x04000001) }},
		{"un transfert d'image", func(c *conn) {
			c.putImage(0x04000001, 0x04000002, 24, 0, 2, 2, make([]render.Color, 4))
		}},
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

// TestPutImageCarriesTheFieldsThatWouldShiftTheImage vérifie les champs dont une
// erreur ne lève rien et décale l'image.
//
// Trois d'entre eux sont nommés par le brief comme des pièges chers : le bourrage à
// gauche doit valoir zéro en ZPixmap, la profondeur doit égaler celle de la fenêtre, et
// les pixels doivent commencer là où l'en-tête s'arrête. Une erreur sur l'un des trois
// ne rend pas une image fausse à moitié — elle rend un escalier diagonal, qui est le
// symptôme d'un défaut de projection ou d'ancre.
func TestPutImageCarriesTheFieldsThatWouldShiftTheImage(t *testing.T) {
	c, _ := connPair(t)

	c.putImage(0x04000001, 0x04000002, 24, 7, 2, 1, []render.Color{
		{R: 1, G: 2, B: 3, A: 255},
		{R: 4, G: 5, B: 6, A: 255},
	})

	out := c.out
	if out[0] != opPutImage || out[1] != zPixmap {
		t.Fatalf("opcode %d et format %d, attendus %d et %d", out[0], out[1], opPutImage, zPixmap)
	}
	if got := order.Uint16(out[12:]); got != 2 {
		t.Errorf("largeur %d, attendu 2", got)
	}
	if got := order.Uint16(out[14:]); got != 1 {
		t.Errorf("hauteur %d, attendu 1", got)
	}
	if got := order.Uint16(out[16:]); got != 0 {
		t.Errorf("abscisse %d, attendu 0 — le découpage se fait en hauteur", got)
	}
	if got := order.Uint16(out[18:]); got != 7 {
		t.Errorf("ordonnée %d, attendu 7", got)
	}
	if out[20] != 0 {
		t.Errorf("bourrage à gauche %d, attendu 0, que le format impose", out[20])
	}
	if out[21] != 24 {
		t.Errorf("profondeur %d, attendu celle de la fenêtre", out[21])
	}

	// Les pixels commencent à la fin de l'en-tête, et le visuel du harnais demande
	// l'échange des octets extrêmes.
	if got := out[putImageHeader : putImageHeader+8]; got[0] != 3 || got[1] != 2 || got[2] != 1 || got[4] != 6 {
		t.Errorf("premiers octets de l'image %v, attendus les composantes échangées", got)
	}
	if len(out) != putImageHeader+8 {
		t.Errorf("requête de %d octets, attendu %d — quatre octets par pixel ne demandent aucun bourrage",
			len(out), putImageHeader+8)
	}
}

// TestPutImageSwitchesToTheExtendedFormWhenItMustGrow vérifie la forme étendue de
// `BIG-REQUESTS`.
//
// Au-delà de ce que seize bits portent, la longueur passe sur trente-deux et les seize
// bits valent zéro. Un serveur qui n'a pas l'extension lirait alors une requête de
// longueur nulle, donc le découpage reste obligatoire tant qu'elle n'est pas activée —
// c'est ce que le compte de lignes garantit, et ce cas-ci ne vérifie que la forme.
func TestPutImageSwitchesToTheExtendedFormWhenItMustGrow(t *testing.T) {
	c, _ := connPair(t)

	// Juste assez de pixels pour dépasser la borne des seize bits.
	pixels := make([]render.Color, requestWordsMax)
	c.putImage(0x04000001, 0x04000002, 24, 0, uint16(len(pixels)), 1, pixels)

	out := c.out
	if got := order.Uint16(out[2:]); got != 0 {
		t.Fatalf("longueur courte %d, attendu 0 pour annoncer la forme étendue", got)
	}

	// La longueur étendue compte le mot qui la porte, donc la requête entière.
	if got := 4 * int(order.Uint32(out[4:])); got != len(out) {
		t.Errorf("longueur étendue %d octets, %d écrits", got, len(out))
	}
}

// TestLinesPerRequestLeavesRoomForTheExtendedLength vérifie que le compte de lignes
// tient dans une requête, marge de la forme étendue comprise.
//
// Un compte trop généreux ne produit pas une image fausse : le serveur refuse la
// requête et coupe la connexion, ce qui se rapporte comme un affichage perdu.
func TestLinesPerRequestLeavesRoomForTheExtendedLength(t *testing.T) {
	c, _ := connPair(t)
	c.setup.maxRequest = 1024

	lines := c.linesPerRequest(64)
	if lines != 3 {
		t.Fatalf("%d ligne(s) par requête, attendu 3", lines)
	}

	// La bande la plus grande, en-tête et marge comprises, doit tenir sous le
	// plafond.
	if got := putImageHeader + 4 + 4*64*lines; got > int(c.setup.maxRequest) {
		t.Errorf("une bande de %d lignes fait %d octets, au-delà du plafond de %d", lines, got, c.setup.maxRequest)
	}
}

// iconOf fabrique une icône opaque dont chaque pixel porte la valeur donnée sur ses
// trois composantes, pour qu'un mot assemblé se reconnaisse d'un coup d'œil.
func iconOf(width, height uint16, shade uint8) *render.Buffer {
	icon := render.NewBuffer(width, height)
	for i := range icon.Pixels {
		icon.Pixels[i] = render.Color{R: shade, G: shade, B: shade, A: 255}
	}
	return icon
}

// TestSetIconsCarriesEverySizeInOneProperty vérifie que toutes les tailles partent
// ensemble, chacune annoncée par ses dimensions.
//
// C'est la différence de fond avec Windows, qui veut une icône par emplacement et
// oblige à choisir : ici la propriété porte une suite d'images et le gestionnaire de
// fenêtres prend celle qui lui convient. Envoyer une seule taille marcherait et
// donnerait une icône floue partout où le gestionnaire en veut une autre.
func TestSetIconsCarriesEverySizeInOneProperty(t *testing.T) {
	c, _ := connPair(t)

	c.setIcons(0x04000001, 0x130, []*render.Buffer{
		iconOf(1, 1, 0x0A),
		iconOf(2, 1, 0x0B),
	})

	// Une image de 1×1 occupe trois mots — largeur, hauteur, un pixel — et une de
	// 2×1 en occupe quatre.
	if got := order.Uint32(c.out[20:]); got != 7 {
		t.Fatalf("longueur annoncée %d éléments, attendu 7 pour deux icônes", got)
	}
	if got := c.out[16]; got != 32 {
		t.Errorf("format %d, attendu 32 — la propriété porte des mots", got)
	}

	data := c.out[changePropertyHeader:]
	if w, h := order.Uint32(data[0:]), order.Uint32(data[4:]); w != 1 || h != 1 {
		t.Errorf("première icône annoncée %d×%d, attendu 1×1", w, h)
	}
	if got := order.Uint32(data[8:]); got != 0xFF0A0A0A {
		t.Errorf("pixel de la première icône %#x, attendu 0xff0a0a0a", got)
	}
	if w, h := order.Uint32(data[12:]), order.Uint32(data[16:]); w != 2 || h != 1 {
		t.Errorf("seconde icône annoncée %d×%d, attendu 2×1", w, h)
	}
	if got := order.Uint32(data[20:]); got != 0xFF0B0B0B {
		t.Errorf("pixel de la seconde icône %#x, attendu 0xff0b0b0b", got)
	}
}

// TestSetIconsSkipsAnIconThatWouldShiftTheRest vérifie qu'une icône dont les pixels ne
// remplissent pas ses dimensions est écartée, et que celle qui suit reste lisible.
//
// La propriété se lit en suivant les tailles annoncées : une image trop courte
// décalerait toutes les suivantes, et le gestionnaire afficherait du bruit à partir de
// là. Le filtre n'est donc pas une politesse — c'est ce qui empêche une planche mal
// remplie par un jeu d'abîmer les icônes voisines.
func TestSetIconsSkipsAnIconThatWouldShiftTheRest(t *testing.T) {
	c, _ := connPair(t)

	c.setIcons(0x04000001, 0x130, []*render.Buffer{
		{Width: 2, Height: 2, Pixels: make([]render.Color, 1)},
		iconOf(1, 1, 0x0C),
	})

	if got := order.Uint32(c.out[20:]); got != 3 {
		t.Fatalf("longueur annoncée %d éléments, attendu 3 pour la seule icône valable", got)
	}

	data := c.out[changePropertyHeader:]
	if w, h := order.Uint32(data[0:]), order.Uint32(data[4:]); w != 1 || h != 1 {
		t.Errorf("icône annoncée %d×%d, attendu celle qui suivait l'écartée", w, h)
	}
	if got := order.Uint32(data[8:]); got != 0xFF0C0C0C {
		t.Errorf("pixel %#x, attendu celui de l'icône valable", got)
	}
}

// TestSetIconsPostsNothingWithoutIcons vérifie qu'aucune propriété n'est posée quand le
// jeu n'en fournit pas.
//
// Poser une propriété vide dirait au gestionnaire de fenêtres que ce jeu n'a pas
// d'icône, là que l'absence de propriété le laisse prendre celle du système — qui est
// le bon défaut.
func TestSetIconsPostsNothingWithoutIcons(t *testing.T) {
	cases := []struct {
		name  string
		icons []*render.Buffer
	}{
		{"aucune icône", nil},
		{"une tranche vide", []*render.Buffer{}},
		{"une icône absente", []*render.Buffer{nil}},
		{"une icône sans surface", []*render.Buffer{{Width: 0, Height: 4}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := connPair(t)
			c.setIcons(0x04000001, 0x130, tc.icons)

			if len(c.out) != 0 {
				t.Errorf("%d octets écrits alors qu'il n'y a pas d'icône à poser", len(c.out))
			}
		})
	}
}

// TestSetIconsWritesStraightAlpha vérifie que les mots partent à alpha non
// prémultiplié.
//
// Le moteur prémultiplie pour que le mélange d'un pixel soit une multiplication, et la
// propriété demande l'inverse. Envoyer le tampon tel quel donnerait une icône assombrie
// partout où elle est translucide — un contour adouci sortirait terne, ce qui ressemble
// à un défaut du dessin et non du backend.
func TestSetIconsWritesStraightAlpha(t *testing.T) {
	c, _ := connPair(t)

	// Un blanc à demi transparent : prémultiplié il vaut 128 sur chaque composante,
	// et la propriété doit recevoir 255.
	icon := render.NewBuffer(1, 1)
	icon.Pixels[0] = render.Premultiply(255, 255, 255, 128)

	c.setIcons(0x04000001, 0x130, []*render.Buffer{icon})

	got := order.Uint32(c.out[changePropertyHeader+8:])
	if got>>24 != 128 {
		t.Errorf("alpha %d, attendu 128", got>>24)
	}
	if red := (got >> 16) & 0xFF; red != 255 {
		t.Errorf("rouge %d, attendu 255 — la composante n'a pas été ramenée à sa pleine valeur", red)
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
