// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La liste close des conversions de pointeur du dépôt, et le relevé qui la tient.

package lozengine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// conversions porte, par fichier, le nombre de conversions de pointeur que le dépôt
// assume. Un site de plus échoue, un site qui disparaît aussi : une liste qui survit à
// ce qu'elle décrit ne garde rien.
//
// **Elle se lit comme une décision, pas comme un relevé.** Chaque entrée a été discutée
// en pull request, parce qu'une conversion de pointeur est ce que le compilateur cesse
// de vérifier — et le dépôt vise des machines dont le boutisme et l'alignement
// diffèrent.
//
// Ce qui n'y figure pas : les lectures de disposition, `Sizeof`, `Offsetof` et
// `Alignof`. Elles ne convertissent rien, et le fichier qui fige les dispositions en
// porte des dizaines à lui seul.
var conversions = map[string]int{
	"internal/backend/icon_windows.go":         4,
	"internal/backend/open_windows.go":         3,
	"internal/backend/pixels.go":               2,
	"internal/backend/pointer_windows.go":      1,
	"internal/backend/present_js.go":           1,
	"internal/backend/present_windows.go":      2,
	"internal/backend/present_windows_test.go": 3,
	"internal/backend/win32_windows.go":        2,
	"internal/backend/window_windows.go":       4,
	"internal/backend/window_windows_test.go":  2,
	"render/layout_test.go":                    4,
}

// unsafeCalls sont les membres d'`unsafe` qui convertissent, par opposition à ceux qui
// mesurent une disposition.
var unsafeCalls = map[string]bool{
	"Pointer": true, "Slice": true, "String": true, "Add": true,
}

// TestConversionsMatchTheClosedList garde la liste close des conversions de pointeur.
//
// **Il lit l'arbre du programme et non le texte des fichiers**, et c'est ce qui le
// distingue du relevé par motif qu'il remplace. Celui-ci comptait toute occurrence du
// nom, donc une godoc qui **expliquait** une conversion en comptait pour une : écrire
// `unsafe.Pointer` en prose faisait rougir le contrôle, et le seul contournement était
// de ne pas nommer ce qu'on explique — une contrainte de style imposée par l'outillage,
// ce qu'un contrôle ne devrait jamais produire.
//
// L'arbre règle aussi le second défaut du motif, qui comptait par ligne avant de compter
// par occurrence : ici une conversion est un nœud, et deux sur une même ligne en font
// deux sans qu'on ait à y penser.
func TestConversionsMatchTheClosedList(t *testing.T) {
	found := map[string]int{}
	fset := token.NewFileSet()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name != "." && (strings.HasPrefix(name, ".") || name == "testdata") {
				return fs.SkipDir
			}
			// Un répertoire qui porte son propre `go.mod` n'est pas de ce module :
			// la fabrique et les essais y vivent, et leur code ne se contrôle pas ici.
			if path != "." {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		if n := countConversions(file); n != 0 {
			found[filepath.ToSlash(path)] = n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du dépôt : %v", err)
	}

	for _, line := range diffCounts(conversions, found) {
		t.Error(line)
	}
}

// countConversions compte les conversions de pointeur d'un fichier déjà analysé.
//
// Le paquet se reconnaît au nom qualifiant le membre, et non à l'import : un identifiant
// local nommé `unsafe` masquerait le paquet, ce qu'aucun code du dépôt ne fait et ce
// qu'un contrôle de cette forme ne peut de toute façon pas trancher sans types.
func countConversions(file *ast.File) int {
	n := 0
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == "unsafe" && unsafeCalls[selector.Sel.Name] {
			n++
		}
		return true
	})
	return n
}

// diffCounts rend ce qui sépare la liste close du relevé, une ligne par écart.
//
// Les deux sens sont dits : un fichier qui en porte plus que prévu, et un qui en porte
// moins — y compris zéro, où l'entrée de la liste n'a plus d'objet. Un renommage sort
// des deux côtés et se plaint deux fois, ce qui est juste : c'est un retrait et un ajout.
func diffCounts(expected, found map[string]int) []string {
	names := map[string]bool{}
	for name := range expected {
		names[name] = true
	}
	for name := range found {
		names[name] = true
	}

	var sorted []string
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	var lines []string
	for _, name := range sorted {
		want, got := expected[name], found[name]
		if want == got {
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"%s porte %d conversion(s) de pointeur, la liste close en déclare %d : "+
				"une conversion de plus se décide en pull request, elle ne s'ajoute pas",
			name, got, want))
	}
	return lines
}
