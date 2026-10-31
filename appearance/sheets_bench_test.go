// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la résolution d'apparence coûte par image, et ce qu'elle alloue — qui doit être
// rien. C'était le seul paquet du module qu'aucune mesure ne regardait.
//
// **Ce que les cas disent ensemble, et c'est une question qui se ferme** : à la charge de
// contrôle du projet, aucun chemin de ce paquet n'atteint le vingtième d'un pour cent du
// budget d'une image, et aucun n'alloue. Le plus lourd est la résolution de huit cents
// aspects, à trois centièmes de pour cent. Ce paquet n'est donc pas un endroit où
// chercher du temps, et le savoir vaut mieux que de l'ignorer : c'est précisément parce
// qu'il n'avait aucun banc que rien ne permettait de l'écarter.
//
// Deuxième fait, celui qui justifie la table de hachage que la godoc de [Sheets]
// revendique : résoudre huit cents aspects d'un seul type ou de six coûte la même chose à
// un pour cent près. La table ne se dégrade pas quand elle disperse, donc le choix tient.

package appearance

import (
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les deux constantes du budget, redites de celles des bancs de `raster` et de
// `sorting`. Ce n'est pas un oubli de factorisation : le tableau des frontières ne donne
// à `appearance` que `render` et `scene`, et un paquet commun pour trois lignes de test
// coûterait une arête de plus à ce tableau.
const frameAt60Hz = 16_666_667

// framePart rapporte la part d'une image à soixante par seconde, sans pondérer : la
// résolution d'apparence est une lecture de table de hachage et de l'arithmétique
// entière, donc elle ne dépend pas de la bande passante mémoire.
func framePart(b *testing.B) {
	perOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	b.ReportMetric(perOp*100/frameAt60Hz, "%image-brut")
}

// benchSheets monte un fournisseur portant le nombre de types demandé, chacun animé.
//
// Les types se nomment par un entier croissant, ce que [Look] permet : le jeu les nomme
// comme il l'entend, et un banc n'a pas besoin de noms parlants.
func benchSheets(types int) *Sheets {
	s := NewSheets()
	for k := range types {
		s.Set(Look{Kind: uint16(k)}, Entry{
			Sheet: uint16(k), Width: 16, Height: 24,
			Tint: render.White(),
			Clip: Clip{Frames: 4},
		})
	}
	return s
}

// BenchmarkAspect mesure la résolution d'apparence à la cible de contrôle du projet, et
// c'est le premier chiffre que ce paquet ait.
//
// **La charge est celle que `tests.md` nomme** : huit cents entités de six types
// changeant de direction sur la même image. C'est le pire cas annoncé du projet, pas un
// régime courant — un jeu ne redemande une apparence que quand elle change.
//
// Ce que les deux cas disent ensemble : la résolution passe par une table de hachage, et
// c'est ce que la godoc de [Sheets] revendique pouvoir se permettre puisque le rendu n'y
// passe jamais. Le cas à un seul type dit ce que coûte la table quand tout tombe sur la
// même entrée, celui à six ce qu'elle coûte quand elle disperse — l'écart entre les deux
// est ce qui dirait qu'une table est le mauvais choix, et non la valeur absolue.
func BenchmarkAspect(b *testing.B) {
	for _, c := range []struct {
		name  string
		types int
	}{
		{"800 aspects de 6 types", 6},
		{"800 aspects d'un seul type", 1},
	} {
		b.Run(c.name, func(b *testing.B) {
			s := benchSheets(c.types)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for i := range 800 {
					s.Aspect(Look{Kind: uint16(i % c.types)}, Facing(i%Directions), i)
				}
			}

			framePart(b)
		})
	}
}

// BenchmarkPlaceLine mesure la pose d'une ligne de texte, dans les deux repères.
//
// **C'est le chemin dont la godoc de `Font.place` déclare une hypothèse d'échappement**,
// et qu'elle dit vouloir éviter : la voie de pose y passe par un drapeau plutôt que par
// une valeur de méthode, parce qu'une valeur de méthode liée est une fermeture et qu'une
// ligne se pose à chaque image. Le banc chiffre ce chemin ;
// [TestPlacingALineAllocatesNothing] garde l'hypothèse elle-même.
//
// Les deux repères coûtent la même chose ou presque, et c'est ce qu'on veut lire : le
// choix entre la surface et l'aire est une décision de mise en page, pas un arbitrage de
// coût.
func BenchmarkPlaceLine(b *testing.B) {
	const line = "SCORE ABCDEFGHIJ"

	for _, c := range []struct {
		name  string
		place func(Font, *scene.Scene) int
	}{
		{"sur la surface", func(f Font, s *scene.Scene) int { return f.Place(s, 8, 8, line) }},
		{"sur l'aire", func(f Font, s *scene.Scene) int { return f.PlaceArea(s, 8, 8, line) }},
	} {
		b.Run(c.name, func(b *testing.B) {
			f, s := ascii(), textScene(b)

			// Une première pose étend la liste des pièces, que les suivantes
			// réemploient après l'avoir vidée. La mesurer avec les autres compterait
			// une allocation qui n'a lieu qu'une fois par partie.
			c.place(f, s)
			s.ClearScreen()

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				c.place(f, s)
				s.ClearScreen()
			}

			framePart(b)
		})
	}
}

// BenchmarkPerFrame mesure les trois calculs qu'un jeu refait par entité et par image,
// et qu'aucun banc n'atteignait.
//
// **Ils sont entrés dans le périmètre avec le paquet, et ils y manquaient tous.** Faire
// entrer `appearance` dans `CHEMIN_CHAUD` a nommé sept fonctions qu'aucun banc ne
// traversait, dont la cadence d'animation et le remappage de direction — qui sont
// exactement des chemins par image, le second étant appelé pour chaque chose quand la
// caméra tourne. `Clip.hold` et `floorDiv` viennent avec, appelés par les deux premiers.
//
// **Ce qu'on en attend est un ordre de grandeur, pas une surveillance.** Chacun est de
// l'arithmétique entière sur quelques opérations : la mesure sert à l'établir une fois,
// et à voir le jour où l'un d'eux cesserait d'être négligeable. La charge est celle de
// la cible de contrôle, huit cents choses par image.
//
// La boucle rend sa valeur dans une somme, et ce n'est pas de la cérémonie : sans elle
// le compilateur a le droit de supprimer des appels dont le résultat ne sert à personne,
// et le banc mesurerait une boucle vide.
func BenchmarkPerFrame(b *testing.B) {
	c := Clip{Frames: 8, Hold: 4}
	f := ascii()
	p := benchSheets(6)

	for _, k := range []struct {
		name string
		run  func(i int) int
	}{
		{"la cadence en boucle", func(i int) int { return c.Loop(i, 0) }},
		{"la cadence jouée une fois", func(i int) int { n, _ := c.Once(i); return n }},
		// Le nombre de directions se demande au fournisseur et ne se lit pas dans la
		// constante, parce que c'est ce qu'un jeu écrit : l'interface le prend en
		// paramètre précisément pour qu'un fournisseur qui en porte seize déclare le
		// sien.
		{"le remappage de direction", func(i int) int {
			d := p.Directions()
			return int(Turn(Facing(i%d), scene.QuarterTurn, d))
		}},
		{"la mesure d'une ligne", func(int) int { w, _ := f.Measure("SCORE ABCDEFGHIJ"); return w }},
	} {
		b.Run(k.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			sum := 0
			for range b.N {
				for i := range 800 {
					sum += k.run(i)
				}
			}

			framePart(b)
			b.ReportMetric(float64(sum&1), "temoin")
		})
	}
}

// TestPlacingALineAllocatesNothing garde l'hypothèse que la godoc de `Font.place`
// déclare, et c'est la première garde qu'elle ait.
//
// **Elle y est écrite comme une crainte plutôt que comme un fait** — « qu'elle tienne
// sur la pile dépend de l'analyse d'échappement, ce qui est exactement le genre
// d'hypothèse qu'on ne veut pas dans une fonction appelée par image sans banc qui la
// garde ». Le banc manquait, et la garde avec.
//
// **Ce que ce cas n'attrape pas, mesuré en l'écrivant plutôt que supposé** : la forme
// même que la godoc redoute. Remise en place, la valeur de méthode liée n'alloue rien —
// elle est appelée sur place, donc l'analyse d'échappement la garde sur la pile, et le
// `range []rune(text)` qu'on pourrait croire coûteux n'alloue pas non plus, le
// compilateur parcourant la chaîne directement. Le drapeau est donc une précaution
// contre une forme qui pourrait échapper un jour, pas une économie d'aujourd'hui.
//
// Ce qu'il attrape, éprouvé en l'écrivant : une tranche construite par appel, qui est la
// façon réaliste dont ce chemin se mettrait à allouer.
//
// Le régime mesuré est celui d'une image ordinaire : la liste des pièces a déjà sa
// place, et une ligne de texte s'y pose.
func TestPlacingALineAllocatesNothing(t *testing.T) {
	f, s := ascii(), textScene(t)

	f.PlaceArea(s, 8, 8, "SCORE ABCDEFGHIJ")
	s.ClearScreen()

	if n := testing.AllocsPerRun(50, func() {
		f.PlaceArea(s, 8, 8, "SCORE ABCDEFGHIJ")
		s.ClearScreen()
	}); n != 0 {
		t.Errorf("%v allocations pour une ligne de texte, attendu aucune", n)
	}
}

// TestResolvingAnAspectAllocatesNothing garde la résolution d'apparence contre
// l'allocation.
//
// Elle ne se fait pas à chaque image — un jeu redemande une apparence quand elle change
// —, mais huit cents entités qui changent de direction sur la même image sont le pire
// cas que le projet annonce, et une allocation par entité y mettrait le ramasse-miettes
// dans le pas.
func TestResolvingAnAspectAllocatesNothing(t *testing.T) {
	s := benchSheets(6)

	if n := testing.AllocsPerRun(50, func() {
		s.Aspect(Look{Kind: 3}, 5, 2)
	}); n != 0 {
		t.Errorf("%v allocations par résolution, attendu aucune", n)
	}
}
