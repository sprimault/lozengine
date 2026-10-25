// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

// Rectangle est une découpe dans une planche, en pixels de celle-ci.
//
// Il ne voyage pas jusqu'au rastériseur : un quad porte déjà sa source, et
// l'indirection coûterait une lecture de plus par quad. La découpe sert à qui
// construit des quads — fournisseur d'apparence, scène scriptée, texte.
type Rectangle struct{ X, Y, Largeur, Hauteur uint16 }
