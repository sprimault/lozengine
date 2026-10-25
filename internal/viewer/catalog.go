// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package viewer

import "sort"

// catalog associe chaque nom de scénario à sa construction. Une fonction et non un
// Scenario tout fait : chaque rejeu part d'un scénario neuf.
var catalog = map[string]func() Scenario{
	"edges":  Edges,
	"blend":  Blend,
	"target": Target,
	"core":   Core,
	"order":  Order,
}

// ByName rend le scénario de ce nom.
func ByName(name string) (Scenario, bool) {
	build, ok := catalog[name]
	if !ok {
		return Scenario{}, false
	}
	return build(), true
}

// Names rend les noms du catalogue, triés, pour qu'un message d'aide ou une suite de
// tests les parcoure dans un ordre stable.
func Names() []string {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
