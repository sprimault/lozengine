// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le rejeu d'un scénario : ce qui se monte une fois, ce qui se refait par image, et
// l'écriture des PNG numérotés. Tout ce qui peut échouer échoue au montage, pour que
// le chemin d'une image n'ait plus d'erreur à rendre.

package viewer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Replay tient ce qu'un scénario réutilise d'une image à l'autre : son atlas, son
// tampon, sa liste de quads et le tampon de sortie s'il agrandit.
type Replay struct {
	scenario Scenario
	atlas    *raster.Atlas
	buffer   *render.Buffer
	out      *render.Buffer
	quads    render.DrawList
}

// NewReplay monte l'atlas du scénario et prépare le tampon à sa résolution.
//
// C'est ici que les erreurs ont le droit d'exister — un atlas se construit une fois,
// avant la première image. Passé ce point, plus rien n'échoue jusqu'à l'écriture.
func NewReplay(s Scenario) (*Replay, error) {
	if s.Atlas == nil || s.Quads == nil {
		return nil, fmt.Errorf("scenario %q: Atlas and Quads are required", s.Name)
	}

	atlas, err := s.Atlas()
	if err != nil {
		return nil, fmt.Errorf("atlas of scenario %q: %w", s.Name, err)
	}
	return &Replay{
		scenario: s,
		atlas:    atlas,
		buffer:   render.NewBuffer(s.Width, s.Height),
	}, nil
}

// Frame rend une image et rend le tampon interne, à la résolution du scénario.
//
// Le tampon est celui du rejeu et se réécrit à chaque appel : le lire, le comparer ou
// l'écrire se fait avant l'image suivante. C'est cette image-là que la suite de
// références compare, jamais une image agrandie — l'échelle est un habillage, pas un
// résultat de rendu.
func (r *Replay) Frame(number int, e InputLine) *render.Buffer {
	r.quads = r.scenario.Quads(number, e, r.quads[:0])
	raster.Clear(r.buffer, r.scenario.Background)
	raster.Draw(r.buffer, r.quads, r.atlas)
	return r.buffer
}

// Write écrit la dernière image rendue dans un PNG numéroté du répertoire, agrandie
// au facteur donné.
//
// Le nom porte le numéro sur quatre chiffres, pour que l'ordre des images soit celui
// des noms dans un répertoire comme dans une visionneuse d'images.
func (r *Replay) Write(dir string, number, factor int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	source := r.buffer
	if factor > 1 {
		// La taille reste en int jusqu'au contrôle : convertie d'abord, elle se
		// repliait sur un facteur élevé et rendait une image minuscule que rien ne
		// distinguait de celle qu'on avait demandée.
		wide := int(r.buffer.Width) * factor
		high := int(r.buffer.Height) * factor
		if wide > render.MaxBufferSide || high > render.MaxBufferSide {
			return fmt.Errorf("factor %d: the image would be %d×%d, beyond the %d per side a buffer holds", factor, wide, high, render.MaxBufferSide)
		}

		if r.out == nil || int(r.out.Width) != wide || int(r.out.Height) != high {
			r.out = render.NewBuffer(uint16(wide), uint16(high))
		}
		raster.Upscale(r.buffer, r.out, factor, r.scenario.Background)
		source = r.out
	}

	path := filepath.Join(dir, fmt.Sprintf("%s-%04d.png", r.scenario.Name, number))
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	if err := raster.WritePNG(source, file); err != nil {
		file.Close()
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	return nil
}

// Run rend toutes les images du scénario et les écrit.
//
// Les entrées se lisent dans l'ordre, une ligne par image, et un rejeu plus long que
// le fichier continue sur des entrées vides.
func (r *Replay) Run(dir string, lines []InputLine, factor int) error {
	for frame := range r.scenario.Frames {
		r.Frame(frame, InputLineAt(lines, frame))
		if err := r.Write(dir, frame, factor); err != nil {
			return err
		}
	}
	return nil
}
