// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'erreur que rend une plateforme dont le backend n'est pas écrit, sortie de son
// fichier sous contrainte pour qu'un contrôle puisse la lire.

package backend

import "errors"

// unsupported rend l'erreur qu'une plateforme sans backend oppose à une ouverture.
//
// **Sans contrainte de construction, et c'est tout l'objet de ce fichier.** Le message
// vivait dans `open_other.go`, dont la contrainte n'est satisfaite par aucune cible du
// dépôt : son cas de test ne tournait **nulle part**. `make test` tourne sur l'hôte, les
// trois runners sont Windows ou Linux, et `make cross` ignore les fichiers de test. Un
// repli qui aurait rendu une fenêtre nulle avec une erreur nulle passait tous les
// contrôles.
//
// Annoter le cas comme non mordant a été écarté : cela laisserait le repli sans garde,
// là que `tests.md` n'accepte cette mention que pour ce qu'on ne peut pas éprouver.
//
// **L'autre voie écartée a fini par être prise, pour une autre raison, et cela ne change
// rien ici.** Exécuter la suite sous `js/wasm` demandait un interpréteur WebAssembly sur
// chaque machine du parc, ce qui paraissait cher pour ce seul cas ; le jalon 5 l'a fait
// pour le déterminisme de la cible entière, et `make wasm-test` exige donc node. Ce
// fichier garde pourtant sa raison d'être : il fait tourner le cas sur les trois runners
// natifs, là que la passe wasm ne couvrirait qu'elle-même — et depuis que `js` a son
// propre backend, elle ne le couvre plus du tout.
//
// Le message nomme la plateforme et le jalon qui l'écrira : « non pris en charge »
// obligerait à chercher dans la feuille de route ce que le moteur attend encore. Il ne
// promet pas de fenêtre à la plateforme qui le lit, puisque certaines n'en auront
// jamais — macOS et iOS passent par le navigateur.
//
// **Il ne nomme plus le navigateur depuis que le jalon 5 l'a livré.** Un message qui
// annonce une famille déjà écrite envoie chercher ce qui est là, et c'est la plateforme
// la moins servie qui le lirait.
func unsupported(goos string) error {
	return errors.New("no window on " + goos +
		": no backend is written for it yet;" +
		" Wayland and DRM/KMS come at milestone 6")
}
