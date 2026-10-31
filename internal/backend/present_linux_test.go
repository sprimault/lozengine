// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0
// Ce qui arrive au serveur quand on présente une image : il doit tenir exactement les
// pixels qu'on lui a donnés, sans quoi la comparaison au pixel ne vaudrait rien.

package backend

import (
	"encoding/binary"
	"math/bits"
	"os"
	"testing"
	"time"

	"github.com/sprimault/lozengine/render"
)

// displayRequired est la variable par laquelle un job d'intégration continue interdit
// au cas d'affichage d'être sauté.
const displayRequired = "LOZENGINE_DISPLAY"

// requireDisplay saute le cas quand aucun serveur d'affichage n'est joignable, et
// **refuse de le sauter** quand l'environnement dit qu'il doit y en avoir un.
//
// Le second régime est tout l'intérêt. Sans lui, un job qui n'aurait pas réussi à
// lancer son serveur passerait au vert en sautant le seul cas qu'il existe pour
// exécuter — et rien ne le signalerait, un saut n'étant pas un échec. C'est le
// raisonnement du job qui exige un échec sous la version inférieure au plancher : un
// contrôle qui ne peut pas échouer n'en est pas un.
//
// Le saut, lui, est nécessaire : ni le poste de développement ni la machine de contrôle
// n'ont de serveur sans écran, et un cas qui y échouerait rendrait `make check`
// inutilisable là où il tourne tous les jours.
func requireDisplay(tb testing.TB) {
	tb.Helper()

	if os.Getenv(displayRequired) != "" || os.Getenv("DISPLAY") != "" {
		return
	}
	tb.Skipf("aucun serveur d'affichage : ni DISPLAY ni %s ne sont posés", displayRequired)
}

// TestTheServerHoldsExactlyWhatWasPresented est la validation du backend X11, et le
// seul cas de tout le paquet qui parle à un vrai serveur.
//
// **Il compare au tampon remis et non aux images de référence du jalon 0**, ce qui est
// un arbitrage du brief : comparer aux références ferait éprouver le rastériseur une
// seconde fois, et ce cas rougirait alors pour des raisons qui ne sont pas les siennes.
// La seule assertion qui lui appartient est *le serveur détient exactement ce que le
// moteur lui a remis*. Une vérité par couche, et pas de jeu de références par backend.
//
// **Les bits hors masques sont masqués des deux côtés.** La présentation ne les écrit
// pas, le protocole les déclarant indéfinis en profondeur 24 : comparer des mots bruts
// rendrait ce cas intermittent, et on chercherait la cause dans le transfert.
//
// **Le motif dépend des deux coordonnées**, parce qu'un tampon uni ne distinguerait ni
// une transposition, ni un décalage de ligne, ni une bande posée au mauvais endroit —
// les trois défauts que le transfert peut produire et dont aucun ne lève d'erreur.
//
// **La fenêtre reste petite exprès** : le serveur ne détient que ce qui est dans
// l'écran, donc une fenêtre qui en dépasserait ferait comparer des pixels que personne
// n'a écrits.
func TestTheServerHoldsExactlyWhatWasPresented(t *testing.T) {
	requireDisplay(t)

	const side = 64

	opened, err := Open(Options{Title: "lozengine", Width: side, Height: side})
	if err != nil {
		t.Fatalf("ouverture de la fenêtre : %v", err)
	}
	defer opened.Close()

	w, ok := opened.(*window)
	if !ok {
		t.Fatalf("Open a rendu un %T, et ce cas a besoin de relire l'image", opened)
	}
	waitMapped(t, w)

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

	got, err := w.c.getImage(w.id, side, side)
	if err != nil {
		t.Fatalf("relecture de l'image : %v", err)
	}

	if want := 4 * len(buffer.Pixels); len(got) < want {
		t.Fatalf("le serveur a rendu %d octets, %d attendus", len(got), want)
	}

	// **La référence se calcule depuis les masques du serveur, et non par la
	// conversion du backend.** La première version de ce cas employait celle-ci des
	// deux côtés : une conversion fausse produisait alors la même erreur dans
	// l'attendu et dans l'envoyé, les deux concordaient, et le cas restait vert. Il
	// a fallu inverser l'ordre des composantes sur un vrai serveur pour le voir —
	// et la godoc affirmait pourtant le contraire.
	//
	// Un test qui éprouve une conversion ne peut pas s'en servir pour dire ce qu'il
	// attend : il lui faut une seconde expression de la même vérité.
	red := bits.TrailingZeros32(w.c.setup.redMask)
	green := bits.TrailingZeros32(w.c.setup.greenMask)
	blue := bits.TrailingZeros32(w.c.setup.blueMask)
	mask := w.c.setup.redMask | w.c.setup.greenMask | w.c.setup.blueMask

	for i, pixel := range buffer.Pixels {
		sent := uint32(pixel.R)<<red | uint32(pixel.G)<<green | uint32(pixel.B)<<blue
		held := imageWord(w.c.setup, got[4*i:]) & mask

		if sent != held {
			t.Fatalf("pixel %d, en (%d, %d) : le serveur détient %#x, on lui a remis %#x",
				i, i%side, i/side, held, sent)
		}
	}
}

// imageWord lit la valeur d'un pixel dans les données d'une image.
//
// **L'ordre des octets d'une image n'est pas celui de la connexion**, et c'est la
// distinction que tout ce lot repose dessus : le client choisit le second, le serveur
// impose le premier. Les confondre marcherait sur un serveur où les deux coïncident —
// celui qu'on éprouve — et produirait des couleurs permutées ailleurs.
func imageWord(s setup, b []byte) uint32 {
	if s.imageOrder == msbFirst {
		return binary.BigEndian.Uint32(b)
	}
	return binary.LittleEndian.Uint32(b)
}

// BenchmarkPresent mesure ce qu'une image remise au serveur coûte par image.
//
// **Il existe pour décider si MIT-SHM valait d'être écrit**, et c'est sa seule raison :
// le transfert d'image par le protocole représente environ quatre fois la taille de
// l'image en trafic mémoire, ce qui est peu sur une machine de bureau et beaucoup sur
// une petite machine comme un Raspberry Pi, qui est une cible annoncée. Le dépôt
// interdit d'optimiser sur une charge devinée,
// donc il fallait un chiffre.
//
// **Le débit est ce qu'on lit, pas la durée**, parce que c'est lui qui se compare à la
// bande passante d'une machine : une durée ne dit rien sans la taille qui l'a produite,
// et c'est le rapport entre les deux qui est un fait du moteur plutôt qu'un fait de
// cette machine.
//
// Les trois tailles ne sont pas un échantillon mais trois questions : la résolution
// interne du moteur, un écran courant, et un écran complet — le transfert grandissant
// avec la surface, c'est la plus grande qui décide.
//
// **Ce que ce banc ne dit pas** : ce que donnerait MIT-SHM. Le mesurer demanderait de
// l'écrire, ce que ce banc existe justement pour décider. Il borne donc le coût à
// battre, et rien de plus.
//
// **Le fait de structure que les deux bancs disent ensemble, et qui a décidé : la
// conversion pèse environ quatre fois le transfert, à toutes les tailles mesurées.**
// MIT-SHM supprime le second et garde la première, donc son gain plafonne au cinquième
// du coût d'une présentation — soit quelques pour cent du budget d'une image, bien en
// dessous du seuil qui autorise à optimiser ici. Le rapport est ce qu'on relit ; les
// durées, elles, sont des faits de la machine qui les a produites.
//
// **Et ce que ce rapport ne couvre pas** : la mesure est prise côté client seulement.
// Le serveur lit aussi ces octets, et MIT-SHM lui épargne une copie qu'aucun banc d'ici
// ne compte. Le gain réel est donc supérieur à ce plafond, d'un facteur qu'il faudrait
// mesurer de l'autre côté.
func BenchmarkPresent(b *testing.B) {
	requireDisplay(b)

	for _, size := range []struct {
		name          string
		width, height uint16
	}{
		{"480×270, la résolution interne du moteur", 480, 270},
		{"1280×720", 1280, 720},
		{"1920×1080, un écran entier", 1920, 1080},
	} {
		b.Run(size.name, func(b *testing.B) {
			opened, err := Open(Options{Title: "lozengine", Width: size.width, Height: size.height})
			if err != nil {
				b.Fatalf("ouverture de la fenêtre : %v", err)
			}
			defer opened.Close()

			w := opened.(*window)
			buffer := render.NewBuffer(size.width, size.height)
			for i := range buffer.Pixels {
				buffer.Pixels[i] = render.Color{R: uint8(i), G: uint8(i >> 8), B: uint8(i >> 16), A: 255}
			}

			// Une première présentation hors de la boucle : elle paie
			// l'agrandissement du tampon d'émission, que les suivantes réemploient.
			w.Present(buffer)

			b.SetBytes(int64(4 * len(buffer.Pixels)))
			b.ResetTimer()

			for range b.N {
				w.Present(buffer)
			}

			// Le relevé se lit après la boucle et jamais pendant : une écriture dans
			// la mesure fabrique le symptôme qu'on cherche.
			b.StopTimer()
			if err := w.Err(); err != nil {
				b.Fatalf("présentation : %v", err)
			}
		})
	}
}

// waitMapped pompe les événements jusqu'à ce que le serveur annonce la fenêtre
// affichée.
//
// La relecture l'exige : le contenu d'une fenêtre masquée n'existe pas pour le serveur,
// qui refuse alors par une erreur de correspondance. L'ouverture ne l'attend pas
// exprès — attendre là bloquerait indéfiniment là où aucun gestionnaire de fenêtres ne
// répond —, donc c'est au cas qui a besoin de l'image de patienter.
func waitMapped(t *testing.T, w *window) {
	t.Helper()

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		w.Input()
		if err := w.Err(); err != nil {
			t.Fatalf("la fenêtre a cessé avant d'être affichée : %v", err)
		}
		if w.mapped {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("la fenêtre n'a pas été affichée en cinq secondes")
}
