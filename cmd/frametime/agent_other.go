// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build !js

// Ce qui tient lieu de moteur de page là où il n'y en a pas : rien, et c'est dit.

package main

// agent rend ce qui exécute le relevé.
//
// **Hors d'une page, il n'y a rien à nommer** : la plateforme et la chaîne suffisent à
// citer un chiffre natif, là qu'un chiffre de page tient d'un triplet dont le moteur est
// le terme le plus volatil. La chaîne vide dit cette absence, et le relevé n'imprime alors
// pas de ligne pour elle.
func agent() string { return "" }
