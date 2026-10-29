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
//
// `CreateDIBSection` et `DeleteObject` en sont sorties avec l'icône de la fenêtre, qui
// les emploie pour de bon : elles sont déclarées avec les autres appels du backend.
var (
	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	selectObject       = gdi32.NewProc("SelectObject")
	deleteDC           = gdi32.NewProc("DeleteDC")
)

// destination monte une image en mémoire et rend son contexte avec ses pixels.
//
// Elle porte le même en-tête que la présentation, ce qui est devenu le cas en passant
// à `BI_RGB` : les deux bouts de la chaîne décrivent donc la disposition du système, et
// ce que le cas éprouve n'est plus l'accord de deux descriptions mais l'exactitude de
// l'échange que [toBGRA] fait en amont.
func destination(t testing.TB, width, height int32) (uintptr, []uint32) {
	t.Helper()

	dc, _, err := createCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatalf("CreateCompatibleDC : %v", err)
	}

	header := newBitmapInfo()
	header.width = width
	header.height = -height

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

// TestToBGRAPlacesEachComponent éprouve l'échange seul, sans passer par le système.
//
// Il rougit sur une permutation que [TestPresentKeepsTheComponentsInOrder] verrait
// aussi, et il n'est pas là pour cela : son objet est de nommer le mot attendu, de
// sorte qu'un échec dise *quelle* composante s'est déplacée plutôt que de montrer deux
// nombres hexadécimaux à comparer de tête. L'alpha y est éprouvé, là où le cas par le
// système ne peut pas le lire — `BI_RGB` sur trente-deux bits laisse son octet
// indéfini.
func TestToBGRAPlacesEachComponent(t *testing.T) {
	cases := []struct {
		name  string
		color render.Color
		word  uint32
	}{
		{"rouge en troisième octet", render.Color{R: 0xFF}, 0x00FF0000},
		{"vert en deuxième", render.Color{G: 0xFF}, 0x0000FF00},
		{"bleu en premier", render.Color{B: 0xFF}, 0x000000FF},
		{"alpha en dernier", render.Color{A: 0xFF}, 0xFF000000},
		{"les quatre distinctes", render.Color{R: 0x11, G: 0x22, B: 0x33, A: 0x44}, 0x44112233},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := toBGRA(nil, []render.Color{c.color})
			if got[0] != c.word {
				t.Errorf("%#08x, attendu %#08x", got[0], c.word)
			}
		})
	}
}

// TestToBGRAKeepsItsBuffer vérifie que le tampon se réemploie tant qu'il suffit, et
// qu'il grandit quand il ne suffit plus.
//
// C'est la propriété dont [TestPresentDoesNotAllocate] dépend, éprouvée ici sur la
// seule fonction plutôt qu'à travers un appel système : une allocation par image dans
// le chemin de rendu vaut huit mégaoctets à 1920×1080.
func TestToBGRAKeepsItsBuffer(t *testing.T) {
	src := make([]render.Color, 64)

	first := toBGRA(nil, src)
	again := toBGRA(first, src)
	if &again[0] != &first[0] {
		t.Error("le tampon a été refait alors que sa capacité suffisait")
	}

	shorter := toBGRA(first, src[:32])
	if len(shorter) != 32 {
		t.Errorf("longueur %d, attendu 32", len(shorter))
	}
	if &shorter[0] != &first[0] {
		t.Error("le tampon a été refait pour une source plus courte")
	}

	bigger := toBGRA(first, make([]render.Color, 128))
	if len(bigger) != 128 {
		t.Errorf("longueur %d, attendu 128", len(bigger))
	}
}

// TestBitmapInfoIsConstant vérifie ce que l'en-tête doit porter avant toute
// présentation, et qu'un appel ne touche que ses dimensions.
//
// La compression en fait partie depuis qu'elle décide du coût : `BI_RGB` est le seul
// en-tête que GDI recopie en bloc, et y revenir par inadvertance ne casserait aucune
// couleur — seulement la cadence, d'un facteur 47. Rien d'autre ne l'attraperait.
func TestBitmapInfoIsConstant(t *testing.T) {
	info := newBitmapInfo()

	if got := uintptr(info.size); got != unsafe.Sizeof(info) {
		t.Errorf("taille déclarée %d, taille réelle %d", got, unsafe.Sizeof(info))
	}
	if info.planes != 1 {
		t.Errorf("plans : %d, attendu 1", info.planes)
	}
	if info.bitCount != 32 {
		t.Errorf("bits par pixel : %d, attendu 32", info.bitCount)
	}
	if info.compression != biRGB {
		t.Errorf("compression : %d, attendu BI_RGB", info.compression)
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
// Le tampon de sortie y est pour tout. La conversion vers l'ordre du système demande
// un second tampon ; le garder d'une image à l'autre est ce qui fait que son coût est
// un parcours et non une allocation de huit mégaoctets par image. Le premier appel
// l'alloue, et c'est l'appel de chauffe d'`AllocsPerRun` qui l'absorbe — comme la
// boucle du moteur, qui présente la même taille des milliers de fois de suite.
func TestPresentDoesNotAllocate(t *testing.T) {
	dc, _ := destination(t, 64, 64)

	w := testWindow(dc)
	buffer := render.NewBuffer(64, 64)

	if n := testing.AllocsPerRun(10, func() { w.Present(buffer) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}

// BenchmarkPresent mesure la remise complète du tampon, échange compris.
//
// **Il manquait, et c'est son absence qui a laissé vivre un facteur 47 pendant trois
// semaines.** Le dépôt n'avait de banc de présentation que pour X11 ; celui-ci dit
// combien coûte le maillon que tout utilisateur de la plateforme exécute à chaque
// image. Avec [BenchmarkToBGRA], qui isole l'échange, il dit aussi où va le temps :
// ce que l'un ne mesure pas, l'appel système le prend.
//
// **La destination est une image en mémoire et non une fenêtre**, ce qui le rend
// exécutable sans session graphique, donc en intégration continue. Mesuré le
// 29 octobre 2026, l'écart à une vraie fenêtre est de 4 % — 23,3 ms contre 22,4 à
// 1920×1080 sur l'ancien chemin —, le coût étant dans GDI côté processeur et non dans
// le pilote. Ce que cela ne mesure donc pas : la composition, le vsync, et tout ce qui
// dépend d'un écran branché.
func BenchmarkPresent(b *testing.B) {
	for _, size := range []struct {
		name          string
		width, height uint16
	}{
		{"480×270, la résolution interne du moteur", 480, 270},
		{"1920×1080, un écran entier", 1920, 1080},
		{"3840×2160", 3840, 2160},
	} {
		b.Run(size.name, func(b *testing.B) {
			dc, _ := destination(b, int32(size.width), int32(size.height))

			w := testWindow(dc)
			buffer := render.NewBuffer(size.width, size.height)
			for i := range buffer.Pixels {
				buffer.Pixels[i] = render.Color{R: uint8(i), G: uint8(i >> 8), B: uint8(i >> 16), A: 255}
			}

			// Une première présentation hors de la boucle : elle paie l'allocation du
			// tampon de sortie, que les suivantes réemploient — comme la boucle du
			// moteur, qui présente la même taille des milliers de fois de suite.
			w.Present(buffer)

			b.SetBytes(int64(4 * len(buffer.Pixels)))
			b.ResetTimer()
			for range b.N {
				w.Present(buffer)
			}
		})
	}
}

// BenchmarkToBGRA mesure l'échange seul, aux tailles de fenêtre qui comptent.
//
// **Le fait de structure que les cas disent ensemble** : le coût est proportionnel à la
// surface et non au contenu, parce qu'il n'y a ni branche ni lecture dépendante dans
// la boucle. C'est ce qui permet de le mettre en regard de la présentation qu'il
// remplace — elle aussi proportionnelle, mais d'un facteur 47 plus grand sur ce
// chemin.
func BenchmarkToBGRA(b *testing.B) {
	sizes := []struct {
		name string
		w, h int
	}{
		{"960x540", 960, 540},
		{"1920x1080", 1920, 1080},
		{"2560x1440", 2560, 1440},
		{"3840x2160", 3840, 2160},
	}
	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			src := make([]render.Color, s.w*s.h)
			dst := make([]uint32, len(src))
			b.ResetTimer()
			for range b.N {
				dst = toBGRA(dst, src)
			}
		})
	}
}
