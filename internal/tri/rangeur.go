// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

// octetsParPasse et passes découpent une clé de soixante-quatre bits en tranches
// d'un octet, rangées de la plus faible à la plus forte.
const (
	bitsParPasse = 8
	seaux        = 1 << bitsParPasse
	passes       = 64 / bitsParPasse
)

// Rangeur ordonne des clés, et avec elles les indices qui disent d'où chacune vient.
//
// Par paquets et non par comparaison : une fonction de comparaison appelée des
// milliers de fois par image coûterait plus que tout le rangement réuni, et il n'y a
// rien à comparer — l'ordre est déjà dans la valeur de la clé.
//
// Ses tampons sont montés une fois. Au-delà de la taille prévue il en alloue de plus
// grands, ce qui arrive au montage d'une scène et pas en cours de partie.
type Rangeur struct {
	cles    []Cle
	sources []uint32
}

// NouveauRangeur monte un rangeur pour un nombre de choses à peindre par image.
func NouveauRangeur(capacite int) *Rangeur {
	if capacite < 0 {
		capacite = 0
	}

	return &Rangeur{
		cles:    make([]Cle, capacite),
		sources: make([]uint32, capacite),
	}
}

// Ranger ordonne les clés par valeur croissante, et déplace avec chacune l'indice de
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
func (r *Rangeur) Ranger(cles []Cle, sources []uint32) {
	n := len(cles)
	if n < 2 {
		return
	}

	if cap(r.cles) < n {
		r.cles = make([]Cle, n)
		r.sources = make([]uint32, n)
	}
	tamponCles, tamponSources := r.cles[:n], r.sources[:n]

	source, destination := cles, tamponCles
	indices, destIndices := sources, tamponSources
	bascules := 0

	for passe := 0; passe < passes; passe++ {
		decalage := uint(passe * bitsParPasse)

		var comptes [seaux]int
		for _, c := range source {
			comptes[byte(c>>decalage)]++
		}

		// Une passe dont tous les octets tombent dans le même seau ne range rien :
		// la sauter évite de recopier la séquence pour la laisser telle quelle. Sur
		// une scène d'une seule couche et d'un seul étage, c'est la moitié des
		// passes, et les deux plus coûteuses sont toujours sautées quand la carte
		// est petite.
		if comptes[byte(source[0]>>decalage)] == n {
			continue
		}

		somme := 0
		for seau, compte := range comptes {
			comptes[seau] = somme
			somme += compte
		}

		for i, c := range source {
			seau := byte(c >> decalage)
			rang := comptes[seau]
			comptes[seau]++

			destination[rang] = c
			destIndices[rang] = indices[i]
		}

		source, destination = destination, source
		indices, destIndices = destIndices, indices
		bascules++
	}

	// Un nombre impair de passes effectives laisse le résultat dans le tampon. Le
	// ramener coûte une recopie, là où forcer une passe de plus en coûterait deux.
	if bascules%2 == 1 {
		copy(cles, source)
		copy(sources, indices)
	}
}
