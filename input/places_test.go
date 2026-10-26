// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le refus de deux noms pour une même place, relevé depuis le paquet lui-même plutôt
// que depuis une liste qu'il faudrait tenir à jour.

package input

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestNoPlaceIsNamedTwice refuse que deux noms désignent la même place.
//
// La faute serait silencieuse et elle est à un bit près : onze codes de balayage
// portent deux places, l'une étendue et l'autre non, donc écrire
// `KeyNumpad4 = extended | 0x4B` au lieu de `0x4B` en ferait un second nom de
// [KeyLeft]. Ni la compilation ni aucun autre cas ne le dirait, et un jeu qui lit le
// pavé numérique verrait les flèches.
//
// **Il ne peut pas rougir tant que le paquet est sain**, ce qui est le cas : c'est
// [TestDuplicatesAreFound] qui éprouve la détection, sur une entrée fabriquée.
func TestNoPlaceIsNamedTwice(t *testing.T) {
	places := namedPlaces(t)

	for value, names := range duplicates(places) {
		t.Errorf("la place %#x porte %d noms : %s",
			value, len(names), strings.Join(names, ", "))
	}
}

// TestTheSweepReachesEveryFormOfPlace vérifie que le relevé atteint les trois façons
// dont une place s'écrit dans ce paquet.
//
// C'est ce qui empêche le contrôle d'unicité d'être vert sans avoir rien vu. Une place
// comme [KeyNumpadDivide] tient son type de son opérande et non d'une annotation, donc
// un relevé qui trierait sur le nom ou sur la forme de la déclaration la manquerait —
// et c'est exactement une de celles qui partagent leur code de base avec une autre.
func TestTheSweepReachesEveryFormOfPlace(t *testing.T) {
	places := namedPlaces(t)

	cases := []struct {
		name  string
		place Key
		want  string
	}{
		{"une place annotée", KeyA, "KeyA"},
		{"une place étendue", KeyLeft, "KeyLeft"},
		{"une place dont le type vient de son opérande", KeyNumpadDivide, "KeyNumpadDivide"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, name := range places[int64(c.place)] {
				if name == c.want {
					return
				}
			}
			t.Errorf("le relevé n'a pas trouvé %s", c.want)
		})
	}
}

// TestDuplicatesAreFound éprouve la détection sur une entrée fabriquée.
//
// Sans lui, le cas qui garde le paquet serait vert quoi qu'il arrive : il ne peut pas
// rougir sur un paquet sain, donc rien ne dirait que la détection elle-même a cessé
// de fonctionner.
func TestDuplicatesAreFound(t *testing.T) {
	found := duplicates(map[int64][]string{
		0x4B: {"KeyLeft", "KeyNumpad4"},
		0x1E: {"KeyA"},
	})

	if len(found) != 1 {
		t.Fatalf("places en double : %d, attendu 1", len(found))
	}
	if names := found[0x4B]; len(names) != 2 {
		t.Errorf("la place 0x4B porte %d noms, attendu 2", len(names))
	}
}

// duplicates rend les places portant plus d'un nom.
func duplicates(places map[int64][]string) map[int64][]string {
	found := map[int64][]string{}
	for value, names := range places {
		if len(names) > 1 {
			found[value] = names
		}
	}
	return found
}

// namedPlaces relève les places que le paquet nomme, indexées par leur valeur.
//
// Le relevé passe par le typeur plutôt que par une liste écrite ici, et c'est tout
// l'intérêt : une place ajoutée entre dans le contrôle sans qu'on y pense. Une liste
// tenue à la main oublierait les places qu'on oublie d'y inscrire, ce qui est
// exactement la faute qu'on cherche.
//
// Le tri se fait sur le type et non sur le préfixe du nom : ce qui compte est qu'une
// constante désigne une place, pas qu'elle s'appelle Key quelque chose.
func namedPlaces(t *testing.T) map[int64][]string {
	t.Helper()

	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal("lecture du paquet:", err)
	}

	var files []*ast.File
	for name, file := range parsed["input"].Files {
		if !strings.HasSuffix(name, "_test.go") {
			files = append(files, file)
		}
	}

	conf := types.Config{Importer: importer.Default()}
	checked, err := conf.Check("input", fset, files, nil)
	if err != nil {
		t.Fatal("typage du paquet:", err)
	}

	places := map[int64][]string{}
	scope := checked.Scope()
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok || !c.Exported() || !isPlace(c.Type()) {
			continue
		}
		if value, exact := constant.Int64Val(c.Val()); exact {
			places[value] = append(places[value], name)
		}
	}

	// Un relevé vide passerait tous les cas sans avoir rien lu, et un contrôle qui
	// ne peut pas échouer n'en est pas un.
	if len(places) == 0 {
		t.Fatal("aucune place relevée : c'est le relevé qui est en panne, pas le paquet")
	}
	return places
}

// isPlace dit si un type est celui d'une place du clavier.
func isPlace(typ types.Type) bool {
	named, ok := typ.(*types.Named)
	return ok && named.Obj().Name() == "Key"
}
