// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'une fenêtre pose autour de son image : le titre de la page, son icône, et le
// curseur au-dessus du canvas.

package backend

import (
	"syscall/js"

	"github.com/sprimault/lozengine/render"
)

// faviconSide est la taille d'icône que le moteur demande pour une page.
//
// **Aucune spécification n'en impose**, contrairement aux deux systèmes : un navigateur
// prend ce que le lien lui donne et le réduit lui-même. Trente-deux est la taille que les
// onglets affichent sur un écran dense, et [pickIcon] rend la plus proche de celles qu'un
// jeu fournit — le moteur ne redimensionne jamais.
const faviconSide = 32

// dress pose le titre, l'icône et le curseur que les options demandent.
//
// **Tout est posé à l'ouverture et rien n'est repris ensuite** : ces trois réglages ne
// changent pas pendant la vie d'une fenêtre, les deux autres familles les traitant de
// même. Une page qui tiendrait à les changer le fait elle-même, et le moteur ne les
// réécrit pas par-dessus.
func (w *window) dress(o Options) {
	js.Global().Get("document").Set("title", o.Title)

	if o.HideCursor {
		// Sur le canvas et non sur la page : le curseur doit disparaître au-dessus du
		// jeu, pas au-dessus de ce que la page met autour.
		w.canvas.Get("style").Set("cursor", "none")
	}

	w.setIcon(pickIcon(o.Icons, faviconSide))
}

// setIcon pose l'icône de la page, et ne fait rien quand le jeu n'en fournit aucune.
//
// **La page encode, pas le moteur.** Un lien d'icône veut une image dans un format que
// le navigateur sait lire, et le moteur n'a pas d'encodeur à sa portée — `raster` ne fait
// pas partie de ce qu'un backend atteint, et déplacer cette frontière pour une icône
// serait cher payé. Un canvas hors écran reçoit donc les pixels et rend l'adresse de
// données par `toDataURL`, ce que toute page sait faire.
//
// Le canvas n'est jamais attaché au document : il ne sert qu'à encoder, et l'attacher
// ferait apparaître un carré dans la page.
//
// **L'élément de lien est réutilisé s'il existe déjà.** Une page qui porte sa propre
// icône la verrait sinon doubler, et c'est le dernier lien qui gagne — donc un moteur qui
// en ajoute un écraserait silencieusement ce qu'une page a voulu.
func (w *window) setIcon(icon *render.Buffer) {
	if icon == nil {
		return
	}

	doc := js.Global().Get("document")

	surface := doc.Call("createElement", "canvas")
	surface.Set("width", icon.Width)
	surface.Set("height", icon.Height)

	ctx := surface.Call("getContext", "2d")
	if !ctx.Truthy() {
		return
	}

	pixels := straightRGBA(icon)
	data := js.Global().Get("Uint8ClampedArray").New(len(pixels))
	js.CopyBytesToJS(data, pixels)
	ctx.Call("putImageData",
		js.Global().Get("ImageData").New(data, int(icon.Width), int(icon.Height)), 0, 0)

	link := doc.Call("querySelector", "link[rel~='icon']")
	if !link.Truthy() {
		link = doc.Call("createElement", "link")
		link.Set("rel", "icon")
		doc.Get("head").Call("appendChild", link)
	}
	link.Set("href", surface.Call("toDataURL", "image/png"))
}
