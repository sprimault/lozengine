// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario du mélange additif : ce qui s'ajoute au fond au lieu de s'y composer,
// sur deux fonds, par les deux chemins du rastériseur, et jusqu'à la saturation.

package viewer

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Additive éprouve le mélange additif, et il est le seul à pouvoir faire échouer trois
// entrées que le mélange ordinaire laisse passer.
//
// **La première est un raccourci devenu faux.** Une séquence de pixels opaques dessinée
// sans teinte se recopie telle quelle, ce qui est juste en composition et écrase le
// fond en additif — et cette recopie ne passe pas par l'arithmétique d'un pixel, donc
// aucun test de `blending` ne l'atteindrait.
//
// Cette séquence-là se dessine depuis une planche **grise**, et c'est la seule façon de
// la prendre en défaut : la planche marquée des autres scénarios est blanche, or un
// blanc ajouté sature au blanc et un blanc recopié reste blanc. Les deux chemins y
// donnent le même pixel, et le scénario serait vert dans les deux états.
//
// **La deuxième est la saturation.** Ajouter du clair sur du clair dépasse la valeur
// pleine, et un plafond manquant ferait reboucler la composante vers le sombre — un
// éclair qui noircit ce qu'il éclaire. D'où la bande claire sous la première rangée.
//
// **La troisième est le chemin par pixel**, que le miroir emprunte au lieu de la
// recopie : les deux doivent ajouter pareil.
func Additive() Scenario {
	const (
		width, height = 48, 32
		side          = 8
		pale          = 210
	)

	return Scenario{
		Name:       "additive",
		Width:      width,
		Height:     height,
		Frames:     1,
		Background: render.Color{R: 30, G: 30, B: 45, A: 255},

		// Deux planches : la marquée, qui montre les axes et porte un bord
		// translucide, et une grise opaque, qui est la seule à distinguer une
		// séquence ajoutée d'une séquence recopiée.
		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(
				markedSheet(side),
				borderedSheet(side, side, render.Color{R: 90, G: 90, B: 110, A: 255}),
			)
		},

		Quads: func(frame int, e InputLine, reserve render.DrawList) render.DrawList {
			quads := reserve

			// La bande claire, où l'ajout doit plafonner au lieu de reboucler.
			quads = append(quads, render.Quad{
				X: 0, Y: 0, Width: width, Height: 10,
				Tint: render.Color{R: pale, G: pale, B: pale, A: 255}, Flags: render.Flat,
			})

			// La planche grise, opaque et sans teinte : c'est elle qui emprunte la
			// recopie, donc elle qui dit si l'additif a été respecté. Sur le clair
			// elle sature, sur le sombre elle éclaircit — et recopiée, elle
			// donnerait son propre gris dans les deux cas.
			// La teinte est explicitement neutre, et c'est elle qui ouvre le chemin
			// de la recopie : une teinte laissée à sa valeur zéro est transparente,
			// donc elle annulerait le pixel et ces quads ne dessineraient rien.
			quads = append(quads,
				render.Quad{X: 2, Y: 1, Width: side, Height: side, Sheet: 1,
					Tint: render.White(), Flags: render.Additive},
				render.Quad{X: 2, Y: 12, Width: side, Height: side, Sheet: 1,
					Tint: render.White(), Flags: render.Additive},
			)

			// La même posée en composition, pour que l'image dise laquelle est
			// laquelle : sur ce fond sombre, les deux ne se ressemblent pas.
			quads = append(quads, render.Quad{
				X: 12, Y: 12, Width: side, Height: side, Sheet: 1,
				Tint: render.White(),
			})

			// Le chemin par pixel, que le miroir emprunte au lieu de la recopie.
			quads = append(quads, render.Quad{
				X: 22, Y: 12, Width: side, Height: side, Sheet: 1,
				Tint:  render.White(),
				Flags: render.Additive | render.MirrorX,
			})

			// L'alpha de la teinte dose ce que la source apporte, les composantes
			// étant prémultipliées.
			quads = append(quads, render.Quad{
				X: 32, Y: 12, Width: side, Height: side,
				Tint:  render.Color{R: 128, G: 128, B: 128, A: 128},
				Flags: render.Additive,
			})

			// Un fondu lumineux, qui est ce que Flat et Additive donnent ensemble. Il
			// traverse les deux fonds, donc il montre l'ajout et la saturation d'un
			// seul coup.
			quads = append(quads, render.Quad{
				X: 2, Y: 22, Width: 44, Height: 8,
				Tint: render.Color{R: 70, G: 0, B: 70, A: 255}, Flags: render.Flat | render.Additive,
			})

			return quads
		},
	}
}
