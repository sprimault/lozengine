// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le backend emprunte au protocole X11 : les opcodes, les masques, les numéros
// d'événement, et rien qui décide de quoi que ce soit.
//
// Tout vient de la spécification du protocole X.Org. Un opcode, un numéro d'événement
// ou la place d'un champ dans une structure sont des faits d'interface, et les relever
// dans une spécification est ce que le projet autorise ; une implémentation ne se
// reprend pas. Le seul fait du backend qui ne soit écrit dans aucune spécification
// est le format du fichier d'autorisation, et il est signalé là où il est lu.

package backend

import (
	"encoding/binary"
	"strconv"
)

// La version du protocole que le client annonce. Elle n'a pas bougé depuis 1987, et un
// serveur qui en parlerait une autre le dirait dans sa réponse au handshake.
const (
	protocolMajor = 11
	protocolMinor = 0
)

// order est l'ordre dans lequel tous les entiers de la connexion se lisent et
// s'écrivent : celui de la machine, puisque c'est celui que le handshake annonce.
//
// Le protocole laisse le client choisir, et le serveur s'y conforme dans les deux sens
// — c'est ce qui épargne une conversion par champ. L'ordre des octets d'un *pixel* est
// une autre question, que le serveur tranche seul.
//
// `binary.NativeEndian` plutôt qu'un choix par `GOARCH` : la bibliothèque standard
// connaît déjà la réponse, et la déduire d'un nom d'architecture est la façon de se
// tromper sur celle qu'on n'avait pas prévue.
var order = binary.NativeEndian

// Les deux valeurs de l'octet d'ordre, en tête du handshake.
const (
	orderLSBFirst = 'l'
	orderMSBFirst = 'B'
)

// handshakeOrder rend l'octet d'ordre que le client annonce, déduit de [order].
func handshakeOrder() byte {
	var b [2]byte
	order.PutUint16(b[:], 1)
	if b[0] == 1 {
		return orderLSBFirst
	}
	return orderMSBFirst
}

// Les trois réponses possibles du serveur au handshake.
const (
	setupFailed       = 0
	setupSuccess      = 1
	setupAuthenticate = 2
)

// eventSize est la taille de tout message venant du serveur, événement comme erreur.
//
// Une réponse de requête commence par ces trente-deux octets et peut les dépasser, sa
// longueur étant portée par un champ. Rien d'autre ne les dépasse jamais, ce qui est
// ce qui permet de découper le flux sans connaître la requête qui l'a provoqué.
const eventSize = 32

// Le premier octet d'un message reçu dit lequel des trois il est : une erreur, une
// réponse, ou un événement dont le numéro vaut deux ou plus.
const (
	msgError = 0
	msgReply = 1
)

// Les événements que la fenêtre traite. Le masque d'événements de la création décide
// lesquels arrivent, et ceux qui ne sont pas sélectionnés ne viennent pas.
const (
	eventKeyPress        = 2
	eventKeyRelease      = 3
	eventButtonPress     = 4
	eventButtonRelease   = 5
	eventMotionNotify    = 6
	eventEnterNotify     = 7
	eventLeaveNotify     = 8
	eventFocusIn         = 9
	eventFocusOut        = 10
	eventKeymapNotify    = 11
	eventDestroyNotify   = 17
	eventUnmapNotify     = 18
	eventMapNotify       = 19
	eventConfigureNotify = 22
	eventClientMessage   = 33
)

// eventSentBit dit qu'un autre client a fabriqué l'événement par `SendEvent` au lieu
// que le serveur l'ait produit.
//
// Il se masque avant toute comparaison de numéro, et l'oublier coûte précisément ce
// que le lot cherche à livrer : la demande de fermeture arrive *toujours* ainsi,
// puisque c'est le gestionnaire de fenêtres qui l'envoie. Le bit resterait mis, le
// numéro ne correspondrait à rien, et la croix ne fermerait pas la fenêtre.
const eventSentBit = 0x80

// Les requêtes que le backend envoie, par leur opcode majeur.
const (
	opCreateWindow   = 1
	opDestroyWindow  = 4
	opMapWindow      = 8
	opInternAtom     = 16
	opChangeProperty = 18
	opCreateGC       = 55
	opPutImage       = 72
	opGetImage       = 73
	opQueryExtension = 98

	// opGetInputFocus ne sert pas à ce qu'elle nomme : c'est la requête à réponse la
	// plus courte du protocole, et on l'emploie pour savoir que le serveur a traité
	// tout ce qui la précédait. Sa réponse ne peut pas arriver avant les erreurs des
	// requêtes antérieures, le serveur traitant un client dans l'ordre reçu.
	opGetInputFocus = 43
)

// copyFromParent vaut zéro pour la profondeur, le visuel et la classe d'une fenêtre :
// c'est ainsi qu'on hérite de la racine au lieu de choisir.
//
// Le brief du jalon l'a tranché : profondeur 24 et visuel de la racine, donc pas de
// table de couleurs à créer. Un visuel de profondeur 32 existe sous un compositeur et
// son quatrième octet est un alpha que celui-ci honore — il n'achète rien ici, et il
// exposerait à une fenêtre translucide si une scène laissait un alpha partiel.
const copyFromParent = 0

// classInputOutput est la classe d'une fenêtre qui peut afficher quelque chose, par
// opposition à celle qui ne reçoit que des entrées.
const classInputOutput = 1

// Les valeurs que la création de fenêtre pose, et leur bit dans le masque.
//
// **La liste s'écrit dans l'ordre croissant des bits**, et s'y tromper ne lève aucune
// erreur : le serveur lirait le masque d'événements comme une couleur de fond. Cela
// donne une fenêtre colorée qui ne reçoit aucun événement, c'est-à-dire « le clavier
// ne marche pas » — un symptôme qu'on va chercher partout ailleurs.
const (
	cwBackPixel = 0x0002
	cwEventMask = 0x0800
)

// Les événements que la fenêtre demande à recevoir.
//
// Seuls ceux qu'elle traite : un événement sélectionné et ignoré traverse la socket et
// la file pour être jeté. `Exposure` n'y est donc pas, le moteur repeignant à chaque
// image de toute façon, et le pointeur non plus.
//
// Les trois du clavier sont demandés dès la création bien que le décodage vienne avec
// le lot qui lit le clavier : le masque ne se pose qu'une fois, à la création, et le
// reprendre ensuite demanderait une requête de plus pour rien.
//
// **maskKeymapState porte un piège pour ce lot-là**, et il vaut mieux le lire ici
// qu'ailleurs : l'événement qu'il fait venir est le seul du protocole **sans numéro de
// séquence**, ses deux octets de séquence portant déjà de l'état de clavier. Tout ce
// qui lit ce numéro doit donc l'exclure. Ce qu'il achète en échange est considérable —
// le serveur renvoie l'état complet du clavier après chaque prise de focus, donc la
// resynchronisation arrive seule, là où Win32 demande de relire les touches une par
// une.
const (
	maskKeyPress        = 1 << 0
	maskKeyRelease      = 1 << 1
	maskButtonPress     = 1 << 2
	maskButtonRelease   = 1 << 3
	maskEnterWindow     = 1 << 4
	maskLeaveWindow     = 1 << 5
	maskPointerMotion   = 1 << 6
	maskKeymapState     = 1 << 14
	maskStructureNotify = 1 << 17
	maskFocusChange     = 1 << 21
)

// Les atomes que la spécification numérote d'avance.
//
// Il n'y a pas à les demander au serveur, ce qui épargne un aller-retour par propriété
// posée. Tout le reste s'obtient par `InternAtom`, dont le prix est le même quelle que
// soit la propriété.
const (
	atomAtom          = 4
	atomCardinal      = 6
	atomString        = 31
	atomWMName        = 39
	atomWMNormalHints = 40
	atomWMSizeHints   = 41
)

// propReplace remplace la valeur d'une propriété, par opposition à ce qui s'y
// ajouterait devant ou derrière.
const propReplace = 0

// Ce que `WM_NORMAL_HINTS` porte : dix-huit mots de trente-deux bits, dont le premier
// dit lesquels des autres sont renseignés.
//
// Le backend n'en renseigne qu'un, la taille minimale, et laisse le reste à zéro. Les
// dix-sept autres mots partent quand même : la structure est de taille fixe, et un
// gestionnaire de fenêtres qui en recevrait moins ignorerait la propriété entière.
const (
	sizeHintsWords    = 18
	sizeHintsMinSize  = 1 << 4
	sizeHintsMinIndex = 5
)

// planeMaskAll demande toutes les couches de bits d'une image, par opposition à une
// seule — ce qui n'a de sens que sur un affichage par plans, que le moteur n'emploie
// pas.
const planeMaskAll = 0xFFFFFFFF

// zPixmap est le format d'image où les composantes d'un pixel sont contiguës, par
// opposition aux formats par plan de bits que le protocole porte aussi.
//
// C'est le seul que le moteur emploie : son tampon est fait de pixels et non de plans.
// En zPixmap, le bourrage à gauche doit valoir zéro et la profondeur doit égaler celle
// de la fenêtre, sous peine d'une erreur de correspondance.
const zPixmap = 2

// Les deux valeurs d'`image-byte-order`, qui dit dans quel sens le serveur veut les
// octets d'un pixel.
//
// Il ne suit pas l'ordre de la connexion, que le client a choisi : les images « sont
// toujours transmises et reçues dans les formats, ordre des octets compris, que le
// serveur spécifie ».
const (
	lsbFirst = 0
	msbFirst = 1
)

// Ce que `BIG-REQUESTS` demande et ce qu'elle lève.
//
// Sans elle, une requête ne peut pas dépasser quatre fois ce que seize bits portent,
// soit 256 Kio : une image 1920×1080 part alors en une trentaine de morceaux. Avec,
// une requête dont les seize bits de longueur valent **zéro** porte sa longueur sur
// trente-deux bits juste après — donc le découpage disparaît, mais le format d'une
// requête change, et seulement pour celles qui dépassent l'ancienne borne.
const (
	bigRequestsName   = "BIG-REQUESTS"
	bigRequestsEnable = 0

	// requestWordsMax est le plus grand nombre de mots qu'une longueur de seize bits
	// porte. Au-delà, la forme étendue est obligatoire.
	requestWordsMax = 65535
)

// Ce que l'extension du clavier sert à demander ici, et rien de plus.
//
// Un seul drapeau nous intéresse : celui qui fait cesser les relâchements que
// l'auto-répétition fabrique. La spécification prévient qu'il n'est pas garanti, d'où
// le champ de la réponse qu'il faut lire au lieu de supposer.
const (
	xkbName           = "XKEYBOARD"
	xkbUseExtension   = 0
	xkbPerClientFlags = 21

	// La version que le client annonce. Le dialogue de version est obligatoire avant
	// toute autre requête de l'extension, qui refuse sinon.
	xkbMajorVersion = 1
	xkbMinorVersion = 0

	// xkbCoreKeyboard désigne le clavier que le serveur présente à tous ses clients,
	// par opposition à un périphérique nommé.
	xkbCoreKeyboard = 0x0100

	// xkbDetectableAutoRepeat fait que le serveur n'envoie plus qu'un relâchement,
	// à la fin, au lieu d'un entre chaque répétition.
	xkbDetectableAutoRepeat = 1 << 0
)

// errorNames nomme les erreurs du protocole par leur code.
//
// Le nom entre dans le message que rend [Window.Err], et il porte beaucoup plus que le
// code : « Match » dit une incohérence entre une requête et son objet, « Window » un
// identifiant qui ne désigne rien. La liste est close depuis la version 11, et un code
// hors liste ne peut venir que d'une extension — d'où le repli plutôt qu'un plantage.
var errorNames = [...]string{
	1:  "Request",
	2:  "Value",
	3:  "Window",
	4:  "Pixmap",
	5:  "Atom",
	6:  "Cursor",
	7:  "Font",
	8:  "Match",
	9:  "Drawable",
	10: "Access",
	11: "Alloc",
	12: "Colormap",
	13: "GContext",
	14: "IDChoice",
	15: "Name",
	16: "Length",
	17: "Implementation",
}

// errorName nomme le code d'une erreur du protocole, ou le rend tel quel quand il
// vient d'une extension.
func errorName(code byte) string {
	if int(code) < len(errorNames) && errorNames[code] != "" {
		return errorNames[code]
	}
	return "extension error " + strconv.Itoa(int(code))
}

// pad rend le nombre d'octets de bourrage qui portent n à un multiple de quatre.
//
// Le protocole aligne ainsi chaque chaîne et chaque requête, et la longueur qu'une
// requête déclare se compte en mots de quatre octets : un bourrage omis décale tout ce
// qui suit sur la connexion, et l'erreur se rapporte alors sur une autre requête.
func pad(n int) int {
	return (4 - n%4) % 4
}

// itoa écrit un entier positif en décimal, sans passer par `strconv`.
//
// Le paquet est déjà atteint par ailleurs et l'emprunter ne coûterait rien ; ce qui
// décide est ailleurs — un code d'erreur et un numéro d'écran sont les deux seuls
// nombres que le backend met dans un texte, et les écrire ici évite de lire une
// fonction de formatage générale pour savoir ce qu'ils deviennent.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
