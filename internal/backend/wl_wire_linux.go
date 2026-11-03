// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le codage des messages du protocole Wayland : un en-tête de deux mots, des arguments
// alignés sur trente-deux bits, et les deux formes d'un identifiant neuf.
//
// La spécification est la description du protocole et son chapitre sur le format du fil,
// jamais une implémentation. Ce que ce fichier en retient tient en quatre phrases : tout
// est fait de mots de trente-deux bits, dans l'ordre d'octets de la machine ; l'en-tête
// porte l'objet émetteur puis, dans un seul mot, la taille en octets dans les seize bits
// de poids fort et l'opcode dans les seize de poids faible ; chaque argument est aligné
// sur trente-deux bits ; et la valeur des octets de bourrage est indéfinie.
//
// **Ce dernier point est une contrainte pour les tests avant d'être une liberté pour le
// code** : un cas qui comparerait un message octet à octet sans masquer son bourrage
// serait intermittent. Les cas de ce fichier comparent donc ce qu'ils ont écrit, et le
// code écrit des zéros pour que la comparaison soit possible — ce qui est un choix
// d'implémentation et non une garantie du protocole.

package backend

import (
	"encoding/binary"
	"fmt"
)

// wlHeaderSize est la taille de l'en-tête d'un message, en octets.
//
// C'est aussi la taille minimale d'un message, puisque la taille annoncée part de
// l'en-tête : la spécification le dit en donnant ce minimum de huit.
const wlHeaderSize = 8

// wlOrder est l'ordre d'octets du protocole.
//
// **C'est le seul endroit du dépôt où `NativeEndian` est le bon choix**, et c'est la
// spécification qui l'impose : « values are represented in the host's byte-order ». Le
// protocole est local par construction — une socket de domaine Unix —, donc les deux
// bouts sont la même machine. Partout ailleurs le dépôt emploie un ordre explicite,
// parce qu'un format qui traverse une frontière ne peut pas dépendre de la machine ; ici
// la dépendance est le format.
var wlOrder = binary.NativeEndian

// wlWriter compose des messages à la suite dans un tampon qu'il réutilise.
//
// **Il n'alloue pas par image dans le régime courant** : le tampon grossit jusqu'à la
// taille du plus gros lot de requêtes d'une image, puis se remet à zéro par `[:0]`. Une
// image qui pose une surface, l'endommage et la commet écrit moins de cent octets, donc
// la croissance s'arrête au premier tour.
//
// L'écriture d'un message encadre ses arguments entre [wlWriter.begin] et
// [wlWriter.end], parce que la taille vit dans l'en-tête et ne se connaît qu'à la fin.
// Oublier `end` laisse une taille nulle, que le compositeur rejette en fermant la
// connexion : c'est brutal, mais c'est immédiat, et un test le voit.
type wlWriter struct {
	buf []byte

	// at est l'offset de l'en-tête du message en cours d'écriture, et opcode son
	// opcode, que `end` recompose avec la taille dans un seul mot. writing dit qu'un
	// message est ouvert, ce qu'un offset ne peut pas dire : zéro est un offset
	// valide, et la valeur zéro de cette structure doit servir telle quelle.
	at      int
	opcode  uint16
	writing bool
}

// begin ouvre un message émis par l'objet id, pour l'opcode donné.
//
// La panique n'est pas de la défensive : elle garde un invariant de programmation, et le
// mode de panne qu'elle remplace est un message dont la taille désigne les octets d'un
// autre — donc un flux que le compositeur lit de travers jusqu'à la fin.
func (w *wlWriter) begin(id uint32, opcode uint16) {
	if w.writing {
		panic("wayland: a message is already being written")
	}
	w.at, w.opcode, w.writing = len(w.buf), opcode, true
	w.buf = append(w.buf, make([]byte, wlHeaderSize)...)
	wlOrder.PutUint32(w.buf[w.at:], id)
}

// end referme le message courant en y inscrivant sa taille et son opcode.
//
// Les deux vivent dans le même mot — taille dans les seize bits de poids fort, opcode
// dans les seize de poids faible —, donc il s'écrit d'un coup et à la fin.
func (w *wlWriter) end() {
	if !w.writing {
		panic("wayland: no message is being written")
	}
	size := uint32(len(w.buf)-w.at) << 16
	wlOrder.PutUint32(w.buf[w.at+4:], size|uint32(w.opcode))
	w.writing = false
}

// uint écrit un entier non signé, ou un identifiant d'objet, ou un identifiant neuf dont
// la description déclare l'interface.
//
// **Les trois sont le même mot, et c'est voulu ici** : la description distingue `uint`,
// `object` et `new_id` pour dire ce qu'ils désignent, pas comment ils s'écrivent. Un
// `object` nul s'écrit zéro, ce que la spécification dit explicitement et que le code
// appelant exprime en passant zéro.
func (w *wlWriter) uint(v uint32) {
	w.buf = wlOrder.AppendUint32(w.buf, v)
}

// newID écrit un identifiant neuf dont la description ne déclare pas l'interface.
//
// **C'est le piège du codage, et il ne se voit pas dans la description** : « a new_id is
// preceded by a string specifying the interface name, and a uint specifying the
// version ». Un `new_id` sans attribut `interface` met donc **trois** valeurs sur le fil
// là qu'un `new_id` qui en porte un n'en met qu'une. Le seul cas du protocole de base est
// `wl_registry.bind`, qui lie n'importe quelle interface et ne peut donc pas la déclarer.
//
// Écrire ici le seul identifiant ferait un message plus court que ce que le compositeur
// attend ; il lirait le reste du flux comme la suite du message, et la connexion
// tomberait sur une erreur qui n'aurait aucun rapport.
func (w *wlWriter) newID(iface string, version, id uint32) {
	w.string(iface)
	w.uint(version)
	w.uint(id)
}

// string écrit une chaîne : sa longueur, son contenu, son terminateur, son bourrage.
//
// **La longueur compte le terminateur nul**, ce que la spécification dit en toutes
// lettres — « an unsigned 32-bit length (including null terminator) ». L'oublier décale
// tout ce qui suit d'un octet, et le compositeur lit alors des arguments plausibles mais
// faux : c'est le mode de panne le plus coûteux à diagnostiquer de ce fichier, puisque
// rien ne le signale avant une erreur de protocole sans rapport.
func (w *wlWriter) string(s string) {
	w.uint(uint32(len(s)) + 1)
	w.buf = append(w.buf, s...)
	w.buf = append(w.buf, 0)
	w.pad()
}

// pad complète le dernier argument jusqu'à la frontière de trente-deux bits.
//
// Elle écrit des zéros là que la spécification laisse ces octets indéfinis, pour que les
// cas de test puissent comparer un message entier. C'est le choix du dépôt et non une
// garantie du protocole, et la tête du fichier le dit.
func (w *wlWriter) pad() {
	for len(w.buf)%4 != 0 {
		w.buf = append(w.buf, 0)
	}
}

// reset vide le tampon en gardant sa capacité, et oublie un message inachevé.
func (w *wlWriter) reset() {
	w.buf = w.buf[:0]
	w.writing = false
}

// wlReader lit les arguments d'un message déjà délimité.
//
// Il ne connaît pas les types attendus : c'est l'appelant qui les lit dans l'ordre que la
// description donne, comme il le fait pour une requête qu'il écrit. Ce que le lecteur
// garde est la seule chose qu'on ne peut pas confier à l'appelant — qu'une lecture au-delà
// du message ne rende pas les octets du message suivant.
type wlReader struct {
	buf []byte
	at  int

	// err retient la première lecture qui a manqué de place. Les suivantes rendent
	// une valeur nulle sans la signaler deux fois, et l'appelant contrôle une fois à
	// la fin : un événement tronqué est une erreur de protocole, pas une valeur à
	// rattraper argument par argument.
	err error
}

// uint lit un entier non signé, un identifiant d'objet ou un nom de global.
func (r *wlReader) uint() uint32 {
	if r.err != nil {
		return 0
	}
	if r.at+4 > len(r.buf) {
		r.fail("uint")
		return 0
	}
	v := wlOrder.Uint32(r.buf[r.at:])
	r.at += 4
	return v
}

// string lit une chaîne et laisse le curseur sur la frontière suivante.
//
// Le terminateur nul est retiré de ce qui est rendu : il fait partie du codage, pas de la
// valeur. Une longueur nulle est refusée plutôt que rendue vide, la spécification
// comptant toujours le terminateur — zéro désigne donc un message mal formé et non une
// chaîne vide, qui vaut un.
func (r *wlReader) string() string {
	n := r.uint()
	if r.err != nil {
		return ""
	}
	if n == 0 || r.at+int(n) > len(r.buf) {
		r.fail("string")
		return ""
	}
	s := string(r.buf[r.at : r.at+int(n)-1])
	r.at += int(n)
	for r.at%4 != 0 {
		r.at++
	}
	return s
}

// int lit un entier signé, qui occupe le même mot qu'un entier non signé.
//
// La conversion est la seule chose que cette méthode apporte, et elle vaut d'être nommée :
// une largeur de `xdg_toplevel.configure` lue en non signé rendrait 4 294 967 295 au lieu
// de -1 sans que rien ne le signale, et c'est une taille de fenêtre qu'un jeu reçoit.
func (r *wlReader) int() int32 {
	return int32(r.uint())
}

// array lit un tableau d'octets et laisse le curseur sur la frontière suivante.
//
// **Une longueur nulle est valide ici, contrairement à une chaîne**, et c'est le cas
// courant : `xdg_toplevel.configure` porte la liste des états de la fenêtre, qui est vide
// pour une fenêtre ordinaire. Il n'y a pas de terminateur à compter, donc zéro désigne
// bien un tableau vide.
//
// La tranche rendue pointe dans le message et ne survit donc pas au découpage suivant.
// C'est ce qui évite une allocation par événement sur un chemin que la boucle traverse à
// chaque pas ; l'appelant qui veut la garder la recopie.
func (r *wlReader) array() []byte {
	n := r.uint()
	if r.err != nil {
		return nil
	}
	if r.at+int(n) > len(r.buf) {
		r.fail("array")
		return nil
	}
	a := r.buf[r.at : r.at+int(n)]
	r.at += int(n)
	for r.at%4 != 0 {
		r.at++
	}
	return a
}

// fail retient la première lecture hors message.
func (r *wlReader) fail(kind string) {
	r.err = fmt.Errorf("wayland: message of %d bytes is too short to hold its %s argument at offset %d",
		len(r.buf), kind, r.at)
}
