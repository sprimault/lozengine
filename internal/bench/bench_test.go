// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le seul cas que ce paquet demande : que la part rapportée soit celle du temps mesuré,
// sans facteur en route.

package bench

import "testing"

// TestPartCarriesNoFactor confronte la métrique que [Part] rapporte au temps que
// `testing` a lui-même mesuré.
//
// **C'est le garde que le dépôt n'avait pas, et il lui a coûté une pondération appliquée à
// l'envers pendant des semaines.** Les bancs multipliaient leur part par quatre pour
// représenter une machine plus étroite ; le facteur vivait dans la formule, le nom de la
// métrique ne le disait pas, et deux rapporteurs ont fini par rendre des unités différentes
// sous le même nom sans qu'aucun contrôle ne le voie.
//
// Ce cas rougit sur exactement cette mutation : réintroduire un facteur dans [Part], ou
// changer le dénominateur, rend une valeur qui ne correspond plus au temps mesuré. Il ne
// dépend d'aucune durée réelle, donc il ne devient pas instable sur une machine chargée —
// ce qu'on lui demande est une égalité entre deux lectures de la même mesure.
func TestPartCarriesNoFactor(t *testing.T) {
	r := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			// Un travail qui n'est pas optimisable, pour que le temps mesuré ne soit
			// pas nul : une division nulle rendrait une part nulle, et le cas
			// passerait en ne comparant rien.
			_ = make([]byte, 64)
		}
		Part(b)
	})

	got, ok := r.Extra["%image"]
	if !ok {
		t.Fatalf("la métrique %%image n'est pas rapportée, seulement %v", keys(r.Extra))
	}

	perOp := float64(r.T.Nanoseconds()) / float64(r.N)
	want := perOp * 100 / FrameAt60Hz

	// **Un pour cent, et non une égalité exacte.** Le cas l'a exigée au premier essai et
	// a échoué : `Part` lit `b.Elapsed()` avant que le banc ne finisse, donc le temps
	// qu'elle divise est un peu plus court que celui que `testing.Benchmark` rapporte —
	// mesuré, deux centièmes de millième d'écart. La tolérance couvre cela et reste cent
	// fois sous ce que la mutation visée produirait : un facteur quatre, ou un
	// dénominateur remplacé par une somme d'étapes, décalent la valeur de plusieurs
	// centaines de pour cent.
	if ecart := got/want - 1; ecart > 0.01 || ecart < -0.01 {
		t.Errorf("part rapportée %g, attendue %g à 1 %% près — un facteur s'est glissé dans Part", got, want)
	}
}

// TestFrameAt60HzIsTheFrameBudget fige le dénominateur du seuil de dix pour cent.
//
// Une seconde divisée par soixante fait 16 666 666,67 nanosecondes, et la constante
// arrondit au-dessus. Le cas rougit si quelqu'un la remplace par la somme d'étapes qu'un
// banc mesure, ce qui est l'erreur de lecture que la godoc du paquet raconte.
func TestFrameAt60HzIsTheFrameBudget(t *testing.T) {
	if got, want := FrameAt60Hz, 1_000_000_000/60+1; got != want {
		t.Errorf("budget d'une image %d ns, attendu %d", got, want)
	}
}

// keys rend les noms des métriques rapportées, pour que l'échec dise ce qu'il a trouvé
// plutôt que ce qu'il cherchait.
func keys(m map[string]float64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
