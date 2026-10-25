// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package input

// Key désigne une place sur le clavier, et non le caractère qu'elle produit.
//
// La valeur est le code de balayage que le système rapporte, et les noms que le moteur
// en donne portent la légende d'un clavier américain : c'est la convention de tout le
// monde, et la seule qui reste vraie quand la disposition change.
//
// Les noms arrivent avec le backend qui lit le clavier. Ce type et les deux régimes de
// lecture existent avant eux parce que c'est le contrat qu'une fenêtre doit remplir,
// quelle que soit la plateforme.
type Key uint16

// keysMax est le nombre de places que l'état sait distinguer.
//
// Cinq cent douze parce qu'un code de balayage étendu tient sur neuf bits : le préfixe
// qui distingue la touche de droite de celle de gauche, ou le pavé numérique du bloc
// directionnel, ajoute un bit au code de base. Au-delà, il n'y a pas de touche.
//
// Le coût est de soixante-quatre octets par jeu de bits, donc cent vingt-huit par état
// d'image. Une tranche allouée serait plus petite sur un clavier ordinaire et coûterait
// une indirection par lecture, dans un chemin parcouru à chaque image.
const keysMax = 512

// words est le nombre d'entiers de soixante-quatre bits qu'un jeu de bits occupe.
const words = keysMax / 64

// locate rend le mot et le bit où une touche se lit, et si elle est dans les bornes.
//
// Hors bornes, les deux régimes rendent faux plutôt que de paniquer : une touche
// inconnue est une donnée du monde extérieur — un clavier exotique, un pilote bavard —
// et non une faute de programmation.
func locate(k Key) (word int, bit uint64, ok bool) {
	if k >= keysMax {
		return 0, 0, false
	}
	return int(k) / 64, 1 << (uint64(k) % 64), true
}
