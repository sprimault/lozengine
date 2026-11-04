// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le montage d'une fenêtre doit tenir : l'ordre que le protocole impose, le commit
// à vide qui déclenche la configuration, un acquittement par séquence et un `pong` par
// `ping`.

package backend

import (
	"strings"
	"testing"

	"github.com/sprimault/lozengine/render"
)

// wlTestFrame est une image de deux pixels sur deux, dont chaque composante est
// distincte.
//
// **Les valeurs sont distinctes exprès** : une image grise ou unie passerait tous les cas
// de disposition, qui ne font que déplacer des composantes. Et aucune ne vaut zéro, qui
// est ce que le bourrage porte.
func wlTestFrame() *render.Buffer {
	return &render.Buffer{
		Width: 2, Height: 2,
		Pixels: []render.Color{
			{R: 1, G: 2, B: 3, A: 255},
			{R: 4, G: 5, B: 6, A: 255},
			{R: 7, G: 8, B: 9, A: 255},
			{R: 10, G: 11, B: 12, A: 255},
		},
	}
}

// wlShiftedFrame rend la même image décalée, pour qu'une seconde présentation se distingue
// de la première dans la mémoire partagée.
func wlShiftedFrame(by uint8) *render.Buffer {
	frame := wlTestFrame()
	for i := range frame.Pixels {
		frame.Pixels[i].R += by
		frame.Pixels[i].G += by
		frame.Pixels[i].B += by
	}
	return frame
}

// wlMount monte une fenêtre sur un compositeur de pacotille.
//
// La structure est bâtie à la main plutôt que par [wlOpen], qui joint la socket que
// l'environnement désigne : ce qui s'éprouve ici est le montage, pas la découverte du
// chemin, dont les cas vivent avec [wlDisplayPath].
func wlMount(tb testing.TB, f *wlFake, o Options) (*wlWindow, error) {
	tb.Helper()

	c := wlFakeConn(tb, f)
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

// La fenêtre Wayland remplit le contrat des sept méthodes, et c'est ici que la
// vérification vit.
//
// **Rien d'autre ne la fait**, et c'est la raison de cette ligne : [wlOpen] rend un type
// concret, donc aucune conversion n'éprouve l'interface avant le lot qui met la famille
// dans la cascade d'ouverture. Une signature qui dériverait — un [input.Input] rendu par
// pointeur, une taille en entiers — ne se verrait qu'à ce moment-là.
var _ Window = (*wlWindow)(nil)

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
		"bind wl_seat",
		"bind xdg_wm_base",
		// Les deux objets d'entrée viennent avant la surface : ils n'en dépendent
		// pas, et le masque de capacités qui les autorise est déjà arrivé.
		"get_keyboard",
		"get_pointer",
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

// TestWLWindowPresentsIntoTheMemoryTheCompositorReads garde la présentation jusqu'aux
// octets.
//
// **Le compositeur de pacotille projette le descripteur qu'il reçoit**, donc ce cas relit
// la mémoire exactement comme un vrai compositeur la lirait. C'est ce qui le sépare d'un
// cas qui vérifierait des requêtes bien formées : une disposition inversée, un pas de ligne
// faux ou un décalage de tampon manqué composent tous des requêtes impeccables.
//
// Les deux dispositions sont éprouvées, et ce sont deux chemins de code : l'ordre du moteur
// passe par une copie en bloc, où l'octet d'alpha tombe dans l'octet que le format déclare
// ignoré ; le repli parcourt les pixels et écrit **zéro** dans cet octet, ce qui est la
// raison pour laquelle les formats à alpha sont refusés ailleurs.
func TestWLWindowPresentsIntoTheMemoryTheCompositorReads(t *testing.T) {
	cases := []struct {
		label   string
		formats []uint32
		want    func(render.Color) [4]byte
	}{
		{
			"l'ordre du moteur, recopié en bloc",
			[]uint32{wlShmXRGB8888, formatXBGR8888},
			func(c render.Color) [4]byte { return [4]byte{c.R, c.G, c.B, c.A} },
		},
		{
			"le repli, qui échange le rouge et le bleu",
			[]uint32{wlShmXRGB8888},
			func(c render.Color) [4]byte { return [4]byte{c.B, c.G, c.R, 0} },
		},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			f := wlNewFake(wlDefaultGlobals(), c.formats)
			w, err := wlMount(t, f, wlTestOptions())
			if err != nil {
				t.Fatalf("montage de la fenêtre : %v", err)
			}

			// Deux images distinctes, et c'est ce qui éprouve le décalage : la
			// seconde présentation prend l'autre tampon, donc écrire au mauvais
			// endroit se verrait. Avec une seule image, un décalage toujours nul
			// passerait, le premier tampon étant à zéro.
			frames := []*render.Buffer{wlTestFrame(), wlShiftedFrame(100)}
			for _, frame := range frames {
				wlSettle(t, w.c)
				w.Present(frame)
			}
			wlSettle(t, w.c)
			if r := f.refusal(); r != "" {
				t.Fatalf("le compositeur a refusé la présentation : %s", r)
			}

			for i, frame := range frames {
				want := make([]byte, 0, 4*len(frame.Pixels))
				for _, pixel := range frame.Pixels {
					b := c.want(pixel)
					want = append(want, b[:]...)
				}

				at := i * len(want)
				got := f.shared(at, len(want))
				if got == nil {
					t.Fatalf("la mémoire partagée ne porte pas %d octets au décalage %d",
						len(want), at)
				}
				for j := range want {
					if got[j] != want[j] {
						t.Fatalf("le tampon %d vaut %v, attendu %v", i, got, want)
					}
				}
			}
		})
	}
}

// TestWLWindowDescribesTheBufferItShares fige la géométrie que `create_buffer` annonce.
//
// Le pas de ligne et le décalage sont les deux nombres qu'un compositeur croit sans les
// vérifier : un pas faux décale chaque ligne d'un peu et rend une image en escalier, un
// décalage faux fait lire l'autre tampon. Les deux composent des requêtes valides.
//
// Le second tampon est éprouvé autant que le premier, parce que c'est son décalage qui dit
// que les deux ne se recouvrent pas.
func TestWLWindowDescribesTheBufferItShares(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888, formatXBGR8888})
	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}

	w.Present(wlTestFrame())
	wlSettle(t, w.c)

	frame := 2 * 4 * 2
	for i, b := range w.pool.buffers {
		got, ok := f.bufferOf(b.id)
		if !ok {
			t.Fatalf("le compositeur ne connaît pas le tampon %d", i)
		}
		want := wlFakeBuffer{at: i * frame, width: 2, height: 2, stride: 8, format: formatXBGR8888}
		if got != want {
			t.Errorf("le tampon %d est décrit %+v, attendu %+v", i, got, want)
		}
	}
}

// TestWLWindowSkipsAPresentationWhenTheCompositorHoldsEveryBuffer garde le saut.
//
// **C'est le régime que le contrat de [Window.Present] autorise en toutes lettres** — « une
// présentation qui ne peut pas avoir lieu … ne fait rien, ce qui est le bon comportement et
// non un échec ». Ce que cela évite est d'écrire dans un tampon que le compositeur lit,
// c'est-à-dire du déchirement qu'aucune erreur ne signale.
//
// Le compositeur retient ici ses tampons, ce que la description permet : elle ne garantit
// le renvoi immédiat que « typically », et seulement quand il garde une copie de la
// surface.
func TestWLWindowSkipsAPresentationWhenTheCompositorHoldsEveryBuffer(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888, formatXBGR8888})
	f.releaseOnCommit = false

	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}

	for i := 0; i < wlBuffers+2; i++ {
		w.Present(wlTestFrame())
		wlSettle(t, w.c)
	}
	if r := f.refusal(); r != "" {
		t.Fatalf("le compositeur a refusé : %s", r)
	}

	attaches := 0
	for _, seen := range f.saw() {
		if strings.HasPrefix(seen, "attach ") {
			attaches++
		}
	}
	if attaches != wlBuffers {
		t.Errorf("la fenêtre a attaché %d tampons, attendu %d : les tours suivants sautent",
			attaches, wlBuffers)
	}
	if w.Err() != nil {
		t.Errorf("un tour sauté a été pris pour une panne : %v", w.Err())
	}
}

// TestWLWindowAlternatesItsBuffers garde que le second tampon sert.
//
// **Sans ce cas, un backend qui réemploie toujours le même passerait**, puisqu'un
// compositeur qui rend vite laisse le premier libre à chaque tour. Le défaut ne se verrait
// alors que devant un compositeur qui retient — donc en production et sur une autre
// machine.
func TestWLWindowAlternatesItsBuffers(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888, formatXBGR8888})
	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}

	for i := 0; i < 3; i++ {
		wlSettle(t, w.c)
		w.Present(wlTestFrame())
	}
	wlSettle(t, w.c)

	attached := []string(nil)
	for _, seen := range f.saw() {
		if strings.HasPrefix(seen, "attach ") {
			attached = append(attached, seen)
		}
	}
	if len(attached) != 3 {
		t.Fatalf("la fenêtre a attaché %d tampons, attendu 3 : %v", len(attached), attached)
	}
	if attached[0] == attached[1] {
		t.Errorf("les deux premières présentations emploient le même tampon : %v", attached)
	}
	if attached[0] != attached[2] {
		t.Errorf("la troisième présentation ne revient pas au premier tampon : %v", attached)
	}
}

// TestWLWindowBuildsANewPoolWhenTheFrameChangesSize garde le remplacement du pool.
//
// Un `wl_buffer` porte ses dimensions, donc ceux d'une autre taille ne servent plus.
// [wlPool] dit pourquoi l'agrandissement en place a été écarté ; ce cas garde la
// conséquence observable, qui est qu'un pool neuf arrive et que les anciens tampons sont
// détruits.
func TestWLWindowBuildsANewPoolWhenTheFrameChangesSize(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888, formatXBGR8888})
	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}

	w.Present(wlTestFrame())
	wlSettle(t, w.c)

	wide := &render.Buffer{Width: 4, Height: 2, Pixels: make([]render.Color, 8)}
	w.Present(wide)
	wlSettle(t, w.c)

	if r := f.refusal(); r != "" {
		t.Fatalf("le compositeur a refusé : %s", r)
	}

	pools, destroyed := 0, 0
	for _, seen := range f.saw() {
		switch {
		case strings.HasPrefix(seen, "create_pool "):
			pools++
		case seen == "destroy_buffer":
			destroyed++
		}
	}
	if pools != 2 {
		t.Errorf("le compositeur a vu %d pools, attendu 2", pools)
	}
	if destroyed != wlBuffers {
		t.Errorf("%d tampons détruits, attendu %d", destroyed, wlBuffers)
	}
	if got, _ := w.pool.buffers[0], 0; got.at != 0 {
		t.Errorf("le premier tampon du pool neuf est au décalage %d, attendu 0", got.at)
	}
}

// TestWLWindowPacesNothing garde la réponse que le jalon a renversée.
//
// Le brief l'annonçait vraie pour les deux nouvelles familles, et c'est faux : le jeton de
// `wl_surface.frame` s'arme au commit, donc dans la présentation — que la boucle saute
// précisément devant une fenêtre qui se dit cadencée. Le témoin de `loop_test.go` garde ce
// que la boucle en fait ; celui-ci garde la réponse.
func TestWLWindowPacesNothing(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888, formatXBGR8888})
	w, err := wlMount(t, f, wlTestOptions())
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}
	if w.Paced() {
		t.Error("la fenêtre se dit cadencée, et la boucle sauterait alors la présentation qui arme le jeton")
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
