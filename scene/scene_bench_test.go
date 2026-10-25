// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/geometrie"
	"github.com/sprimault/lozengine/rendu"
)

// BenchmarkChangerDeDirection mesure le second critère de contrôle du projet : huit
// cents entités de six types changent de direction sur la même image.
//
// Changer de direction, pour une scène, c'est rhabiller une entité — la position ne
// bouge pas, l'image qu'elle montre change. C'est le geste qu'un jeu répète le plus
// après le déplacement, et le seul qui touche des centaines d'entités d'un coup.
//
// Le cas « déplacer » est là pour la comparaison : les deux écrivent dans la même
// tranche par la même poignée, et seul change ce qu'on y copie. Mesuré trois fois plus
// cher à rhabiller qu'à déplacer, pour des tailles voisines — l'écart n'est donc pas
// expliqué, et il ne vaut pas d'être cherché tant que les deux restent sous le
// centième d'une image.
func BenchmarkChangerDeDirection(b *testing.B) {
	const entites = 800

	s, err := Nouvelle(64, 64, geometrie.Dimetrique(32, 16), Options{Entites: entites})
	if err != nil {
		b.Fatalf("montage : %v", err)
	}

	// Six types, donc six aspects à faire tourner. Un seul ne montrerait pas qu'une
	// entité change réellement d'image.
	types := make([]Aspect, 6)
	for i := range types {
		types[i] = Aspect{
			Planche: uint16(i), Largeur: 12, Hauteur: 24, Teinte: rendu.Blanc(),
		}
	}

	poignees := make([]Poignee, 0, entites)
	for i := range entites {
		p, ok := s.Poser(Entite{X: float64(i % 64), Y: float64(i / 64), Aspect: types[i%6]})
		if !ok {
			b.Fatalf("entité %d refusée", i)
		}
		poignees = append(poignees, p)
	}

	b.Run("rhabiller", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for n := range b.N {
			for i, p := range poignees {
				s.Habiller(p, types[(i+n)%6])
			}
		}
	})

	b.Run("déplacer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for n := range b.N {
			for i, p := range poignees {
				s.Deplacer(p, float64(i%64)+float64(n%2)*0.5, float64(i/64), 0)
			}
		}
	})
}
