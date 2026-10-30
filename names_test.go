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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// french porte les mots qu'un identifiant du dépôt a réellement portés, relevés au
// moment de les retirer. Ce sont des mots entiers : la comparaison se fait après
// découpe du camelCase, donc `cas` n'accuse pas `cases` et `vert` n'accuse pas
// `vertical`. Le pluriel est traité par la comparaison, pas par la liste.
//
// Volontairement absents, et c'est le choix qui compte : les mots que l'anglais
// emploie aussi — `place`, `figure`, `image`, `part`, `procedure` —, et ceux de moins
// de quatre lettres qu'une découpe pourrait rendre par accident.
//
// **La liste s'allonge, et le premier code écrit après elle l'a prouvé.** Le backend
// Windows est arrivé avec treize noms français de plus, dont deux — `nom` et `cas` —
// que le relevé d'origine avait nommés sans qu'ils entrent ici, et `fenetres` que le
// singulier ne couvrait pas. Une liste de mots déjà vus ne protège de rien si on ne
// l'allonge pas au moment où elle manque.
var french = map[string]bool{
	"aplat": true, "apres": true, "atome": true, "avant": true, "avec": true,
	"blanc": true, "bleu": true, "borne": true, "cas": true, "chemin": true,
	"clair": true, "classe": true, "coin": true, "couleur": true,
	"decalage": true, "demi": true, "dernier": true, "drapeaux": true,
	"droite": true, "echecs": true, "ecrit": true, "ecrivain": true,
	"enregistre": true, "enregistrement": true, "facteur": true, "fenetre": true,
	"fichier": true, "gauche": true, "gris": true, "haut": true, "haute": true,
	"hauteur": true, "inscrite": true, "interne": true, "jetable": true,
	"largeur": true, "loin": true, "majorant": true, "nom": true, "noyau": true,
	"numero": true, "octets": true, "ordre": true, "pion": true, "pire": true,
	"planche": true, "poignee": true, "pointeur": true, "ponderation": true,
	"pose": true, "premier": true, "proche": true, "profondeur": true,
	"quarts": true, "racine": true, "regenerer": true, "regler": true,
	"rien": true, "rouge": true, "sans": true, "somme": true, "sortie": true,
	"taille": true, "tampon": true, "teinte": true, "titre": true, "tuile": true,
	"vert": true, "veut": true, "voile": true,
}

// isFrench dit si un mot découpé est de la liste, au singulier comme au pluriel.
//
// Le pluriel se traite ici et non dans la liste, parce que l'y écrire deux fois est
// la façon sûre d'en oublier un : `fenetre` y était, `fenetres` non, et c'est passé.
// Retirer un `s` final n'ouvre pas de faux positif sur les mots anglais du dépôt —
// `cases` donne `case`, `windows` donne `window`, aucun des deux n'étant dans la
// liste.
func isFrench(word string) bool {
	if french[word] {
		return true
	}
	return strings.HasSuffix(word, "s") && french[strings.TrimSuffix(word, "s")]
}

// TestIdentifiersAreNotFrench refuse tout identifiant du dépôt portant un mot de
// [french], et il couvre une classe là où un scénario ne couvre qu'un cas : la
// bascule des noms s'est faite en quatre lots, et ce qui la referait reculer est
// qu'un test neuf se copie sur un test voisin.
//
// Ce qu'il n'attrape pas : un mot français inédit. Fermer cela demanderait un
// dictionnaire, que personne ne tiendrait à jour ; la liste s'allonge d'un mot le jour
// où on en retire un du code.
//
// **Son périmètre est le module et non le disque** : un répertoire qui porte son propre
// `go.mod` est un autre module, donc du code que ce dépôt ne gouverne pas, et le juger
// ici ferait rougir la suite sur des fichiers qu'aucune de ses règles n'atteint. Le
// dépôt n'était vert que par hasard avant cela — le seul module imbriqué commence aussi
// par un point, ce qui le faisait sauter pour une autre raison.
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
# le pion part du bord, sans voile
pawn 60 30 no-veil
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

// TestIsFrench garde le trou par lequel le backend Windows est passé : la liste portait
// `fenetre` et l'identifiant s'appelait `fenetres`.
//
// Les deux cas anglais comptent autant que les français : si retirer un `s` final
// ouvrait des faux positifs, le contrôle deviendrait pénible et finirait désarmé.
func TestIsFrench(t *testing.T) {
	cases := []struct {
		word string
		want bool
	}{
		{"fenetre", true},
		{"fenetres", true},
		{"tuiles", true},
		{"nom", true},
		{"cas", true},
		{"cases", false},
		{"windows", false},
		{"procedure", false},
		{"rows", false},
		{"vertical", false},
	}
	for _, c := range cases {
		t.Run(c.word, func(t *testing.T) {
			if got := isFrench(c.word); got != c.want {
				t.Errorf("isFrench(%q) rend %v, attendu %v", c.word, got, c.want)
			}
		})
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
			if isFrench(word) {
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
