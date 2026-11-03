// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les constantes de `xdg-shell`, l'extension par laquelle une surface devient une fenêtre
// de bureau.
//
// Elle est relevée dans `xdg-shell.xml`, une seconde description que `wl_linux.go` ne
// porte pas : le protocole de base sait créer une surface et lui donner des pixels, il ne
// sait pas ce qu'est une fenêtre. C'est cette extension qui apporte le titre, la taille
// négociée et la demande de fermeture — et le compositeur l'annonce comme un global
// ordinaire, `xdg_wm_base`.

package backend

// Les opcodes de `xdg_wm_base`, version 1 : deux requêtes et un événement.
//
// `destroy` est à l'opcode 0 et `create_positioner` à 1, tous deux sans emploi : fermer la
// connexion détruit tout ce qu'elle porte, et le positionneur ouvre la voie aux menus que
// le moteur ne pose pas. Les trous de la numérotation sont donc ceux de la description.
const (
	wlXDGWMBaseGetXDGSurface uint16 = 2
	wlXDGWMBasePong          uint16 = 3

	wlXDGWMBasePing uint16 = 0
)

// wlXDGErrUnresponsive est le code par lequel un compositeur coupe un client muet.
//
// Sa description le dit sans détour — « the client didn't respond to a ping event in
// time ». Le protocole ne fixe ni le délai ni ce qui arrive sans réponse : « it's
// unspecified what will happen if the client doesn't respond to the ping request, or in
// what timeframe ». La parade n'est donc pas de répondre vite, c'est de répondre depuis le
// chemin que la boucle de jeu traverse à chaque pas — un `pong` différé jusqu'au prochain
// envoi est permis, un `pong` qu'aucun chemin n'émet ferme la fenêtre.
const wlXDGErrUnresponsive uint32 = 6

// Les opcodes de `xdg_surface`, version 1.
//
// `get_popup` est à l'opcode 2 et `set_window_geometry` à 3, tous deux sans emploi ici :
// le moteur n'ouvre pas de menu, et la géométrie par défaut — la boîte englobante de la
// surface — est exactement ce qu'une fenêtre de jeu veut.
const (
	wlXDGSurfaceGetToplevel  uint16 = 1
	wlXDGSurfaceAckConfigure uint16 = 4

	wlXDGSurfaceConfigure uint16 = 0
)

// Les codes d'erreur de `xdg_surface` qu'un défaut de montage produit, de type `uint32`.
//
// Ils sont énumérés parce que `go doc` abrège un bloc de constantes après la première, et
// parce que ce sont les trois seuls qu'une faute de notre côté puisse lever :
// [wlXDGErrNotConstructed] pour une requête émise avant que le rôle soit donné,
// [wlXDGErrUnconfiguredBuffer] pour un tampon attaché avant le premier `configure`,
// [wlXDGErrInvalidSerial] pour un numéro de série acquitté sans avoir été émis.
//
// **Le second est celui que l'ordre de montage protège**, et sa description nomme
// exactement le geste interdit : « attaching a buffer to an unconfigured surface ».
// L'énumération ne commence pas à zéro, la description partant de un.
const (
	wlXDGErrNotConstructed     uint32 = 1
	wlXDGErrUnconfiguredBuffer uint32 = 3
	wlXDGErrInvalidSerial      uint32 = 4
)

// Les opcodes de `xdg_toplevel`, version 1.
//
// Le moteur n'emploie que deux requêtes sur quatorze, et ce qu'il laisse relève du
// gestionnaire de fenêtres ou du joueur — `move`, `resize`, `set_maximized`,
// `set_fullscreen`, `set_minimized`.
//
// **`set_min_size` en fait partie, et c'est un choix plutôt qu'un oubli** : aucune des
// familles livrées ne déclare de taille minimale, la surface de dessin suivant la fenêtre
// depuis la 0.5.4. Une fenêtre réduite sous la résolution interne est donc un état
// légitime et non un état à interdire, et le déclarer ici ferait de Wayland la seule
// famille où le joueur ne peut pas rétrécir.
const (
	wlXDGToplevelSetTitle uint16 = 2
	wlXDGToplevelSetAppID uint16 = 3

	wlXDGToplevelConfigure uint16 = 0
	wlXDGToplevelClose     uint16 = 1
)
