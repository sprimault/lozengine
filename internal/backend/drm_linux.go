// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les faits d'interface du réglage de mode du noyau : les requêtes qu'on lui adresse et
// les structures qu'elles transportent. Rien ici ne décide de quoi que ce soit — la
// découverte, la présentation et les entrées vivent dans les fichiers qui les écrivent.
//
// **Ce sont des faits relevés dans les en-têtes UAPI du noyau**, dont la
// `Linux-syscall-note` existe précisément pour que du code non-GPL les emploie. Chaque
// déclaration cite l'interface et la requête qui la définissent, jamais un chemin de
// machine ni un numéro de ligne : un numéro de ligne d'en-tête ne se reproduit pas d'un
// noyau à l'autre.
//
// **Les codes de format n'y sont pas, et c'est voulu** : `DRM_FORMAT_XBGR8888` est le
// même code à quatre caractères que `wl_shm` annonce, donc [formatXBGR8888] le porte déjà
// pour les deux familles. Le relever une seconde fois donnerait deux noms à une valeur.

package backend

import "unsafe"

// La composition du mot d'une requête `ioctl`, telle que `asm-generic/ioctl.h` la définit.
//
// **Quatre champs dans un mot de trente-deux bits** : le numéro sur huit bits, le type
// d'interface sur huit, la taille de l'argument sur quatorze, et la direction sur deux.
// C'est ce qui explique qu'une requête dépende de la taille de sa structure, et donc que
// les constantes ci-dessous la dérivent au lieu de l'écrire.
//
// **Cette composition est celle d'`asm-generic`, et quatre architectures en ont une
// autre** — alpha, mips, powerpc et sparc, dont aucune n'est dans les cibles du moteur.
// Les cinq qui y sont, `amd64`, `386`, `arm`, `arm64` et `riscv64`, emploient toutes
// celle-ci.
const (
	drmRequestNumberShift = 0
	drmRequestTypeShift   = 8
	drmRequestSizeShift   = 16
	drmRequestDirShift    = 30
)

// Les directions qu'une requête déclare, du point de vue de l'espace utilisateur.
//
// `_IOC_WRITE` vaut un et `_IOC_READ` deux ; une requête qui remplit la structure qu'on
// lui donne porte les deux, ce que `_IOWR` compose. `_IO` n'en porte aucune et ne
// transporte rien.
const (
	drmDirNone  = 0 << drmRequestDirShift
	drmDirWrite = 1 << drmRequestDirShift
	drmDirRead  = 2 << drmRequestDirShift

	drmDirBoth = drmDirRead | drmDirWrite
)

// drmRequestType est le type d'interface que toutes les requêtes du réglage de mode
// portent, `DRM_IOCTL_BASE` valant le caractère `d`.
const drmRequestType = 'd' << drmRequestTypeShift

// Les requêtes du réglage de mode, composées de leur numéro et de la taille de leur
// argument.
//
// **Elles se dérivent de la structure Go et ne sont pas recopiées**, et c'est ce qui
// place le mode de panne au bon endroit : une structure mal transposée produit un mot
// cohérent mais faux, que le noyau refuse par un `EINVAL` muet, sur une machine qui a un
// écran. Dérivée, l'erreur se voit dans [TestDRMLayoutMatchesTheKernelHeaders], qui fige
// la taille et le décalage de chaque champ, et dans
// [TestDRMRequestsMatchTheHeaderMacros], qui compare les mots aux valeurs que le
// préprocesseur rend.
//
// `unsafe.Sizeof` d'un type concret **est** une expression constante en Go, donc rien
// ici ne se calcule à l'exécution.
//
// Les numéros viennent de `drm.h` : `DRM_IOCTL_SET_CLIENT_CAP` en `_IOW`,
// `SET_MASTER` et `DROP_MASTER` en `_IO`, et tout le reste du réglage de mode en `_IOWR`.
const (
	drmGetCap       = drmDirBoth | uintptr(unsafe.Sizeof(drmCap{}))<<drmRequestSizeShift | drmRequestType | 0x0c
	drmSetClientCap = drmDirWrite | uintptr(unsafe.Sizeof(drmClientCap{}))<<drmRequestSizeShift | drmRequestType | 0x0d

	drmSetMaster  = drmDirNone | drmRequestType | 0x1e
	drmDropMaster = drmDirNone | drmRequestType | 0x1f

	drmModeGetResources = drmDirBoth | uintptr(unsafe.Sizeof(drmModeCardRes{}))<<drmRequestSizeShift | drmRequestType | 0xa0
	drmModeGetCrtc      = drmDirBoth | uintptr(unsafe.Sizeof(drmModeCrtc{}))<<drmRequestSizeShift | drmRequestType | 0xa1
	drmModeSetCrtc      = drmDirBoth | uintptr(unsafe.Sizeof(drmModeCrtc{}))<<drmRequestSizeShift | drmRequestType | 0xa2
	drmModeGetEncoder   = drmDirBoth | uintptr(unsafe.Sizeof(drmModeGetEncoderArg{}))<<drmRequestSizeShift | drmRequestType | 0xa6
	drmModeGetConnector = drmDirBoth | uintptr(unsafe.Sizeof(drmModeGetConnectorArg{}))<<drmRequestSizeShift | drmRequestType | 0xa7

	// `DRM_IOCTL_MODE_RMFB` prend un entier non signé et non une structure, seul cas
	// du relevé : son argument est l'identifiant du tampon à retirer.
	drmModeRmFB = drmDirBoth | uintptr(unsafe.Sizeof(uint32(0)))<<drmRequestSizeShift | drmRequestType | 0xaf

	drmModePageFlip = drmDirBoth | uintptr(unsafe.Sizeof(drmModeCrtcPageFlip{}))<<drmRequestSizeShift | drmRequestType | 0xb0

	drmModeCreateDumbReq  = drmDirBoth | uintptr(unsafe.Sizeof(drmModeCreateDumb{}))<<drmRequestSizeShift | drmRequestType | 0xb2
	drmModeMapDumbReq     = drmDirBoth | uintptr(unsafe.Sizeof(drmModeMapDumb{}))<<drmRequestSizeShift | drmRequestType | 0xb3
	drmModeDestroyDumbReq = drmDirBoth | uintptr(unsafe.Sizeof(drmModeDestroyDumb{}))<<drmRequestSizeShift | drmRequestType | 0xb4

	drmModeAddFB2 = drmDirBoth | uintptr(unsafe.Sizeof(drmModeFBCmd2{}))<<drmRequestSizeShift | drmRequestType | 0xb8
)

// drmCapability est une capacité du pilote que `DRM_IOCTL_GET_CAP` interroge.
//
// Deux valeurs sont relevées. [drmCapDumbBuffer] est celle qui décide : un pilote qui rend
// zéro ne sait pas fabriquer de tampon simple, donc la carte n'est pas utilisable par ce
// chemin — c'est le cas d'un périphérique de rendu sans sortie, qui existe sur la même
// machine sous un autre nœud. [drmCapDumbPreferredDepth] est relevée pour ne pas la
// supposer : le moteur écrit en trente-deux bits par pixel, et un pilote qui préférerait
// autre chose l'accepte quand même, donc elle informe sans décider.
type drmCapability uint64

// Les capacités de pilote de `drm.h`, dans la numérotation qu'il déclare.
const (
	drmCapDumbBuffer         drmCapability = 0x1
	drmCapDumbPreferredDepth drmCapability = 0x3
)

// drmClientCapability est une capacité que le client demande au pilote par
// `DRM_IOCTL_SET_CLIENT_CAP`.
//
// Deux valeurs sont relevées, et **le moteur n'en demande aucune** : il pose son mode par
// `SETCRTC`, l'interface héritée, donc [drmClientCapAtomic] ne lui sert pas ; et
// [drmClientCapWritebackConnectors], qui fait **apparaître** le connecteur d'écriture
// différée dans l'énumération, ne sert qu'au dispositif d'épreuve — un jeu qui
// l'obtiendrait allumerait, sur une machine sans écran, une sortie que personne ne
// regarde.
//
// **Les deux ne se demandent que dans cet ordre, et c'est mesuré le 4 novembre 2026** :
// l'écriture différée demandée seule est **refusée**, le noyau ne la servant qu'à un
// client qui a déjà pris le modèle atomique. Demandée après lui, elle passe, et
// l'énumération montre dix connecteurs au lieu de neuf — dont un writeback `connected` qui
// annonce trente-deux modes, donc avec de quoi choisir. Le brief la croyait indépendante,
// et le cas de découverte a échoué deux fois dessus.
type drmClientCapability uint64

// Les capacités de `drm.h`, dans la numérotation qu'il déclare.
const (
	drmClientCapAtomic              drmClientCapability = 3
	drmClientCapWritebackConnectors drmClientCapability = 5
)

// drmConnectorStatus est l'état d'un connecteur, tel que le champ `connection` de
// `DRM_IOCTL_MODE_GETCONNECTOR` le rend.
//
// Trois valeurs : [drmConnectorConnected], [drmConnectorDisconnected] et
// [drmConnectorUnknown].
//
// **Elles ne sont pas dans l'UAPI, et il faut le dire parce que la godoc du champ y
// renvoie quand même** : `drm_mode.h` écrit « See enum drm_connector_status », et cette
// énumération vit dans `include/drm/drm_connector.h`, un en-tête interne au noyau. Les
// valeurs sont donc relevées là, où elles sont écrites en clair plutôt que laissées à
// l'ordre de déclaration.
//
// Ce que la même godoc du noyau impose au lot qui découvre, et qui n'est pas une
// préférence : un connecteur `unknown` ne s'allume que s'il n'y en a **aucun** de
// `connected`.
type drmConnectorStatus uint32

// Les trois états de `drm_connector.h`.
const (
	drmConnectorConnected    drmConnectorStatus = 1
	drmConnectorDisconnected drmConnectorStatus = 2
	drmConnectorUnknown      drmConnectorStatus = 3
)

// drmModeNameLen est la longueur du nom d'un mode, `DRM_DISPLAY_MODE_LEN`.
//
// Le nom est un tableau de caractères et non une chaîne terminée : il peut occuper ses
// trente-deux octets sans séparateur final, donc le lire demande de s'arrêter au premier
// zéro **ou** à la fin.
const drmModeNameLen = 32

// Les types et les drapeaux d'un mode, du champ `type` et du champ `flags` de
// [drmModeInfo].
//
// [drmModeTypePreferred] est le seul critère sûr pour choisir un mode : la liste qu'un
// connecteur rend est ordonnée à la convenance du pilote, donc prendre la première sans
// regarder ce drapeau marche souvent et pas toujours. [drmModeTypeDriver] dit que le mode
// vient du pilote et non d'un réglage de l'espace utilisateur.
//
// Les deux drapeaux sont relevés pour être **écartés** : sous un mode entrelacé, une image
// ne porte qu'une ligne sur deux, et sous un mode doublé chaque ligne compte double. Le
// moteur écrit un tampon linéaire, donc il choisit un mode qui n'a ni l'un ni l'autre.
const (
	drmModeTypePreferred = 1 << 3
	drmModeTypeDriver    = 1 << 6

	drmModeFlagInterlace  = 1 << 4
	drmModeFlagDoubleScan = 1 << 5
)

// Les drapeaux de `DRM_IOCTL_MODE_PAGE_FLIP`.
//
// [drmPageFlipEvent] demande que la bascule soit annoncée par un événement lisible sur
// le descripteur de la carte, ce qui est la seule façon de savoir qu'un tampon est libre.
// [drmPageFlipAsync] demande au pilote de ne pas attendre le retour de balayage, donc
// d'accepter le déchirement ; le moteur ne le pose pas.
const (
	drmPageFlipEvent = 0x01
	drmPageFlipAsync = 0x02
)

// Les types d'événement que le descripteur de la carte rend, de `drm.h`.
//
// Seul [drmEventFlipComplete] intéresse le moteur ; [drmEventVblankType] est relevé parce
// qu'il arrive sur le même descripteur et qu'une boucle qui ne le reconnaît pas le
// prendrait pour une bascule.
const (
	drmEventVblankType   = 0x01
	drmEventFlipComplete = 0x02
)

// drmCap est l'argument de `DRM_IOCTL_GET_CAP`, soit `struct drm_get_cap`.
//
// Elle a la même forme que [drmClientCap] et reste un type distinct, parce que les deux
// structures sont distinctes dans l'UAPI : les fondre ferait dépendre la requête de l'une
// de la taille de l'autre, et un noyau qui élargirait une seule des deux changerait les
// deux mots de requête.
type drmCap struct {
	capability uint64
	value      uint64
}

// drmClientCap est l'argument de `DRM_IOCTL_SET_CLIENT_CAP`, soit
// `struct drm_set_client_cap`.
type drmClientCap struct {
	capability uint64
	value      uint64
}

// drmModeCardRes est l'argument de `DRM_IOCTL_MODE_GETRESOURCES`, soit
// `struct drm_mode_card_res`.
//
// **Elle se remplit en deux appels et c'est le protocole de toutes les énumérations du
// réglage de mode** : un premier appel avec les pointeurs à zéro rend les comptes, un
// second avec les tableaux alloués rend les identifiants. Les deux sont nécessaires parce
// qu'un nombre de connecteurs ne se devine pas.
//
// Les quatre premiers champs sont des **adresses de tableaux** passées en soixante-quatre
// bits quelle que soit l'architecture, ce qui est le cas de toutes les structures du
// réglage de mode : un pointeur de trente-deux bits s'y écrit dans un champ de
// soixante-quatre.
type drmModeCardRes struct {
	fbIDPtr        uint64
	crtcIDPtr      uint64
	connectorIDPtr uint64
	encoderIDPtr   uint64

	countFBs        uint32
	countCrtcs      uint32
	countConnectors uint32
	countEncoders   uint32

	minWidth  uint32
	maxWidth  uint32
	minHeight uint32
	maxHeight uint32
}

// drmModeInfo décrit un mode d'affichage, soit `struct drm_mode_modeinfo`.
//
// **Ce que le moteur en lit tient en trois champs** — `hdisplay`, `vdisplay` et le nom —,
// et les dix autres décrivent les temps de balayage que le pilote impose. Ils sont
// transportés quand même : la structure part telle quelle à `SETCRTC`, qui veut le mode
// entier tel que le connecteur l'a annoncé.
//
// `vrefresh` est en hertz entiers, donc il ne suffit pas à distinguer deux modes de
// cadence voisine ; c'est `clock` et les temps qui le font.
type drmModeInfo struct {
	clock uint32

	hdisplay   uint16
	hsyncStart uint16
	hsyncEnd   uint16
	htotal     uint16
	hskew      uint16

	vdisplay   uint16
	vsyncStart uint16
	vsyncEnd   uint16
	vtotal     uint16
	vscan      uint16

	vrefresh uint32

	flags uint32
	typ   uint32
	name  [drmModeNameLen]byte
}

// drmModeGetConnectorArg est l'argument de `DRM_IOCTL_MODE_GETCONNECTOR`, soit
// `struct drm_mode_get_connector`.
//
// Même protocole en deux appels que [drmModeCardRes], et une subtilité de plus : le
// premier appel **sonde le matériel**, ce qui peut prendre des millisecondes et faire
// clignoter une sortie en cours d'usage. C'est pourquoi la découverte ne se refait pas
// par image.
//
// `pad` doit être nul, ce que la valeur zéro de la structure donne.
type drmModeGetConnectorArg struct {
	encodersPtr   uint64
	modesPtr      uint64
	propsPtr      uint64
	propValuesPtr uint64

	countModes    uint32
	countProps    uint32
	countEncoders uint32

	encoderID       uint32
	connectorID     uint32
	connectorType   uint32
	connectorTypeID uint32

	connection drmConnectorStatus
	mmWidth    uint32
	mmHeight   uint32
	subpixel   uint32

	pad uint32
}

// drmModeGetEncoderArg est l'argument de `DRM_IOCTL_MODE_GETENCODER`, soit
// `struct drm_mode_get_encoder`.
//
// `possibleCrtcs` est un **masque de bits sur l'index** du contrôleur dans l'énumération
// des ressources, et non sur son identifiant : le bit zéro désigne le premier `crtc_id`
// que `GETRESOURCES` a rendu. Prendre l'identifiant pour l'index donne un masque qui ne
// correspond à rien, et le pilote refuse le mode sans dire pourquoi.
type drmModeGetEncoderArg struct {
	encoderID   uint32
	encoderType uint32

	crtcID uint32

	possibleCrtcs  uint32
	possibleClones uint32
}

// drmModeCrtc est l'argument de `DRM_IOCTL_MODE_GETCRTC` et de `SETCRTC`, soit
// `struct drm_mode_crtc`.
//
// **La même structure sert à lire l'état et à le poser**, ce qui est ce qui permet de
// rendre l'affichage au système en repassant exactement ce qu'on avait lu. `modeValid` à
// zéro avec un `fbID` nul éteint le contrôleur.
type drmModeCrtc struct {
	setConnectorsPtr uint64
	countConnectors  uint32

	crtcID uint32
	fbID   uint32

	x uint32
	y uint32

	gammaSize uint32
	modeValid uint32
	mode      drmModeInfo
}

// drmModeCreateDumb est l'argument de `DRM_IOCTL_MODE_CREATE_DUMB`, soit
// `struct drm_mode_create_dumb`.
//
// **Le pas de ligne est rendu par le pilote et ne se calcule pas** : il aligne la largeur
// à sa convenance, donc une largeur non multiple de son pas donne un `pitch` plus grand
// que `width × 4`. C'est le piège que le lot qui présente doit mesurer sur une largeur
// **non alignée** — 1366 et non 1920, dont le pas tombe juste par hasard.
//
// `height` précède `width`, dans cet ordre, et l'inversion ne lève aucune erreur : elle
// rend un tampon de la bonne taille et de la mauvaise forme.
type drmModeCreateDumb struct {
	height uint32
	width  uint32
	bpp    uint32
	flags  uint32

	handle uint32
	pitch  uint32
	size   uint64
}

// drmModeMapDumb est l'argument de `DRM_IOCTL_MODE_MAP_DUMB`, soit
// `struct drm_mode_map_dumb`.
//
// Elle ne projette rien elle-même : elle rend le **décalage** à passer à `mmap` sur le
// descripteur de la carte. `pad` doit être nul.
type drmModeMapDumb struct {
	handle uint32
	pad    uint32
	offset uint64
}

// drmModeDestroyDumb est l'argument de `DRM_IOCTL_MODE_DESTROY_DUMB`, soit
// `struct drm_mode_destroy_dumb`.
type drmModeDestroyDumb struct {
	handle uint32
}

// drmModeFBCmd2 est l'argument de `DRM_IOCTL_MODE_ADDFB2`, soit
// `struct drm_mode_fb_cmd2`.
//
// **`ADDFB2` et non l'`ADDFB` historique**, et ce n'est pas un choix de style : mesuré par
// le brief, en trente-deux bits par pixel sur une profondeur de vingt-quatre, l'ancienne
// requête ne peut exprimer que `XR24` — l'ordre que la présentation Windows fabrique par
// conversion. `ADDFB2` prend un code de format explicite, donc elle sait demander
// [formatXBGR8888], qui est la disposition du moteur.
//
// Les quatre places de chaque tableau sont les plans d'un format multiplanaire ; un
// format à un seul plan n'emploie que la première et laisse les autres nulles.
type drmModeFBCmd2 struct {
	fbID        uint32
	width       uint32
	height      uint32
	pixelFormat uint32
	flags       uint32

	handles  [4]uint32
	pitches  [4]uint32
	offsets  [4]uint32
	modifier [4]uint64
}

// drmModeCrtcPageFlip est l'argument de `DRM_IOCTL_MODE_PAGE_FLIP`, soit
// `struct drm_mode_crtc_page_flip`.
//
// `userData` revient tel quel dans l'événement de bascule, ce qui est la façon d'associer
// une annonce à ce qui l'a demandée. `reserved` doit être nul.
type drmModeCrtcPageFlip struct {
	crtcID   uint32
	fbID     uint32
	flags    uint32
	reserved uint32
	userData uint64
}

// drmEventHeader est l'en-tête commun des événements de la carte, soit
// `struct drm_event`.
//
// **La longueur porte l'événement entier, en-tête compris**, donc une boucle de lecture
// avance de `length` et non de la taille de la structure qu'elle a reconnue. C'est ce qui
// lui permet de sauter un type qu'elle ne connaît pas au lieu de se désynchroniser.
type drmEventHeader struct {
	typ    uint32
	length uint32
}

// drmEventVblank est l'événement de bascule et de retour de balayage, soit
// `struct drm_event_vblank`.
//
// `crtcID` vaut zéro sur les noyaux qui ne le renseignent pas, ce que l'en-tête dit en
// propres termes. Un client qui pilote un seul contrôleur n'en a pas besoin ; un client
// qui en pilote plusieurs se sert de `userData`, qu'il a choisi lui-même.
type drmEventVblank struct {
	base     drmEventHeader
	userData uint64
	tvSec    uint32
	tvUsec   uint32
	sequence uint32
	crtcID   uint32
}
