// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le chemin d'entrée Windows alloue par image, figé au compte mesuré.

package backend

import "testing"

// TestInputPathAllocatesExactlyWhatWasMeasured fige ce que le chemin d'entrée alloue par
// image, et c'est la seule garde que ce chemin ait jamais eue.
//
// **La règle du dépôt interdit d'allouer par image, et ce chemin y fait exception.** Elle
// est nommée et chiffrée plutôt que tacite : `LazyProc.Call` porte `//go:uintptrescapes`,
// donc il alloue quelle que soit son arité et force le pointé sur le tas. La pompe en fait
// un par message et un pour la file vide, la taille client trois de plus.
//
// **Ce qui a tranché est la mesure, et elle n'est pas discutable** : 115 ns et 96 octets
// pour `Input` sur une file vide, 48 ns et 40 octets pour `Size`. Les deux ensemble valent
// 163 ns par image, soit **un millième de pour cent** du budget d'une image à soixante par
// seconde. Passer ces appels à `SyscallN` sur des tranches gardées — ce que la présentation
// fait — demanderait quatre réécritures et trois champs de plus sur un chemin dont la
// couverture est récente, pour ce millième. L'arbitrage est dans `DECISIONS.md`.
//
// **Ce n'est donc pas une garde à zéro, c'est une garde de non-régression** : le compte
// exact est exigé, un de plus échoue, un de moins aussi. C'est la forme que la cible `vet`
// emploie pour ses conversions de pointeur — une liste qui survit à ce qu'elle décrit ne
// garde rien —, et elle dit ici qu'un appel ajouté à la pompe se verra.
func TestInputPathAllocatesExactlyWhatWasMeasured(t *testing.T) {
	var w window
	track(t, &w)

	for _, c := range []struct {
		name string
		want float64
		run  func()
	}{
		{"la pompe sur une file vide", 2, func() { w.Input() }},
		{"la taille de la zone utile", 3, func() { w.Size() }},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(50, c.run); got != c.want {
				t.Errorf("%v allocations par image, %v mesurées et attendues"+
					" — un appel système de plus ou de moins sur ce chemin",
					got, c.want)
			}
		})
	}
}
