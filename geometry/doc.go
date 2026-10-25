// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package geometry convertit entre le repère du monde et celui de l'écran.
//
// Le monde compte en cellules, l'écran en pixels, et la conversion est une matrice
// construite une fois par un de ses quatre constructeurs. Aucune constante de ratio
// ne vit ailleurs : pas de division par deux disséminée dans le code, pas de
// disposition en losanges écrite à la main.
//
// Tout y est en virgule fixe à partir de l'entrée. Une même scène rend ainsi la
// même image au pixel près du poste au Raspberry Pi, ce que le flottant ne
// garantirait pas — la spécification du langage autorise à fusionner « x·y + z » en
// une instruction unique, que certaines architectures pratiquent et d'autres non, et
// une matrice n'est faite que de cela.
package geometry
