// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'icône de la fenêtre sous Windows : deux tailles demandées au système, deux
// poignées construites depuis les pixels du jeu, et le message qui les pose.

package backend

import (
	"syscall"
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// setIcons pose la grande et la petite icône de la fenêtre, et rend les poignées à
// libérer.
//
// Deux tailles plutôt qu'une, parce que le système les emploie à deux endroits — la
// barre des tâches et le coin du cadre — et qu'il réduirait lui-même la grande pour
// obtenir l'autre. La réduction d'un dessin fait pixel par pixel se voit, et le jeu
// qui fournit les deux tailles n'a pas à la subir.
//
// Un échec ne fait rien : une fenêtre sans icône porte celle du système, ce qui n'est
// pas un défaut dont un jeu doive s'occuper au lancement.
func (w *window) setIcons(icons []*render.Buffer) {
	if len(icons) == 0 {
		return
	}

	for _, size := range [...]struct {
		metric uintptr
		which  uintptr
	}{
		{smCXIcon, iconBig},
		{smCXSmallIcon, iconSmall},
	} {
		side, _, _ := getSystemMetrics.Call(size.metric)
		if side == 0 {
			continue
		}

		icon := createIcon(pickIcon(icons, uint16(side)))
		if icon == 0 {
			continue
		}

		sendMessageW.Call(uintptr(w.hwnd), wmSetIcon, size.which, icon)
		w.icons = append(w.icons, icon)
	}
}

// createIcon construit la poignée d'icône que le système attend depuis les pixels du
// jeu, et rend zéro quand elle n'a pas pu l'être.
//
// Les pixels sont posés **non prémultipliés**, et c'est un fait mesuré plutôt que
// documenté : la documentation de l'appel ne dit rien de l'alpha, et une icône créée
// avec un pixel à moitié transparent ressort telle quelle d'un aller-retour par le
// système — il ne convertit donc pas, et c'est au dessin que le shell prémultiplie.
// Lui passer du prémultiplié assombrirait tout ce qui n'est pas opaque.
//
// Le masque est un bitmap d'un bit laissé à zéro. Un DIB à trente-deux bits porte son
// alpha, donc le masque ne sert plus à découper la forme, mais l'appel en exige un.
func createIcon(icon *render.Buffer) uintptr {
	if icon == nil {
		return 0
	}

	header := bitmapInfoHeader{
		size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		width:    int32(icon.Width),
		height:   int32(icon.Height),
		planes:   1,
		bitCount: 32,
	}

	var bits *uint32
	color, _, _ := createDIBSection.Call(0, uintptr(unsafe.Pointer(&header)),
		dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if color == 0 {
		return 0
	}
	defer deleteObject.Call(color)

	// La hauteur est positive, donc le système lit les lignes du bas vers le haut
	// là où le moteur les compte depuis le haut. La présentation s'en tire par une
	// hauteur négative ; ici elle ne peut pas, l'appel refusant un DIB inversé, donc
	// la recopie remet les lignes dans l'ordre qu'il attend.
	rows := straightARGB(icon)
	width := int(icon.Width)
	out := unsafe.Slice(bits, len(rows))
	for y := 0; y < int(icon.Height); y++ {
		copy(out[y*width:(y+1)*width], rows[(int(icon.Height)-1-y)*width:])
	}

	mask, _, _ := createBitmap.Call(uintptr(icon.Width), uintptr(icon.Height), 1, 1, 0)
	if mask == 0 {
		return 0
	}
	defer deleteObject.Call(mask)

	info := iconInfo{icon: 1, mask: syscall.Handle(mask), color: syscall.Handle(color)}
	handle, _, _ := createIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	return handle
}

// releaseIcons rend au système les poignées d'icône que la fenêtre a prises.
//
// Le système copie les bitmaps au moment de créer l'icône, donc les deux temporaires
// partent dès createIcon ; la poignée, elle, vit aussi longtemps que la fenêtre.
func (w *window) releaseIcons() {
	for _, icon := range w.icons {
		destroyIcon.Call(icon)
	}
	w.icons = nil
}

// iconInfo décrit une icône au système : son point chaud, son masque et ses couleurs.
//
// Le champ icon vaut un pour une icône et zéro pour un curseur, auquel cas les deux
// suivants seraient le point chaud. Une icône n'en a pas.
type iconInfo struct {
	icon        int32
	xHotspot    uint32
	yHotspot    uint32
	mask, color syscall.Handle
}
