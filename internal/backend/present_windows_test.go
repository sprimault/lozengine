// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La présentation éprouvée sans écran, en la faisant peindre dans une image en
// mémoire : un contexte compatible est un contexte valide, donc le chemin complet se
// parcourt et les pixels se relisent.
//
// C'est ce qui attrape les deux défauts que rien d'autre ne verrait — des composantes
// échangées, et une image retournée de haut en bas.

package backend

import (
	"testing"
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// Les procédures que seule la destination d'essai emploie. Elles vivent ici parce que
// le backend n'en a pas besoin : les déclarer à côté des siennes laisserait du code
// que rien n'appelle.
var (
	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	createDIBSection   = gdi32.NewProc("CreateDIBSection")
	selectObject       = gdi32.NewProc("SelectObject")
	deleteObject       = gdi32.NewProc("DeleteObject")
	deleteDC           = gdi32.NewProc("DeleteDC")
)

// biRGB est la disposition que le système suppose sans masques : du bleu dans l'octet
// de poids faible. La destination d'essai l'emploie, pour que le test lise ce qu'un
// écran lirait.
const biRGB = 0

// destination monte une image en mémoire et rend son contexte avec ses pixels.
//
// Elle est volontairement décrite sans masques : c'est l'autre bout de la chaîne, et
// si la présentation annonçait mal la disposition de sa source, les deux ne
// s'accorderaient pas.
func destination(t *testing.T, width, height int32) (uintptr, []uint32) {
	t.Helper()

	dc, _, err := createCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatalf("CreateCompatibleDC : %v", err)
	}

	header := bitmapInfoHeader{
		size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		width:       width,
		height:      -height,
		planes:      1,
		bitCount:    32,
		compression: biRGB,
	}

	// Le système écrit l'adresse des pixels dans un pointeur déjà typé, plutôt que
	// dans un entier qu'il faudrait reconvertir : `go vet` refuse à bon droit qu'un
	// `uintptr` redevienne un pointeur, puisqu'il ne retient rien. Cette mémoire
	// appartient au système et vit jusqu'à `DeleteObject`.
	var bits *uint32
	bitmap, _, err := createDIBSection.Call(
		dc, uintptr(unsafe.Pointer(&header)), dibRGBColors,
		uintptr(unsafe.Pointer(&bits)), 0, 0,
	)
	if bitmap == 0 {
		deleteDC.Call(dc)
		t.Fatalf("CreateDIBSection : %v", err)
	}
	selectObject.Call(dc, bitmap)

	t.Cleanup(func() {
		deleteObject.Call(bitmap)
		deleteDC.Call(dc)
	})

	return dc, unsafe.Slice(bits, int(width*height))
}

// testWindow monte une fenêtre qui n'en est pas une : juste ce qu'il faut pour que la
// présentation parcoure son chemin, c'est-à-dire un contexte et ce qui se monte à
// l'ouverture.
func testWindow(dc uintptr) *window {
	return &window{dc: dc, info: newBitmapInfo(), args: newPresentArgs()}
}

// TestPresentKeepsTheComponentsInOrder vérifie que les composantes arrivent où il
// faut, et c'est le seul cas à le pouvoir.
//
// Un échange du rouge et du bleu ne casse rien, ne lève aucune erreur, et ne se voit
// qu'à l'image — sur une scène grise, il ne se voit même pas. Les couleurs du cas ont
// donc leurs trois composantes distinctes.
func TestPresentKeepsTheComponentsInOrder(t *testing.T) {
	dc, pixels := destination(t, 2, 1)

	buffer := render.NewBuffer(2, 1)
	buffer.Pixels[0] = render.Color{R: 0x40, G: 0x80, B: 0xC0, A: 0xFF}
	buffer.Pixels[1] = render.Color{R: 0x11, G: 0x22, B: 0x33, A: 0xFF}

	testWindow(dc).Present(buffer)

	want := []uint32{0x004080C0, 0x00112233}
	for i, got := range pixels {
		if got&0x00FFFFFF != want[i] {
			t.Errorf("pixel %d : %06x, attendu %06x", i, got&0x00FFFFFF, want[i])
		}
	}
}

// TestPresentDoesNotFlipTheImage vérifie le sens des lignes.
//
// Un DIB compte ses lignes depuis le bas, le moteur depuis le haut, et une hauteur
// positive suffirait à retourner toute l'image. Le défaut est spectaculaire mais
// invisible sur une image symétrique, donc le cas emploie deux lignes de couleurs
// différentes.
func TestPresentDoesNotFlipTheImage(t *testing.T) {
	dc, pixels := destination(t, 1, 2)

	buffer := render.NewBuffer(1, 2)
	buffer.Pixels[0] = render.Color{R: 0xFF, A: 0xFF} // la ligne du haut
	buffer.Pixels[1] = render.Color{B: 0xFF, A: 0xFF} // celle du bas

	testWindow(dc).Present(buffer)

	if pixels[0]&0x00FFFFFF != 0x00FF0000 {
		t.Errorf("ligne du haut : %06x, attendu ff0000 — l'image est retournée", pixels[0]&0x00FFFFFF)
	}
	if pixels[1]&0x00FFFFFF != 0x000000FF {
		t.Errorf("ligne du bas : %06x, attendu 0000ff", pixels[1]&0x00FFFFFF)
	}
}

// TestBitmapMasksMatchTheColourLayout relie les masques annoncés au système à la
// disposition réelle de la couleur du moteur.
//
// C'est le cas qui rougirait si `render.Color` changeait d'ordre : les deux vivent
// dans des paquets différents, rien ne les relie à la compilation, et la présentation
// se mettrait à peindre faux sans qu'une seule ligne de ce fichier change.
func TestBitmapMasksMatchTheColourLayout(t *testing.T) {
	info := newBitmapInfo()

	cases := []struct {
		name  string
		color render.Color
		mask  uint32
	}{
		{"rouge", render.Color{R: 0xFF}, info.red},
		{"vert", render.Color{G: 0xFF}, info.green},
		{"bleu", render.Color{B: 0xFF}, info.blue},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			word := *(*uint32)(unsafe.Pointer(&c.color))
			if word != c.mask {
				t.Errorf("la composante occupe %#08x, le masque annonce %#08x", word, c.mask)
			}
		})
	}
}

// TestBitmapInfoIsConstant vérifie ce que l'en-tête doit porter avant toute
// présentation, et qu'un appel ne touche que ses dimensions.
func TestBitmapInfoIsConstant(t *testing.T) {
	info := newBitmapInfo()

	if got := uintptr(info.header.size); got != unsafe.Sizeof(info.header) {
		t.Errorf("taille déclarée %d, taille réelle %d", got, unsafe.Sizeof(info.header))
	}
	if info.header.planes != 1 {
		t.Errorf("plans : %d, attendu 1", info.header.planes)
	}
	if info.header.bitCount != 32 {
		t.Errorf("bits par pixel : %d, attendu 32", info.header.bitCount)
	}
	if info.header.compression != biBitfields {
		t.Errorf("compression : %d, attendu BI_BITFIELDS", info.header.compression)
	}
}

// TestPresentWithoutAnythingToShow vérifie les trois cas où il n'y a rien à poser : la
// boucle les rencontrera, et aucun ne doit lever quoi que ce soit.
func TestPresentWithoutAnythingToShow(t *testing.T) {
	dc, _ := destination(t, 1, 1)

	cases := []struct {
		name   string
		window *window
		buffer *render.Buffer
	}{
		{"sans contexte", testWindow(0), render.NewBuffer(1, 1)},
		{"sans tampon", testWindow(dc), nil},
		{"tampon sans pixel", testWindow(dc), render.NewBuffer(0, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.window.Present(c.buffer)
		})
	}
}

// TestPresentDoesNotAllocate vérifie la propriété dont le budget d'une image dépend :
// la présentation ne touche pas au tas.
//
// Les masques y sont pour tout. Décrire la disposition du moteur au système permet de
// lui passer le tampon où il est ; sans eux, il faudrait un second tampon et un
// parcours complet de l'image à chaque présentation.
func TestPresentDoesNotAllocate(t *testing.T) {
	dc, _ := destination(t, 64, 64)

	w := testWindow(dc)
	buffer := render.NewBuffer(64, 64)

	if n := testing.AllocsPerRun(10, func() { w.Present(buffer) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
