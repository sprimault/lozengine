// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La découverte de l'affichage : quelle carte, quel connecteur, quel mode, quel
// contrôleur. Rien n'est posé ici — ce fichier regarde, et le lot qui présente décide.

package backend

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

// drmDeviceDir est le répertoire où le noyau expose ses cartes.
const drmDeviceDir = "/dev/dri"

// drmCard est une carte ouverte et ce que la découverte en a tiré.
//
// **L'état initial du contrôleur est lu ici et pas ailleurs**, et c'est une contrainte
// d'ordre et non de commodité : il faut le lire **avant** d'avoir rien posé, sinon ce
// qu'on rendrait au système en partant serait ce qu'on y a mis. Le lot qui rend
// l'affichage le repasse tel quel.
type drmCard struct {
	file *os.File

	// fd est le descripteur de la carte, relevé une fois à l'ouverture.
	//
	// **Il est gardé, et `os.File.Fd` ne se rappelle jamais, pour une raison mesurée le
	// 4 novembre 2026** : Go ajoute ce nœud à son scrutateur, donc `Fd` **remet le
	// descripteur en mode bloquant** à chaque appel — c'est écrit dans sa
	// documentation. Un `SetNonblock` posé au montage était donc défait par le premier
	// `Fd` du tour suivant, et la lecture des événements de bascule bloquait pour de
	// bon : mesuré, le cas de présentation a tenu les six cents secondes du délai de
	// `go test` sans rien faire.
	//
	// Le relevé passe par [syscall.RawConn.Control], qui rend le descripteur sans
	// toucher à son mode, là où `Fd` le change.
	fd int

	// connector, crtc et mode sont ce que la découverte a retenu : le connecteur qui
	// porte un écran, le contrôleur qui peut l'alimenter, et le mode à poser.
	connector uint32
	crtc      uint32
	mode      drmModeInfo

	// saved est l'état du contrôleur avant qu'on y touche. Un contrôleur qui
	// n'alimentait rien a un `mode_valid` nul, ce qui se repasse aussi bien.
	saved drmModeCrtc

	// frames sont les tampons que la présentation fait tourner, order la disposition
	// que le pilote a accordée, et front celui qui est à l'écran.
	frames [drmFrames]drmFrame
	order  pixelOrder
	front  int

	// lit dit que le mode a été posé, donc que les tours suivants basculent au lieu de
	// rallumer. Il ne se relève jamais : un écran allumé le reste jusqu'à la
	// fermeture.
	lit bool

	// closed retient que la carte ne présentera plus rien, et err ce qui l'a mise fin
	// autrement que par la volonté du joueur. Seule la première cause est gardée.
	closed bool
	err    error
}

// errNoDisplay dit qu'aucune carte n'offre d'écran à allumer.
//
// **C'est une erreur de famille et non une panne**, ce qui compte pour la cascade
// d'ouverture : une machine qui a un serveur graphique garde ses connecteurs occupés, et
// une machine sans écran n'en a aucun de branché. Dans les deux cas la famille n'est pas
// la bonne, et c'est ce que l'ouverture doit pouvoir distinguer d'une carte cassée.
var errNoDisplay = errors.New("no DRM card offers a connected display")

// drmOpenCard ouvre la première carte capable de fabriquer un tampon simple.
//
// **Le critère est la capacité et non le nom**, parce que `/dev/dri` porte deux sortes de
// nœuds : les cartes et les périphériques de rendu seuls, qui n'ont aucune sortie. Trier
// par nom servirait tant que la première carte est la bonne, ce qui n'est pas une
// propriété mais une habitude.
//
// Les cartes sont essayées dans l'ordre de leur nom, qui est l'ordre où le noyau les a
// numérotées, et la première qui répond est retenue. L'erreur rendue est celle de la
// dernière essayée : une machine sans aucune carte et une machine dont la carte refuse ne
// se distinguent pas du point de vue de l'appelant, qui passera à la famille suivante.
func drmOpenCard() (*os.File, int, error) {
	entries, err := os.ReadDir(drmDeviceDir)
	if err != nil {
		return nil, 0, fmt.Errorf("listing %s: %w", drmDeviceDir, err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if n := e.Name(); len(n) > 4 && n[:4] == "card" {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	last := error(errNoDisplay)
	for _, name := range names {
		path := filepath.Join(drmDeviceDir, name)

		// **`O_NONBLOCK` dès l'ouverture, et non un `SetNonblock` plus tard** : la
		// présentation lit les événements de bascule à chaque tour et ne doit jamais
		// y attendre, et c'est le drapeau d'ouverture que Go respecte — un réglage
		// posé après coup est défait par le premier `os.File.Fd`.
		file, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
		if err != nil {
			last = err
			continue
		}

		fd, err := drmRawFd(file)
		if err != nil {
			file.Close()
			last = err
			continue
		}
		if dumb, err := drmCapabilityOf(fd, drmCapDumbBuffer); err != nil || dumb == 0 {
			file.Close()
			if err != nil {
				last = err
			} else {
				last = fmt.Errorf("%s cannot make dumb buffers", path)
			}
			continue
		}
		return file, fd, nil
	}
	return nil, 0, last
}

// drmCapabilityOf interroge une capacité du pilote.
func drmCapabilityOf(fd int, which drmCapability) (uint64, error) {
	arg := drmCap{capability: uint64(which)}
	if err := drmIoctl(fd, drmGetCap, &arg); err != nil {
		return 0, fmt.Errorf("asking the driver for capability %d: %w", which, err)
	}
	return arg.value, nil
}

// drmRawFd relève le descripteur d'un fichier sans toucher à son mode.
//
// **C'est la seule façon de l'obtenir sans le remettre en bloquant**, et c'est tout ce qui
// la justifie : `os.File.Fd` le change pour un nœud que Go a confié à son scrutateur, et sa
// documentation le dit. [syscall.RawConn.Control] appelle sa fonction avec le descripteur
// tel qu'il est, le temps de l'appel.
//
// Le descripteur qu'elle rend ne vaut que tant que le fichier reste ouvert, ce qui est la
// durée de vie de la carte qui le porte.
func drmRawFd(file *os.File) (int, error) {
	raw, err := file.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("getting raw access to %s: %w", file.Name(), err)
	}

	fd := -1
	if err := raw.Control(func(handle uintptr) { fd = int(handle) }); err != nil {
		return 0, fmt.Errorf("reading the descriptor of %s: %w", file.Name(), err)
	}
	return fd, nil
}

// drmEnableClientCap demande une capacité client au pilote.
//
// **Le moteur n'en demande aucune, et c'est un choix qu'il faut énoncer plutôt que de
// laisser deviner.** La seule qui changerait la découverte est
// [drmClientCapWritebackConnectors], qui fait apparaître le connecteur d'écriture différée
// dans l'énumération — mesuré par le brief, neuf connecteurs sans elle et dix avec. Un jeu
// qui l'obtiendrait allumerait sur une machine sans écran un connecteur que personne ne
// regarde, et croirait afficher : le mode de panne serait un jeu qui tourne et ne se voit
// pas.
//
// Elle existe donc pour le **dispositif d'épreuve**, qui est le seul à vouloir ce
// connecteur, et c'est la raison d'être de [drmDiscoverOn] : un cas ouvre la carte, pose
// la capacité, puis découvre, sans que le chemin du moteur change d'une ligne.
func drmEnableClientCap(fd int, which drmClientCapability) error {
	arg := drmClientCap{capability: uint64(which), value: 1}
	if err := drmIoctl(fd, drmSetClientCap, &arg); err != nil {
		return fmt.Errorf("asking the driver for client capability %d: %w", which, err)
	}
	return nil
}

// drmResources énumère les connecteurs et les contrôleurs de la carte.
//
// **Deux appels, et c'est le protocole de toutes les énumérations du réglage de mode** :
// le premier, avec des adresses nulles, rend les comptes ; le second, avec les tableaux
// alloués, rend les identifiants. Un seul appel ne peut pas suffire, un nombre de
// connecteurs ne se devinant pas.
//
// **Les comptes ne sont pas réécrits entre les deux appels**, ce qui est un piège de cette
// interface : le noyau remplit au plus ce qu'on annonce, donc annoncer plus que ce qu'on a
// alloué lui ferait écrire au-delà du tableau. Les champs de compte sont posés depuis la
// longueur des tranches et non depuis ce que le premier appel a rendu.
func drmResources(fd int) (connectors, crtcs []uint32, err error) {
	var counts drmModeCardRes
	if err := drmIoctl(fd, drmModeGetResources, &counts); err != nil {
		return nil, nil, fmt.Errorf("counting the card's resources: %w", err)
	}
	if counts.countConnectors == 0 || counts.countCrtcs == 0 {
		return nil, nil, errNoDisplay
	}

	connectors = make([]uint32, counts.countConnectors)
	crtcs = make([]uint32, counts.countCrtcs)

	res := drmModeCardRes{
		connectorIDPtr:  drmArrayAddr(connectors),
		crtcIDPtr:       drmArrayAddr(crtcs),
		countConnectors: uint32(len(connectors)),
		countCrtcs:      uint32(len(crtcs)),
	}
	if err := drmIoctlInto(fd, drmModeGetResources, &res, connectors, crtcs); err != nil {
		return nil, nil, fmt.Errorf("listing the card's resources: %w", err)
	}
	return connectors, crtcs, nil
}

// drmConnector lit un connecteur, ses modes et les encodeurs qui peuvent l'alimenter.
//
// **Le premier appel sonde le matériel**, et il faut le savoir parce que cela coûte :
// quelques millisecondes, et sur une sortie en cours d'usage un clignotement visible.
// C'est la raison pour laquelle la découverte a lieu une fois à l'ouverture et jamais par
// image.
func drmConnector(fd int, id uint32) (drmModeGetConnectorArg, []drmModeInfo, []uint32, error) {
	counts := drmModeGetConnectorArg{connectorID: id}
	if err := drmIoctl(fd, drmModeGetConnector, &counts); err != nil {
		return counts, nil, nil, fmt.Errorf("probing connector %d: %w", id, err)
	}

	modes := make([]drmModeInfo, counts.countModes)
	encoders := make([]uint32, counts.countEncoders)

	conn := drmModeGetConnectorArg{
		connectorID:   id,
		modesPtr:      drmArrayAddr(modes),
		encodersPtr:   drmArrayAddr(encoders),
		countModes:    uint32(len(modes)),
		countEncoders: uint32(len(encoders)),
	}
	if err := drmIoctlInto(fd, drmModeGetConnector, &conn, modes, encoders); err != nil {
		return conn, nil, nil, fmt.Errorf("reading connector %d: %w", id, err)
	}
	return conn, modes, encoders, nil
}

// drmPickMode choisit le mode à poser parmi ceux qu'un connecteur annonce.
//
// **Le drapeau de préférence est le seul critère sûr**, l'ordre de la liste étant à la
// convenance du pilote : prendre la première entrée marche souvent, et « souvent » n'est
// pas une propriété. À défaut de mode préféré, la plus grande surface est retenue, ce qui
// est le choix qu'un afficheur de kiosque veut et qui ne dépend d'aucun ordre.
//
// **Les modes entrelacés et doublés sont écartés avant tout choix**, et pas par goût : le
// moteur écrit un tampon linéaire d'une ligne par ligne d'écran, là qu'un mode entrelacé
// n'en affiche qu'une sur deux par image et qu'un mode doublé compte chaque ligne deux
// fois. L'image y serait à demi haute ou deux fois trop, sans qu'aucune erreur le dise.
func drmPickMode(modes []drmModeInfo) (drmModeInfo, bool) {
	var best drmModeInfo
	found := false

	for _, m := range modes {
		if m.flags&(drmModeFlagInterlace|drmModeFlagDoubleScan) != 0 {
			continue
		}
		if m.hdisplay == 0 || m.vdisplay == 0 {
			continue
		}

		if m.typ&drmModeTypePreferred != 0 {
			return m, true
		}
		if !found || int(m.hdisplay)*int(m.vdisplay) > int(best.hdisplay)*int(best.vdisplay) {
			best, found = m, true
		}
	}
	return best, found
}

// drmPickCrtc trouve un contrôleur capable d'alimenter le connecteur donné.
//
// **Deux voies, et la première est celle qu'il faut essayer d'abord** : si le connecteur
// porte déjà un encodeur actif, le contrôleur qui lui est accroché est celui que le
// matériel a lui-même choisi, donc celui dont la combinaison est sûre. La seconde parcourt
// les encodeurs possibles du connecteur et lit leur masque.
//
// **`possibleCrtcs` est un masque sur l'index et non sur l'identifiant**, et c'est le
// piège de cette interface : le bit zéro désigne le premier contrôleur que
// [drmResources] a rendu, pas le contrôleur numéro zéro. Prendre l'identifiant pour
// l'index donne un masque qui ne correspond à rien, et le pilote refuse le mode sans dire
// pourquoi.
func drmPickCrtc(fd int, conn drmModeGetConnectorArg, encoders, crtcs []uint32) (uint32, error) {
	if conn.encoderID != 0 {
		enc := drmModeGetEncoderArg{encoderID: conn.encoderID}
		if err := drmIoctl(fd, drmModeGetEncoder, &enc); err == nil && enc.crtcID != 0 {
			return enc.crtcID, nil
		}
	}

	for _, id := range encoders {
		enc := drmModeGetEncoderArg{encoderID: id}
		if err := drmIoctl(fd, drmModeGetEncoder, &enc); err != nil {
			continue
		}
		if crtc, ok := drmCrtcForMask(enc.possibleCrtcs, crtcs); ok {
			return crtc, nil
		}
	}
	return 0, fmt.Errorf("connector %d has no usable display controller", conn.connectorID)
}

// drmCrtcForMask lit un masque d'encodeur et rend le premier contrôleur qu'il autorise.
//
// **Elle est séparée de son appelant pour une seule raison : c'est elle qui porte le
// piège, et sous cette forme il s'éprouve sans matériel.** Le masque porte sur l'**index**
// du contrôleur dans l'énumération des ressources, jamais sur son identifiant — le bit
// zéro désigne le premier `crtc_id` rendu. Un masque lu sur l'identifiant désigne un
// contrôleur qui n'a rien à voir, ou aucun, et le pilote refuse ensuite le mode sans dire
// pourquoi.
//
// Un bit au-delà de ce que l'énumération a rendu est ignoré : le masque fait
// trente-deux places et une carte en porte rarement autant.
func drmCrtcForMask(mask uint32, crtcs []uint32) (uint32, bool) {
	for i, crtc := range crtcs {
		if mask&(1<<uint(i)) != 0 {
			return crtc, true
		}
	}
	return 0, false
}

// drmDiscover ouvre une carte et retient de quoi allumer un écran.
//
// **L'ordre des deux passes est imposé par le noyau et non par nous** : un connecteur
// `unknown` ne s'allume que s'il n'y en a **aucun** de `connected`, ce que la
// documentation de `enum drm_connector_status` dit en propres termes. Une seule passe qui
// accepterait les deux états prendrait un connecteur douteux alors qu'un écran est
// branché ailleurs.
//
// Un connecteur sans mode utilisable est écarté comme s'il n'était pas branché : il n'y a
// rien à y poser, et le distinguer n'apprendrait rien à l'appelant, qui passera de toute
// façon à la famille suivante.
func drmDiscover() (*drmCard, error) {
	file, fd, err := drmOpenCard()
	if err != nil {
		return nil, err
	}
	return drmDiscoverOn(file, fd)
}

// drmDiscoverOn découvre sur une carte déjà ouverte, et prend sa propriété.
//
// **La couture existe pour le dispositif d'épreuve et pour rien d'autre** : une machine
// sans sortie physique n'a de connecteur `connected` que son connecteur d'écriture
// différée, qui n'entre dans l'énumération qu'avec une capacité client que le moteur ne
// demande pas — [drmEnableClientCap] dit pourquoi. Un cas ouvre donc la carte lui-même,
// pose la capacité, puis entre ici.
//
// Elle prend la propriété du fichier dans les deux sens : la carte rendue le porte, et un
// échec le ferme. Un appelant qui garderait la main dessus aurait à défaire une découverte
// à moitié faite.
func drmDiscoverOn(file *os.File, fd int) (*drmCard, error) {
	connectors, crtcs, err := drmResources(fd)
	if err != nil {
		file.Close()
		return nil, err
	}

	for _, accept := range [2]drmConnectorStatus{drmConnectorConnected, drmConnectorUnknown} {
		for _, id := range connectors {
			conn, modes, encoders, err := drmConnector(fd, id)
			if err != nil || conn.connection != accept {
				continue
			}

			mode, ok := drmPickMode(modes)
			if !ok {
				continue
			}
			crtc, err := drmPickCrtc(fd, conn, encoders, crtcs)
			if err != nil {
				continue
			}

			card := &drmCard{file: file, fd: fd, connector: id, crtc: crtc, mode: mode}
			card.saved = drmModeCrtc{crtcID: crtc}
			if err := drmIoctl(fd, drmModeGetCrtc, &card.saved); err != nil {
				file.Close()
				return nil, fmt.Errorf("reading the initial state of controller %d: %w", crtc, err)
			}
			return card, nil
		}
	}

	file.Close()
	return nil, errNoDisplay
}

// Close remet l'affichage comme il était et rend la carte au système.
//
// **L'état du contrôleur se remet ici, chez qui l'a posé**, et non dans un lot ultérieur :
// un chemin qui allume un écran sans savoir l'éteindre laisse la machine dans l'état où il
// l'a mise à chaque exécution, y compris celle d'un cas de test. Ce qui reste au lot qui
// rend l'affichage est ce que ce fichier ne pose pas — le mode de la console et la
// maîtrise.
//
// **Le contrôleur se remet dans son état lu avant tout**, et c'est ce qui rend la remise
// juste quoi qu'on ait fait : un contrôleur qui n'alimentait rien a un `mode_valid` nul,
// donc le repasser l'éteint ; un contrôleur qui alimentait quelque chose retrouve son mode
// et son tampon.
//
// L'ordre des trois gestes n'est pas indifférent : éteindre avant de détruire les tampons,
// sinon le pilote refuserait de détruire ce qu'il balaie ; et détruire avant de fermer le
// descripteur, par lequel les requêtes passent.
func (c *drmCard) Close() {
	if c.file == nil {
		return
	}
	if c.lit {
		c.restore()
		c.lit = false
	}
	c.unmount()
	c.file.Close()
	c.file = nil
	c.closed = true
}

// restore repasse au contrôleur l'état que la découverte avait lu.
//
// Son échec n'est pas rapporté : on est en train de partir, et il n'y a plus personne à
// qui le dire. Ce qui reste vrai est que le descripteur se ferme ensuite, ce qui fait
// reprendre l'affichage au système de toute façon.
func (c *drmCard) restore() {
	state := c.saved
	_ = drmIoctl(c.fd, drmModeSetCrtc, &state)
}

// size rend la taille de l'image que le mode retenu demande.
func (c *drmCard) size() (width, height uint16) {
	return c.mode.hdisplay, c.mode.vdisplay
}

// name rend le nom du mode retenu, tel que le pilote l'a écrit.
//
// Le nom est un tableau de caractères et non une chaîne terminée : il peut occuper ses
// trente-deux octets sans séparateur final, donc la lecture s'arrête au premier zéro **ou**
// à la fin.
func (c *drmCard) name() string {
	n := c.mode.name[:]
	for i, b := range n {
		if b == 0 {
			n = n[:i]
			break
		}
	}
	return string(n)
}

// drmUsable dit si une erreur de découverte veut « cette famille n'est pas la bonne »
// plutôt que « l'affichage est cassé ».
//
// **C'est ce que la cascade d'ouverture lit**, et la distinction compte : une permission
// refusée sur `/dev/dri` et une absence de connecteur branché mènent toutes deux à essayer
// autre chose, là qu'un pilote qui refuse une requête qu'il devrait honorer est une panne
// dont il faut rendre compte.
func drmUsable(err error) bool {
	return !errors.Is(err, errNoDisplay) &&
		!errors.Is(err, os.ErrNotExist) &&
		!errors.Is(err, os.ErrPermission) &&
		!errors.Is(err, syscall.ENODEV)
}
