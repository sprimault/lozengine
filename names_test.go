// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le refus des noms français déjà rencontrés une fois, pour que la bascule des
// identifiants ne se défasse pas à mesure qu'un test neuf se copie sur son voisin.

package lozengine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// french porte les mots qu'un identifiant du dépôt a réellement portés, relevés au
// moment de les retirer. Ce sont des mots entiers : la comparaison se fait après
// découpe du camelCase, donc `cas` n'accuse pas `cases` et `vert` n'accuse pas
// `vertical`.
//
// Volontairement absents, et c'est le choix qui compte : les mots que l'anglais
// emploie aussi — `place`, `figure`, `image`, `part` —, et ceux de moins de quatre
// lettres qu'une découpe pourrait rendre par accident.
var french = map[string]bool{
	"aplat": true, "apres": true, "avant": true, "avec": true, "blanc": true,
	"bleu": true, "borne": true, "chemin": true, "clair": true, "coin": true,
	"couleur": true, "decalage": true, "demi": true, "drapeaux": true,
	"echecs": true, "ecrit": true, "ecrivain": true, "facteur": true,
	"fenetre": true, "fichier": true, "gris": true, "haut": true, "haute": true,
	"hauteur": true, "interne": true, "jetable": true, "largeur": true,
	"loin": true, "majorant": true, "noyau": true, "numero": true,
	"octets": true, "ordre": true, "pion": true, "pire": true, "planche": true,
	"planches": true, "ponderation": true, "pose": true, "premier": true,
	"proche": true, "profondeur": true, "quarts": true, "racine": true,
	"regenerer": true, "regler": true, "rien": true, "rouge": true,
	"sans": true, "somme": true, "sortie": true, "tampon": true,
	"teinte": true, "tuile": true, "tuiles": true, "vert": true, "veut": true,
	"voile": true,
}

// TestIdentifiersAreNotFrench refuse tout identifiant du dépôt portant un mot de
// [french], et il couvre une classe là où un scénario ne couvre qu'un cas : la
// bascule des noms s'est faite en quatre lots, et ce qui la referait reculer est
// qu'un test neuf se copie sur un test voisin.
//
// Ce qu'il n'attrape pas : un mot français inédit. Fermer cela demanderait un
// dictionnaire, que personne ne tiendrait à jour ; la liste s'allonge d'un mot le jour
// où on en retire un du code.
func TestIdentifiersAreNotFrench(t *testing.T) {
	fset := token.NewFileSet()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name != "." && (strings.HasPrefix(name, ".") || name == "testdata") {
				return fs.SkipDir
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
		for _, f := range frenchIdentifiers(fset, file) {
			t.Errorf("%s: %q holds the French word %q", f.position, f.name, f.word)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository: %v", err)
	}
}

// TestFrenchIdentifiersAreDetected est ce qui donne sa valeur au test ci-dessus, et
// sans lui le dépôt étant déjà propre, personne ne saurait si le relevé regarde quoi
// que ce soit.
//
// Il porte aussi la raison d'être d'un test là où une cible de `make` aurait suffi en
// apparence : la source éprouvée contient une godoc française et une chaîne brute de
// prose française, que l'arbre syntaxique ne rend pas. Un filtre textuel aurait à les
// distinguer du code, et une chaîne brute s'étend sur plusieurs lignes.
func TestFrenchIdentifiersAreDetected(t *testing.T) {
	const source = `package essai

// Le tampon reçoit la couleur : une godoc reste en français, et rien de ce commentaire
// ne doit être relevé.
var lines = ` + "`" + `
# le pion part du bord
pion 60 30 sans-voile
` + "`" + `

var tampon int
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "essai.go", source, 0)
	if err != nil {
		t.Fatalf("parsing the sample: %v", err)
	}

	found := frenchIdentifiers(fset, file)
	if len(found) != 1 {
		t.Fatalf("relevé : %v, attendu le seul identifiant", found)
	}
	if found[0].name != "tampon" || found[0].word != "tampon" {
		t.Errorf("relevé : %q sur %q", found[0].name, found[0].word)
	}
}

// TestWords vérifie la découpe dont dépend la comparaison par mot entier : sans elle,
// `cases` porterait `cas` et `vertical` porterait `vert`.
func TestWords(t *testing.T) {
	cases := []struct {
		identifier string
		want       []string
	}{
		{"tampon", []string{"tampon"}},
		{"plancheSolClair", []string{"planche", "sol", "clair"}},
		{"cases", []string{"cases"}},
		{"vertical", []string{"vertical"}},
		{"writePNG", []string{"write", "png"}},
		{"MaxBufferSide", []string{"max", "buffer", "side"}},
	}
	for _, c := range cases {
		t.Run(c.identifier, func(t *testing.T) {
			got := words(c.identifier)
			if strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Errorf("découpe : %v, attendu %v", got, c.want)
			}
		})
	}
}

// finding est un identifiant fautif et le mot qui l'accuse.
type finding struct {
	position string
	name     string
	word     string
}

// frenchIdentifiers relève les identifiants d'un fichier analysé qui portent un mot
// de [french].
//
// Il parcourt tous les identifiants et non les seules déclarations : un champ nommé en
// français se lit aussi à l'usage, et le paquet qui le déclare n'est pas toujours celui
// qu'on est en train de corriger.
func frenchIdentifiers(fset *token.FileSet, file *ast.File) []finding {
	var found []finding

	ast.Inspect(file, func(n ast.Node) bool {
		identifier, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		for _, word := range words(identifier.Name) {
			if french[word] {
				found = append(found, finding{
					position: fset.Position(identifier.Pos()).String(),
					name:     identifier.Name,
					word:     word,
				})
			}
		}
		return true
	})
	return found
}

// words découpe un identifiant à chaque majuscule et rend ses mots en minuscules.
//
// Une suite de majuscules reste un seul mot — `writePNG` rend `write` et `png` —,
// parce que les acronymes du dépôt n'ont pas à être comparés lettre par lettre.
func words(identifier string) []string {
	var out []string
	start := 0

	for i, r := range identifier {
		if i == 0 || !unicode.IsUpper(r) {
			continue
		}
		// Une majuscule qui suit une majuscule prolonge un acronyme.
		if previous := rune(identifier[i-1]); unicode.IsUpper(previous) {
			continue
		}
		out = append(out, strings.ToLower(identifier[start:i]))
		start = i
	}
	return append(out, strings.ToLower(identifier[start:]))
}
