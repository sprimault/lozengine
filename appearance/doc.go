// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package appearance résout ce qu'une chose montre : de ce qu'elle est, de l'état où
// elle se trouve, de la direction vers laquelle elle regarde et du moment de son
// animation, un fournisseur tire de quoi remplir un quad.
//
// **Le moteur n'appelle jamais un fournisseur.** C'est le jeu qui le consulte, quand
// l'apparence d'une entité change, et qui écrit le résultat par
// [scene.Scene.SetAspect]. Résoudre par entité et par image serait un appel dans le
// chemin de rendu, que l'invariant interdit : un appel venu d'un autre langage coûte
// des centaines de nanosecondes, ce qu'un chemin parcouru des milliers de fois par
// image ne peut pas absorber. Et le faire en amont ne coûte rien, une entité changeant
// d'apparence bien plus rarement qu'elle n'est dessinée.
//
// Tout y est donc facultatif. Un jeu qui compose ses aspects à la main n'importe pas
// ce paquet, et c'est le cas des exemples les plus courts.
package appearance
