// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package sorting

// passBits et passes découpent une clé de soixante-quatre bits en tranches d'un
// octet, rangées de la plus faible à la plus forte.
const (
	passBits = 8
	buckets  = 1 << passBits
	passes   = 64 / passBits
)

// Radix ordonne des clés, et avec elles les indices qui disent d'où chacune vient.
//
// Par paquets et non par comparaison : une fonction de comparaison appelée des
// milliers de fois par image coûterait plus que tout le rangement réuni, et il n'y a
// rien à comparer — l'ordre est déjà dans la valeur de la clé.
//
// Ses tampons sont montés une fois. Au-delà de la taille prévue il en alloue de plus
// grands, ce qui arrive au montage d'une scène et pas en cours de partie.
type Radix struct {
	keys    []Key
	sources []uint32
}

// NewRadix monte un rangeur pour un nombre de choses à peindre par image.
func NewRadix(capacity int) *Radix {
	if capacity < 0 {
		capacity = 0
	}

	return &Radix{
		keys:    make([]Key, capacity),
		sources: make([]uint32, capacity),
	}
}

// Sort ordonne les clés par valeur croissante, et déplace avec chacune l'indice de
// même rang. Les deux tranches ont la même longueur.
//
// Les indices sont ce qui permet de ne pas déplacer les quads eux-mêmes : un quad pèse
// vingt octets, un indice quatre, et le rangement les recopie huit fois.
//
// Le rangement est stable, et il le doit pour deux raisons sans rapport l'une avec
// l'autre. La première est mécanique : ranger par passes successives, de l'octet le
// plus faible au plus fort, n'est juste que si chaque passe préserve l'ordre que les
// précédentes ont établi — sinon la dernière défait les autres, et le résultat n'est
// même plus ordonné. La seconde est visible : deux choses de clés égales se
// peindraient dans un ordre changeant d'une image à l'autre, et deux sprites
// superposés se relaieraient au premier plan. C'est le scintillement, et il se voit
// tout de suite.
func (r *Radix) Sort(keys []Key, sources []uint32) {
	n := len(keys)
	if n < 2 {
		return
	}

	if cap(r.keys) < n {
		r.keys = make([]Key, n)
		r.sources = make([]uint32, n)
	}
	bufferKeys, bufferSources := r.keys[:n], r.sources[:n]

	source, destination := keys, bufferKeys
	indices, destIndices := sources, bufferSources
	flips := 0

	for pass := 0; pass < passes; pass++ {
		shift := uint(pass * passBits)

		var counts [buckets]int
		for _, c := range source {
			counts[byte(c>>shift)]++
		}

		// Une passe dont tous les octets tombent dans le même seau ne range rien :
		// la sauter évite de recopier la séquence pour la laisser telle quelle. Sur
		// une scène d'une seule couche et d'un seul étage, c'est la moitié des
		// passes, et les deux plus coûteuses sont toujours sautées quand la carte
		// est petite.
		if counts[byte(source[0]>>shift)] == n {
			continue
		}

		sum := 0
		for bucket, count := range counts {
			counts[bucket] = sum
			sum += count
		}

		for i, c := range source {
			bucket := byte(c >> shift)
			rank := counts[bucket]
			counts[bucket]++

			destination[rank] = c
			destIndices[rank] = indices[i]
		}

		source, destination = destination, source
		indices, destIndices = destIndices, indices
		flips++
	}

	// Un nombre impair de passes effectives laisse le résultat dans le tampon. Le
	// ramener coûte une recopie, là où forcer une passe de plus en coûterait deux.
	if flips%2 == 1 {
		copy(keys, source)
		copy(sources, indices)
	}
}
