// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le chemin d'entrée X11 alloue par image, et il ne doit rien allouer.

package backend

import "testing"

// TestInputDoesNotAllocate garde le chemin d'entrée X11 contre l'allocation par image, et
// c'est la première garde qu'il ait.
//
// **Le régime mesuré est celui d'une image ordinaire : la socket est vide.** C'est le cas
// courant — un joueur n'enfonce pas une touche à chaque pas —, et c'est aussi celui où une
// allocation se verrait le moins, donc celui qu'il faut éprouver. `Input` appelle `poll`
// puis `receive`, qui demande à la socket ce qu'elle tient déjà sans attendre.
//
// **Ce qui allouait, et pourquoi aucun contrôle ne le voyait** : `receive` construisait sa
// fermeture de lecture à chaque appel et la passait à `syscall.RawConn.Read`, une méthode
// d'interface. La valeur de fonction s'échappe donc, et les deux variables qu'elle capture
// partent sur le tas avec elle. Les deux autres chemins par image du backend — la
// présentation et la conversion de pixels — avaient leur garde ; l'entrée n'en avait
// aucune, dans aucun des deux backends.
//
// La fermeture est désormais montée une fois et gardée sur la connexion. **Jamais une
// valeur de méthode à sa place** : celle-ci est elle-même une fermeture, allouée à chaque
// prise, et ce cas ne la distinguerait pas de l'ancienne écriture.
func TestInputDoesNotAllocate(t *testing.T) {
	w, _ := windowPair(t)

	// Une première lecture monte ce qui se garde — la fermeture, les tampons —, comme
	// l'appel de chauffe d'AllocsPerRun le ferait pour le reste.
	w.Input()

	if n := testing.AllocsPerRun(50, func() { w.Input() }); n != 0 {
		t.Errorf("%v allocations par image sur une socket vide, attendu aucune", n)
	}
}

// BenchmarkInput chiffre le chemin d'entrée, seul chemin par image des backends qui n'avait
// pas de banc.
//
// Socket vide, le régime d'une image ordinaire. Ce qu'il mesure est le coût de demander à
// la socket ce qu'elle tient : un appel système qui rend `EAGAIN`, et rien d'autre.
func BenchmarkInput(b *testing.B) {
	w, _ := windowPair(b)
	w.Input()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w.Input()
	}
}
