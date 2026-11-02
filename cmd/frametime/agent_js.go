// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le moteur de page qui exécute le relevé, nommé parce qu'un chiffre sous WebAssembly
// n'existe que pour un triplet.

package main

import "syscall/js"

// agent rend ce qui exécute le relevé, pour que le chiffre soit citable.
//
// **Un chiffre sous WebAssembly n'existe que pour un triplet** — machine, moteur de la
// page, chaîne Go —, et les trois interagissent : le même agrandissement a varié d'un
// facteur deux entre quatre combinaisons, l'effet du moteur changeant de signe selon la
// chaîne. Un relevé qui ne nomme pas le sien n'est donc pas reproductible, et il invite à
// comparer ce qui ne se compare pas.
//
// La marque se lit par les indications du client quand elles existent, et par la chaîne
// d'agent sinon : la première est courte et exacte, la seconde est longue et menteuse par
// conception, mais elle est toujours là.
func agent() string {
	navigator := js.Global().Get("navigator")
	if !navigator.Truthy() {
		return "moteur de page inconnu"
	}

	if data := navigator.Get("userAgentData"); data.Truthy() {
		brands := data.Get("brands")
		for i := range brands.Length() {
			brand := brands.Index(i)
			name := brand.Get("brand").String()
			// La marque de brouillage que les navigateurs insèrent pour décourager
			// la détection ; la nommer ici serait prendre du bruit pour un fait.
			if name == "" || name == "Not:A-Brand" {
				continue
			}
			return name + " " + brand.Get("version").String()
		}
	}
	return navigator.Get("userAgent").String()
}
