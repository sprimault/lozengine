// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les constantes du protocole Wayland : les objets qu'on ne choisit pas, les opcodes, et
// les bornes que la spécification impose au client.
//
// Tout ce qui est ici est relevé dans `wayland.xml`, qui est la description du protocole,
// et dans son chapitre sur le format du fil. Chaque bloc nomme l'interface et la version
// dont il vient — jamais un chemin de fichier ni un numéro de ligne, qui ne se
// reproduisent pas d'une distribution à l'autre.

package backend

// wlDisplayID est l'identifiant de l'objet que le serveur accorde d'office.
//
// « ID 1 is automatically assigned to the singleton wl_display instance when a client
// connects » : il n'est donc ni demandé ni annoncé, et c'est le seul objet dont un client
// connaisse l'identifiant avant d'avoir parlé.
const wlDisplayID = 1

// Les bornes de l'espace d'identifiants, telles que la spécification du fil les donne.
//
// **La contrainte qui surprend n'est pas la borne, c'est la densité** : « the IDs are
// densely packed in the sense that the ID N will not be used until N-1 has been used »,
// et le texte la qualifie d'exigence stricte. On ne peut donc pas sauter un identifiant,
// même pour aligner quelque chose ou pour réserver une plage. Réutiliser un identifiant
// que `wl_display.delete_id` a rendu reste permis, puisque tous ceux qui le précèdent ont
// déjà servi.
//
// Zéro désigne l'objet nul, et la plage haute appartient au serveur : « IDs allocated by
// the server are in the range [0xff000000, 0xffffffff] ».
const (
	wlFirstID uint32 = 2
	wlLastID  uint32 = 0xfeffffff
)

// Les opcodes des requêtes de `wl_display`, version 1.
const (
	wlDisplaySync        uint16 = 0
	wlDisplayGetRegistry uint16 = 1
)

// Les opcodes des événements de `wl_display`, version 1.
const (
	wlDisplayError    uint16 = 0
	wlDisplayDeleteID uint16 = 1
)

// Les codes de l'événement `wl_display.error`, de type `uint32`.
//
// Ils sont énumérés parce que `go doc` abrège un bloc de constantes après la première, et
// parce qu'une erreur de protocole est ce qu'on lit en premier quand un backend tombe :
// [wlErrInvalidObject] pour un objet que le serveur n'a pas trouvé, [wlErrInvalidMethod]
// pour une méthode absente de l'interface ou une requête mal formée, [wlErrNoMemory] pour
// un serveur à court de mémoire, [wlErrImplementation] pour un défaut du compositeur.
//
// **La deuxième est celle qu'un défaut de codage produit**, et son libellé le dit —
// « malformed request » —, donc un message mal composé se présente comme une méthode qui
// n'existe pas.
const (
	wlErrInvalidObject  uint32 = 0
	wlErrInvalidMethod  uint32 = 1
	wlErrNoMemory       uint32 = 2
	wlErrImplementation uint32 = 3
)

// Les opcodes de `wl_registry`, version 1 : une requête et deux événements.
const (
	wlRegistryBind uint16 = 0

	wlRegistryGlobal       uint16 = 0
	wlRegistryGlobalRemove uint16 = 1
)

// wlCompositorCreateSurface est l'opcode de la requête qui crée une surface.
//
// `create_region` est la seconde et dernière requête de l'interface, à l'opcode 1 : elle
// sert à décrire une zone opaque ou sensible au pointeur, ce qu'une fenêtre de jeu peut
// laisser au défaut — toute la surface, et rien d'opaque déclaré.
const wlCompositorCreateSurface uint16 = 0

// Les opcodes de `wl_surface`, à la version que `wl_compositor` a accordée.
//
// **Trois d'entre eux ne sont pas à la version 1, et c'est ce qui décide de leur emploi** :
// `set_buffer_transform` arrive en 2, `set_buffer_scale` en 3, `damage_buffer` en 4. Sous
// une version plus basse, la parade n'est pas de les éviter un par un mais de ne pas
// changer d'échelle : la surface et le tampon partagent alors leurs coordonnées, et
// `damage` dit la même chose que `damage_buffer`.
const (
	wlSurfaceDestroy      uint16 = 0
	wlSurfaceAttach       uint16 = 1
	wlSurfaceDamage       uint16 = 2
	wlSurfaceFrame        uint16 = 3
	wlSurfaceCommit       uint16 = 6
	wlSurfaceSetScale     uint16 = 8
	wlSurfaceDamageBuffer uint16 = 9

	wlSurfaceEnter                    uint16 = 0
	wlSurfaceLeave                    uint16 = 1
	wlSurfacePreferredBufferScale     uint16 = 2
	wlSurfacePreferredBufferTransform uint16 = 3
)

// wlShmFormat est l'opcode de l'événement par lequel `wl_shm` annonce un format, et il
// en envoie un par format qu'il accepte.
const wlShmFormat uint16 = 0

// Les deux valeurs de `wl_shm.format` qui ne sont pas des codes à quatre caractères.
//
// **C'est une exception historique, et elle vaut sur le seul format que le protocole
// garantisse.** La description donne `argb8888` à zéro et `xrgb8888` à un, là que tous les
// autres portent leur code — `xbgr8888` vaut `0x34324258`, qui est `XB24`. Un client qui
// passerait une valeur de `wl_shm.format` au constructeur commun chercherait donc
// `0x34325258` et ne reconnaîtrait pas le format de repli, celui dont il a le plus besoin.
//
// La traduction vit ici et non dans [pixelOrderOfFourCC], parce que ces deux nombres ne
// veulent rien dire ailleurs : sous DRM/KMS, zéro n'est pas un format, c'est l'absence de
// format.
const (
	wlShmARGB8888 uint32 = 0
	wlShmXRGB8888 uint32 = 1
)

// wlCallbackDone est l'opcode de l'unique événement de `wl_callback`, version 1.
//
// La description le marque `destructor` : l'objet disparaît en le rendant, donc son
// identifiant revient par `wl_display.delete_id` et ne se réutilise pas avant.
const wlCallbackDone uint16 = 0

// wlErrorName nomme un code d'erreur du protocole, ou le rend tel quel quand il n'est pas
// de ceux que la version 1 définit.
//
// Un compositeur peut émettre un code qu'on ne connaît pas : le message doit alors porter
// le nombre plutôt que de taire ce qu'il n'a pas su nommer.
func wlErrorName(code uint32) string {
	switch code {
	case wlErrInvalidObject:
		return "invalid_object"
	case wlErrInvalidMethod:
		return "invalid_method"
	case wlErrNoMemory:
		return "no_memory"
	case wlErrImplementation:
		return "implementation"
	default:
		return "unknown"
	}
}
