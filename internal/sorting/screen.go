// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le relevé de ce qui est posé en pixels de l'image : les seules choses qui entrent
// dans la liste sans passer par la projection.

package sorting

import "github.com/sprimault/lozengine/scene"

// collectScreen relève ce que la scène porte en pixels de l'image.
//
// **L'origine de la vue s'ajoute avant de poser**, parce que le placement la retranche
// pour tout le monde : une pièce d'écran doit rester où le jeu l'a mise quand la caméra
// bouge, donc on compense exactement ce que le chemin commun va retirer. Passer par ce
// chemin plutôt que d'en écrire un second garde le découpage, le remplissage du quad
// et l'ancre en un seul endroit.
//
// **Le rang de pose va dans la profondeur, et c'est ce qui rend le tri rapide ici.**
// Sans lui, tous les quads d'une ligne de texte partageraient couche, étage, hauteur
// et planche, donc une seule et même clé : ils tomberaient tous dans le même groupe de
// départage, qui est un tri par insertion, et cent glyphes y coûteraient dix mille
// comparaisons. Avec lui, les clés diffèrent, le tri par paquets les range, et l'ordre
// obtenu est l'ordre de pose — ce que [scene.Scene.PlaceScreen] promet.
//
// La hauteur reste nulle : une pièce d'écran n'est pas au-dessus du sol, elle est
// ailleurs. C'est la couche qui dit où elle se range, et elle seule.
func (t *Sorter) collectScreen() {
	pieces := scene.ScreenPieces(t.scene)
	if len(pieces) == 0 {
		return
	}

	for i, p := range pieces {
		depth := uint32(min(i, MaxDepth))

		// L'ancre du chemin commun est le pied de l'image, centré ; une pièce d'écran
		// se place par son coin haut-gauche, qui est ce qu'on sait mesurer d'une
		// interface. On vise donc le point dont cette ancre tire ce coin, et le
		// décalage d'ancre de l'aspect continue de s'ajouter par-dessus.
		//
		// **Le compenser ici et non ailleurs est ce qui garde le découpage juste** :
		// une pièce posée en haut de l'image serait sinon remontée d'une hauteur
		// d'image au-dessus du bord, et écartée en silence.
		ex := t.viewX + p.X + int32(p.Width)/2
		ey := t.viewY + p.Y + int32(p.Height)

		t.place(p.Aspect, ex, ey, depth, 0, uint32(i), 0)
	}
}
