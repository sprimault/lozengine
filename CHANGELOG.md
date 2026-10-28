# Lozengine · Journal des modifications · Changelog

Format [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/), versions
[SemVer](https://semver.org/lang/fr/). Chaque version porte ses entrées en français puis
en anglais : les notes de publication reprennent la section telle quelle.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), versioning:
[SemVer](https://semver.org/). Every version carries its entries in French then in
English: release notes reuse the section as is.

Tant que le jalon 9 de la [feuille de route](ROADMAP.md) n'est pas atteint, la version
reste en `0.x` et les signatures publiques peuvent changer à chaque version mineure. Le
gel de l'API et le passage en `1.0.0` interviennent après validation du moteur par un
second jeu.

**Le mineur s'incrémente à chaque jalon franchi, sans en porter le numéro** : le titre
de chaque section dit de quel jalon elle vient. Les deux suites n'ont pas à coïncider,
et vouloir les aligner imposerait soit un `v0.0.0`, que Go confond avec l'absence de
version, soit une renumérotation de la feuille de route que l'historique ne suivrait
pas.

Until milestone 9 of the [roadmap](ROADMAP.md) (in French) is reached, the version
stays in `0.x` and public signatures may change with every minor release. The API
freeze and the move to `1.0.0` happen once the engine has been validated by a
second game.

**The minor is bumped at every milestone reached, without carrying its number**: each
section's title says which milestone it comes from. The two sequences need not
coincide, and aligning them would impose either a `v0.0.0`, which Go mistakes for the
absence of a version, or a renumbering of the roadmap that history would not follow.

<!--
Catégories : Ajouté, Modifié, Déprécié, Retiré, Corrigé, Sécurité.
Categories: Added, Changed, Deprecated, Removed, Fixed, Security.

Une entrée par changement visible depuis l'extérieur du moteur, écrite dans les deux
langues sous la même version, le bloc français puis le bloc anglais séparés par une
ligne de trois astérisques. Les remaniements internes sans effet sur l'API, les tests
et l'outillage n'y figurent pas.

Ce commentaire reste au-dessus des versions : `make notes` extrait une section
jusqu'au titre suivant, donc tout ce qui traîne après la dernière entrerait dans les
notes de publication.
-->

## [Non publié]

### Ajouté

- `Aspect.AnchorX` et `AnchorY` décalent l'image autour du point qu'elle vise, en
  pixels. De quoi recaler une planche dont le pied n'est pas au milieu, sans la
  redécouper.
- `Provider` résout ce qu'une chose montre, de ce qu'elle est, de son état, de sa
  direction et du moment de son animation. Le jeu le consulte quand une apparence
  change ; `Turn` accorde une direction avec l'orientation de la caméra.
- `Sheets` est le fournisseur de référence : des planches dessinées, en colonnes de
  directions et en lignes d'images. Huit directions, et c'est lui seul qui le déclare.
- `Clip` dit la cadence d'une animation et rend son image par `Loop` ou par `Once`.
  Rien n'est gardé d'une image à la suivante : tout se dérive du numéro de pas.
- `Aspect.Reveal` redessine une chose au-dessus de ce qui la cache, dans la couche
  qu'elle nomme. `SceneOptions.RevealAfter` dit à partir de combien de pixels cachés,
  et `RevealTint` sous quelle teinte — c'est la chose qu'on révèle, pas le décor
  qu'on efface.
- La souris est lue sur les deux plateformes : `Input.Pointer` rend sa position **en
  pixels de l'image**, la boucle défaisant l'agrandissement et les bandes.
  `ButtonHeld` et `ButtonJustPressed` ont les deux régimes des touches, et `Wheel`
  compte les crans du pas.
- `Scene.At` rend la case qu'un pixel désigne, à l'étage demandé : c'est le pixel que
  `Input.Pointer` donne, sans conversion à faire. `Options.HideCursor` retire le
  curseur du système pour un jeu qui dessine le sien.

### Modifié

- L'étage élève désormais à l'écran en plus de ranger, pour une cellule de décor comme
  pour une entité : un décor à deux niveaux se décrit. `z` est la hauteur au-dessus de
  son étage, et porter les deux élève deux fois.

***

### Added

- `Aspect.AnchorX` and `AnchorY` offset the image around the point it targets, in
  pixels. Enough to realign a sheet whose foot is off-centre, without recutting it.
- `Provider` resolves what a thing shows, from what it is, its state, its facing and
  the moment of its animation. The game consults it when an appearance changes; `Turn`
  matches a facing with the camera orientation.
- `Sheets` is the reference provider: drawn sheets, with facings as columns and frames
  as rows. Eight facings, and it alone declares that number.
- `Clip` states the pace of an animation and yields its frame through `Loop` or
  `Once`. Nothing is kept between frames: everything derives from the step number.
- `Aspect.Reveal` redraws a thing above whatever hides it, in the layer it names.
  `SceneOptions.RevealAfter` says from how many hidden pixels, and `RevealTint` under
  which tint — it is the thing that gets revealed, not the decor that gets erased.
- The mouse is read on both platforms: `Input.Pointer` gives its position **in image
  pixels**, the loop undoing the upscale and the bars. `ButtonHeld` and
  `ButtonJustPressed` have the two regimes of keys, and `Wheel` counts the step's
  notches.
- `Scene.At` gives the cell a pixel points at, on the requested floor: that pixel is
  what `Input.Pointer` returns, with no conversion to do. `Options.HideCursor` removes
  the system cursor for a game that draws its own.

### Changed

- The floor now raises on screen as well as ranks, for a decor cell as for an entity:
  a two-level decor can be described. `z` is the height above its own floor, and
  carrying both raises twice.

## [0.4.0] — 2026-10-28 — Jalon 3 · Backend Linux par X11

Un jeu s'écrit sous Windows et sous Linux. Le navigateur est le jalon 5 : ailleurs le
moteur se compile et calcule ses images, et rien ne les affiche.

### Ajouté

- Linux est pris en charge, par le protocole X11 écrit à la main : le moteur y ouvre
  une fenêtre, l'affiche, lit le clavier et porte son icône. Aucune bibliothèque C,
  comme sous Windows.

### Corrigé

- La fenêtre ne se réduit plus sous la taille de l'image. En dessous, elle se
  découpait : les bords sortaient de la vue sans que rien ne le dise.

***

A game can be written on Windows and on Linux. The browser is milestone 5: elsewhere
the engine builds and computes its frames, and nothing displays them.

### Added

- Linux is supported, through the X11 protocol written by hand: the engine opens a
  window there, displays it, reads the keyboard and carries its icon. No C library,
  just as on Windows.

### Fixed

- The window no longer shrinks below the size of the image. Under it, the image was
  cropped: its edges left the view with nothing to say so.

## [0.3.2] — 2026-10-26

### Ajouté

- `Quitter` est ce qu'un jeu écrit en plus de `Game` quand il veut pouvoir partir de
  lui-même : une entrée de menu, une touche d'échappement. Sans elle, seule la
  fermeture de la fenêtre arrêtait la boucle.
- `MaxSheets` se lit depuis le paquet racine, comme les autres plafonds : c'est le
  nombre de planches qu'un atlas porte, et `Atlas.Add` refuse au-delà.

### Corrigé

- Une entité passe devant la cellule qu'elle foule, où qu'elle se place dedans. Dans la
  moitié nord d'une case, le sol lui passait sur les pieds.
- La fenêtre impose la flèche du système comme curseur. Elle gardait la forme que le
  pointeur avait en y entrant, dont la double flèche de redimensionnement.

***

### Added

- `Quitter` is what a game writes in addition to `Game` when it wants to leave of its
  own accord: a menu entry, an escape key. Without it, only closing the window stopped
  the loop.
- `MaxSheets` can be read from the root package, like the other ceilings: it is how
  many sheets an atlas holds, and `Atlas.Add` refuses beyond it.

### Fixed

- An entity draws in front of the cell it treads, wherever it stands inside it. In the
  northern half of a cell, the ground painted over its feet.
- The window sets the system arrow as its cursor. It kept whatever shape the pointer
  had on entering, the resize arrow included.

## [0.3.1] — 2026-10-26

La fenêtre n'existe toujours que sous Windows : Linux par X11 est le jalon 3, le
navigateur le jalon 5. Ailleurs le moteur calcule ses images, et rien ne les affiche.

### Ajouté

- Le pavé numérique se nomme : `KeyNumpad0` à `KeyNumpad9`, le séparateur décimal, les
  quatre opérateurs, son Entrée et `KeyNumLock`. Ce sont des places à part, lisibles
  quel que soit l'état du verrou numérique.
- `Options.Icons` porte l'icône de la fenêtre, une planche par taille. Le moteur retient
  la plus proche de ce que le système demande et n'en redimensionne aucune ; vide,
  la fenêtre garde l'icône du système.
- Le drapeau `Additive` ajoute un quad au fond au lieu de l'y composer, chaque
  composante plafonnant à sa valeur pleine. C'est le mélange de ce qui émet de la
  lumière, et combiné à `Flat` il donne un fondu lumineux.
- `Premultiply` construit une couleur depuis les composantes séparées qu'affiche un
  éditeur d'images, et `Color.Valid` dit si une teinte écrite à la main est cohérente.
  C'est la porte d'une couleur isolée ; pour une planche entière,
  `NewSheetFromStraight`.
- `Color.Scale` atténue une couleur, ses quatre composantes ensemble : c'est ce que
  demande un éclat qui s'éteint, et amputer les trois couleurs seules assombrirait au
  lieu d'estomper. La fraction est entière, donc l'atténuation est la même partout.
- `NewSheetFromStraight` monte une planche depuis des octets RVBA dont l'alpha n'est
  pas encore appliqué, pour qui tient ses pixels ailleurs que dans un PNG. La
  conversion a lieu en entrant.

### Corrigé

- Une touche tenue au moment où la fenêtre prend le focus ne pose plus une seconde
  place à tort. Onze codes de balayage en désignent deux, et celle qui était posée par
  erreur ne recevait jamais son relâchement : elle restait tenue pour de bon.

***

The window still exists on Windows only: Linux through X11 is milestone 3, the browser
milestone 5. Elsewhere the engine computes its frames, and nothing displays them.

### Added

- The numeric keypad has names: `KeyNumpad0` to `KeyNumpad9`, the decimal separator,
  the four operators, its Enter key and `KeyNumLock`. These are places of their own,
  readable whatever Num Lock is set to.
- `Options.Icons` carries the window icon, one sheet per size. The engine keeps the one
  closest to what the system asks for and resizes none of them; left empty, the window
  keeps the system icon.
- The `Additive` flag adds a quad to the background instead of compositing over it,
  each channel capping at its full value. It is the blend for anything that emits
  light, and combined with `Flat` it gives a luminous fade.
- `Premultiply` builds a colour from the separate channels an image editor shows, and
  `Color.Valid` states whether a hand-written tint is coherent. That is the door for a
  single colour; for a whole sheet, `NewSheetFromStraight`.
- `Color.Scale` dims a colour, all four channels together: that is what a fading
  sparkle needs, and cutting the three colour channels alone would darken rather than
  fade. The fraction is an integer, so the dimming is the same everywhere.
- `NewSheetFromStraight` builds a sheet from RGBA bytes whose alpha is not applied
  yet, for whoever holds their pixels somewhere other than a PNG. The conversion
  happens on the way in.

### Fixed

- A key held as the window takes focus no longer sets a second place by mistake.
  Eleven scan codes name two places, and the one set in error never received its
  release: it stayed held for good.

## [0.3.0] — 2026-10-25 — Jalon 2 · Backend Windows

Un jeu s'écrit, mais sous Windows seulement : Linux par X11 est le jalon 3, le navigateur
le jalon 5. Ailleurs le moteur se compile et calcule ses images, et rien ne les affiche.

### Ajouté

- `Input` dit ce que le clavier tient à une image et ce qui vient d'être enfoncé, par
  `Held` et `JustPressed`. Une `Key` désigne une place, pas un caractère.
- Les planches se chargent depuis le paquet racine : `OpenSheet` par chemin,
  `LoadSheet` par lecteur, plus `Sheet`, `Atlas`, `Rectangle` et `NewSheet`.
- `geometry`, `scene` et `raster` sont publics : les champs et les méthodes de
  `Matrix`, `Scene`, `Entity`, `Aspect` et `Sheet` se lisent dans leur
  documentation.
- `MaxSide` et `MaxTiers` disent la plus grande grille possible et le nombre de
  choses qu'une cellule empile.
- La scène se décrit depuis le paquet racine : `NewScene`, `Scene`, `Entity`,
  `Aspect`, `Handle`, `Orientation` et ses quatre quarts de tour, `SceneOptions`.
- `MaxLayer` et `MaxFloor` disent combien de couches et d'étages l'ordre de dessin
  distingue. Une pose qui les dépasse est refusée, au lieu d'être ramenée au plafond.
- `Buffer` est l'image dans laquelle le moteur dessine, et un hôte peut l'allouer
  lui-même : `NewBuffer` et `MaxBufferSide` viennent avec.
- `windows/386` entre dans les cibles vérifiées à la compilation, la famille Windows
  étant annoncée sur ses trois architectures. Cette cible **compile**, rien de plus.
- Les touches se nomment : lettres, chiffres, flèches, modificateurs distingués à
  gauche et à droite, fonctions et navigation. Un nom désigne une place, donc `KeyW`
  reste la touche au-dessus de `KeyS` quelle que soit la disposition du joueur.
- **Un jeu s'écrit** : `Run` ouvre une fenêtre et tient la boucle à pas fixe, le jeu
  n'écrivant qu'une méthode, `Game.Step`. `Options` porte le titre, la résolution
  interne, le pas, le fond et l'atlas, et tout sauf l'atlas a un défaut.
- `Engine` est la même capacité sans fenêtre ni horloge, pour un hôte qui garde sa
  boucle : `NewEngine`, puis un `Step` par image et `Frame` pour l'image obtenue.
- `State` fabrique des entrées sans clavier, de quoi éprouver la méthode d'un jeu sans
  ouvrir de fenêtre.
- `Options.Bars` est la couleur des bandes qui entourent l'image quand la fenêtre ne
  tombe pas sur un multiple exact de la résolution interne. Elle vaut le noir opaque et
  ne suit donc pas le fond du monde, qu'un jeu peut vouloir clair.

### Modifié

- **Les paquets et les identifiants passent à l'anglais, chemins d'import compris** :
  `lozengine/rendu` devient `lozengine/render`, `geometrie` devient `geometry`. Un jeu
  écrit pour la `0.2.0` ne compile plus sans renommer ses imports et ses appels.
- Les messages d'erreur sont en anglais. La godoc et les commentaires restent en
  français.

***

### Added

- `Input` states which keys a frame holds and which were just pressed, through `Held`
  and `JustPressed`. A `Key` names a place, not a character.
- Sheets are loaded from the root package: `OpenSheet` by path, `LoadSheet` by reader,
  plus `Sheet`, `Atlas`, `Rectangle` and `NewSheet`.
- `geometry`, `scene` and `raster` are public: the fields and methods of `Matrix`,
  `Scene`, `Entity`, `Aspect` and `Sheet` can be read in their documentation.
- `MaxSide` and `MaxTiers` state the largest possible grid and how many things a
  cell stacks.
- A scene is described from the root package: `NewScene`, `Scene`, `Entity`,
  `Aspect`, `Handle`, `Orientation` and its four quarter turns, `SceneOptions`.
- `MaxLayer` and `MaxFloor` state how many layers and floors the draw order tells
  apart. A placement beyond them is refused rather than clamped to the ceiling.
- `Buffer` is the image the engine draws into, and a host may allocate it itself:
  `NewBuffer` and `MaxBufferSide` come with it.
- `windows/386` joins the targets verified at build time, the Windows family being
  announced for all three of its architectures. That target **compiles**, nothing more.
- Keys have names: letters, digits, arrows, modifiers told apart left and right,
  function and navigation keys. A name states a place, so `KeyW` stays the key above
  `KeyS` whatever the player's layout.
- **A game can be written**: `Run` opens a window and holds the fixed-step loop, the
  game writing a single method, `Game.Step`. `Options` carries the title, internal
  resolution, step, background and atlas, and all but the atlas have a default.
- `Engine` is the same capability without a window or a clock, for a host that keeps
  its own loop: `NewEngine`, then one `Step` per frame and `Frame` for the image.
- `State` builds inputs without a keyboard, enough to exercise a game's method
  without opening a window.
- `Options.Bars` is the colour of the bars surrounding the image when the window is not
  an exact multiple of the internal resolution. It is opaque black, so it does not
  follow the world's background, which a game may want light.

### Changed

- **Packages and identifiers move to English, import paths included**:
  `lozengine/rendu` becomes `lozengine/render`, `geometrie` becomes `geometry`. A game
  written against `0.2.0` no longer compiles without renaming its imports and calls.
- Error messages are in English. Godoc and comments stay in French.

## [0.2.0] — 2026-10-24 — Jalon 1 · Noyau isométrique

Le moteur a désormais une scène et un ordre de dessin, mais rien n'en sort encore :
seule la projection s'atteint de l'extérieur, et il n'y a ni fenêtre, ni boucle.

### Ajouté

- `Matrice` convertit entre les cellules du monde et les pixels de l'écran, dans les
  deux sens, et rend au passage de quoi ordonner un point. Quatre constructeurs, et
  un calcul entier : une même scène rend les mêmes pixels d'une machine à l'autre.
- `PlanchesMax` dit combien de planches un atlas peut porter, ce que la godoc de
  `Quad` annonçait sans le nommer.

***

The engine now has a scene and a draw order, but nothing of them comes out yet: only
the projection is reachable from the outside, and there is still no window, no loop.

### Added

- `Matrice` converts between world cells and screen pixels, both ways, and also
  returns what orders a point. Four constructors, integer arithmetic: one and the
  same scene renders the same pixels from one machine to the next.
- `PlanchesMax` states how many sheets an atlas can hold, which `Quad`'s doc comment
  announced without naming it.

## [0.1.1] — 2026-10-23

Le moteur ne rend toujours pas de jeu : ni fenêtre, ni géométrie, ni scène.

### Ajouté

- Le paquet `rendu` est public : `github.com/sprimault/lozengine/rendu`. Les champs de
  `Quad` et de `Couleur` se lisent dans sa documentation.
- Cinq architectures de plus vérifiées à la compilation : `windows/arm64`,
  `linux/arm64`, `linux/arm`, `linux/386`, `js/wasm`. Elles compilent, rien de plus.

### Modifié

- Plancher abaissé à Go 1.22, ce que le code exige réellement.

***

The engine still renders no game: no window, no geometry, no scene.

### Added

- The `rendu` package is public: `github.com/sprimault/lozengine/rendu`. The fields of
  `Quad` and `Couleur` can be read in its documentation.
- Five more architectures verified at build time: `windows/arm64`, `linux/arm64`,
  `linux/arm`, `linux/386`, `js/wasm`. They compile, nothing more.

### Changed

- Floor lowered to Go 1.22, which is what the code actually requires.

## [0.1.0] — 2026-09-22 — Jalon 0 · Rendu hors écran

**Ce que la 0.1.0 donne, et ce qu'elle ne donne pas.** Le moteur publie sa primitive de
dessin et rien d'autre : ni fenêtre, ni boucle, ni géométrie, ni scène. Le rastériseur
qui consomme ces quads existe, il dessine, il écrit des PNG et une suite de
non-régression compare ses images à des références sans ouvrir de fenêtre — mais il vit
sous `internal/` et reste hors de portée d'un hôte. Ce qui est publié l'est donc pour
être décrit, pas encore pour être rendu de l'extérieur. En revanche ces quatre types
sont définitifs de fait : ils traverseront la frontière C, et une disposition mémoire
ne se révise pas une fois qu'un hôte compilé la suppose.

### Ajouté

- `Quad`, la primitive unique du moteur : un rectangle de planche posé dans le tampon,
  déjà projeté, cullé et trié. Vingt octets, alignement de deux, champs de taille fixe,
  même disposition en Go et en C — c'est ce qui permettra à un hôte de pousser un
  tableau de quads dans son propre pipeline sans conversion par quad et par image. Il
  porte sa destination, sa source dans la planche, l'index de celle-ci, une teinte de
  modulation et ses drapeaux, **mais pas sa clé de tri** : une liste est ordonnée par
  construction, et la répartition des bits de la clé n'a pas à devenir irrévocable en
  traversant l'ABI. Sa valeur zéro ne dessine rien, ce qui est l'échec sûr pour un hôte
  qui remplit la structure de zéros. Trois limites en découlent, documentées et jamais
  vérifiées à l'image : un tampon de 32 767 pixels de côté, une planche de 65 535, et
  **1 024 planches** — ce dernier chiffre venant des dix bits que la clé de tri leur
  réserve.
- `Rendu`, la suite ordonnée des quads d'une image : l'ordre du tableau est l'ordre de
  dessin, et c'est tout ce qu'un backend a le droit d'en savoir. Un tableau nommé, et
  non une structure qui en porterait un : la contrainte de disposition veut les quads
  contigus, et un hôte en reçoit le pointeur du premier avec leur nombre, sans
  conversion ni propriété mémoire à négocier.
- `Couleur`, une couleur RVBA sur quatre octets à alpha prémultiplié, et `Blanc`, la
  teinte neutre. Le prémultiplié ramène le mélange d'un pixel à une multiplication et
  une addition par composante ; le PNG stockant l'inverse, la conversion appartient au
  chargement et jamais au chemin de rendu. `Blanc` est une fonction et non une variable
  de paquet, qu'un consommateur pourrait changer sous les pieds de tous les autres.
- Les drapeaux `MiroirX` et `MiroirY`, qui lisent la source à l'envers — un fournisseur
  d'apparence tire ainsi l'est de l'ouest sans rien ajouter à son atlas —, et `Aplat`,
  qui peint le rectangle en `Teinte` sans lire aucune source. Sans ce dernier, un
  panneau d'interface ou un fondu au noir demanderait d'étirer un texel, c'est-à-dire
  la mise à l'échelle fractionnaire que le moteur s'interdit.

***

**What 0.1.0 gives, and what it does not.** The engine publishes its drawing primitive
and nothing else: no window, no loop, no geometry, no scene. The rasterizer consuming
those quads does exist, it draws, it writes PNGs, and a regression suite compares its
images against references without opening a window — but it lives under `internal/` and
stays out of a host's reach. What is published is therefore published to be described,
not yet to be rendered from the outside. These four types are, on the other hand,
final in practice: they will cross the C boundary, and a memory layout cannot be
revised once a compiled host assumes it.

### Added

- `Quad`, the engine's single primitive: a sheet rectangle placed in the buffer,
  already projected, culled and sorted. Twenty bytes, two-byte alignment, fixed-size
  fields, the same layout in Go and in C — which is what will let a host push an array
  of quads into its own pipeline with no conversion per quad and per frame. It carries
  its destination, its source within the sheet, that sheet's index, a modulation tint
  and its flags, **but not its sort key**: a list is ordered by construction, and the
  key's bit layout has no business becoming irrevocable by crossing the ABI. Its zero
  value draws nothing, which is the safe failure for a host that zeroes the struct.
  Three limits follow, documented and never checked per frame: a buffer of 32,767
  pixels a side, a sheet of 65,535, and **1,024 sheets** — that last figure coming from
  the ten bits the sort key reserves for them.
- `Rendu`, the ordered list of an image's quads: array order is drawing order, and that
  is all a backend is entitled to know. A named array, not a struct holding one: the
  layout constraint wants the quads contiguous, and a host receives the pointer to the
  first one with their count, with no conversion and no memory ownership to negotiate.
- `Couleur`, a four-byte premultiplied-alpha RGBA color, and `Blanc`, the neutral tint.
  Premultiplication reduces blending a pixel to one multiplication and one addition per
  channel; PNG storing the opposite, the conversion belongs to loading and never to the
  render path. `Blanc` is a function rather than a package variable, which a consumer
  could change under everyone else's feet.
- The `MiroirX` and `MiroirY` flags, which read the source backwards — an appearance
  provider thus derives east from west without adding anything to its atlas — and
  `Aplat`, which paints the rectangle in `Teinte` without reading any source. Without
  the latter, a UI panel or a fade to black would require stretching a texel, that is,
  the fractional scaling the engine forbids itself.
