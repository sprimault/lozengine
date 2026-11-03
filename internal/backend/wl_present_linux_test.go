// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'un vrai compositeur détient quand on lui présente une image, et ce que ce cas ne
// dit pas de lui.

package backend

import (
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sprimault/lozengine/render"
)

// compositorRequired est la variable par laquelle un job d'intégration continue interdit
// au cas de présentation Wayland d'être sauté.
//
// **Une seconde variable et non `LOZENGINE_DISPLAY`**, parce que les deux régimes ne se
// recouvrent pas : un job Wayland n'a pas forcément de serveur X, et un job X11 n'a pas de
// compositeur. Une variable unique ferait échouer chacun sur le cas de l'autre.
const compositorRequired = "LOZENGINE_WAYLAND"

// requireCompositor saute le cas quand aucun compositeur n'est joignable, et **refuse de
// le sauter** quand l'environnement dit qu'il doit y en avoir un.
//
// Le second régime est tout l'intérêt, et c'est le raisonnement de [requireDisplay] : sans
// lui, un job qui n'aurait pas réussi à lancer son compositeur passerait au vert en sautant
// le seul cas qu'il existe pour exécuter — et un saut n'est pas un échec.
//
// Le saut, lui, est nécessaire : ni le poste de développement ni la machine de contrôle ne
// portent de compositeur, et un cas qui y échouerait rendrait `make check` inutilisable là
// où il tourne tous les jours.
func requireCompositor(tb testing.TB) {
	tb.Helper()

	if os.Getenv(compositorRequired) != "" || wlNamed() {
		return
	}
	tb.Skipf("aucun compositeur : ni WAYLAND_DISPLAY ni %s ne sont posés", compositorRequired)
}

// TestTheCompositorHoldsExactlyWhatWasPresented est la validation du backend Wayland, et
// le seul cas du paquet qui parle à un vrai compositeur.
//
// **Ce qu'il affirme est plus direct qu'en X11, et la raison est la mémoire partagée.**
// Là-bas les octets traversent une socket, donc le cas doit les redemander au serveur ;
// ici le client écrit **dans la mémoire que le compositeur lit**, donc la relire est relire
// ce qu'il détient. Il n'y a pas de transfert à éprouver, seulement une conversion et un
// décalage.
//
// **Ce qu'il ne dit pas, et c'est à dire** : que le compositeur ait composé cette image à
// l'écran. Aucun client ne peut l'observer — une capture passe par le compositeur lui-même
// —, et c'est le job d'intégration continue qui la prend. Ce cas prouve le protocole et la
// justesse des octets, rien de plus.
//
// **Il n'affirme pas non plus que le tampon soit rendu.** `wl_buffer.release` dirait que le
// compositeur a fini de lire, mais sa description ne le promet que « typically », et un
// compositeur qui garde ses deux tampons fait sauter une présentation — ce que le contrat
// de fenêtre autorise. En faire une assertion rendrait ce cas intermittent pour une
// liberté que le protocole accorde.
//
// **La position d'octet attendue se dérive du code que le compositeur a annoncé**, et non
// de la disposition que le backend en a construite. C'est la leçon du cas X11 : un test qui
// éprouve une conversion ne peut pas s'en servir pour dire ce qu'il attend, sans quoi une
// conversion fausse produit la même erreur des deux côtés et le cas reste vert.
//
// **Le motif dépend des deux coordonnées**, parce qu'un tampon uni ne distinguerait ni une
// transposition, ni un décalage de ligne, ni un pas de ligne mal calculé — les trois
// défauts que l'écriture peut produire et dont aucun ne lève d'erreur.
func TestTheCompositorHoldsExactlyWhatWasPresented(t *testing.T) {
	requireCompositor(t)

	const side = 64

	w, err := wlOpen(Options{Title: "lozengine", Width: side, Height: side})
	if err != nil {
		t.Fatalf("ouverture de la fenêtre : %v", err)
	}
	defer w.Close()

	buffer := render.NewBuffer(side, side)
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			buffer.Pixels[y*side+x] = render.Color{
				R: uint8(x * 4),
				G: uint8(y * 4),
				B: uint8((x ^ y) * 4),
				A: 255,
			}
		}
	}

	w.Present(buffer)
	if err := w.Err(); err != nil {
		t.Fatalf("présentation : %v", err)
	}
	if w.pool == nil {
		t.Fatal("la présentation n'a monté aucune mémoire partagée")
	}

	// Les positions d'octet du format annoncé, relevées dans le commentaire que la
	// table des formats porte : le code dit la disposition du mot de trente-deux bits
	// du bit fort au bit faible, et ce qu'on lit ici est l'ordre en mémoire, qui se lit
	// à l'envers.
	var red, green, blue int
	switch code := wlFourCCOf(w.g.format); code {
	case formatXBGR8888:
		red, green, blue = 0, 1, 2
	case formatXRGB8888:
		red, green, blue = 2, 1, 0
	default:
		t.Fatalf("le compositeur a accordé le format %s, dont ce cas ne connaît pas la disposition",
			fourccName(code))
	}

	held := w.pool.mem[w.pool.buffers[w.pool.last].at:]
	if want := w.pool.stride * side; len(held) < want {
		t.Fatalf("la mémoire partagée porte %d octets après le décalage du tampon, %d attendus",
			len(held), want)
	}

	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			sent := buffer.Pixels[y*side+x]
			at := y*w.pool.stride + x*4

			if got, want := held[at+red], sent.R; got != want {
				t.Fatalf("en (%d, %d), le rouge que le compositeur lira vaut %d, on a écrit %d",
					x, y, got, want)
			}
			if got, want := held[at+green], sent.G; got != want {
				t.Fatalf("en (%d, %d), le vert que le compositeur lira vaut %d, on a écrit %d",
					x, y, got, want)
			}
			if got, want := held[at+blue], sent.B; got != want {
				t.Fatalf("en (%d, %d), le bleu que le compositeur lira vaut %d, on a écrit %d",
					x, y, got, want)
			}
		}
	}
}

// wlOutputSize mesure la sortie du compositeur par une première capture.
//
// **Deux conceptions plus simples ont été réfutées par la mesure, et c'est pourquoi la
// taille se mesure au lieu de se demander.** La première supposait qu'un compositeur sans
// écran met son client au format de sa sortie : sous cage il n'en propose aucune, et la
// fenêtre restait à soixante-quatre pixels de côté dans une sortie de 1280×720. La seconde
// demandait le plein écran par `xdg_toplevel.set_fullscreen` : cage ne l'honore pas, et
// l'attente de la nouvelle taille a expiré sur sa borne.
//
// Ce qui reste vrai ne dépend d'aucune faveur du compositeur : **une surface Wayland prend
// la taille du tampon qu'on y attache**. Présenter à la taille de la sortie la couvre donc
// par construction, et la place de la fenêtre n'a pas à être devinée — une surface de la
// taille exacte de la sortie ne peut être nulle part ailleurs qu'à son origine.
//
// Le tampon de ce premier tour n'est pas comparé : il n'existe que pour qu'une image soit
// composée, puisqu'une capture mesure une sortie peinte et non une sortie déclarée.
func wlOutputSize(tb testing.TB, w *wlWindow) (width, height uint16) {
	tb.Helper()

	opened, openedHeight := w.Size()
	w.Present(render.NewBuffer(opened, openedHeight))
	if err := w.Err(); err != nil {
		tb.Fatalf("première présentation : %v", err)
	}
	wlFrameDone(tb, w)

	bounds := wlCapture(tb).Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		tb.Fatalf("la capture de mesure fait %d×%d", bounds.Dx(), bounds.Dy())
	}
	return uint16(bounds.Dx()), uint16(bounds.Dy())
}

// wlFrameDone demande un rappel d'image et attend que le compositeur l'honore.
//
// **C'est la seule synchronisation du protocole qui dise « l'image est composée »**, et
// c'est ce dont une capture a besoin : un `commit` traité n'est pas encore une image
// rendue, donc capturer juste après rendrait la sortie du tour précédent. Un aller-retour
// ne suffirait pas — il prouve que le compositeur a **lu** la requête, pas qu'il a peint.
//
// Le rappel est demandé par le cas et non par le backend, qui ne s'en sert pas : `Paced`
// rend faux pour cette famille, précisément parce que le jeton ne se réarmerait pas hors de
// la présentation. Ici il n'y a pas de boucle à cadencer, seulement une image à attendre.
//
// L'attente est bornée parce qu'un compositeur qui ne compose pas laisserait le cas en
// place indéfiniment, et une expiration nommée se lit là qu'un blocage ne dit rien.
func wlFrameDone(tb testing.TB, w *wlWindow) {
	tb.Helper()

	id, err := w.c.newID()
	if err != nil {
		tb.Fatalf("identifiant du rappel d'image : %v", err)
	}

	w.c.out.begin(w.surface, wlSurfaceFrame)
	w.c.out.uint(id)
	w.c.out.end()
	w.c.out.begin(w.surface, wlSurfaceCommit)
	w.c.out.end()
	if err := w.c.flush(); err != nil {
		tb.Fatalf("demande du rappel d'image : %v", err)
	}

	// Le gestionnaire de la fenêtre ignore les objets qu'il ne connaît pas, donc le
	// rappel se capte en s'insérant devant lui — comme un aller-retour le fait.
	done := false
	previous := w.c.handler
	w.c.handler = func(sender uint32, opcode uint16, r *wlReader) error {
		if sender == id && opcode == wlCallbackDone {
			r.uint()
			done = true
			return nil
		}
		if previous == nil {
			return nil
		}
		return previous(sender, opcode, r)
	}
	defer func() { w.c.handler = previous }()

	if err := w.c.bound(5 * time.Second); err != nil {
		tb.Fatalf("borne de l'attente : %v", err)
	}
	defer w.c.bound(0)

	for !done {
		if err := w.c.fill(); err != nil {
			tb.Fatalf("attente de l'image composée : %v", err)
		}
		if err := w.c.dispatch(); err != nil {
			tb.Fatalf("attente de l'image composée : %v", err)
		}
	}
}

// wlCapture prend l'image de la sortie par l'outil de capture du compositeur.
//
// `grim` parle le protocole de copie d'écran de wlroots, que le compositeur de la voie
// principale porte. Son absence saute le cas plutôt que de l'échouer : c'est un outil du
// dispositif d'épreuve, comme `Xvfb` ou `xdotool` le sont du job X11, et il ne fait partie
// d'aucune des deux machines de développement.
func wlCapture(tb testing.TB) image.Image {
	tb.Helper()

	tool, err := exec.LookPath("grim")
	if err != nil {
		tb.Skipf("grim est absent : aucune capture possible sous ce compositeur")
	}

	path := filepath.Join(tb.TempDir(), "capture.png")
	out, err := exec.Command(tool, path).CombinedOutput()
	if err != nil {
		tb.Fatalf("capture par grim : %v\n%s", err, out)
	}

	file, err := os.Open(path)
	if err != nil {
		tb.Fatalf("ouverture de la capture : %v", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		tb.Fatalf("décodage de la capture : %v", err)
	}
	return img
}

// TestTheCompositorComposedExactlyWhatWasPresented est la validation que seule une capture
// peut donner.
//
// **C'est la seule assertion du dépôt qui traverse la composition**, et elle dit ce que
// [TestTheCompositorHoldsExactlyWhatWasPresented] ne peut pas : que le compositeur a peint
// l'image, et non seulement qu'il a de quoi la peindre. Les deux ensemble couvrent la
// chaîne entière — la mémoire que le client écrit, puis l'écran que le compositeur compose.
//
// **La comparaison est exacte et ne calcule aucune position**, et c'est le montage qui le
// permet : un backend présente le tampon **tel quel**, l'agrandissement entier et les
// bandes étant l'affaire de la boucle du paquet racine. En présentant un tampon de la
// taille que le compositeur a imposée, la capture **est** l'image présentée — donc il n'y a
// ni facteur à défaire, ni bande à écarter, ni pixel dont la place dépende d'une
// négociation.
//
// **L'alpha ne se compare pas.** Le format accordé déclare son quatrième octet ignoré, donc
// le compositeur est libre de rendre ce qu'il veut à cette place et la capture l'ouvre en
// opaque. Les trois composantes de couleur, elles, se comparent au pixel.
//
// **La capture porte la sortie entière et non la fenêtre**, ce qui n'est juste que si la
// fenêtre la couvre : [wlOutputSize] mesure la sortie, et présenter à cette taille la
// couvre par construction. Deux voies plus directes ont été réfutées par la mesure, et sa
// godoc les nomme.
func TestTheCompositorComposedExactlyWhatWasPresented(t *testing.T) {
	requireCompositor(t)

	w, err := wlOpen(Options{Title: "lozengine", Width: 64, Height: 64})
	if err != nil {
		t.Fatalf("ouverture de la fenêtre : %v", err)
	}
	defer w.Close()

	width, height := wlOutputSize(t, w)

	buffer := render.NewBuffer(width, height)
	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			buffer.Pixels[y*int(width)+x] = render.Color{
				R: uint8(x * 4),
				G: uint8(y * 4),
				B: uint8((x ^ y) * 4),
				A: 255,
			}
		}
	}

	w.Present(buffer)
	if err := w.Err(); err != nil {
		t.Fatalf("présentation : %v", err)
	}
	wlFrameDone(t, w)

	got := wlCapture(t)
	bounds := got.Bounds()
	if bounds.Dx() != int(width) || bounds.Dy() != int(height) {
		t.Fatalf("la capture fait %d×%d, la fenêtre %d×%d : elle ne porte pas la même surface",
			bounds.Dx(), bounds.Dy(), width, height)
	}

	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			sent := buffer.Pixels[y*int(width)+x]
			r, g, b, _ := got.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()

			// Les composantes d'une couleur du paquet image sont sur seize bits et
			// prémultipliées ; les nôtres sont sur huit et opaques, donc le retour
			// au même domaine est une division par l'échelle.
			if uint8(r>>8) != sent.R || uint8(g>>8) != sent.G || uint8(b>>8) != sent.B {
				t.Fatalf("en (%d, %d), le compositeur a composé (%d, %d, %d), on a présenté (%d, %d, %d)",
					x, y, r>>8, g>>8, b>>8, sent.R, sent.G, sent.B)
			}
		}
	}
}
