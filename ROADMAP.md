# Lozengine · Feuille de route

Moteur de rendu isométrique 2D en Go, dont le cœur n'a aucune dépendance. Rendu
logiciel. Cibles : Windows, Linux et le navigateur en WebAssembly.

État : jalon 0 franchi, jalon 1 en cours. Le moteur rend hors écran et sa suite de
non-régression compare des images de référence sans ouvrir de fenêtre ; il n'a encore
ni géométrie, ni scène, ni fenêtre.

## Jalons

### 0 · Rendu hors écran

Rastériseur logiciel d'atlas RGBA, écriture PNG, visionneuse rejouant une scène
scriptée.

Fin : la suite de tests compare des rendus à des images de référence, sans ouvrir de
fenêtre.

### 1 · Noyau isométrique

Géométrie et projection paramétrable, modèle de scène, culling, clé de tri, tri
topologique des volumes multi-cellules.

La projection étant une matrice, la grille carrée vue de dessus est un constructeur de
plus, pas un second moteur : le dimétrique reste le sujet, le reste vient avec.

Fin : une carte de 500×500 portant 5000 entités produit une liste de quads ordonnée.

### 2 · Backend Windows

Fenêtre, boucle de messages, entrées, présentation et mise à l'échelle entière par
appels système directs. Paquet racine qui assemble le tout : boucle à pas fixe, défauts
utilisables, et une seule méthode à écrire côté jeu — l'ordre de dessin vient de la clé
de tri, l'échelle du facteur entier.

La boucle s'appuie sur une étape appelable, qu'un hôte pilote lui-même.

Fin : un exemple de moins de 30 lignes affiche une grille et un personnage déplaçable.

### 3 · Backend Linux

Client X11 écrit sur le protocole, sans bibliothèque C.

Fin : le même exemple tourne sous X11 et sous XWayland.

### 4 · Apparence et interaction

Interface de fournisseur d'apparence, implémentation de référence en planches PNG,
animation, occlusion, étages, désignation à la souris. Texte depuis un atlas de glyphes
et couche d'interface, qui est une couche haute de la clé de tri et non une exception au
chemin de rendu.

Fin : un décor à deux niveaux se traverse sans perdre le personnage.

### 5 · Audio

Mixeur logiciel et sorties système sur les deux plateformes.

Fin : huit sons simultanés et une musique en boucle, sans rupture ni décalage
perceptible, sous Windows et sous Linux.

### 6 · Apparence calculée

Éclairage par cartes de normales et de hauteur, atlas dynamique avec cache et budget
amorti, fabrique de sprites depuis des modèles volumiques.

Fin : 800 entités de 6 types changent de direction sur la même image sans pic.

### 7 · Backend web

Client WebAssembly : canvas, entrées, présentation. Le rendu logiciel s'y transporte tel
quel — le tampon part en `putImageData`, qui ne met rien à l'échelle, et l'agrandissement
entier revient à `image-rendering: pixelated` sur un canvas dimensionné par CSS, comme il
revient à `StretchDIBits` sous Windows.

`syscall/js` étant dans la bibliothèque standard, la cible n'ouvre pas la liste close.
Elle demande en revanche de livrer `wasm_exec.js`, le fichier de liaison fourni avec la
chaîne Go : un artefact à distribuer avec la page, pas une dépendance du moteur. Le son y
passe par WebAudio, troisième sortie du mixeur, qui ne change pas pour autant.

Go n'émettant pas de SIMD en WebAssembly et n'y ayant pas de vrais threads, c'est la
cible où une charge élevée saturera en premier. Le budget d'image s'y mesure, il ne s'y
suppose pas.

Fin : le même exemple qu'aux jalons 2 et 3 tourne dans un navigateur, et son temps par
image y est mesuré.

### 8 · Validation par un second jeu

Portage d'un jeu au tour par tour sur le moteur, face au jeu temps réel qui a servi à
l'écrire. Gel des signatures publiques.

Fin : le second jeu tourne sur le moteur sans qu'aucune signature publique ait eu à
changer pour lui. Toute signature qu'il a fallu reprendre est corrigée avant le gel,
puisque c'est exactement ce que cette étape sert à découvrir.

### 9 · Publication

Documentation, exemples, intégration continue sur les deux plateformes.

Fin : l'intégration continue est verte sur Windows et sur Linux, et un clone neuf
compile et passe la suite sans autre installation qu'une chaîne Go.

### 10 · Interopérabilité

Bibliothèque native consommable depuis Rust, C++ et tout langage parlant l'ABI C, en
`c-shared` et `c-archive`, avec en-tête généré et exemples minimaux.

Le noyau seul est exposé, jamais la fenêtre ni la boucle : un jeu hôte possède les
siennes. Deux contrats au choix de l'hôte, rastérisation dans un tampon qu'il alloue, ou
sortie d'un tableau contigu de quads qu'il pousse dans son propre pipeline, y compris un
pipeline GPU.

Cette cible est la seule à demander `CGO_ENABLED=1` et un compilateur C sur la machine de
compilation. Elle n'ajoute aucune dépendance au moteur, dont le `go.mod` reste vide.

Fin : un exemple Rust et un exemple C++ affichent une scène rendue par le moteur.

## Hors périmètre

- **macOS natif.** Cocoa ne s'atteint qu'à travers l'exécution Objective-C, ce qui
  demande cgo ou une couche comme purego — l'une comme l'autre sortiraient de la liste
  close. La frontière backend le garde atteignable si la contrainte change, et le backend
  web y donne accès entre-temps, dans un navigateur.
- **Accélération GPU.** Exclue par la règle de dépendance. Le rendu est logiciel, en
  résolution interne basse mise à l'échelle par un facteur entier.
- **Wayland natif.** Envisagé après le jalon 9. XWayland couvre le besoin d'ici là.
- **Moteur de jeu généraliste.** Le moteur rend une scène et lit des entrées. Ni
  physique, ni système d'entités, ni éditeur, ni script. Et le rendu logiciel ne vise pas
  le nombre de sprites qu'une carte graphique absorbe : la projection se paramètre, la
  puissance non.

## Principe directeur

**Le cœur du moteur n'a aucune dépendance, sur aucune plateforme.** Géométrie, scène,
tri, structures et rastériseur n'emploient que la bibliothèque standard, tests compris,
et un contrôle le vérifie paquet par paquet. La fenêtre, les entrées et le son passent
par des appels système directs et par des protocoles écrits sur socket, jamais par le
chargement d'une bibliothèque C.

Deux paquets périphériques font exception, sous licence permissive et par décision
inscrite : le rendu de polices au jalon 4, la sortie audio au jalon 5. Les réécrire
coûterait des semaines pour un résultat qui ne distinguerait en rien le moteur — ce qui
n'est pas le cas du protocole X11, dont les semaines sont assumées parce qu'il est le
sujet. La liste et les critères d'entrée sont dans `docs/go.md`.

Seule la cible d'interopérabilité du jalon 10 demande un compilateur C, sans rien
ajouter au moteur lui-même.

La bibliothèque native du jalon 10 ne remet pas ce principe en cause : produire un
`c-shared` ou un `c-archive` réclame un compilateur C sur la machine de compilation, mais
n'ajoute rien au moteur. Les binaires Windows et Linux continuent de se construire sans
cgo, et le code reste identique.
