// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'erreur que rend une plateforme dont le backend n'est pas écrit, sortie de son
// fichier sous contrainte pour qu'un contrôle puisse la lire.

package backend

import "errors"

// unsupported rend l'erreur qu'une plateforme sans backend oppose à une ouverture.
//
// **Sans contrainte de construction, et c'est tout l'objet de ce fichier.** Le message
// vivait dans `open_other.go`, sous `!windows && !linux` : aucune cible du dépôt
// n'exécute un `GOOS` qui le compile, donc son cas de test ne tournait **nulle part**.
// `make test` tourne sur l'hôte, les trois runners sont Windows ou Linux, `make cross`
// ignore les fichiers de test, et le `js/wasm` de `make vet` ne fait que compiler. Un
// repli qui aurait rendu une fenêtre nulle avec une erreur nulle passait tous les
// contrôles.
//
// Deux autres voies ont été écartées. Annoter le cas comme non mordant laisserait le
// repli sans garde, là que `tests.md` n'accepte cette mention que pour ce qu'on ne peut
// pas éprouver. Et exécuter `GOOS=js go test` demanderait un interpréteur WebAssembly
// sur chaque machine du parc, pour un gain que ce découpage donne gratuitement.
//
// Le message nomme la plateforme et le jalon qui l'écrira : « non pris en charge »
// obligerait à chercher dans la feuille de route ce que le moteur attend encore. Il ne
// promet aucune des deux familles à la plateforme qui le lit, puisque certaines n'en
// auront jamais — macOS et iOS passent par le navigateur.
func unsupported(goos string) error {
	return errors.New("no window on " + goos +
		": no backend is written for it yet; the browser comes at milestone 5," +
		" Wayland and DRM/KMS at milestone 6")
}
