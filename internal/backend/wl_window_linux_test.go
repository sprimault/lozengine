// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le montage d'une fenêtre doit tenir : l'ordre que le protocole impose, le commit
// à vide qui déclenche la configuration, un acquittement par séquence et un `pong` par
// `ping`.

package backend

import (
	"strings"
	"testing"
)

// wlMount monte une fenêtre sur un compositeur de pacotille.
//
// La structure est bâtie à la main plutôt que par [wlOpen], qui joint la socket que
// l'environnement désigne : ce qui s'éprouve ici est le montage, pas la découverte du
// chemin, dont les cas vivent avec [wlDisplayPath].
func wlMount(t *testing.T, f *wlFake, o Options) (*wlWindow, error) {
	t.Helper()

	c := wlFakeConn(t, f)
	w := &wlWindow{c: c, width: o.Width, height: o.Height}
	return w, w.mount(o)
}

// wlSettle attend que le compositeur ait traité tout ce qui lui a été envoyé.
//
// **Un aller-retour est la seule barrière que le protocole offre**, et il en faut une :
// le dernier envoi du montage part après que la boucle d'attente est sortie, donc un cas
// qui lirait aussitôt ce que le compositeur a vu pourrait le lire avant l'acquittement.
// Le compositeur traitant ses requêtes dans l'ordre, la réponse au `sync` prouve que tout
// ce qui précède est passé.
func wlSettle(t *testing.T, c *wlConn) {
	t.Helper()

	if err := c.roundtrip(); err != nil {
		t.Fatalf("aller-retour de stabilisation : %v", err)
	}
}

// wlTestOptions est l'ouverture que les cas de ce fichier demandent.
func wlTestOptions() Options {
	return Options{Title: "essai", Width: 800, Height: 600}
}

// TestWLWindowMountsInTheOrderTheProtocolImposes fige la suite des requêtes d'ouverture.
//
// **C'est le cas qui porte le lot**, et il garde quatre choses qu'aucun autre n'attrape
// ensemble : que le rôle est donné avant toute autre requête à l'`xdg_surface`, que le
// premier commit part **sans tampon**, que l'acquittement vient après la configuration et
// non avant, et que le `ping` reçoit sa réponse.
//
// Le dernier point est celui qu'on oublierait : le compositeur de pacotille sonde le
// client avant de le configurer et refuse un acquittement laissé sans `pong`, donc un
// routage qui ignorerait l'événement ferait tomber ce cas avec `unresponsive` plutôt que
// de laisser une fenêtre fragile passer.
//
// La valeur de l'identifiant d'application n'est pas éprouvée ici — elle dépend du nom du
// binaire de test —, seulement sa place dans la suite.
func TestWLWindowMountsInTheOrderTheProtocolImposes(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}
	wlSettle(t, w.c)

	want := []string{
		"bind wl_compositor",
		"bind wl_shm",
		"bind xdg_wm_base",
		"create_surface",
		"get_xdg_surface",
		"get_toplevel",
		"set_title essai",
		"set_app_id " + wlAppID(),
		"commit",
		"pong 1",
		"ack_configure 2",
	}

	got := f.saw()
	if len(got) != len(want) {
		t.Fatalf("le compositeur a vu %d requêtes, attendu %d :\n%s", len(got), len(want),
			strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("requête %d : %q, attendu %q", i, got[i], want[i])
		}
	}
	if r := f.refusal(); r != "" {
		t.Errorf("le compositeur a refusé le montage : %s", r)
	}
}

// TestWLWindowKeepsItsOpeningSizeWhenTheCompositorProposesNothing garde la lecture de
// zéro.
//
// **Zéro n'est pas une taille, c'est une abstention**, et la description le dit : « if the
// width or height arguments are zero, it means the client should decide its own window
// dimension ». Le prendre pour une taille ouvrirait une fenêtre de zéro pixel, donc un
// tampon vide et une image que rien ne montre.
//
// Le cas éprouve aussi une valeur négative, que le protocole autorise sur le fil puisqu'il
// déclare ces arguments signés : lue en non signé, elle ferait une fenêtre de soixante-cinq
// mille pixels de côté.
func TestWLWindowKeepsItsOpeningSizeWhenTheCompositorProposesNothing(t *testing.T) {
	for _, proposed := range []int32{0, -1} {
		f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
		f.proposeW, f.proposeH = proposed, proposed

		w, err := wlMount(t, f, wlTestOptions())
		if err != nil {
			t.Fatalf("montage avec une proposition de %d : %v", proposed, err)
		}
		if width, height := w.Size(); width != 800 || height != 600 {
			t.Errorf("une proposition de %d donne %d×%d, attendu 800×600", proposed, width, height)
		}
	}
}

// TestWLWindowTakesTheSizeTheCompositorProposes garde le sens de la négociation.
//
// C'est le compositeur qui décide de la taille d'une fenêtre sous ce protocole, là que les
// deux autres familles donnent au client ce qu'il demande. Un montage qui garderait la
// taille d'ouverture peindrait à côté de ce que le compositeur affiche.
func TestWLWindowTakesTheSizeTheCompositorProposes(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	f.proposeW, f.proposeH = 1280, 720

	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}
	if width, height := w.Size(); width != 1280 || height != 720 {
		t.Errorf("la fenêtre fait %d×%d, attendu 1280×720", width, height)
	}
}

// TestWLWindowAcksOnlyTheLastConfigure garde ce que la description permet d'économiser.
//
// « If the client receives multiple configure events before it can respond to one, it only
// has to ack the last configure event. » Acquitter les deux serait permis, mais garder un
// numéro par séquence obligerait à les retenir tous — et le mode de panne du régime
// inverse est pire : acquitter le **premier** laisse le compositeur attendre l'accusé du
// dernier, donc une fenêtre qui ne prend jamais sa nouvelle taille.
func TestWLWindowAcksOnlyTheLastConfigure(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	f.sequences = 2

	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}
	wlSettle(t, w.c)

	acks := []string(nil)
	for _, seen := range f.saw() {
		if strings.HasPrefix(seen, "ack_configure ") {
			acks = append(acks, seen)
		}
	}
	// Le sondage prend le numéro 1, les deux séquences les numéros 2 et 3.
	if len(acks) != 1 || acks[0] != "ack_configure 3" {
		t.Errorf("les acquittements sont %v, attendu le seul numéro 3", acks)
	}
}

// TestWLWindowClosesOnTheToplevelCloseEvent garde la demande de fermeture.
//
// C'est la croix de la barre de titre, et elle n'est qu'une demande : « this is only a
// request that the user intends to close the window ». Le moteur l'honore comme X11
// honore `WM_DELETE_WINDOW`, en déclarant la fenêtre fermée sans rien détruire — c'est au
// jeu de finir son pas.
func TestWLWindowClosesOnTheToplevelCloseEvent(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	f.closing = true

	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}
	if !w.Closed() {
		t.Error("la fenêtre ne se déclare pas fermée après la demande du compositeur")
	}
}

// TestWLFakeRefusesWhatTheProtocolForbids éprouve que le compositeur de pacotille mord.
//
// **Sans ce cas, le montage pourrait passer parce que le juge dort**, et c'est le défaut
// que `tests.md` nomme : un cas qui ne peut pas rougir ne garde rien. Chaque séquence ici
// est fautive d'une seule façon, et la requête se remet directement au compositeur plutôt
// que par un client de travers — un second montage écrit pour échouer dériverait du
// premier.
func TestWLFakeRefusesWhatTheProtocolForbids(t *testing.T) {
	// Le rôle, puis le commit qui déclenche la configuration : le préambule que trois
	// des quatre séquences partagent.
	role := func(f *wlFake) { wlFeed(f, f.xdg, wlXDGSurfaceGetToplevel, 22) }
	commit := func(f *wlFake) { wlFeed(f, f.surface, wlSurfaceCommit) }

	cases := []struct {
		label string
		drive func(*wlFake)
		says  string
	}{
		{
			"un tampon attaché avant la première configuration",
			func(f *wlFake) {
				role(f)
				wlFeed(f, f.surface, wlSurfaceAttach, 99, 0, 0)
			},
			"attached before the first configure",
		},
		{
			"une requête émise avant que le rôle soit donné",
			func(f *wlFake) {
				wlFeed(f, f.xdg, wlXDGSurfaceAckConfigure, 1)
			},
			"before the surface was given a role",
		},
		{
			"un numéro de série jamais émis",
			func(f *wlFake) {
				role(f)
				commit(f)
				wlFeed(f, f.wmBase, wlXDGWMBasePong, 1)
				wlFeed(f, f.xdg, wlXDGSurfaceAckConfigure, 999)
			},
			"serial 999 was never sent",
		},
		{
			"un sondage laissé sans réponse",
			func(f *wlFake) {
				role(f)
				commit(f)
				wlFeed(f, f.xdg, wlXDGSurfaceAckConfigure, 2)
			},
			"ping 1 went unanswered",
		},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			f := wlFakeMounted()
			c.drive(f)

			refusal := f.refusal()
			if refusal == "" {
				t.Fatal("le compositeur a tout accepté, attendu un refus")
			}
			if !strings.Contains(refusal, c.says) {
				t.Errorf("le refus dit %q, attendu qu'il nomme %q", refusal, c.says)
			}
		})
	}
}

// TestWLAppIDNamesSomethingACompositorCanLookUp garde ce qui remplace une icône.
//
// Le protocole ne transporte pas d'image de fenêtre : ce qu'un compositeur cherche est une
// entrée `.desktop` portant cet identifiant. Une chaîne vide, ou un chemin au lieu d'un
// nom, ne correspondrait à rien — donc une icône générique et des fenêtres regroupées de
// travers, sans qu'aucune erreur ne le signale.
func TestWLAppIDNamesSomethingACompositorCanLookUp(t *testing.T) {
	id := wlAppID()
	if id == "" {
		t.Fatal("l'identifiant d'application est vide")
	}
	if strings.ContainsAny(id, `/\`) {
		t.Errorf("l'identifiant vaut %q, attendu un nom et non un chemin", id)
	}
}
