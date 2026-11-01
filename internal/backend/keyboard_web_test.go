// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la table des places doit tenir : qu'elle soit totale, qu'aucune place n'y soit
// atteinte deux fois, et que les deux familles qui se ressemblent restent distinctes.

package backend

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/sprimault/lozengine/input"
)

// TestEveryDeclaredPlaceIsReachable exige que chaque place déclarée par `input` ait un
// `code` qui y mène.
//
// **C'est le seul cas qui rougira le jour où une place s'ajoutera au moteur sans entrer
// ici**, et c'est pour cela qu'il relève les places depuis les sources plutôt que de
// compter. Un dénombrement contre quatre-vingt-six resterait vert : la place neuve ferait
// quatre-vingt-sept d'un côté et la table en garderait quatre-vingt-six, ce que seule une
// comparaison nom par nom voit.
//
// Le défaut qu'il attrape est une touche muette dans le navigateur et nulle part ailleurs,
// donc invisible à toute autre couche.
//
// **Il lit le disque, et `make wasm-test` l'écarte sous Windows** pour cela : l'hôte node
// n'y gère pas l'ouverture d'un répertoire. Il tourne sur l'hôte par `make test` et sous
// `js/wasm` en intégration continue, donc il est couvert deux fois.
func TestEveryDeclaredPlaceIsReachable(t *testing.T) {
	reached := map[input.Key]bool{}
	for _, k := range codePlaces {
		reached[k] = true
	}

	for name, value := range declaredPlaces(t) {
		if !reached[input.Key(value)] {
			t.Errorf("la place %s n'est atteinte par aucun code de la page", name)
		}
	}
}

// TestNoPlaceIsReachedTwice refuse que deux `code` mènent à la même place.
//
// La faute serait un copier-coller, et son effet est qu'une touche en commande une autre —
// un jeu qui lit le pavé verrait les flèches. Ni la compilation ni le cas de totalité ne le
// diraient : une place atteinte deux fois reste atteinte.
func TestNoPlaceIsReachedTwice(t *testing.T) {
	seen := map[input.Key][]string{}
	for code, k := range codePlaces {
		seen[k] = append(seen[k], code)
	}

	for k, codes := range seen {
		if len(codes) > 1 {
			t.Errorf("la place %#x est atteinte par %d codes : %s",
				uint16(k), len(codes), strings.Join(codes, ", "))
		}
	}
}

// TestTheRowDigitsUseTheirDomNames garde les dix renommages.
//
// Les vingt-six lettres portent le même nom des deux côtés, ce qui invite à écrire `Key0`
// pour le chiffre zéro par symétrie. La page ne l'envoie jamais : elle dit `Digit0`. Le cas
// rougit sur cette erreur-là, qui laisserait les dix chiffres de la rangée muets alors que
// les lettres répondent — un symptôme qu'on chercherait longtemps du côté de l'écouteur.
func TestTheRowDigitsUseTheirDomNames(t *testing.T) {
	for d := range 10 {
		dom := string(rune('0' + d))
		if _, ok := codePlaces["Digit"+dom]; !ok {
			t.Errorf("le code Digit%s ne mène à aucune place", dom)
		}
		if _, ok := codePlaces["Key"+dom]; ok {
			t.Errorf("le code Key%s est dans la table, et la page ne l'envoie jamais", dom)
		}
	}
}

// TestTheNumpadIsNotTheNavigationBlock vérifie que le pavé numérique et le bloc de
// navigation mènent à des places distinctes.
//
// **C'est la confusion que le moteur redoute le plus, et elle est à un bit près** : onze
// codes de balayage portent deux places, l'une étendue et l'autre non, donc `Numpad4` et
// `ArrowLeft` partagent `0x4B` à ce bit près. La godoc de `input` le documente pour ses
// propres constantes ; ici le risque est de traduire un `code` du pavé vers la place de la
// flèche. Le cas rougit sur cette inversion.
func TestTheNumpadIsNotTheNavigationBlock(t *testing.T) {
	pads := []string{"Numpad0", "Numpad1", "Numpad2", "Numpad3", "Numpad4",
		"Numpad5", "Numpad6", "Numpad7", "Numpad8", "Numpad9"}
	others := []string{"ArrowUp", "ArrowLeft", "ArrowRight", "ArrowDown",
		"Home", "End", "PageUp", "PageDown", "Insert", "Delete"}

	for _, pad := range pads {
		for _, other := range others {
			if codePlaces[pad] == codePlaces[other] {
				t.Errorf("%s et %s mènent à la même place", pad, other)
			}
		}
	}
}

// TestPlaceOfCodeRefusesWhatTheEngineDoesNotName vérifie le second retour.
//
// **Il garde un piège de forme plutôt qu'une correspondance** : une lecture de table rend
// la valeur zéro pour une clé absente, et cette valeur se convertit en [input.Key] sans
// rien refuser. Un appelant qui ignorerait le booléen poserait donc une place qui n'existe
// pas à chaque touche inconnue — et la page en envoie beaucoup, le moteur ne nommant que
// quatre-vingt-six des deux cents et quelques `code` de la spécification.
//
// Le cas emploie `F13`, qui est un `code` réel que le moteur ne déclare pas : l'oubli
// qu'il attrape est de rendre `true` par défaut, ou de ne rien rendre du tout.
func TestPlaceOfCodeRefusesWhatTheEngineDoesNotName(t *testing.T) {
	if _, ok := placeOfCode("F13"); ok {
		t.Error("F13 n'est pas une place du moteur, et la table l'accepte")
	}
	if _, ok := placeOfCode(""); ok {
		t.Error("un code vide est accepté")
	}

	k, ok := placeOfCode("KeyW")
	if !ok {
		t.Fatal("KeyW n'est pas reconnue")
	}
	if k != input.KeyW {
		t.Errorf("KeyW mène à %#x, attendu %#x", uint16(k), uint16(input.KeyW))
	}
}

// declaredPlaces relève les places que le paquet `input` déclare, depuis ses sources.
//
// Le relevé plutôt qu'une liste, pour la raison que le cas de totalité donne : une liste
// tenue à la main serait une seconde copie de ce qu'on veut contrôler, et les deux
// dériveraient ensemble. C'est la technique de `places_test.go` du paquet `input`, lue
// depuis l'extérieur : le chemin est relatif parce que rien dans la chaîne d'outils ne
// donne les constantes d'un paquet déjà compilé.
func declaredPlaces(t *testing.T) map[string]int64 {
	t.Helper()

	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, "../../input", nil, 0)
	if err != nil {
		t.Fatal("lecture du paquet input:", err)
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
		t.Fatal("typage du paquet input:", err)
	}

	out := map[string]int64{}
	scope := checked.Scope()
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok || !c.Exported() || !strings.HasPrefix(name, "Key") {
			continue
		}
		named, ok := c.Type().(*types.Named)
		if !ok || named.Obj().Name() != "Key" {
			continue
		}
		if value, exact := constant.Int64Val(c.Val()); exact {
			out[name] = value
		}
	}
	if len(out) == 0 {
		t.Fatal("aucune place relevée : le balayage ne lit pas ce qu'il croit lire")
	}
	return out
}
