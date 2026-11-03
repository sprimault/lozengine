// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le clavier X11 ajoute à la table du noyau : le décalage de huit que son
// protocole impose, et la relecture de l'état entier au moment où la fenêtre prend le
// focus.
//
// La correspondance elle-même vit dans [evdevKeyOf], que les familles Linux partagent —
// un code de touche X11 **est** un code du noyau décalé, et la table est donc une table
// evdev. Ce qui reste ici est ce qui ne vaut que pour ce protocole.
//
// **Il faut résister à la symétrie avec Windows, et c'est tout l'enjeu de ce fichier.**
// Deux dispositifs que le backend Windows a dû construire n'ont ici aucune
// contrepartie : le pavé numérique ne se devine pas d'après un verrou, parce qu'evdev
// le distingue nativement du bloc directionnel ; et l'état du clavier ne se relit pas
// touche par touche, parce que le serveur l'envoie entier de lui-même.

package backend

import "github.com/sprimault/lozengine/input"

// keycodeShift est l'écart entre un code de touche X11 et le code evdev du noyau.
//
// Le protocole réserve les codes 0 à 7 — il veut que les codes valides commencent à
// huit — et le pilote de clavier y pose les codes du noyau décalés d'autant. C'est le
// seul calcul de toute la traduction pour la grande majorité des touches.
const keycodeShift = 8

// x11KeyOf traduit un code de touche X11 vers la place du moteur, et dit si elle existe.
//
// **Le refus sous huit est propre à ce protocole et reste donc ici.** X11 veut que les
// codes valides commencent à huit et le pilote y pose les codes du noyau décalés
// d'autant ; en evdev, les codes 1 à 8 sont Échap et les chiffres 1 à 7. Porter ce refus
// dans [evdevKeyOf] perdrait ces huit touches en silence sous les deux autres familles.
//
// **La touche de pause n'est traduite nulle part, et c'est une correspondance refusée
// plutôt qu'oubliée.** Le jeu 1 l'envoyait en trois octets et Windows ne la rapporte pas
// uniformément : poser une équivalence la rendrait lisible sous X11 et pas sous Windows,
// pour la même valeur. Une place qui ne veut pas dire la même chose partout vaut moins
// qu'une place absente, puisque c'est l'universalité de la convention qui fait sa valeur.
func x11KeyOf(code byte) (input.Key, bool) {
	if code < keycodeShift {
		return 0, false
	}
	return evdevKeyOf(uint32(code) - keycodeShift)
}

// relearnKeys reprend l'état du clavier depuis le vecteur de bits que le serveur
// envoie.
//
// **C'est ce qui remplace la relecture touche par touche du backend Windows**, et
// l'écart de coût est le fait marquant du clavier X11 : là-bas la reprise du focus
// interroge le système pour chacune des places possibles, ici le serveur envoie l'état
// complet de lui-même après chaque prise de focus — à condition de l'avoir demandé, ce
// que le masque d'événements fait depuis le lot qui a créé la fenêtre.
//
// L'oubli préalable est voulu : ce que le vecteur ne nomme pas n'est pas enfoncé, et
// repartir de l'état réel est précisément l'objet de cet événement. Les touches tenues
// se rallument donc comme des appuis neufs, ce qui est aussi le choix de Windows.
//
// Le vecteur couvre les codes à partir de huit, un bit par touche : l'octet n porte
// les codes 8+8n à 15+8n, du bit de poids faible au bit de poids fort.
func (w *x11Window) relearnKeys(bitmap []byte) {
	w.state.Forget()

	for at, mask := range bitmap {
		for bit := 0; bit < 8; bit++ {
			if mask&(1<<bit) == 0 {
				continue
			}
			code := keycodeShift + 8*at + bit
			if code > 255 {
				// Un code de touche tient sur un octet, donc le vecteur ne peut
				// pas en porter plus de trente et un. Au-delà, c'est le serveur
				// qui parle d'autre chose que de ce protocole.
				return
			}
			if k, ok := x11KeyOf(uint8(code)); ok {
				w.state.Press(k)
			}
		}
	}
}
