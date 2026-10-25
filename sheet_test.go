// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// writePNG écrit un PNG opaque de deux pixels de côté au chemin donné.
//
// Le fichier se fabrique au lieu d'être versionné : une image de deux pixels n'apprend
// rien à personne dans le dépôt, et la suite de non-régression a ses propres
// références pour ce qui se juge à l'œil.
func writePNG(t *testing.T, path string) {
	t.Helper()

	buffer := render.NewBuffer(2, 2)
	raster.Clear(buffer, White())

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("création de %s : %v", path, err)
	}
	defer file.Close()

	if err := raster.WritePNG(buffer, file); err != nil {
		t.Fatalf("écriture du PNG : %v", err)
	}
}

// TestOpenSheet vérifie ce que LoadSheet ne couvre pas : ouvrir le fichier, le
// décoder, et le refermer.
func TestOpenSheet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuile.png")
	writePNG(t, path)

	sheet, err := OpenSheet(path)
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	if sheet.Width != 2 || sheet.Height != 2 {
		t.Errorf("planche de %d×%d, attendu 2×2", sheet.Width, sheet.Height)
	}
	if n := len(sheet.Pixels); n != 4 {
		t.Errorf("%d pixels, attendu 4", n)
	}
}

// TestOpenSheetRefuses vérifie que les deux échecs remontent, et ils sont de nature
// différente : l'un vient du système de fichiers, l'autre du décodage.
//
// Le second est celui qui compte ici, parce qu'il traverse une fonction de plus : sans
// lui, rien ne dirait qu'une erreur de décodage sort bien d'OpenSheet au lieu d'être
// avalée avec le fichier qu'on referme.
func TestOpenSheetRefuses(t *testing.T) {
	dir := t.TempDir()
	notAPNG := filepath.Join(dir, "texte.png")
	if err := os.WriteFile(notAPNG, []byte("ceci n'est pas une image"), 0o644); err != nil {
		t.Fatalf("écriture de %s : %v", notAPNG, err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"fichier absent", filepath.Join(dir, "absent.png")},
		{"contenu qui n'est pas un PNG", notAPNG},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := OpenSheet(c.path); err == nil {
				t.Error("ouverture acceptée")
			}
		})
	}
}
