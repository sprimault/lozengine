// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la découverte doit choisir, et ce qu'une vraie carte en dit.

package backend

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

// cardRequired est la variable par laquelle un job d'intégration continue interdit au cas
// de découverte réelle d'être sauté.
//
// **Une troisième variable, et c'est la même raison que pour les deux premières** : un job
// qui a une carte n'a pas de compositeur, et un job qui a un compositeur n'a pas forcément
// de carte à lui. Une variable unique ferait échouer chacun sur le régime de l'autre.
const cardRequired = "LOZENGINE_DRM"

// requireCard saute le cas quand aucune carte n'est joignable, et **refuse de le sauter**
// quand l'environnement dit qu'il doit y en avoir une.
//
// Le second régime est tout l'intérêt : sans lui, un job qui n'aurait pas chargé son
// pilote virtuel passerait au vert en sautant le seul cas qui existe pour l'exécuter, et un
// saut n'est pas un échec.
// **Elle passe par [drmDiscoverOn] et pose la capacité d'écriture différée**, ce que le
// moteur ne fait pas. La raison est mesurée et elle a fait échouer la première version de
// ce cas : sur une machine sans sortie physique, le seul connecteur `connected` est celui
// d'écriture différée, et il n'entre dans l'énumération qu'avec cette capacité. Sans elle,
// la découverte ne voit que des connecteurs débranchés et le cas ne peut pas tourner — ce
// qui ne dit rien du moteur, dont le refus est juste.
func requireCard(tb testing.TB) *drmCard {
	tb.Helper()

	file, fd, err := drmOpenCard()
	if err == nil {
		// **L'ordre des deux capacités n'est pas indifférent, et c'est mesuré** :
		// demandée seule, celle d'écriture différée est refusée — le noyau ne la
		// sert qu'à un client qui a déjà pris le modèle atomique. Demandée après,
		// elle passe, et l'énumération montre dix connecteurs au lieu de neuf.
		//
		// Le refus n'arrête pas le cas : une carte qui a une vraie sortie n'a besoin
		// d'aucune des deux, et c'est le régime d'une machine avec écran.
		if drmEnableClientCap(fd, drmClientCapAtomic) == nil {
			_ = drmEnableClientCap(fd, drmClientCapWritebackConnectors)
		}

		var card *drmCard
		if card, err = drmDiscoverOn(file, fd); err == nil {
			tb.Cleanup(card.Close)
			return card
		}
	}
	if os.Getenv(cardRequired) != "" {
		tb.Fatalf("%s est posé et la découverte a échoué : %v", cardRequired, err)
	}
	tb.Skipf("aucune carte avec un écran : %v", err)
	return nil
}

// TestDRMPicksAModeTheEngineCanWriteInto garde le choix d'un mode.
//
// Quatre régimes, et chacun est un défaut qu'un choix naïf produirait : prendre la
// première entrée d'une liste dont l'ordre appartient au pilote ; retenir un mode
// entrelacé, où le moteur écrirait deux fois plus de lignes que l'écran n'en montre par
// image ; retenir un mode doublé, où il en écrirait deux fois trop peu ; et accepter une
// entrée nulle, qu'un connecteur débranché peut annoncer.
//
// **Ce qui le fait rougir** : remplacer le drapeau de préférence par l'ordre de la liste,
// ce que le premier cas attrape puisque le mode préféré n'y est pas en tête ; et retirer
// l'un des deux drapeaux écartés, que son cas attrape seul.
func TestDRMPicksAModeTheEngineCanWriteInto(t *testing.T) {
	mode := func(w, h uint16, typ, flags uint32) drmModeInfo {
		return drmModeInfo{hdisplay: w, vdisplay: h, typ: typ, flags: flags}
	}

	for _, c := range []struct {
		name  string
		modes []drmModeInfo
		want  [2]uint16
		ok    bool
	}{
		{
			"le mode préféré, même en seconde place",
			[]drmModeInfo{
				mode(1920, 1080, drmModeTypeDriver, 0),
				mode(1280, 720, drmModeTypeDriver|drmModeTypePreferred, 0),
			},
			[2]uint16{1280, 720}, true,
		},
		{
			"sans préférence, la plus grande surface",
			[]drmModeInfo{
				mode(800, 600, drmModeTypeDriver, 0),
				mode(1920, 1080, drmModeTypeDriver, 0),
				mode(1280, 1024, drmModeTypeDriver, 0),
			},
			[2]uint16{1920, 1080}, true,
		},
		{
			"un mode entrelacé est écarté, même préféré",
			[]drmModeInfo{
				mode(1920, 1080, drmModeTypePreferred, drmModeFlagInterlace),
				mode(1280, 720, drmModeTypeDriver, 0),
			},
			[2]uint16{1280, 720}, true,
		},
		{
			"un mode doublé est écarté, même préféré",
			[]drmModeInfo{
				mode(1920, 1080, drmModeTypePreferred, drmModeFlagDoubleScan),
				mode(640, 480, drmModeTypeDriver, 0),
			},
			[2]uint16{640, 480}, true,
		},
		{
			"une entrée de dimension nulle ne se choisit pas",
			[]drmModeInfo{mode(0, 0, drmModeTypePreferred, 0)},
			[2]uint16{0, 0}, false,
		},
		{"aucun mode", nil, [2]uint16{0, 0}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := drmPickMode(c.modes)
			if ok != c.ok {
				t.Fatalf("choix rendu %v, attendu %v", ok, c.ok)
			}
			if ok && (got.hdisplay != c.want[0] || got.vdisplay != c.want[1]) {
				t.Errorf("mode choisi %d×%d, attendu %d×%d",
					got.hdisplay, got.vdisplay, c.want[0], c.want[1])
			}
		})
	}
}

// TestDRMReadsTheControllerMaskAsAnIndex garde le piège de l'interface.
//
// **Le masque porte sur l'index et non sur l'identifiant**, et les deux premiers cas le
// disent ensemble : un masque de bit zéro désigne le **premier** contrôleur rendu, quel que
// soit son identifiant. Un seul des deux ne suffirait pas — avec des identifiants
// commençant à zéro, index et identifiant coïncideraient et la lecture fausse passerait.
//
// Ce qui le fait rougir : lire `crtcs[0]` sans masque, ou indexer par l'identifiant.
func TestDRMReadsTheControllerMaskAsAnIndex(t *testing.T) {
	for _, c := range []struct {
		name  string
		mask  uint32
		crtcs []uint32
		want  uint32
		ok    bool
	}{
		{"le premier bit désigne le premier rendu", 0b001, []uint32{41, 42, 43}, 41, true},
		{"le second bit, le second rendu", 0b010, []uint32{41, 42, 43}, 42, true},
		{"le premier bit autorisé gagne", 0b110, []uint32{41, 42, 43}, 42, true},
		{"un bit hors de l'énumération est ignoré", 0b1000, []uint32{41, 42, 43}, 0, false},
		{"un masque vide n'autorise rien", 0, []uint32{41}, 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := drmCrtcForMask(c.mask, c.crtcs)
			if ok != c.ok || got != c.want {
				t.Errorf("masque %#b rend (%d, %v), attendu (%d, %v)", c.mask, got, ok, c.want, c.ok)
			}
		})
	}
}

// TestDRMModeNameStopsAtTheEndOrTheSeparator garde la lecture d'un nom de mode.
//
// Le nom est un tableau de trente-deux caractères et non une chaîne terminée : il peut les
// occuper tous sans séparateur final. Lire jusqu'au premier zéro **sans** borner à la fin
// dépasserait le tableau sur ce cas-là, et c'est le seul qui le montre.
func TestDRMModeNameStopsAtTheEndOrTheSeparator(t *testing.T) {
	var card drmCard

	copy(card.mode.name[:], "1920x1080")
	if got, want := card.name(), "1920x1080"; got != want {
		t.Errorf("nom lu %q, attendu %q", got, want)
	}

	for i := range card.mode.name {
		card.mode.name[i] = 'x'
	}
	if got, want := card.name(), len(card.mode.name); len(got) != want {
		t.Errorf("un nom qui occupe ses %d octets se lit sur %d", want, len(got))
	}

	card.mode.name = [drmModeNameLen]byte{}
	if got := card.name(); got != "" {
		t.Errorf("un nom vide se lit %q", got)
	}
}

// TestDRMUsableTellsAWrongFamilyFromABrokenDriver garde ce que la cascade d'ouverture lit.
//
// **La distinction décide du comportement et non d'un message** : une famille qui n'est pas
// la bonne fait passer à la suivante, un pilote qui refuse une requête qu'il devrait honorer
// est une panne dont il faut rendre compte. Les confondre ferait l'un des deux défauts — un
// jeu qui ne s'ouvre pas sur une machine où une autre famille marche, ou une panne avalée
// en silence.
//
// Les erreurs enveloppées y sont parce que c'est sous cette forme qu'elles arrivent : la
// découverte enveloppe tout ce qu'elle rend.
func TestDRMUsableTellsAWrongFamilyFromABrokenDriver(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want bool
	}{
		{"aucun écran branché", errNoDisplay, false},
		{"aucun écran branché, enveloppé", errors.Join(errors.New("ouverture"), errNoDisplay), false},
		{"pas de /dev/dri", os.ErrNotExist, false},
		{"permission refusée sur la carte", os.ErrPermission, false},
		{"la carte a disparu", syscall.ENODEV, false},
		{"le pilote refuse une requête", syscall.EINVAL, true},
		{"mémoire insuffisante", syscall.ENOMEM, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := drmUsable(c.err); got != c.want {
				t.Errorf("drmUsable(%v) rend %v, attendu %v", c.err, got, c.want)
			}
		})
	}
}

// TestDRMArrayAddressIsZeroForTheCountingPass garde ce que le premier appel d'une
// énumération demande.
//
// **Zéro n'est pas une absence de valeur ici, c'est la valeur** : le noyau répond par les
// comptes réels quand on lui donne des adresses nulles avec des comptes nuls. Indexer un
// premier élément qui n'existe pas serait l'erreur naturelle, et elle ferait paniquer la
// découverte au lieu de la commencer.
func TestDRMArrayAddressIsZeroForTheCountingPass(t *testing.T) {
	if got := drmArrayAddr([]uint32(nil)); got != 0 {
		t.Errorf("une tranche nulle rend l'adresse %#x, attendu 0", got)
	}
	if got := drmArrayAddr([]uint32{}); got != 0 {
		t.Errorf("une tranche vide rend l'adresse %#x, attendu 0", got)
	}
	if got := drmArrayAddr(make([]uint32, 4)); got == 0 {
		t.Error("une tranche de quatre places rend une adresse nulle")
	}
}

// TestTheCardOffersAModeAndAController est la validation de la découverte sur du vrai
// matériel.
//
// **C'est le seul cas de la famille qui parle à un pilote**, et ce qu'il affirme est ce que
// les quatre lots suivants supposent : qu'une carte a été trouvée, qu'un connecteur porte
// un écran, qu'un mode non nul en est sorti, et qu'un contrôleur peut l'alimenter. Les cas
// purs ci-dessus gardent les choix ; celui-ci garde que la chaîne entière des appels se
// parle.
//
// **Il ne pose rien et n'a donc besoin d'aucun privilège** : la découverte lit, et
// l'ouverture en lecture-écriture du nœud suffit à lire. C'est ce qui permet de l'exécuter
// sur une machine partagée sans rien y déranger — aucun mode n'est posé, aucune maîtrise
// n'est prise.
//
// **Ce qu'il ne dit pas** : que le mode retenu s'affiche. Un connecteur d'écriture
// différée est `connected` sans qu'aucun écran existe, et c'est le régime de la machine de
// contrôle. Le lot qui présente est celui qui le montrera.
//
// **Et il ne dit pas non plus que le moteur trouverait cette carte** : son montage pose une
// capacité que le moteur ne demande pas, pour la raison que [drmEnableClientCap] porte. Ce
// qu'il garde est la chaîne des appels — énumérer, sonder, choisir, lire l'état initial —,
// pas la politique de choix d'une famille.
func TestTheCardOffersAModeAndAController(t *testing.T) {
	card := requireCard(t)

	width, height := card.size()
	if width == 0 || height == 0 {
		t.Errorf("le mode retenu fait %d×%d", width, height)
	}
	if card.connector == 0 {
		t.Error("aucun connecteur retenu")
	}
	if card.crtc == 0 {
		t.Error("aucun contrôleur retenu")
	}
	if card.saved.crtcID != card.crtc {
		t.Errorf("l'état initial lu porte le contrôleur %d, attendu %d",
			card.saved.crtcID, card.crtc)
	}

	t.Logf("carte découverte : connecteur %d, contrôleur %d, mode %q en %d×%d",
		card.connector, card.crtc, card.name(), width, height)
}
