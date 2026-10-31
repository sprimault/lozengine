// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le backend emprunte à Windows : les procédures, les constantes et les
// structures, et rien qui décide de quoi que ce soit.
//
// Les valeurs ne sont pas recopiées d'un en-tête ni d'un autre moteur : chacune a été
// éprouvée par son effet avant d'entrer ici — une opération raster par ce qu'elle
// peint, un numéro de message par le moment où il arrive. Les tailles de structures se
// prennent par `unsafe.Sizeof` plutôt que de s'écrire, parce qu'elles diffèrent entre
// `amd64` et `386` et que la seule façon de ne pas s'y tromper est de ne pas les
// connaître.

package backend

import (
	"syscall"
	"unsafe"
)

// Les deux bibliothèques du système, nommées sans chemin : elles figurent dans
// `KnownDLLs`, donc le chargeur prend la section déjà mappée et ne parcourt aucun
// chemin de recherche. Une bibliothèque plantée à côté du binaire ne peut pas s'y
// substituer, et il n'y a donc ni `LoadLibraryEx` ni chemin en dur à écrire.
var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
)

// Les procédures employées, résolues au premier appel puis gardées.
//
// `LazyProc.Call` et non `SyscallN` : l'écart mesuré est de six nanosecondes sur
// soixante, et la pompe à messages en fait quelques dizaines par image, soit le
// millième de pour cent d'une image. Garder les adresses dans des variables
// n'apporterait rien non plus — un nanomètre de différence —, `Addr` étant un
// `sync.Once` déjà franchi.
var (
	registerClassExW = user32.NewProc("RegisterClassExW")
	createWindowExW  = user32.NewProc("CreateWindowExW")
	destroyWindow    = user32.NewProc("DestroyWindow")
	defWindowProcW   = user32.NewProc("DefWindowProcW")
	showWindow       = user32.NewProc("ShowWindow")
	peekMessageW     = user32.NewProc("PeekMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessageW = user32.NewProc("DispatchMessageW")
	getClientRect    = user32.NewProc("GetClientRect")
	adjustWindowRect = user32.NewProc("AdjustWindowRect")
	getDC            = user32.NewProc("GetDC")
	releaseDC        = user32.NewProc("ReleaseDC")

	// Le suivi du pointeur, sans lequel le système ne dit jamais qu'il est sorti de
	// la fenêtre. Le nom porte un suffixe parce que le type qu'il prend s'appelle
	// déjà ainsi.
	trackMouseEventProc = user32.NewProc("TrackMouseEvent")

	// Poser le curseur, ce qui sert à ne pas en poser du tout.
	setCursor = user32.NewProc("SetCursor")

	// Les deux que la reprise du focus emploie, pour relire ce que le clavier tient
	// déjà. La traduction suit la disposition, ce qui est voulu ici.
	mapVirtualKeyW   = user32.NewProc("MapVirtualKeyW")
	getAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	getKeyState      = user32.NewProc("GetKeyState")

	sendMessageW       = user32.NewProc("SendMessageW")
	getSystemMetrics   = user32.NewProc("GetSystemMetrics")
	createIconIndirect = user32.NewProc("CreateIconIndirect")
	destroyIcon        = user32.NewProc("DestroyIcon")
	loadCursorW        = user32.NewProc("LoadCursorW")

	// Dans kernel32 et non user32, ce qu'un `LazyProc` ne dit qu'à l'appel : la
	// compilation et les tests sans écran laissaient passer l'erreur, et c'est la
	// vérification manuelle qui l'a trouvée.
	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	// La conscience de densité, que Windows ne porte qu'à partir de la version 1703 de
	// Windows 10 — la mise à jour d'avril 2017. Elle se cherche avant de s'appeler,
	// comme celle qui suit : voir [procAddr].
	setProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")

	// La densité de l'écran, pour ouvrir la fenêtre à la taille que le jeu demande en
	// points et non en pixels physiques. Absente avant la version 1607 de Windows 10
	// — août 2016 —, et cherchée avant de s'appeler : voir [densityFactor].
	getDpiForSystem = user32.NewProc("GetDpiForSystem")

	// La remise du tampon à l'écran. Son adresse se prend une fois à l'ouverture de la
	// fenêtre, jamais par image : voir [window.presentProc].
	setDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
	createDIBSection  = gdi32.NewProc("CreateDIBSection")
	createBitmap      = gdi32.NewProc("CreateBitmap")
	deleteObject      = gdi32.NewProc("DeleteObject")
)

// procAddr rend l'adresse d'une procédure chargée à la demande, et l'erreur de
// recherche quand le système ne la porte pas.
//
// C'est la seule façon d'atteindre une procédure dont la présence n'est pas acquise.
// `LazyProc.Call` et `LazyProc.Addr` passent tous deux par `mustFind`, donc ils
// **paniquent** là où un moteur doit se replier ou refuser proprement — un appel
// système manquant est une version de Windows plus ancienne que prévu, pas un défaut
// de programmation.
//
// Les vingt-sept autres procédures du backend s'appellent directement : toutes sont
// antérieures à Windows 2000, donc leur absence signifierait un système qui ne peut
// de toute façon pas exécuter un binaire Go.
func procAddr(p *syscall.LazyProc) (uintptr, error) {
	if err := p.Find(); err != nil {
		return 0, err
	}
	return p.Addr(), nil
}

// Les styles de classe et de fenêtre.
const (
	// csOwnDC donne à la fenêtre son propre contexte de périphérique, qui n'a donc
	// pas à être repris à chaque image.
	csVRedraw = 0x0001
	csHRedraw = 0x0002
	csOwnDC   = 0x0020

	// wsOverlappedWindow est la fenêtre ordinaire : bordure, titre, menu système,
	// boutons de réduction et d'agrandissement.
	wsOverlappedWindow = 0x00CF0000

	// cwUseDefault laisse le système placer la fenêtre.
	cwUseDefault = ^uintptr(0x7FFFFFFF)
)

// Les commandes de ShowWindow.
const (
	swShow = 5
)

// Les messages que le backend traite. Tous les autres vont à `DefWindowProcW` : un
// cycle de vie trivial en apporte trente-deux, dont deux enregistrés dynamiquement par
// le système, et aucun backend ne peut présumer de la liste.
const (
	wmDestroy   = 0x0002
	wmSize      = 0x0005
	wmKillFocus = 0x0008
	wmClose     = 0x0010

	// Les quatre messages de touche. Les variantes « système » arrivent quand Alt
	// est tenu, et pour F10 seule : les ignorer laisserait un déplacement coincé dès
	// qu'un joueur garde Alt enfoncé.
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	// La prise de focus, par laquelle on rattrape ce qu'un doigt déjà posé n'a pas
	// envoyé ici.
	wmSetFocus = 0x0007

	// La demande des bornes de redimensionnement, par laquelle le plancher se pose.
	// Elle ouvre la séquence de création, avant que la poignée soit connue d'ici.
	wmGetMinMaxInfo = 0x0024
)

// mapvkVscToVkEx demande le code virtuel d'une place, préfixe des touches étendues
// compris.
const mapvkVscToVkEx = 3

// mapvkVkToVsc demande la place d'un code virtuel, sans préfixe. C'est la traduction
// inverse de [mapvkVscToVkEx], et elle sert à retrouver les places du pavé numérique
// depuis les codes virtuels qu'elles produisent quand le verrou est allumé.
const mapvkVkToVsc = 0

// vkNumLock est le code virtuel du verrou numérique. Son bit de poids faible dit s'il
// est allumé, là où le bit de poids fort dirait que la touche est enfoncée.
const vkNumLock = 0x90

// wmSetIcon pose une icône sur la fenêtre, et iconBig ou iconSmall dit laquelle des
// deux : la grande sert la barre des tâches, la petite le coin du cadre.
const (
	wmSetIcon = 0x0080

	iconSmall = 0
	iconBig   = 1
)

// smCXIcon et smCXSmallIcon sont les côtés que le système veut pour ses deux icônes.
// On les demande plutôt que de les écrire : ils suivent la densité de l'écran, donc
// 32 et 16 ne valent qu'à cent pour cent.
const (
	smCXIcon      = 11
	smCXSmallIcon = 49
)

// idcArrow est le curseur ordinaire, celui d'une fenêtre qui n'a rien à dire du
// pointeur.
//
// Une classe enregistrée sans curseur n'en impose aucun, et le système laisse alors au
// pointeur la forme qu'il avait en entrant dans la zone utile : la flèche de
// redimensionnement reste affichée après qu'on a tiré un bord vers l'intérieur. Le
// défaut vient du jalon 2, où le champ existait et n'était pas rempli.
const idcArrow = 32512

// wmSetCursor demande à la fenêtre quel curseur poser, et htClient dit que le pointeur
// est sur la zone de jeu plutôt que sur une bordure.
//
// C'est par là qu'une fenêtre cache le curseur, et non par sa classe : celle-ci est
// enregistrée une fois pour toutes les fenêtres du processus, donc y poser un curseur
// vide le retirerait aussi à celles qui le veulent.
const (
	wmSetCursor = 0x0020
	htClient    = 1
)

// sizeMinimized est le `wParam` de `wmSize` qui annonce la réduction. C'est par lui
// que la minimisation se lit, et non en interrogeant la fenêtre à chaque image.
const sizeMinimized = 1

// pmRemove retire de la file le message que `PeekMessageW` rend.
const pmRemove = 0x0001

// dibRGBColors dit que les couleurs du DIB sont littérales, sans passer par une
// palette.
const dibRGBColors = 0

// biRGB dit que les composantes sont là où le système les suppose : le bleu dans
// l'octet de poids faible, puis le vert, le rouge et l'octet inutilisé.
//
// **C'est le seul des deux en-têtes que GDI sait recopier en bloc**, et l'écart est
// d'un facteur 47 : mesuré le 29 octobre 2026 sur une fenêtre réelle, la même
// présentation coûte 0,47 ms en `BI_RGB` et 22,1 ms en décrivant les masques du
// moteur, à 1920×1080. GDI accepte n'importe quel masque, mais n'a de chemin rapide
// que pour le sien — hors de là il convertit pixel par pixel, à 10,7 ms par
// mégapixel, ce que `SetDIBitsToDevice` ne signale par rien. La preuve est isolée :
// les mêmes `BI_BITFIELDS` avec les masques standard coûtent 0,475 ms, donc c'est la
// disposition qui est lente et non la façon de la décrire.
//
// La conversion est donc faite ici, où elle coûte 1,5 ms de plus, et c'est ce que
// [window.Present] explique.
const biRGB = 0

// newBitmapInfo décrit au système la disposition du tampon que la présentation lui
// donne.
//
// Les dimensions se posent à chaque présentation, le reste ne change jamais. La
// hauteur sera négative : le moteur compte ses lignes depuis le haut, là où un DIB les
// compte depuis le bas par défaut, et une hauteur négative est la façon de le dire.
func newBitmapInfo() bitmapInfoHeader {
	return bitmapInfoHeader{
		size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		planes:      1,
		bitCount:    32,
		compression: biRGB,
	}
}

// perMonitorAwareV2 est la poignée opaque de conscience de densité la plus complète.
//
// Sans elle, le système montre au processus un écran plus petit que le vrai et étire
// l'image pour remplir l'écran : le facteur entier serait calculé sur une taille
// fausse puis multiplié par une fraction, ce que l'échelle entière obligatoire existe
// pour interdire — et rien, dans les nombres que le moteur lit, ne le trahirait.
const perMonitorAwareV2 = ^uintptr(3)

// baseDPI est la densité que le système appelle cent pour cent : quatre-vingt-seize
// points par pouce, et c'est une convention de Windows et non une mesure.
const baseDPI = 96

// densityFactor rend par combien multiplier une taille de fenêtre pour qu'elle paraisse
// à la taille demandée sur l'écran courant.
//
// **Un facteur entier, comme tout le reste du moteur**, et l'arrondi va au plus proche
// avec les demis vers le haut : à 150 % une fenêtre reste donc à sa taille demandée en
// points plutôt que d'en faire les deux tiers. Trop petit est pire que trop grand à
// cette résolution — un jeu en 480×270 descendrait à 320×180 points, où l'on ne
// distingue plus un personnage de vingt-quatre pixels.
//
// **La densité se lit après avoir déclaré la conscience, jamais avant.** Mesuré le
// 29 octobre 2026 : le même appel rend 96 avant [perMonitorAwareV2] et 120 après, sur
// un écran réglé à 125 %. Lu trop tôt, il rendrait toujours cent pour cent et ce lot
// n'aurait aucun effet.
//
// Un échec rend un, donc la taille demandée telle quelle : `GetDpiForSystem` n'existe
// pas avant Windows 10 1607, et un moteur qui plante pour une question de confort
// d'affichage serait pire que le confort qu'il cherche. C'est aussi pourquoi la
// procédure se cherche au lieu de s'appeler, et [procAddr] dit ce que coûte l'oubli.
func densityFactor() int {
	addr, err := procAddr(getDpiForSystem)
	if err != nil {
		return 1
	}
	dpi, _, _ := syscall.SyscallN(addr)
	return factorForDPI(int(dpi))
}

// factorForDPI rend le facteur qu'une densité mérite, séparée de l'appel système pour
// qu'un cas de test puisse l'éprouver sur les échelles que Windows propose — aucune
// machine du parc n'étant réglée au-delà de 125 %.
func factorForDPI(dpi int) int {
	return max((dpi+baseDPI/2)/baseDPI, 1)
}

// rect est le rectangle que rendent `GetClientRect` et qu'attend `AdjustWindowRect`.
type rect struct {
	left, top, right, bottom int32
}

// point est la paire de coordonnées de Win32 : la position que portent les messages de
// souris, dont la pompe n'a pas l'usage mais que la structure d'un message contient, et
// les cinq tailles de [minMaxInfo].
type point struct {
	x, y int32
}

// minMaxInfo porte les bornes de taille et de position qu'une fenêtre accepte, et le
// système la passe par adresse dans le `lParam` de `wmGetMinMaxInfo`.
//
// Les cinq champs y sont pour la disposition seule : le système lit la structure
// entière au retour du message, donc en omettre un décalerait les suivants. Seul
// minTrack est écrit.
type minMaxInfo struct {
	reserved    point
	maxSize     point
	maxPosition point
	minTrack    point
	maxTrack    point
}

// message est ce que la pompe reçoit.
//
// Sa taille diffère entre les deux architectures — quarante-huit octets contre
// vingt-huit —, et c'est le compilateur qui le sait.
type message struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// windowClass est l'enregistrement par lequel le système connaît une classe de
// fenêtre : son style, sa procédure de messages et ses icônes.
type windowClass struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

// bitmapInfoHeader décrit au système l'image que la présentation lui donne.
//
// Elle ne porte aucun pointeur, donc sa taille est la même sur les deux
// architectures : le chemin de présentation n'a aucune précaution à prendre de ce
// côté.
type bitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

// newPresentArgs monte la liste d'arguments de la présentation, les cases qui ne
// changent jamais étant déjà posées.
//
// L'ordre est celui de l'appel : contexte, coin de destination, largeur et hauteur,
// coin de source, première ligne et nombre de lignes, pixels, description, usage des
// couleurs. Les cases laissées à zéro le restent — la destination et la source sont
// toujours à leur origine, puisque l'agrandissement et le centrage sont faits en
// amont par le rastériseur.
func newPresentArgs() []uintptr {
	args := make([]uintptr, 12)
	args[11] = dibRGBColors
	return args
}

// clientSize rend les dimensions de la zone utile, en pixels.
//
// Zéro quand la fenêtre est réduite, ce que l'appel système rend tel quel : c'est à
// l'appelant de ne pas en tirer un facteur d'agrandissement.
func clientSize(hwnd uintptr) (width, height uint16) {
	var r rect
	if ret, _, _ := getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ret == 0 {
		return 0, 0
	}
	return uint16(r.right - r.left), uint16(r.bottom - r.top)
}

// outerSize rend la taille de fenêtre qui donne la zone utile demandée.
//
// `CreateWindowExW` reçoit la taille de la fenêtre entière, bordures et barre de titre
// comprises : demander 480×270 donne une zone utile de 464×231 sur un thème ordinaire,
// et le facteur entier tomberait alors à un cran de moins que ce que l'écran permet.
// L'écart dépend du thème, donc il se calcule et ne s'écrit pas.
func outerSize(width, height uint16, style uintptr) (uint16, uint16) {
	r := rect{right: int32(width), bottom: int32(height)}
	if ret, _, _ := adjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), style, 0); ret == 0 {
		return width, height
	}
	return uint16(r.right - r.left), uint16(r.bottom - r.top)
}
