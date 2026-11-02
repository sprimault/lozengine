// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Dans quelle colonne du relevé de recouvrement un quad compte, quand sa voie de pose
// ne dit pas ce qu'il est.

package scene

// Masking dit comment un quad compte dans le relevé de recouvrement : comme du décor,
// comme une entité, ou selon la voie par laquelle il a été posé.
//
// Ses valeurs sont [MaskAsPlaced], [MaskAsDecor] et [MaskAsEntity].
//
// # Pourquoi ce réglage existe
//
// Le relevé sépare ce que le décor cache de ce que les entités cachent —
// [render.Overlap.ByDecor], [render.Overlap.CoveredByDecor] et la bascule
// `TintByDecor` des options de révélation. Sans ce champ, cette séparation suit la voie
// de pose : ce qui vient de [Scene.PlaceCell] compte comme décor, ce qui vient de
// [Scene.Place] compte comme entité.
//
// **Or la voie de pose est aussi un arbitrage de coût**, et la godoc de [MaxTiers]
// recommande d'en changer : une strate pleine ajoute un quad par cellule visible, donc
// quelques objets épars sur une grande carte se décrivent mieux en entités. Un jeu qui
// suit cette recommandation reclassait d'un coup tout ce qu'il déplaçait, alors que rien
// de ce que ces objets **sont** n'avait changé. Relevé sur un jeu réel, où sept cent
// vingt-deux quads masquants ont changé de colonne en même temps.
//
// La conséquence était observable : une chose cachée à la fois par un quad posé en
// cellule et par un quad posé en entité ne vérifiait plus
// `CoveredByDecor == Covered`, et un hôte qui lit cette distinction pour choisir entre
// deux rendus basculait de branche sans l'avoir décidé.
//
// # Ce qu'il ne touche pas, et c'est ce qui compte le plus
//
// **L'ordre de dessin ne change pas.** La voie de pose décide aussi du rang
// d'empilement — une entité passe devant la cellule qu'elle foule, quel que soit le
// côté —, et ce rang est un invariant du moteur. Ce champ ne porte que le relevé : un
// muret déclaré décor se dessine exactement là où il se dessinait.
//
// **Et il ne dit pas si un quad compte, seulement dans quelle colonne.**
// [Aspect.Uncounted] l'emporte donc : un quad qui ne compte nulle part n'a pas de
// colonne. Les deux champs ne se recouvrent pas.
type Masking uint8

// Les trois façons dont un quad peut compter dans le relevé de recouvrement.
//
// Le bloc annote son type sur la première ligne : sans cela, `go doc` ne montre pas ces
// valeurs dans la documentation de [Masking].
const (
	// MaskAsPlaced laisse la voie de pose décider, et c'est la valeur zéro : ce qui
	// vient de [Scene.PlaceCell] compte comme décor, le reste comme entité. C'est ce
	// que faisait tout le moteur avant que ce champ existe, donc un jeu qui ne le
	// renseigne pas ne voit rien changer.
	MaskAsPlaced Masking = iota

	// MaskAsDecor compte le quad comme du décor quelle que soit sa voie de pose. C'est
	// ce qu'un jeu pose sur un muret, une haie ou un véhicule garé qu'il décrit en
	// entité pour ne pas ouvrir une strate sur toute la carte.
	MaskAsDecor

	// MaskAsEntity compte le quad comme une entité quelle que soit sa voie de pose.
	// C'est l'inverse, et il sert ce qu'un jeu range dans une strate parce que le décor
	// l'y range naturellement, mais qui se comporte comme une créature pour qui le
	// regarde — une plante mouvante, une tourelle.
	MaskAsEntity
)
