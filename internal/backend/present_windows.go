// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La remise du tampon à GDI, et la conversion qu'elle demande : le moteur écrit le
// rouge dans l'octet de poids faible, le système le veut en troisième position, et
// c'est ici que l'échange se fait.

package backend

import (
	"syscall"
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// toBGRA écrit dans dst les pixels de src avec le rouge et le bleu échangés, et rend
// dst ajusté à la longueur voulue.
//
// **Les composantes se lisent et le mot se compose, plutôt que de voir le tampon
// comme des entiers.** Une conversion de pointeur irait plus vite — 1,06 ms contre
// 1,51 à 1920×1080, mesuré le 29 octobre 2026 en quatre passes dont une reprise de la
// base — mais l'écart est de 0,45 ms sur un budget d'image de 16,7, là où le seuil qui
// autorise à optimiser le chemin de rendu est de 10 % pondérés. Le backend X11 a
// refusé la même conversion pour la même raison, et deux backends qui font le même
// échange n'ont pas à le faire de deux façons.
//
// **Un mot par pixel, et non quatre octets** : composer l'entier coûte 1,51 ms là où
// écrire les quatre composantes séparément en coûte 1,64, et c'est un mot de
// trente-deux bits que l'appel système relit de toute façon.
//
// **L'octet de poids fort part tel quel.** En `BI_RGB` sur trente-deux bits le système
// l'ignore, mais le recopier ne coûte rien de plus que de poser un zéro, et un tampon
// dont l'alpha survit se relit en test comme le moteur l'a peint.
func toBGRA(dst []uint32, src []render.Color) []uint32 {
	if cap(dst) < len(src) {
		dst = make([]uint32, len(src))
	}
	dst = dst[:len(src)]

	for i, c := range src {
		dst[i] = uint32(c.B) | uint32(c.G)<<8 | uint32(c.R)<<16 | uint32(c.A)<<24
	}
	return dst
}

// Present pose le tampon à l'écran.
//
// **Les pixels sont recopiés dans l'ordre du système avant de partir, et c'est la
// forme la moins chère des deux.** Le tampon que le noyau rend est du RVBA à alpha
// prémultiplié, disposition qui ne se négocie pas ; GDI, lui, ne recopie en bloc que
// la sienne. Lui décrire la nôtre par des masques est permis et ne demande aucune
// copie, mais le fait alors convertir pixel par pixel : 22,1 ms à 1920×1080 contre
// 0,47, soit un facteur 47 — et la copie faite ici coûte 1,5 ms, donc 2,0 ms en tout.
// Le détail de la mesure est sur [biRGB].
//
// C'est la frontière du noyau qui se paie, et elle était prévue ainsi : chaque backend
// pose ce tampon comme sa plateforme l'exige et la conversion est à lui. X11 fait le
// même échange dans [pixelOrder.appendPixels], le navigateur voudra du non
// prémultiplié.
//
// Sans erreur, et le contrat le veut ainsi pour une raison mesurée : sur une fenêtre
// réduite, l'appel rend le nombre de lignes demandé alors que rien n'est peint. Il n'y
// a donc pas d'échec à rapporter, seulement une présentation qui n'a pas eu lieu — et
// personne à qui s'en plaindre soixante fois par seconde.
func (w *window) Present(t *render.Buffer) {
	if w.dc == 0 || t == nil || len(t.Pixels) == 0 {
		return
	}

	// Le tampon de sortie est gardé d'une image à l'autre : le reconstruire par image
	// allouerait huit mégaoctets à 1080p dans le chemin de rendu.
	w.bgra = toBGRA(w.bgra, t.Pixels)

	w.info.width = int32(t.Width)
	w.info.height = -int32(t.Height)

	// Les arguments passent par une tranche gardée, et c'est le seul appel du backend
	// à le faire : douze arguments variadiques en allouent une à chaque présentation,
	// donc soixante fois par seconde. Ailleurs, `LazyProc.Call` reste préférable —
	// six nanosecondes de plus et rien sur le tas.
	w.args[0] = w.dc
	w.args[3] = uintptr(t.Width)
	w.args[4] = uintptr(t.Height)
	w.args[8] = uintptr(t.Height)
	w.args[9] = uintptr(unsafe.Pointer(&w.bgra[0]))
	w.args[10] = uintptr(unsafe.Pointer(&w.info))

	syscall.SyscallN(setDIBitsToDevice.Addr(), w.args...)
}
