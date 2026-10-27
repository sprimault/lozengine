// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La touche désignée par sa place sur le clavier, et le découpage en mots de
// soixante-quatre bits qui tient cinq cent douze places sans rien allouer.

package input

// Key désigne une place sur le clavier, et non le caractère qu'elle produit.
//
// **La valeur est la convention du moteur, et non celle d'un système.** Elle coïncide
// avec les codes de balayage du jeu 1, ceux que les claviers rapportent depuis toujours
// et que Windows transmet tels quels ; chaque backend traduit vers elle ce que sa
// plateforme lui donne, et sous X11 cette traduction est une soustraction de huit. Un
// jeu lit donc les mêmes valeurs partout.
//
// Les noms que le moteur en donne portent la légende d'un clavier américain : c'est la
// convention de tout le monde, et la seule qui reste vraie quand la disposition change.
//
// Quatre-vingt-six places portent un nom : les lettres de [KeyA] à [KeyZ], les chiffres
// de [Key1] à [Key9] puis [Key0] dans l'ordre de la rangée, les quatre flèches,
// [KeyEscape], [KeyBackspace], [KeyTab], [KeyEnter] et [KeySpace], les modificateurs
// distingués à gauche et à droite, les douze touches de fonction, le bloc de
// navigation de [KeyHome] à [KeyDelete], et le pavé numérique de [KeyNumpad0] à
// [KeyNumLock]. Elles sont nommées ici parce qu'interroger le type ne montre que la
// première de chaque bloc, la documentation s'abrégeant ensuite.
//
// Une place sans nom se lit quand même, par sa valeur dans cette convention : elle
// désigne la même place sur toutes les plateformes, puisque c'est vers elle que les
// backends traduisent.
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
