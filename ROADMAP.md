# Lozengine · Feuille de route

Moteur de rendu isométrique 2D en Go, dont le cœur n'a aucune dépendance. Rendu
logiciel. Cibles : Windows, Linux et le navigateur en WebAssembly.

État : jalon 4 franchi. Un jeu se décrit : le moteur ouvre une fenêtre, lit le clavier
et la souris, tient une boucle à pas fixe et n'attend du jeu qu'une seule méthode. Il
décide de l'ordre de dessin, y compris entre deux niveaux de décor, et désigne la case
sous le pointeur. Sous Windows et sous Linux — le navigateur vient au jalon 5.

## Jalons

### 0 · Rendu hors écran

Rastériseur logiciel d'atlas RGBA, écriture PNG, visionneuse rejouant une scène
scriptée.

Fin : la suite de tests compare des rendus à des images de référence, sans ouvrir de
fenêtre.

### 1 · Noyau isométrique

Géométrie et projection paramétrable, modèle de scène, retrait de ce qui sort de
l'image, clé de tri, départage de ce que la profondeur laisse à égalité.

La projection étant une matrice, la grille carrée vue de dessus est un constructeur de
plus, pas un second moteur : le dimétrique reste le sujet, le reste vient avec.

Fin : une carte de 500×500 portant 5000 entités produit une liste de quads ordonnée.

### 2 · Backend Windows

Fenêtre, boucle de messages, entrées, présentation et mise à l'échelle entière par
appels système directs. Paquet racine qui assemble le tout : boucle à pas fixe, défauts
utilisables, et une seule méthode à écrire côté jeu — l'ordre de dessin vient de la clé
de tri, l'échelle du facteur entier.

La boucle s'appuie sur une étape appelable, qu'un hôte pilote lui-même.

Fin : un exemple affiche une grille et un personnage déplaçable, et le hello world tient
en moins de 30 lignes.

Deux exigences et non une, parce qu'elles mesurent deux choses : la première dit que la
chaîne entière tourne, la seconde que les valeurs par défaut suffisent à démarrer. Les
fondre en demanderait une troisième que personne ne voulait — un déplacement aux quatre
directions coûte à lui seul seize lignes, et c'est de la logique de jeu que le moteur
n'a pas le droit d'écrire à la place du jeu.

### 3 · Backend Linux par X11

Client X11 écrit sur le protocole, sans bibliothèque C. Wayland natif et DRM/KMS sont
le jalon 6 : empilés ici, ils feraient un jalon qu'on ne finit pas.

Fin : le même exemple tourne sous X11 et sous XWayland.

### 4 · Apparence et interaction

Interface de fournisseur d'apparence, implémentation de référence en planches PNG,
animation, occlusion, étages, désignation à la souris. Texte depuis un atlas de glyphes
et couche d'interface, qui est une couche haute de la clé de tri et non une exception au
chemin de rendu.

Fin : un décor à deux niveaux se traverse sans perdre le personnage.

### 5 · Backend web

Client WebAssembly : canvas, entrées, présentation. Le rendu logiciel s'y transporte tel
quel — le moteur agrandit lui-même, bandes comprises, et le tampon part en
`putImageData`, qui ne met rien à l'échelle. C'est le contrat des cinq familles de
backends : présenter les pixels tels quels, et n'étirer jamais.

`syscall/js` étant dans la bibliothèque standard, la cible n'ouvre pas la liste close.
Elle demande en revanche de livrer `wasm_exec.js`, le fichier de liaison fourni avec la
chaîne Go : un artefact à distribuer avec la page, pas une dépendance du moteur.

C'est ce jalon qui donne Android et iOS, par le navigateur : un jeu en page plein écran
ne se distingue pas d'une application pour qui y joue. Il passe donc devant l'audio, dont
aucun appareil n'attend la sortie. Les entrées tactiles en font partie, faute de quoi la
cible ne sert que le poste ; la densité de pixels de la page aussi, qu'aucune autre
plateforme ne pose.

Le dimensionnement de la vue n'attend plus ce jalon et a été livré avant lui : les bandes
qu'une vue figée laissait étaient un défaut des backends déjà en place, et non une
capacité que le navigateur apportait.

Go n'émettant pas de SIMD en WebAssembly et n'y ayant pas de vrais threads, c'est la
cible où une charge élevée saturera en premier. Le budget d'image s'y mesure, il ne s'y
suppose pas.

Fin : le même exemple qu'aux jalons 2 et 3 tourne dans un navigateur, du poste au
téléphone, et son temps par image y est mesuré.

### 6 · Linux sans serveur graphique

Client Wayland écrit sur le protocole, et un chemin DRM/KMS avec `evdev` pour les
machines qui n'ont aucun serveur à qui parler.

Wayland est le compositeur par défaut des distributions récentes et de Raspberry Pi OS :
XWayland couvre le besoin jusqu'ici, au prix d'une couche de traduction qu'un moteur
logiciel paie en latence. DRM/KMS vise le Pi en kiosque et les bornes embarquées.

Même nature de travail qu'au jalon 3 — un protocole décrit, parlé sur une socket Unix,
`wayland.xml` étant sa description même — et des `ioctl` relevés dans les en-têtes UAPI
du noyau. `WriteMsgUnix`, `UnixRights`, `Ftruncate` et `Mmap` sont dans la bibliothèque
standard ; seul `memfd_create` manque à `syscall`, et son numéro se déclare comme se
déclarent les opcodes.

Deux protocoles entiers, chacun comparable à X11 en volume. Les empiler sur le jalon 3
ferait un jalon qu'on ne finit pas, d'où celui-ci.

Fin : le même exemple tourne sous un compositeur Wayland et sur un connecteur DRM/KMS,
sans serveur graphique.

### 7 · Audio

Mixeur logiciel et sorties système sur les deux plateformes, plus WebAudio comme
troisième sortie du même mixeur.

Fin : huit sons simultanés et une musique en boucle, sans rupture ni décalage
perceptible, sous Windows, sous Linux et dans un navigateur.

### 8 · Apparence calculée

Éclairage par cartes de normales et de hauteur, atlas dynamique avec cache et budget
amorti, fabrique de sprites depuis des modèles volumiques.

Fin : 800 entités de 6 types changent de direction sur la même image sans pic.

### 9 · Validation par un second jeu

Portage d'un jeu au tour par tour sur le moteur, face au jeu temps réel qui a servi à
l'écrire. Gel des signatures publiques.

Fin : le second jeu tourne sur le moteur sans qu'aucune signature publique ait eu à
changer pour lui. Toute signature qu'il a fallu reprendre est corrigée avant le gel,
puisque c'est exactement ce que cette étape sert à découvrir.

### 10 · Publication

Documentation, exemples, intégration continue sur les deux plateformes.

Fin : l'intégration continue est verte sur Windows et sur Linux, et un clone neuf
compile et passe la suite sans autre installation qu'une chaîne Go.

### 11 · Interopérabilité

Bibliothèque native consommable depuis Rust, C++ et tout langage sachant appeler du C —
cette convention d'appel, qu'on nomme ABI, étant le terrain commun de presque tous. En
`c-shared` et `c-archive`, avec en-tête généré et exemples minimaux.

Le noyau seul est exposé, jamais la fenêtre ni la boucle : un jeu hôte possède les
siennes. Deux contrats au choix de l'hôte, rastérisation dans un tampon qu'il alloue, ou
sortie d'un tableau contigu de quads qu'il pousse dans son propre pipeline, y compris un
pipeline GPU.

Cette cible est la seule à demander `CGO_ENABLED=1` et un compilateur C sur la machine de
compilation. Elle n'ajoute aucune dépendance au moteur, dont le `go.mod` reste vide.

**Android natif passe par ici, et n'est pas hors périmètre.** Le moteur n'y ouvre pas de
fenêtre lui-même : l'application fournit la sienne et consomme le moteur par cette même
convention d'appel, en le laissant rastériser dans le tampon qu'elle alloue. C'est mot
pour mot l'un des deux contrats ci-dessus, donc rien de particulier à écrire pour ce
système.

La licence du kit de développement natif d'Android n'y fait pas obstacle : elle ne pose
aucune condition sur ce qu'on produit avec lui, et c'est un compilateur, pas une
dépendance du moteur — `go.mod` reste vide, et la chaîne d'outils vit sur la machine de
compilation. Quelle voie retenir exactement s'instruit au moment d'écrire ce jalon.

Fin : un exemple Rust et un exemple C++ affichent une scène rendue par le moteur.

## Hors périmètre

- **Une fenêtre native sur macOS ou sur iOS.** Pas servie, et pas pour la même raison.
  Pour iOS, Go refuse de lier `ios/arm64` sans cgo, et la compilation exige de toute
  façon un Mac avec Xcode : aucune ligne de ce dépôt ne lèverait ce refus. Pour macOS,
  Cocoa ne s'atteint que par la convention d'appel d'Objective-C, dont le point d'entrée
  s'obtient par `dlsym` — donc par cgo et un compilateur C, ou par un paquet de plus
  hors de la liste close —, et l'éprouver demande une machine qui n'est pas au parc. Le
  jalon 5 couvre les deux systèmes par le navigateur.

  **Ce qui n'est pas hors périmètre, et qu'il ne faut pas confondre** : qu'un jeu tourne
  sur ces systèmes. La bibliothèque native du jalon 11 le permet dans l'autre sens, en
  laissant l'application fournir sa fenêtre.
- **Accélération GPU.** Exclue par la règle de dépendance. Le rendu est logiciel, en
  résolution interne basse mise à l'échelle par un facteur entier.
- **Moteur de jeu généraliste.** Le moteur rend une scène et lit des entrées. Ni
  physique, ni système d'entités, ni éditeur, ni script. Et le rendu logiciel ne vise pas
  le nombre de sprites qu'une carte graphique absorbe : la projection se paramètre, la
  puissance non.

## Principe directeur

**Le cœur du moteur n'a aucune dépendance, sur aucune plateforme.** Géométrie, scène,
tri, structures et rastériseur n'emploient que la bibliothèque standard, tests compris,
et un contrôle le vérifie paquet par paquet. La fenêtre, les entrées et le son passent
par les modules que le système fournit lui-même — `syscall` sous Windows — et par des
protocoles écrits sur socket sous Linux. Le critère est que le moteur n'embarque aucune
bibliothèque C tierce, n'exige aucun compilateur C et ne distribue rien à côté de son
binaire : `go.mod` vide, `CGO_ENABLED=0`.

Deux paquets périphériques y sont admis d'avance, sous licence permissive et par
décision inscrite : le rendu de polices et la sortie audio du jalon 7. Les réécrire
coûterait des semaines pour un résultat qui ne distinguerait en rien le moteur — ce qui
n'est pas le cas du protocole X11, dont les semaines sont assumées parce qu'il est le
sujet.

**Admis ne veut pas dire employé, et le premier ne l'est toujours pas.** Le jalon 4
affiche du texte depuis une planche de glyphes, qui est une image comme une autre :
rastériser une police à l'exécution n'avait alors plus d'objet, et `go.mod` est resté
vide. La place reste ouverte pour le jour où une police se lirait à l'exécution. La
liste et les critères d'entrée sont dans `docs/go.md`.

Seule la cible d'interopérabilité du jalon 11 demande un compilateur C, sans rien
ajouter au moteur lui-même.

La bibliothèque native du jalon 11 ne remet pas ce principe en cause : produire un
`c-shared` ou un `c-archive` réclame un compilateur C sur la machine de compilation, mais
n'ajoute rien au moteur. Les binaires Windows et Linux continuent de se construire sans
cgo, et le code reste identique.
