// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'ouverture doit tenir : prendre la version que les deux acceptent, reconnaître
// le format de repli malgré sa valeur historique, et refuser en nommant ce qui manque.

package backend

import (
	"os"
	"strings"
	"testing"
)

// TestWLFourCCOfTranslatesTheTwoHistoricFormats garde la traduction des deux valeurs qui
// ne sont pas des codes.
//
// **C'est le piège du lot, et il porte sur le format dont on a le plus besoin.** La
// description donne `argb8888` à zéro et `xrgb8888` à un, là que tous les autres portent
// leur code à quatre caractères. Or `xrgb8888` est le repli garanti : sans traduction, un
// compositeur qui l'annonce serait pris pour un compositeur qui n'offre rien
// d'exploitable, et l'ouverture échouerait en annonçant que le format `?` n'est pas
// reconnu.
//
// Le cas rougit si la traduction disparaît, et aussi si elle s'applique à un format qui
// porte déjà son code — `0x34324258` doit traverser tel quel.
func TestWLFourCCOfTranslatesTheTwoHistoricFormats(t *testing.T) {
	cases := []struct {
		label  string
		format uint32
		want   uint32
	}{
		{"argb8888, valeur historique zéro", wlShmARGB8888, fourcc('A', 'R', '2', '4')},
		{"xrgb8888, valeur historique un", wlShmXRGB8888, formatXRGB8888},
		{"xbgr8888, qui porte déjà son code", formatXBGR8888, formatXBGR8888},
		{"un format inconnu, tel quel", fourcc('N', 'V', '1', '2'), fourcc('N', 'V', '1', '2')},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			if got := wlFourCCOf(c.format); got != c.want {
				t.Errorf("la valeur %#x donne %s (%#x), attendu %s (%#x)",
					c.format, fourccName(got), got, fourccName(c.want), c.want)
			}
		})
	}
}

// TestWLPixelOrderPrefersTheEngineLayout garde le choix du format et son ordre de
// préférence.
//
// La préférence n'est pas un détail de performance : `XB24` place les composantes dans
// l'ordre où [render.Color] les porte, donc le tampon part sans être parcouru. Le repli
// `XR24` coûte l'échange qu'X11 paie déjà.
//
// **Le quatrième cas est celui qui compte le plus** : un compositeur qui n'annonce que les
// deux formats garantis, et dont le seul exploitable arrive sous sa valeur historique.
func TestWLPixelOrderPrefersTheEngineLayout(t *testing.T) {
	cases := []struct {
		label   string
		formats []uint32
		kind    pixelKind
	}{
		{
			"les deux, donc l'ordre du moteur",
			[]uint32{wlShmARGB8888, wlShmXRGB8888, formatXBGR8888},
			pixelDirect,
		},
		{
			"le repli seul, sous sa valeur historique",
			[]uint32{wlShmARGB8888, wlShmXRGB8888},
			pixelSwapped,
		},
		{
			"l'ordre du moteur seul",
			[]uint32{formatXBGR8888},
			pixelDirect,
		},
		{
			"le repli annoncé par son code et non par sa valeur",
			[]uint32{formatXRGB8888},
			pixelSwapped,
		},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			o, err := wlPixelOrder(c.formats)
			if err != nil {
				t.Fatalf("choix du format : %v", err)
			}
			if o.kind != c.kind {
				t.Errorf("la disposition retenue est classée %d, attendu %d", o.kind, c.kind)
			}
		})
	}
}

// TestWLPixelOrderRefusesWhatItCannotWrite garde les deux refus.
//
// Un compositeur qui n'offre que de l'alpha est refusé pour la raison que
// [pixelOrderOfFourCC] porte : les dispositions rapides écriraient zéro dans l'octet
// d'alpha et la surface serait transparente. Et une liste vide est refusée parce qu'elle
// signale autre chose — le protocole impose deux formats, donc n'en recevoir aucun veut
// dire que l'aller-retour n'a pas eu lieu.
//
// Les deux messages nomment ce qu'on a reçu, en clair : un code en hexadécimal ne se
// reconnaît pas, et c'est la seule information qu'un intégrateur aura.
func TestWLPixelOrderRefusesWhatItCannotWrite(t *testing.T) {
	t.Run("rien du tout", func(t *testing.T) {
		if _, err := wlPixelOrder(nil); err == nil {
			t.Fatal("la liste vide est acceptée, attendu un refus")
		}
	})

	t.Run("de l'alpha seulement", func(t *testing.T) {
		_, err := wlPixelOrder([]uint32{wlShmARGB8888, fourcc('A', 'B', '2', '4')})
		if err == nil {
			t.Fatal("les formats à alpha sont acceptés, attendu un refus")
		}
		if !strings.Contains(err.Error(), "AR24") {
			t.Errorf("le refus dit %q, attendu qu'il nomme les formats offerts en clair", err)
		}
	})
}

// wlFakeGlobal est un global que le compositeur de pacotille annonce.
type wlFakeGlobal struct {
	name    uint32
	iface   string
	version uint32
}

// wlServe répond aux requêtes d'ouverture comme un compositeur le ferait.
//
// Il ne juge rien — c'est le lot de la surface qui apportera un compositeur qui refuse —,
// mais il tient les deux choses dont l'ouverture dépend : un aller-retour reçoit sa
// réponse, et les formats n'arrivent qu'après que `wl_shm` a été lié. Sans cette seconde
// règle, le cas passerait avec un seul aller-retour et ne dirait rien du dispositif.
func wlServe(t *testing.T, server *os.File, globals []wlFakeGlobal, formats []uint32) {
	t.Helper()

	go func() {
		in := make([]byte, 4096)
		have := 0
		registry, shm := uint32(0), uint32(0)

		for {
			n, err := server.Read(in[have:])
			if err != nil || n == 0 {
				return
			}
			have += n

			for have >= wlHeaderSize {
				second := wlOrder.Uint32(in[4:])
				size := int(second >> 16)
				if size < wlHeaderSize || have < size {
					break
				}
				sender, opcode := wlOrder.Uint32(in), uint16(second&0xffff)
				r := wlReader{buf: in[wlHeaderSize:size]}

				var out wlWriter
				switch {
				case sender == wlDisplayID && opcode == wlDisplayGetRegistry:
					registry = r.uint()
					for _, g := range globals {
						out.begin(registry, wlRegistryGlobal)
						out.uint(g.name)
						out.string(g.iface)
						out.uint(g.version)
						out.end()
					}

				case sender == wlDisplayID && opcode == wlDisplaySync:
					out.begin(r.uint(), wlCallbackDone)
					out.uint(0)
					out.end()

				case sender == registry && opcode == wlRegistryBind:
					r.uint()
					iface, _, id := r.string(), r.uint(), r.uint()
					if iface == "wl_shm" {
						shm = id
						for _, f := range formats {
							out.begin(shm, wlShmFormat)
							out.uint(f)
							out.end()
						}
					}
				}

				if len(out.buf) > 0 {
					if _, err := server.Write(out.buf); err != nil {
						return
					}
				}
				copy(in, in[size:have])
				have -= size
			}
		}
	}()
}

// wlOpenPair monte une connexion servie par le compositeur de pacotille.
func wlOpenPair(t *testing.T, globals []wlFakeGlobal, formats []uint32) *wlConn {
	t.Helper()

	c, server := wlConnPair(t)
	wlServe(t, server, globals, formats)
	return c
}

// wlDefaultGlobals est ce qu'un compositeur ordinaire annonce de ce qui nous intéresse.
func wlDefaultGlobals() []wlFakeGlobal {
	return []wlFakeGlobal{
		{1, "wl_compositor", 6},
		{2, "wl_shm", 2},
		{3, "wl_output", 4},
		{4, "xdg_wm_base", 7},
	}
}

// TestWLBindGlobalsTakesTheLowerVersion garde le choix de version.
//
// **Les deux bords comptent, et pour deux raisons différentes.** Demander plus que l'offre
// est une erreur de protocole, donc le `min` protège la connexion ; demander moins que
// l'offre est ce qui rend le client compatible avec un compositeur plus récent, donc le
// `min` protège aussi la portée. Ici `wl_shm` est offert en 2 et voulu en 1, `xdg_wm_base`
// en 7 et voulu en 1, `wl_compositor` en 6 et voulu en 6.
func TestWLBindGlobalsTakesTheLowerVersion(t *testing.T) {
	c := wlOpenPair(t, wlDefaultGlobals(), []uint32{wlShmARGB8888, wlShmXRGB8888, formatXBGR8888})

	g, err := c.wlBindGlobals()
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}

	if g.compositorVersion != 6 {
		t.Errorf("wl_compositor est lié en version %d, attendu 6", g.compositorVersion)
	}
	if g.wmBaseVersion != 1 {
		t.Errorf("xdg_wm_base est lié en version %d, attendu 1 : on ne demande pas plus que ce qu'on emploie",
			g.wmBaseVersion)
	}
	if g.compositor == 0 || g.shm == 0 || g.wmBase == 0 {
		t.Errorf("un identifiant est resté nul : compositor %d, shm %d, wm_base %d",
			g.compositor, g.shm, g.wmBase)
	}
	if g.pixels.kind != pixelDirect {
		t.Errorf("la disposition retenue est classée %d, attendu %d", g.pixels.kind, pixelDirect)
	}
}

// TestWLBindGlobalsTakesTheOfferedVersionWhenItIsLower garde l'autre bord du même choix.
//
// Sans ce cas, le précédent passerait aussi avec un code qui demande toujours la version
// voulue : les trois offres y sont supérieures ou égales. Ici `wl_compositor` n'est offert
// qu'en 3, donc la version liée doit descendre — et c'est le régime d'un compositeur
// ancien, que la feuille de route cite nommément.
func TestWLBindGlobalsTakesTheOfferedVersionWhenItIsLower(t *testing.T) {
	globals := []wlFakeGlobal{
		{1, "wl_compositor", 3},
		{2, "wl_shm", 1},
		{3, "xdg_wm_base", 1},
	}
	c := wlOpenPair(t, globals, []uint32{wlShmXRGB8888})

	g, err := c.wlBindGlobals()
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	if g.compositorVersion != 3 {
		t.Errorf("wl_compositor est lié en version %d, attendu 3 : l'offre est plus basse que le besoin",
			g.compositorVersion)
	}
	if g.pixels.kind != pixelSwapped {
		t.Errorf("la disposition retenue est classée %d, attendu %d", g.pixels.kind, pixelSwapped)
	}
}

// TestWLBindGlobalsNamesWhatIsMissing garde le refus d'un global absent.
//
// Le message nomme l'interface **et** ce qu'elle apporte, parce que les deux manquent à
// qui lit l'erreur : savoir que `xdg_wm_base` est absent ne dit pas qu'un compositeur sans
// lui ne peut pas donner de fenêtre. C'est le régime d'un compositeur minimal ou d'un
// environnement embarqué.
func TestWLBindGlobalsNamesWhatIsMissing(t *testing.T) {
	globals := []wlFakeGlobal{
		{1, "wl_compositor", 6},
		{2, "wl_shm", 1},
	}
	c := wlOpenPair(t, globals, []uint32{formatXBGR8888})

	_, err := c.wlBindGlobals()
	if err == nil {
		t.Fatal("l'ouverture a réussi sans xdg_wm_base, attendu un refus")
	}
	for _, want := range []string{"xdg_wm_base", "get_xdg_surface"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("le refus dit %q, attendu qu'il porte %q", err, want)
		}
	}
}

// TestWLRegistryForgetsARemovedGlobal garde l'écoute de `global_remove`.
//
// Un global peut disparaître entre son annonce et notre `bind` — un écran débranché retire
// son `wl_output` —, et lier un nom retiré est une erreur de protocole. Le cas éprouve la
// table directement : le faire par le compositeur de pacotille demanderait de courir avec
// lui, ce qui rendrait le cas intermittent.
func TestWLRegistryForgetsARemovedGlobal(t *testing.T) {
	r := &wlRegistry{id: 2, offers: make(map[string]wlOffer)}

	var w wlWriter
	w.begin(2, wlRegistryGlobal)
	w.uint(9)
	w.string("wl_output")
	w.uint(4)
	w.end()
	rd := wlReader{buf: w.buf[wlHeaderSize:]}
	if err := r.handle(2, wlRegistryGlobal, &rd); err != nil {
		t.Fatalf("annonce du global : %v", err)
	}
	if _, ok := r.offers["wl_output"]; !ok {
		t.Fatal("le global annoncé n'est pas dans la table")
	}

	w.reset()
	w.begin(2, wlRegistryGlobalRemove)
	w.uint(9)
	w.end()
	rd = wlReader{buf: w.buf[wlHeaderSize:]}
	if err := r.handle(2, wlRegistryGlobalRemove, &rd); err != nil {
		t.Fatalf("retrait du global : %v", err)
	}
	if offer, ok := r.offers["wl_output"]; ok {
		t.Errorf("le global retiré est toujours dans la table sous le nom %d", offer.name)
	}
}

// TestWLBindPutsTheInterfaceOnTheWire garde, au niveau du registre, ce que le codage du
// fil garde au niveau d'un argument.
//
// Les deux cas ne gardent pas la même chose, et ce n'est pas un doublon : celui du codage
// dit que `newID` écrit trois valeurs, celui-ci dit que le `bind` **l'emploie** plutôt que
// d'écrire un identifiant seul. Une requête qui passerait par `uint` compilerait et
// enverrait un message de douze octets au lieu de vingt-huit.
func TestWLBindPutsTheInterfaceOnTheWire(t *testing.T) {
	c, server := wlConnPair(t)

	reg := &wlRegistry{id: 2, offers: map[string]wlOffer{
		"wl_compositor": {name: 1, version: 6},
		"wl_shm":        {name: 2, version: 1},
		"xdg_wm_base":   {name: 3, version: 1},
	}}
	var g wlGlobals
	if err := c.bind(reg, &g); err != nil {
		t.Fatalf("liaison : %v", err)
	}

	in := make([]byte, 4096)
	n, err := server.Read(in)
	if err != nil {
		t.Fatalf("lecture des requêtes : %v", err)
	}

	// Trois liaisons, dont la première porte « wl_compositor » : treize caractères,
	// son terminateur et trois octets de bourrage.
	r := wlReader{buf: in[wlHeaderSize:n]}
	if got, want := r.uint(), uint32(1); got != want {
		t.Errorf("le nom du global vaut %d, attendu %d", got, want)
	}
	if got, want := r.string(), "wl_compositor"; got != want {
		t.Fatalf("l'interface sur le fil est %q, attendu %q : le bind n'écrit pas un new_id non déclaré",
			got, want)
	}
	if got, want := r.uint(), uint32(6); got != want {
		t.Errorf("la version sur le fil vaut %d, attendu %d", got, want)
	}
	if got := r.uint(); got == 0 {
		t.Error("l'identifiant sur le fil est nul")
	}
}
