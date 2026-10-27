// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"bytes"
	"strings"
	"testing"
)

// setupFormat est une entrée de la table des formats : une profondeur, et la place
// qu'un de ses pixels occupe sur la connexion.
type setupFormat struct {
	depth, bitsPerPixel uint8
}

// setupVisual est un visuel tel qu'un cas de test le pose, avec ses trois masques.
type setupVisual struct {
	id                           uint32
	redMask, greenMask, blueMask uint32
}

// setupScreen est un écran tel qu'un cas de test le pose.
type setupScreen struct {
	root, rootVisual, black uint32
	depth                   uint8
	visuals                 []setupVisual
}

// setupBlockOf assemble le bloc de longueur variable que le serveur envoie après
// l'en-tête du handshake.
//
// Il est construit ici plutôt que capturé sur un vrai serveur, et c'est ce qui rend
// les cas utiles : un handshake capturé ne porte qu'une seule combinaison de
// profondeur et de place par pixel, et c'est justement leur écart qui doit être
// éprouvé.
func setupBlockOf(vendor string, formats []setupFormat, screens ...setupScreen) []byte {
	b := make([]byte, 32)

	order.PutUint32(b[4:], 0x04000000) // resource-id-base
	order.PutUint32(b[8:], 0x001FFFFF) // resource-id-mask
	order.PutUint16(b[16:], uint16(len(vendor)))
	order.PutUint16(b[18:], 0x0040) // maximum-request-length, en mots de quatre octets
	b[20] = byte(len(screens))
	b[21] = byte(len(formats))
	b[22] = 0 // image-byte-order, petit-boutien
	b[26] = 8
	b[27] = 255

	b = append(b, vendor...)
	b = append(b, make([]byte, pad(len(vendor)))...)

	for _, f := range formats {
		b = append(b, f.depth, f.bitsPerPixel, 32, 0, 0, 0, 0, 0)
	}

	for _, s := range screens {
		at := len(b)
		b = append(b, make([]byte, 40)...)
		order.PutUint32(b[at:], s.root)
		order.PutUint32(b[at+12:], s.black)
		order.PutUint32(b[at+32:], s.rootVisual)
		b[at+38] = s.depth
		b[at+39] = 1 // une seule profondeur autorisée, qui porte tous les visuels

		at = len(b)
		b = append(b, make([]byte, 8)...)
		b[at] = s.depth
		order.PutUint16(b[at+2:], uint16(len(s.visuals)))

		for _, v := range s.visuals {
			at = len(b)
			b = append(b, make([]byte, 24)...)
			order.PutUint32(b[at:], v.id)
			b[at+4] = 4 // TrueColor
			order.PutUint32(b[at+8:], v.redMask)
			order.PutUint32(b[at+12:], v.greenMask)
			order.PutUint32(b[at+16:], v.blueMask)
		}
	}
	return b
}

// trueColor24 est l'écran ordinaire d'un serveur courant : profondeur 24, un visuel
// vrai-couleur, et un second visuel pour qu'il y ait à choisir.
func trueColor24() setupScreen {
	return setupScreen{
		root: 0x0000012C, rootVisual: 0x00000021, black: 0, depth: 24,
		visuals: []setupVisual{
			{id: 0x00000020, redMask: 0xF800, greenMask: 0x07E0, blueMask: 0x001F},
			{id: 0x00000021, redMask: 0xFF0000, greenMask: 0x00FF00, blueMask: 0x0000FF},
		},
	}
}

// TestParseSetupTellsDepthFromBitsPerPixel verrouille le piège le plus cher du jalon :
// la profondeur et la place d'un pixel sur la connexion sont deux nombres différents.
//
// Une profondeur de 24 se transmet presque toujours sur 32 bits, et les deux chiffres
// sont dans le même handshake à dix octets d'écart. Les confondre ne lève aucune
// erreur : le serveur lit ce qu'il attend, il en manque un quart, et l'image sort en
// escalier diagonal — le symptôme exact d'un défaut de projection ou d'ancre, qu'on
// irait chercher à l'autre bout du moteur.
func TestParseSetupTellsDepthFromBitsPerPixel(t *testing.T) {
	block := setupBlockOf("essai", []setupFormat{{1, 1}, {24, 32}}, trueColor24())

	s, err := parseSetup(block, 0)
	if err != nil {
		t.Fatalf("handshake refusé : %v", err)
	}

	if s.depth != 24 {
		t.Errorf("profondeur %d, attendu 24", s.depth)
	}
	if s.bitsPerPixel != 32 {
		t.Errorf("place par pixel %d, attendu 32 — c'est la table des formats qui le dit, pas la profondeur", s.bitsPerPixel)
	}
}

// TestParseSetupTakesTheMasksOfTheRootVisual vérifie que les masques retenus sont ceux
// du visuel de la racine, et non ceux du premier visuel rencontré.
//
// C'est de ces masques et de l'ordre des octets d'image que se dérive la conversion des
// pixels, jamais de l'architecture de la machine. Prendre ceux du premier visuel
// donnerait ici du seize bits là où l'écran en fait vingt-quatre, et les couleurs
// sortiraient fausses sans qu'aucune erreur ne soit levée.
func TestParseSetupTakesTheMasksOfTheRootVisual(t *testing.T) {
	block := setupBlockOf("essai", []setupFormat{{24, 32}}, trueColor24())

	s, err := parseSetup(block, 0)
	if err != nil {
		t.Fatalf("handshake refusé : %v", err)
	}

	if s.visual != 0x00000021 {
		t.Errorf("visuel racine %#x, attendu 0x21", s.visual)
	}
	if s.redMask != 0xFF0000 || s.greenMask != 0x00FF00 || s.blueMask != 0x0000FF {
		t.Errorf("masques %#x/%#x/%#x, attendus ceux du visuel racine", s.redMask, s.greenMask, s.blueMask)
	}
}

// TestParseSetupChoosesTheScreenAsked vérifie que le numéro d'écran de `DISPLAY` est
// suivi.
//
// Les écrans ne sont pas à offset fixe : chacun porte une liste de visuels de longueur
// variable, donc le second ne s'atteint qu'en traversant le premier. Un parcours faux
// rendrait des champs lus au milieu d'une structure, c'est-à-dire des valeurs
// plausibles et fausses.
func TestParseSetupChoosesTheScreenAsked(t *testing.T) {
	second := trueColor24()
	second.root = 0x000002BC
	second.rootVisual = 0x00000020
	second.visuals = []setupVisual{{id: 0x00000020, redMask: 0xFF0000, greenMask: 0x00FF00, blueMask: 0x0000FF}}

	block := setupBlockOf("essai", []setupFormat{{24, 32}}, trueColor24(), second)

	s, err := parseSetup(block, 1)
	if err != nil {
		t.Fatalf("handshake refusé : %v", err)
	}

	if s.root != 0x000002BC {
		t.Errorf("racine %#x, attendu celle du second écran", s.root)
	}
	if s.visual != 0x00000020 {
		t.Errorf("visuel %#x, attendu celui du second écran", s.visual)
	}
}

// TestParseSetupReadsTheResourceRange vérifie la plage d'identifiants et la taille
// maximale d'une requête.
//
// Les deux sont des concessions du serveur : créer une fenêtre hors de la plage est
// refusé, et dépasser la taille coupe la connexion. Ni l'une ni l'autre ne se devine.
func TestParseSetupReadsTheResourceRange(t *testing.T) {
	block := setupBlockOf("essai", []setupFormat{{24, 32}}, trueColor24())

	s, err := parseSetup(block, 0)
	if err != nil {
		t.Fatalf("handshake refusé : %v", err)
	}

	if s.idBase != 0x04000000 || s.idMask != 0x001FFFFF {
		t.Errorf("plage d'identifiants %#x/%#x, attendue 0x04000000/0x001FFFFF", s.idBase, s.idMask)
	}
	// La longueur maximale est annoncée en mots de quatre octets, et gardée en
	// octets : c'est dans cette unité que la comparaison avec une image se fera.
	if s.maxRequest != 4*0x40 {
		t.Errorf("requête maximale %d octets, attendu %d", s.maxRequest, 4*0x40)
	}
	if s.vendor != "essai" {
		t.Errorf("nom du serveur %q, attendu %q", s.vendor, "essai")
	}
}

// TestParseSetupRefusesWhatItCannotUse vérifie que les handshakes inutilisables sont
// refusés au lieu de rendre des champs lus au hasard.
func TestParseSetupRefusesWhatItCannotUse(t *testing.T) {
	cases := []struct {
		name   string
		block  []byte
		screen int
	}{
		{
			"un écran que le serveur n'a pas",
			setupBlockOf("essai", []setupFormat{{24, 32}}, trueColor24()),
			4,
		},
		{
			"aucun format pour la profondeur de sa propre racine",
			setupBlockOf("essai", []setupFormat{{1, 1}, {16, 16}}, trueColor24()),
			0,
		},
		{
			"un visuel racine absent des visuels décrits",
			setupBlockOf("essai", []setupFormat{{24, 32}}, setupScreen{
				root: 1, rootVisual: 0x999, depth: 24,
				visuals: []setupVisual{{id: 0x20, redMask: 0xFF0000}},
			}),
			0,
		},
		{
			"un bloc trop court pour porter ses champs fixes",
			make([]byte, 16),
			0,
		},
		{
			"un bloc qui s'arrête dans le nom du serveur",
			setupBlockOf("essai", []setupFormat{{24, 32}}, trueColor24())[:34],
			0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseSetup(c.block, c.screen); err == nil {
				t.Error("handshake accepté")
			}
		})
	}
}

// TestReadSetupReportsWhatTheServerSaid vérifie que le refus et l'exigence
// d'autorisation rendent la raison écrite par le serveur.
//
// C'est le message le plus utile de tout le backend : il distingue « il n'y a pas de
// serveur » de « le serveur est là et n'a pas voulu de notre cookie », ce qui se
// corrige en trouvant le bon fichier plutôt qu'en cherchant un défaut dans le client.
func TestReadSetupReportsWhatTheServerSaid(t *testing.T) {
	cases := []struct {
		name   string
		answer byte
		reason string
	}{
		{"un refus motivé", setupFailed, "Client is not authorized"},
		{"une autorisation exigée", setupAuthenticate, "no protocol specified"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rest := append([]byte(c.reason), make([]byte, pad(len(c.reason)))...)

			head := make([]byte, 8)
			head[0] = c.answer
			head[1] = byte(len(c.reason))
			order.PutUint16(head[6:], uint16(len(rest)/4))

			_, err := readSetup(bytes.NewReader(append(head, rest...)))
			if err == nil {
				t.Fatal("réponse acceptée")
			}
			if !strings.Contains(err.Error(), c.reason) {
				t.Errorf("le message ne porte pas la raison du serveur : %v", err)
			}
		})
	}
}

// TestReadSetupKeepsOnlyTheVariableBlock vérifie qu'une réussite rend le bloc qui suit
// l'en-tête, et lui seul.
//
// Les huit octets d'en-tête ne portent rien d'utile en cas de réussite, et les laisser
// décalerait tous les offsets du décodage de huit — ce qui ne lèverait aucune erreur et
// rendrait des champs plausibles.
func TestReadSetupKeepsOnlyTheVariableBlock(t *testing.T) {
	block := setupBlockOf("essai", []setupFormat{{24, 32}}, trueColor24())

	head := make([]byte, 8)
	head[0] = setupSuccess
	order.PutUint16(head[6:], uint16(len(block)/4))

	got, err := readSetup(bytes.NewReader(append(head, block...)))
	if err != nil {
		t.Fatalf("handshake refusé : %v", err)
	}
	if !bytes.Equal(got, block) {
		t.Errorf("bloc de %d octets rendu, %d attendus", len(got), len(block))
	}
}
