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

Until milestone 7 of the [roadmap](ROADMAP.md) (in French) is reached, the version
stays in `0.x` and public signatures may change with every minor release. The API
freeze and the move to `1.0.0` happen once the engine has been validated by a
second game.

## [Non publié]

### Ajouté

- `Quad`, la primitive de dessin du moteur : un rectangle de planche posé dans le
  tampon, en disposition compatible C — vingt octets, champs de taille fixe. Il porte
  sa destination, sa source, sa planche, une teinte de modulation et ses drapeaux de
  miroir et d'aplat, mais pas sa clé de tri.
- `Rendu`, la suite ordonnée des quads d'une image : l'ordre du tableau est l'ordre de
  dessin.
- `Couleur`, une couleur RVBA à alpha prémultiplié, et `Blanc`, la teinte neutre.
- Les drapeaux `MiroirX`, `MiroirY` et `Aplat`.

### Added

- `Quad`, the engine's drawing primitive: a sheet rectangle placed in the buffer, with
  a C-compatible layout — twenty bytes, fixed-size fields. It carries its destination,
  its source, its sheet, a modulation tint and its mirror and flat-fill flags, but not
  its sort key.
- `Rendu`, the ordered list of an image's quads: array order is drawing order.
- `Couleur`, a premultiplied-alpha RGBA color, and `Blanc`, the neutral tint.
- The `MiroirX`, `MiroirY` and `Aplat` flags.

<!--
Catégories : Ajouté, Modifié, Déprécié, Retiré, Corrigé, Sécurité.
Categories: Added, Changed, Deprecated, Removed, Fixed, Security.

Une entrée par changement visible depuis l'extérieur du moteur, écrite dans les deux
langues sous la même version. Les remaniements internes sans effet sur l'API, les
tests et l'outillage n'y figurent pas.
-->
