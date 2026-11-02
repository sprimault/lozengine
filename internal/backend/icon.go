// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le choix d'une icône parmi celles qu'un jeu fournit, et la conversion de ses pixels.
// Les deux sont ici plutôt que dans un backend, parce qu'aucun des deux ne dépend
// d'une plateforme et que tous les backends en auront besoin.

package backend

import "github.com/sprimault/lozengine/render"

// pickIcon rend l'icône dont le côté approche le plus la taille demandée, et nil quand
// le jeu n'en fournit aucune.
//
// **Choisir une taille est propre à Windows**, qui veut une icône par emplacement — une
// grande pour Alt-Tab, une petite pour la barre de titre. X11 envoie au contraire toutes
// les tailles dans une seule propriété et laisse le gestionnaire de fenêtres prendre ce
// qui lui convient, donc il ne passe pas par ici. Ce qui est commun à tous est la
// conversion des pixels, par straightARGB.
//
// **Le moteur ne redimensionne pas**, et c'est la règle d'échelle qui l'impose : un
// agrandissement entier ne sait pas passer de 24 à 16 pixels, et une réduction douce
// brouillerait un dessin fait pixel par pixel. On prend donc la taille la plus proche
// et on laisse la plateforme réduire si elle y tient — mal, mais chez elle.
//
// À égalité d'écart, la plus grande gagne : réduire garde les contours, agrandir les
// invente.
func pickIcon(icons []*render.Buffer, side uint16) *render.Buffer {
	var best *render.Buffer
	var bestGap int

	for _, icon := range icons {
		if icon == nil || icon.Width == 0 || icon.Height == 0 {
			continue
		}

		gap := int(icon.Width) - int(side)
		if gap < 0 {
			gap = -gap
		}

		switch {
		case best == nil, gap < bestGap, gap == bestGap && icon.Width > best.Width:
			best, bestGap = icon, gap
		}
	}
	return best
}

// straightARGB rend les pixels d'une icône en mots ARGB à alpha **non** prémultiplié, la
// disposition que les plateformes attendent d'une icône.
//
// **C'est la valeur du mot qui compte, pas son image en mémoire** : l'alpha occupe les
// huit bits de poids fort, puis le rouge, le vert et le bleu. C'est ce que Windows lit
// dans un pixel de DIB et c'est aussi ce que la propriété `_NET_WM_ICON` d'X11 demande,
// donc les deux plateformes consomment ces mots tels quels. Nommer cette disposition
// BGRA décrirait l'ordre où les octets tombent sur une machine petit-boutienne, ce qui
// n'est ni ce qu'on écrit ni ce que les deux formats spécifient.
//
// Deux conversions en une passe. L'ordre des composantes d'abord : le moteur tient du
// RVBA, où le rouge vient en premier. L'alpha ensuite : le moteur prémultiplie pour que
// le mélange d'un pixel soit une multiplication, et une icône se décrit à l'envers.
//
// **La division arrondit au plus proche, et ce n'est pas du luxe.** Tronquer coûte un
// cran sur un tiers des valeurs à alpha partiel — un vert de 32 à alpha 64 ressort à
// 127 au lieu de 128 —, ce qui est la même perte que l'aller-retour par un PNG. La
// demi-division ajoutée avant de diviser la supprime là où elle peut l'être.
//
// Un pixel entièrement transparent n'a pas de couleur à retrouver et ressort nul.
func straightARGB(icon *render.Buffer) []uint32 {
	out := make([]uint32, len(icon.Pixels))

	for i, pixel := range icon.Pixels {
		if pixel.A == 0 {
			continue
		}

		alpha := uint32(pixel.A)

		out[i] = alpha<<24 |
			straight(pixel.R, alpha)<<16 |
			straight(pixel.G, alpha)<<8 |
			straight(pixel.B, alpha)
	}
	return out
}

// straightRGBA rend les pixels d'une icône en octets RVBA à alpha **non** prémultiplié.
//
// **Des octets et non des mots, contrairement à [straightARGB]**, et les deux coexistent
// parce que leurs destinataires ne demandent pas la même chose : une icône de système se
// décrit par la valeur d'un mot, là qu'un tampon de page se décrit par la suite de ses
// octets. Convertir l'un en l'autre dépendrait du boutisme de la machine, ce que le
// déterminisme du moteur interdit d'écrire.
//
// C'est la disposition qu'un `ImageData` attend, à l'alpha près : celui-ci veut de la
// couleur droite, là que le moteur prémultiplie. La division est celle de [straight],
// arrondi compris.
//
// Un pixel entièrement transparent ressort nul, ses quatre octets avec.
func straightRGBA(icon *render.Buffer) []byte {
	out := make([]byte, len(icon.Pixels)*4)

	for i, pixel := range icon.Pixels {
		if pixel.A == 0 {
			continue
		}

		alpha := uint32(pixel.A)
		out[i*4] = byte(straight(pixel.R, alpha))
		out[i*4+1] = byte(straight(pixel.G, alpha))
		out[i*4+2] = byte(straight(pixel.B, alpha))
		out[i*4+3] = pixel.A
	}
	return out
}

// straight rend une composante à sa pleine valeur, l'alpha qui la prémultipliait étant
// donné.
//
// Le plafond n'est pas de la défensive en trop : dans un prémultiplié correct une
// composante ne dépasse jamais son alpha, mais une planche qu'un jeu remplit lui-même
// peut porter n'importe quoi, et le quotient déborderait alors de son octet pour aller
// écraser les voisins. Un pixel faux resterait un pixel faux ; ici il en abîmerait
// trois autres.
func straight(channel uint8, alpha uint32) uint32 {
	value := (uint32(channel)*255 + alpha/2) / alpha
	if value > 255 {
		return 255
	}
	return value
}
