// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le codage du fil doit tenir : la taille et l'opcode dans un seul mot, le
// terminateur compté dans la longueur d'une chaîne, et les trois valeurs d'un
// identifiant neuf dont l'interface n'est pas déclarée.

package backend

import (
	"strings"
	"testing"
)

// TestWLWriterPutsTheSizeAndOpcodeInOneWord fige la composition du second mot de
// l'en-tête.
//
// **Les deux moitiés sont interchangeables à l'œil et pas au protocole** : taille dans
// les seize bits de poids fort, opcode dans les seize de poids faible. Les inverser donne
// un message dont la taille vaut l'opcode — donc une taille de un ou deux octets, sous le
// minimum de huit —, et le compositeur ferme la connexion sans dire pourquoi.
//
// Le cas vérifie aussi que la taille part de l'en-tête et non des arguments, ce que la
// spécification dit en donnant huit pour minimum.
func TestWLWriterPutsTheSizeAndOpcodeInOneWord(t *testing.T) {
	var w wlWriter
	w.begin(1, 0x0004)
	w.uint(0xdeadbeef)
	w.end()

	if got, want := len(w.buf), 12; got != want {
		t.Fatalf("le message fait %d octets, attendu %d : l'en-tête plus un argument", got, want)
	}
	if got, want := wlOrder.Uint32(w.buf[0:]), uint32(1); got != want {
		t.Errorf("l'émetteur vaut %#x, attendu %#x", got, want)
	}

	second := wlOrder.Uint32(w.buf[4:])
	if got, want := second>>16, uint32(12); got != want {
		t.Errorf("la taille annoncée vaut %d, attendu %d", got, want)
	}
	if got, want := second&0xffff, uint32(0x0004); got != want {
		t.Errorf("l'opcode vaut %#x, attendu %#x", got, want)
	}
}

// TestWLWriterCountsTheNullTerminatorInAStringLength garde la longueur d'une chaîne.
//
// **Le mode de panne est le plus coûteux à diagnostiquer du codage** : une longueur sans
// le terminateur décale d'un octet tout ce qui suit dans le message, et le compositeur lit
// alors des arguments plausibles mais faux. L'erreur de protocole qui finit par tomber
// n'a aucun rapport avec la cause, et elle tombe sur une autre requête.
//
// Le cas fige aussi le bourrage, qui porte la chaîne de onze à douze octets.
func TestWLWriterCountsTheNullTerminatorInAStringLength(t *testing.T) {
	var w wlWriter
	w.begin(2, 0)
	w.string("wl_shm")
	w.end()

	if got, want := wlOrder.Uint32(w.buf[wlHeaderSize:]), uint32(7); got != want {
		t.Errorf("la longueur annoncée vaut %d, attendu %d : six caractères et le terminateur", got, want)
	}
	if got, want := len(w.buf), wlHeaderSize+12; got != want {
		t.Errorf("le message fait %d octets, attendu %d : quatre de longueur, sept de contenu, un de bourrage",
			got, want)
	}
	if got := w.buf[wlHeaderSize+4+6]; got != 0 {
		t.Errorf("l'octet qui suit le contenu vaut %d, attendu 0 : le terminateur est écrit", got)
	}
}

// TestWLWriterWritesThreeValuesForAnUndeclaredNewID garde le piège de `wl_registry.bind`.
//
// **Ce cas est le seul du dépôt à pouvoir attraper cette erreur**, et elle ne se voit pas
// dans la description du protocole : l'argument y est déclaré `new_id` comme celui de
// `create_surface`, et rien ne distingue les deux qu'un attribut `interface` absent. La
// spécification du fil l'énonce — « a new_id is preceded by a string specifying the
// interface name, and a uint specifying the version » —, donc trois valeurs là qu'un
// `new_id` déclaré n'en écrit qu'une.
//
// Il rougit si le codage n'écrit que l'identifiant : le message ferait alors douze octets
// au lieu de vingt-huit, et le compositeur lirait la suite du flux comme la fin de ce
// message.
func TestWLWriterWritesThreeValuesForAnUndeclaredNewID(t *testing.T) {
	var w wlWriter
	w.begin(2, 0)
	w.uint(7)
	w.newID("wl_compositor", 6, 3)
	w.end()

	// Huit d'en-tête, quatre pour le nom du global, vingt pour la chaîne de treize
	// caractères avec son terminateur et son bourrage, quatre pour la version, quatre
	// pour l'identifiant.
	if got, want := len(w.buf), 8+4+20+4+4; got != want {
		t.Fatalf("le message fait %d octets, attendu %d : l'interface et la version précèdent l'identifiant",
			got, want)
	}

	r := wlReader{buf: w.buf[wlHeaderSize:]}
	if got, want := r.uint(), uint32(7); got != want {
		t.Errorf("le nom du global vaut %d, attendu %d", got, want)
	}
	if got, want := r.string(), "wl_compositor"; got != want {
		t.Errorf("l'interface vaut %q, attendu %q", got, want)
	}
	if got, want := r.uint(), uint32(6); got != want {
		t.Errorf("la version vaut %d, attendu %d", got, want)
	}
	if got, want := r.uint(), uint32(3); got != want {
		t.Errorf("l'identifiant vaut %d, attendu %d", got, want)
	}
	if r.err != nil {
		t.Errorf("la relecture a manqué de place : %v", r.err)
	}
}

// TestWLReaderRefusesToReadPastTheMessage garde la seule chose que le lecteur apporte.
//
// Un événement tronqué — par un compositeur fautif, ou par un découpage du flux mal fait —
// ne doit pas rendre les octets du message suivant comme s'ils étaient les siens. Le mode
// de panne serait un état de fenêtre construit sur les arguments de quelqu'un d'autre,
// sans qu'aucune erreur ne le signale.
//
// Le cas éprouve les deux lectures et les deux bords : un entier à qui il manque un octet,
// une chaîne dont la longueur dépasse le message, et une longueur nulle — que la
// spécification ne peut pas produire, puisqu'elle compte toujours le terminateur.
func TestWLReaderRefusesToReadPastTheMessage(t *testing.T) {
	cases := []struct {
		label string
		buf   []byte
		read  func(*wlReader)
	}{
		{"un entier à qui il manque un octet", make([]byte, 3), func(r *wlReader) { r.uint() }},
		{"une chaîne sans sa longueur", make([]byte, 2), func(r *wlReader) { r.string() }},
		{
			"une chaîne dont la longueur dépasse le message",
			func() []byte {
				b := make([]byte, 8)
				wlOrder.PutUint32(b, 64)
				return b
			}(),
			func(r *wlReader) { r.string() },
		},
		{
			"une longueur de chaîne nulle, que le protocole ne produit pas",
			make([]byte, 8),
			func(r *wlReader) { r.string() },
		},
		{
			"un tableau dont la longueur dépasse le message",
			func() []byte {
				b := make([]byte, 8)
				wlOrder.PutUint32(b, 64)
				return b
			}(),
			func(r *wlReader) { r.array() },
		},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			r := wlReader{buf: c.buf}
			c.read(&r)
			if r.err == nil {
				t.Fatalf("la lecture a réussi, attendu un refus")
			}
			if !strings.Contains(r.err.Error(), "too short") {
				t.Errorf("le refus dit %q, attendu qu'il nomme la troncature", r.err)
			}
		})
	}
}

// TestWLReaderReadsASignedIntAndAnArray garde les deux lectures que la configuration
// d'une fenêtre demande.
//
// **Les deux ont un mode de panne silencieux, et ce sont deux silences différents.** Une
// largeur négative lue en non signé rend 4 294 967 295 au lieu de -1, donc une taille de
// fenêtre absurde que rien ne signale ; et un tableau dont on n'avance pas le curseur
// laisse la lecture suivante tomber sur son contenu.
//
// Le cas compose exactement ce que `xdg_toplevel.configure` porte — deux entiers signés
// puis un tableau — pour que le bourrage du tableau soit celui du protocole et non un
// montage de circonstance.
func TestWLReaderReadsASignedIntAndAnArray(t *testing.T) {
	var w wlWriter
	w.begin(3, wlXDGToplevelConfigure)
	w.uint(0xffffffff) // -1 en complément à deux, tel qu'un compositeur l'écrirait
	w.uint(720)
	w.uint(2)
	w.buf = append(w.buf, 4, 0)
	w.pad()
	w.uint(0xcafe)
	w.end()

	r := wlReader{buf: w.buf[wlHeaderSize:]}
	if got, want := r.int(), int32(-1); got != want {
		t.Errorf("la largeur vaut %d, attendu %d", got, want)
	}
	if got, want := r.int(), int32(720); got != want {
		t.Errorf("la hauteur vaut %d, attendu %d", got, want)
	}
	if got := r.array(); len(got) != 2 || got[0] != 4 || got[1] != 0 {
		t.Errorf("le tableau vaut %v, attendu [4 0]", got)
	}
	if got, want := r.uint(), uint32(0xcafe); got != want {
		t.Errorf("l'argument qui suit le tableau vaut %#x, attendu %#x : le bourrage n'a pas été franchi",
			got, want)
	}
	if r.err != nil {
		t.Errorf("la lecture a manqué de place : %v", r.err)
	}
}

// TestWLReaderReadsAnEmptyArray garde la différence avec une chaîne.
//
// Une longueur nulle est refusée pour une chaîne, qui compte toujours son terminateur, et
// valide pour un tableau, qui n'en a pas. C'est le cas **courant** de
// `xdg_toplevel.configure` : une fenêtre ordinaire n'est ni maximisée, ni en plein écran,
// ni en cours de redimensionnement, donc sa liste d'états est vide. La refuser ferait
// échouer le montage de toute fenêtre normale.
func TestWLReaderReadsAnEmptyArray(t *testing.T) {
	r := wlReader{buf: make([]byte, 4)}
	if got := r.array(); len(got) != 0 {
		t.Errorf("le tableau porte %d octets, attendu aucun", len(got))
	}
	if r.err != nil {
		t.Errorf("un tableau vide est refusé : %v", r.err)
	}
}

// TestWLReaderKeepsTheFirstFailure garde que les lectures suivantes ne masquent pas la
// première.
//
// Sans cela, un appelant qui lit quatre arguments d'affilée verrait la dernière erreur et
// non celle qui a réellement eu lieu, donc un message qui nomme le mauvais argument. Et
// une lecture qui repartirait à zéro après un échec rendrait des valeurs tirées du
// bourrage.
func TestWLReaderKeepsTheFirstFailure(t *testing.T) {
	r := wlReader{buf: make([]byte, 2)}
	r.uint()
	first := r.err
	if first == nil {
		t.Fatal("la première lecture a réussi, attendu un refus")
	}

	if got := r.uint(); got != 0 {
		t.Errorf("la seconde lecture rend %d, attendu 0", got)
	}
	if r.err != first {
		t.Errorf("l'erreur retenue a changé : %v, attendu %v", r.err, first)
	}
}

// TestWLWriterResetKeepsTheBuffer garde la réutilisation du tampon d'une image à l'autre.
//
// C'est ce qui évite une allocation par image sur le chemin de présentation. Le cas
// éprouve aussi qu'un message laissé ouvert par une erreur plus haut n'empêche pas l'image
// suivante d'écrire, ce qui serait une panique au lieu d'une image perdue.
//
// **La capacité se mesure juste avant la remise à zéro et pas avant**, et la première
// version de ce cas s'y est trompée : elle la relevait, ouvrait un message de plus, puis
// accusait la remise à zéro d'avoir réalloué — alors que c'était l'`append` de l'en-tête,
// qui faisait passer le tampon de seize à trente-deux octets. Un cas qui accuse le mauvais
// appel est pire qu'un cas absent.
func TestWLWriterResetKeepsTheBuffer(t *testing.T) {
	var w wlWriter
	w.begin(1, 0)
	w.uint(0)
	w.end()

	// Un message laissé ouvert, qui est le régime d'une erreur survenue au milieu
	// d'une écriture : le tour suivant doit pouvoir repartir.
	w.begin(1, 1)
	w.uint(0)

	capacity := cap(w.buf)
	w.reset()

	if len(w.buf) != 0 {
		t.Errorf("le tampon garde %d octets après remise à zéro, attendu 0", len(w.buf))
	}
	if cap(w.buf) < capacity {
		t.Errorf("la capacité est tombée de %d à %d : la remise à zéro rend le tampon au tas",
			capacity, cap(w.buf))
	}
	w.begin(1, 2)
	w.end()
}
