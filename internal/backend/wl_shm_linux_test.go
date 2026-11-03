// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la mémoire partagée doit tenir avant qu'un compositeur la lise : un fichier sans
// nom, de la taille demandée, et une rotation des tampons qui ne réécrit jamais celui que
// le compositeur détient.

package backend

import (
	"os"
	"testing"
)

// TestWLMemoryFileHasNoNameAndTheRightSize garde les deux propriétés du fichier partagé.
//
// **Le retrait du répertoire n'est pas une politesse** : le fichier porte une image par
// tour et vit tant que la fenêtre, donc le laisser visible le donnerait à lire à n'importe
// quel processus du compte, et un plantage en laisserait un par exécution. Le descripteur
// suffit à tout ce qui suit — le compositeur en reçoit un duplicata.
//
// La taille est éprouvée parce que `wl_shm.create_pool` fait projeter au compositeur
// exactement ce qu'on annonce : un fichier plus court et il lit au-delà de sa fin, ce qui
// lui vaut un signal et non une erreur de protocole.
func TestWLMemoryFileHasNoNameAndTheRightSize(t *testing.T) {
	const size = 4096

	file, err := wlMemoryFile(size)
	if err != nil {
		t.Fatalf("création du fichier partagé : %v", err)
	}
	t.Cleanup(func() { file.Close() })

	if _, err := os.Stat(file.Name()); err == nil {
		t.Errorf("le fichier est encore visible sous %q", file.Name())
	}

	info, err := file.Stat()
	if err != nil {
		t.Fatalf("mesure du fichier par son descripteur : %v", err)
	}
	if info.Size() != size {
		t.Errorf("le fichier fait %d octets, attendu %d", info.Size(), size)
	}
}

// TestWLPoolTurnsItsBuffers garde la rotation et son refus.
//
// Trois régimes, et le troisième est celui qui compte : tout détenu rend -1, ce que la
// présentation lit comme un tour à sauter. Rendre un index quelconque ferait écrire dans un
// tampon que le compositeur lit, donc du déchirement sans erreur.
//
// Le cas n'ouvre aucune connexion : la rotation est une propriété de la structure, et
// l'éprouver par un compositeur mêlerait ce qu'elle décide à ce que le protocole transporte.
func TestWLPoolTurnsItsBuffers(t *testing.T) {
	p := &wlPool{last: wlBuffers - 1}
	for i := range p.buffers {
		p.buffers[i].id = uint32(100 + i)
	}

	first := p.free()
	if first != 0 {
		t.Fatalf("le premier tampon choisi est %d, attendu 0", first)
	}

	p.buffers[first].held = true
	p.last = first
	second := p.free()
	if second == first {
		t.Fatalf("le second tampon choisi est encore %d", second)
	}

	p.buffers[second].held = true
	p.last = second
	if got := p.free(); got != -1 {
		t.Errorf("tous les tampons sont détenus et le choix rend %d, attendu -1", got)
	}

	if !p.returned(p.buffers[first].id) {
		t.Errorf("l'identifiant %d n'est pas reconnu comme un tampon du pool", p.buffers[first].id)
	}
	if got := p.free(); got != first {
		t.Errorf("après le renvoi, le choix rend %d, attendu %d", got, first)
	}
	if p.returned(999) {
		t.Error("un identifiant étranger est pris pour un tampon du pool")
	}
}
