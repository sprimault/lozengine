// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La présentation d'une image dans la page : une vue d'octets sur le tampon du moteur,
// un `ImageData` posé dessus, et un seul appel par image.

package backend

import (
	"runtime"
	"syscall/js"
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// linearMemory est le nom sous lequel la page expose la mémoire du module au moteur.
//
// **La page doit poser ce global elle-même**, après avoir instancié le module :
// `globalThis.lozengineMemory = inst.exports.mem`. C'est une ligne, et elle fait la
// différence entre une présentation qui copie huit mégaoctets par image et une qui
// n'en copie aucun.
//
// **L'objet attendu est le `WebAssembly.Memory`, pas son `ArrayBuffer`** : celui-ci
// change d'identité quand la mémoire croît, donc un global qui le porterait se
// périmerait sans prévenir. Le tampon se relit sur l'objet à chaque reconstruction.
//
// **Et il se lit de l'API de WebAssembly, jamais de l'objet du fichier de liaison.**
// `go.mem` est un interne de `wasm_exec.js` qui a déjà changé de place d'une version de
// la chaîne à l'autre ; `inst.exports` est la spécification.
//
// Son absence n'est pas une erreur : la présentation copie alors, et le jeu tourne. Elle
// se voit au relevé du temps par image, qui est le lot qui mesure.
const linearMemory = "lozengineMemory"

// presentation garde d'une image à l'autre ce qu'une image ne doit pas reconstruire.
//
// Rien ici n'est alloué par image dans le régime courant : la vue et l'`ImageData` sont
// montés une fois par taille de canvas, et la présentation n'est plus qu'un appel.
type presentation struct {
	// data est l'`ImageData` que `putImageData` pose, et view la vue d'octets sur
	// laquelle il est construit. Les deux vont ensemble : un `ImageData` ne se
	// réassocie pas à un autre tampon.
	data js.Value
	view js.Value

	// memory est le `WebAssembly.Memory` du module quand la page l'expose, et la
	// valeur indéfinie sinon. C'est elle qui décide du régime, et elle ne change pas
	// pendant la vie de la fenêtre.
	memory js.Value

	// addr et length sont l'adresse du tampon dans la mémoire du module et sa longueur
	// en octets. **Ce sont les deux gardes du régime aliasé, et aucune ne suffit** :
	// la croissance du tas détache le tampon sans changer l'adresse, là qu'une
	// réallocation de la surface change l'adresse sans détacher. Le second cas est le
	// plus dangereux des deux — la pose réussit et la page affiche l'image précédente,
	// en silence, après chaque redimensionnement.
	addr   uintptr
	length int

	// width et height sont la taille de l'`ImageData` monté. Un canvas redimensionné
	// en demande un autre, sa taille étant figée à la construction.
	width, height uint16

	// pinner retient le tampon là où il est tant que la page en détient une vue.
	//
	// La documentation d'`unsafe.Pointer` déclare invalide ce qu'on fait ici sans lui —
	// « the garbage collector will not update that uintptr's value if the object
	// moves » —, et l'épinglage remplace « le ramasse-miettes de Go ne déplace rien
	// aujourd'hui » par une garantie du langage.
	pinner runtime.Pinner
}

// Present pose le tampon dans la page, tel quel.
//
// **Aucune conversion de disposition.** Un contexte `{alpha: false}` ignore le quatrième
// octet par la spécification, et [render.Color] est déjà l'ordre qu'`ImageData` attend :
// les octets du moteur traversent inchangés. Windows échange le rouge et le bleu, X11
// suit l'ordre du visuel qu'il a choisi.
//
// Ce n'est pas ce qui distingue la page des autres, et l'annoncer ainsi s'est périmé :
// Wayland et DRM/KMS ne convertissent pas non plus dès qu'ils obtiennent `XB24`. Ce
// qu'elle a de particulier est la **copie**, qu'une vue posée sur la mémoire du module
// supprime en plus.
//
// Un tampon plus court que sa propre taille ne présente rien. Ce n'est pas de la
// défensive : la longueur entre dans la construction d'un `ImageData`, et une
// incohérence y lève une exception de la page, qu'un module Go ne rattrape pas. Un hôte
// qui remplit un [render.Buffer] lui-même peut écrire ce couple, là que `NewBuffer` ne
// le peut pas.
func (w *window) Present(t *render.Buffer) {
	if w.closed || t == nil {
		return
	}

	pixels := int(t.Width) * int(t.Height)
	if pixels == 0 || len(t.Pixels) < pixels {
		return
	}

	w.frame.refresh(t, pixels)
	w.ctx.Call("putImageData", w.frame.data, 0, 0)
}

// refresh porte le tampon à la page, en reconstruisant ce qui ne vaut plus.
//
// Dans le régime aliasé et à taille constante, elle ne fait rien du tout : la vue
// désigne déjà les octets que le rastériseur vient d'écrire. C'est ce qui retire la
// moitié du chemin de présentation, et c'est pourquoi les gardes se vérifient à chaque
// image plutôt que de se supposer.
func (p *presentation) refresh(t *render.Buffer, pixels int) {
	addr := uintptr(unsafe.Pointer(&t.Pixels[0]))
	length := pixels * 4

	if p.stale(addr, length, t) {
		p.rebuild(addr, length, t)
	}
	if !p.aliased() {
		js.CopyBytesToJS(p.view, bytesOf(t.Pixels[:pixels]))
	}
}

// aliased dit si la vue désigne la mémoire du module plutôt qu'un tampon de la page.
func (p *presentation) aliased() bool {
	return !p.memory.IsUndefined()
}

// stale dit si la vue et l'`ImageData` doivent être remontés.
//
// Les deux premiers cas valent dans les deux régimes ; les deux derniers sont les gardes
// du régime aliasé, et leur ordre compte peu — ce qui compte est qu'elles soient toutes
// les deux là. Le dernier appelle la page une fois par image, et c'est le prix de la
// seule garde qui voit un tampon détaché.
func (p *presentation) stale(addr uintptr, length int, t *render.Buffer) bool {
	switch {
	case p.data.IsUndefined():
		return true
	case p.width != t.Width || p.height != t.Height:
		return true
	case !p.aliased():
		return false
	case p.addr != addr || p.length != length:
		return true
	default:
		return p.view.Get("byteLength").Int() == 0
	}
}

// rebuild monte une vue sur le tampon courant et l'`ImageData` qui va avec.
//
// L'`ArrayBuffer` se relit sur l'objet `WebAssembly.Memory` et ne se garde pas : c'est
// lui que la croissance du tas remplace, et le garder ferait construire la vue suivante
// sur un tampon détaché.
func (p *presentation) rebuild(addr uintptr, length int, t *render.Buffer) {
	p.release()

	clamped := js.Global().Get("Uint8ClampedArray")
	if p.aliased() {
		p.pinner.Pin(&t.Pixels[0])
		p.view = clamped.New(p.memory.Get("buffer"), addr, length)
	} else {
		p.view = clamped.New(length)
	}

	p.data = js.Global().Get("ImageData").New(p.view, int(t.Width), int(t.Height))
	p.addr, p.length = addr, length
	p.width, p.height = t.Width, t.Height
}

// release libère ce que la présentation retient, et laisse de quoi tout remonter.
//
// Le régime n'est pas oublié : `memory` survit, parce qu'il décrit la page et non le
// tampon. Appeler deux fois est sans effet.
func (p *presentation) release() {
	p.pinner.Unpin()
	p.view, p.data = js.Undefined(), js.Undefined()
	p.addr, p.length = 0, 0
	p.width, p.height = 0, 0
}
