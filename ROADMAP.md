# Lozengine · Feuille de route

Moteur de rendu isométrique 2D en Go, sans aucune dépendance externe. Rendu logiciel.
Cibles : Windows et Linux.

État : jalon 0 en cours. Le moteur ne rend rien d'utilisable pour l'instant.

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

### 7 · Validation par un second jeu

Portage d'un jeu au tour par tour sur le moteur, face au jeu temps réel qui a servi à
l'écrire. Gel des signatures publiques.

Fin : le second jeu tourne sur le moteur sans qu'aucune signature publique ait eu à
changer pour lui. Toute signature qu'il a fallu reprendre est corrigée avant le gel,
puisque c'est exactement ce que cette étape sert à découvrir.

### 8 · Publication

Documentation, exemples, intégration continue sur les deux plateformes.

Fin : l'intégration continue est verte sur Windows et sur Linux, et un clone neuf
compile et passe la suite sans autre installation qu'une chaîne Go.

### 9 · Interopérabilité

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

- **macOS.** Impossible sans dépendance ni compilateur C. La frontière backend le garde
  atteignable si la contrainte change.
- **Accélération GPU.** Exclue par la règle de dépendance. Le rendu est logiciel, en
  résolution interne basse mise à l'échelle par un facteur entier.
- **Wayland natif.** Envisagé après le jalon 8. XWayland couvre le besoin d'ici là.
- **Moteur de jeu généraliste.** Le moteur rend une scène et lit des entrées. Ni
  physique, ni système d'entités, ni éditeur, ni script. Et le rendu logiciel ne vise pas
  le nombre de sprites qu'une carte graphique absorbe : la projection se paramètre, la
  puissance non.

## Principe directeur

Aucune dépendance externe, sur aucune plateforme. `go.mod` ne contient aucune ligne
`require`, et cela vaut pour les tests, les exemples et l'outillage. Seule la cible
d'interopérabilité du jalon 9 demande un compilateur C, sans rien ajouter au moteur
lui-même.

La bibliothèque native du jalon 9 ne remet pas ce principe en cause : produire un
`c-shared` ou un `c-archive` réclame un compilateur C sur la machine de compilation, mais
n'ajoute rien au moteur. Les binaires Windows et Linux continuent de se construire sans
cgo, et le code reste identique.
