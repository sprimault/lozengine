// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package sorting décide l'ordre de dessin : il écarte ce qui sort de l'image, compose
// une clé par chose à peindre, et range les quads sur ces clés.
//
// L'ordre tient tout entier dans un nombre. Comparer deux clés revient à comparer
// leurs couches, puis leurs étages à couche égale, et ainsi de suite jusqu'au dernier
// champ — sans une seule condition à écrire, et sans fonction de comparaison appelée
// des milliers de fois par image.
package sorting
