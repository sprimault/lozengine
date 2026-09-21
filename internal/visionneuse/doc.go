// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package visionneuse rejoue une scène scriptée sans écran et en écrit les images.
//
// C'est l'outillage qui rend le rastériseur jugeable : une scène décrite en Go, un
// rendu, des PNG numérotés qu'on regarde ou qu'on compare à des références. Rien ici
// n'appartient au moteur, et le moteur n'en dépend pas.
//
// La contrainte qui gouverne tout le paquet est le déterminisme. Ni horloge, ni
// aléatoire, ni ressource lue sur le disque : deux rejeux doivent rendre exactement
// les mêmes pixels, faute de quoi une comparaison au pixel près ne veut rien dire.
package visionneuse
