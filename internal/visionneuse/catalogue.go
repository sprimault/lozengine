// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import "sort"

// catalogue associe chaque nom de scénario à sa construction. Une fonction et non un
// Scenario tout fait : chaque rejeu part d'un scénario neuf.
var catalogue = map[string]func() Scenario{
	"mire": Mire,
}

// Par rend le scénario de ce nom.
func Par(nom string) (Scenario, bool) {
	construire, ok := catalogue[nom]
	if !ok {
		return Scenario{}, false
	}
	return construire(), true
}

// Noms rend les noms du catalogue, triés, pour qu'un message d'aide ou une suite de
// tests les parcoure dans un ordre stable.
func Noms() []string {
	noms := make([]string, 0, len(catalogue))
	for nom := range catalogue {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	return noms
}
