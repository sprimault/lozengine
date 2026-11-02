// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les deux gardes de la vue aliasée, éprouvées sans page : elles ne demandent qu'un
// tableau d'octets, que l'exécuteur de la suite porte comme un navigateur.

package backend

import (
	"syscall/js"
	"testing"

	"github.com/sprimault/lozengine/render"
)

// detached imite un tampon que la croissance du tas a détaché : la page en garde l'objet,
// et sa longueur est tombée à zéro.
func detached() js.Value {
	return js.ValueOf(map[string]any{"byteLength": 0})
}

// attached rend une vue d'octets réelle, de la longueur demandée.
func attached(length int) js.Value {
	return js.Global().Get("Uint8ClampedArray").New(length)
}

// exposedMemory imite le global par lequel une page expose la mémoire du module.
//
// N'importe quel objet convient : le régime se décide sur la présence du global, et seule
// une reconstruction de la vue en lit le tampon — ce qu'aucun cas d'ici ne fait, faute de
// page pour porter un `ImageData`.
func exposedMemory() js.Value {
	return js.ValueOf(map[string]any{})
}

// TestStaleSeesBothWaysAViewGoesBad garde les deux gardes du régime aliasé, et c'est le
// cas qui porte tout le lot : leur absence ne se voit pas à l'écran autrement que par une
// image qui ne change plus.
//
// **Aucune des deux ne suffit, et c'est le fait que l'instruction du jalon avait
// manqué** : la croissance du tas détache le tampon sans changer son adresse, donc seule
// la longueur en octets le dit ; une réallocation de la surface change l'adresse sans
// détacher, donc la pose réussit et la page affiche l'image précédente, en silence, après
// chaque redimensionnement de fenêtre.
//
// Chacun des deux derniers cas rougit si l'on retire la garde qu'il vise, et aucun ne
// rougit si l'on retire l'autre.
func TestStaleSeesBothWaysAViewGoesBad(t *testing.T) {
	const addr, length = 4096, 32
	buffer := render.NewBuffer(4, 2)

	for _, c := range []struct {
		name string
		p    presentation
		addr uintptr
		want bool
	}{
		{
			"rien n'est encore monté",
			presentation{},
			addr, true,
		},
		{
			"le canvas a changé de taille",
			presentation{
				data: js.ValueOf(map[string]any{}),
				view: attached(length), addr: addr, length: length,
				width: 8, height: 2,
			},
			addr, true,
		},
		{
			"régime copié, rien n'a changé",
			presentation{
				data: js.ValueOf(map[string]any{}),
				view: attached(length), addr: 0, length: 0,
				width: 4, height: 2,
			},
			addr, false,
		},
		{
			"régime copié, le tampon a déménagé",
			presentation{
				data: js.ValueOf(map[string]any{}),
				view: attached(length), addr: 0, length: 0,
				width: 4, height: 2,
			},
			addr * 2, false,
		},
		{
			"régime aliasé, rien n'a changé",
			presentation{
				data: js.ValueOf(map[string]any{}), memory: exposedMemory(),
				view: attached(length), addr: addr, length: length,
				width: 4, height: 2,
			},
			addr, false,
		},
		{
			"régime aliasé, la surface a été réallouée",
			presentation{
				data: js.ValueOf(map[string]any{}), memory: exposedMemory(),
				view: attached(length), addr: addr, length: length,
				width: 4, height: 2,
			},
			addr * 2, true,
		},
		{
			"régime aliasé, le tas a grandi et détaché le tampon",
			presentation{
				data: js.ValueOf(map[string]any{}), memory: exposedMemory(),
				view: detached(), addr: addr, length: length,
				width: 4, height: 2,
			},
			addr, true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := c.p
			if got := p.stale(c.addr, length, buffer); got != c.want {
				t.Errorf("stale rend %v, attendu %v", got, c.want)
			}
		})
	}
}

// TestAliasedFollowsWhatThePageExposes vérifie que le régime se décide sur la présence du
// global, et sur rien d'autre.
//
// C'est ce qui permet à une page qui n'expose pas sa mémoire de tourner quand même, en
// copiant — le jeu marche, et seul le relevé du temps par image le distingue.
func TestAliasedFollowsWhatThePageExposes(t *testing.T) {
	var copied presentation
	if copied.aliased() {
		t.Error("le régime aliasé est pris sans que la page ait exposé sa mémoire")
	}

	aliased := presentation{memory: exposedMemory()}
	if !aliased.aliased() {
		t.Error("le régime copié est pris alors que la page expose sa mémoire")
	}
}

// TestReleaseKeepsTheRegimeAndDropsTheView garde la distinction que `release` doit tenir :
// la vue décrit un tampon, qui change ; la mémoire décrit la page, qui ne change pas.
//
// Oublier la mémoire ferait retomber la fenêtre en régime copié au premier
// redimensionnement, et le coût de présentation doublerait sans qu'aucune erreur ne le
// dise.
func TestReleaseKeepsTheRegimeAndDropsTheView(t *testing.T) {
	p := presentation{
		data: js.ValueOf(map[string]any{}), memory: exposedMemory(),
		view: attached(32), addr: 4096, length: 32, width: 4, height: 2,
	}

	p.release()

	if !p.aliased() {
		t.Error("le régime est oublié avec la vue")
	}
	if !p.data.IsUndefined() || !p.view.IsUndefined() {
		t.Error("la vue ou son ImageData survit à la libération")
	}
	if p.addr != 0 || p.length != 0 || p.width != 0 || p.height != 0 {
		t.Errorf("les gardes survivent à la libération : %d, %d, %d×%d",
			p.addr, p.length, p.width, p.height)
	}
}
