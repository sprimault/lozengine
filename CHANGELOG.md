# Lozengine · Journal des modifications · Changelog

Format [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/), versions
[SemVer](https://semver.org/lang/fr/). Chaque version porte ses entrées en français puis
en anglais : les notes de publication reprennent la section telle quelle.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), versioning:
[SemVer](https://semver.org/). Every version carries its entries in French then in
English: release notes reuse the section as is.

Tant que le jalon 7 de la [feuille de route](ROADMAP.md) n'est pas atteint, la version
reste en `0.x` et les signatures publiques peuvent changer à chaque version mineure. Le
gel de l'API et le passage en `1.0.0` interviennent après validation du moteur par un
second jeu.

**Le mineur s'incrémente à chaque jalon franchi, sans en porter le numéro** : le titre
de chaque section dit de quel jalon elle vient. Les deux suites n'ont pas à coïncider,
et vouloir les aligner imposerait soit un `v0.0.0`, que Go confond avec l'absence de
version, soit une renumérotation de la feuille de route que l'historique ne suivrait
pas.

Until milestone 7 of the [roadmap](ROADMAP.md) (in French) is reached, the version
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
