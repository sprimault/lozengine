# Conception

Ce document fait foi. Un désaccord entre le code et ce qui est écrit ici est un
défaut du code, jamais l'inverse.

Les décisions ci-dessous sont tranchées. Elles vivent dans les signatures
publiques ou dans les structures que tout le reste consomme, donc elles ne se
corrigent pas après coup : les rouvrir est une discussion, pas un correctif.

## Ce que le moteur résout

Un moteur de rendu isométrique ne se distingue pas par sa façon de dessiner un
sprite, mais par sa façon de décider lequel passe devant. Dans une bibliothèque
de dessin 2D, cet ordre est le problème de celui qui appelle : il dessine dans
l'ordre où il veut que ça se superpose, et en isométrique il finit par écrire
lui-même une clé de tri, des couches, des étages, et le départage de ce que la
profondeur laisse à égalité.

Lozengine prend ce travail à sa charge. On lui décrit ce qui existe et où ; il
décide de l'ordre. Tout le reste de la conception en découle.

Conséquence directe sur la forme de l'API : une scène se **décrit**, elle ne se
dessine pas. Il n'existe pas de fonction de dessin publique prenant une position
d'écran, parce qu'elle court-circuiterait la seule chose que le moteur apporte.

## Repère et projection

- **Monde distinct de l'écran.** Cellules flottantes, `x` et `y` dans la grille,
  `z` en hauteur au-dessus du sol. Sous les deux projections en losange, `x` croît
  vers le bas-gauche de l'écran et `y` vers le bas-droite ; sous la vue de dessus,
  où les axes ne tournent pas, `x` va vers la droite et `y` vers le bas.
- **Ancre.** Le point du monde visé est le contact au sol, centre de case par
  défaut, et il ne se règle pas. Ce qui se règle est l'endroit où l'image se pose
  autour du pixel qu'il atteint — le pied de l'image, centré, par défaut —, par
  l'apparence et jamais par appel. La pose en pixels de l'interface fait exception :
  son origine est le coin haut-gauche, et l'ancre s'y ajoute de la même façon.
- **Hauteur.** `z` reste un champ séparé, jamais fondu dans la matrice de
  projection. L'écran est la projection de `(x, y)` moins `z` fois un facteur ;
  la clé de tri se calcule sur `(x+y)` puis `z`. Fondre les deux perd
  l'information à l'entrée du pipeline et rend le tri irréparable.
- **Projection.** Matrice affine avec son inverse calculée une fois et gardée.
  Quatre voies de construction : le dimétrique au ratio libre, l'isométrie vraie
  qui **dérive le sien**, la grille carrée vue de dessus, et la matrice brute pour
  les cas exotiques. Aucune constante de ratio ailleurs que dans un constructeur
  de matrice : pas de division par deux disséminée dans le code.

  L'isométrie dérive son ratio parce qu'un constructeur qui n'impose rien n'en est
  pas un : son rapport est déterminé — `1/√3`, soit 18 pour une largeur de 32
  contre 16 pour le dimétrique —, et le laisser en paramètre en ferait un
  dimétrique sous un autre nom. La racine carrée est à arrondi correct par
  IEEE-754, comme la division, et s'évalue une fois à la construction.
- **Hauteur de l'étage.** Le facteur par lequel `z` monte à l'écran vit dans la
  matrice, et vaut par défaut la hauteur de la cellule — le cas d'un cube
  isométrique. Un ajusteur le change, plutôt qu'un paramètre de plus à chaque
  constructeur. S'il vivait ailleurs, l'appelant appliquerait la soustraction et le
  calcul se disséminerait, ce que la clause ci-dessus interdit.
- **Sortie absolue, vue retranchée au moment du retrait.** La projection rend des
  pixels **absolus**, en `int32`. La scène possède l'origine de la vue ; c'est le
  tri qui la retranche, au moment où il retire ce qui tombe hors de l'image, et c'est
  lui qui remplit le `Quad`. Deux étages, et la raison est le débordement : en absolu
  sur 16 bits, la plus grande carte représentable serait de 2 047 cases de côté à
  tuile 32 — seize fois moins que la limite de la clé de tri, et un chiffre qui
  dépendrait de la taille de tuile.

  **Les deux se font ensemble parce que c'est le même calcul** : situer un point par
  rapport aux bords de l'image, c'est à la fois lui donner sa place dedans et savoir
  s'il y tombe. C'est aussi le seul endroit où la réduction à seize bits ne peut pas
  déborder : ce qui est trop loin est déjà parti, et ce qui dépasse d'un bord commence
  au plus à sa largeur en arrière.
- **Caméra.** Quatre orientations discrètes à 90°. Elles touchent le remappage des
  coordonnées par la scène et le décalage de direction demandé au fournisseur
  d'apparence — pas la matrice, qui reste la même pour les quatre vues.

  **La rotation est un remappage de coordonnées, jamais un changement de signe.**
  Une rotation en `(−y, x)` rendrait la profondeur signée, demanderait un biais et
  ferait tomber de moitié la plage du champ. La scène possédant l'étendue de sa
  grille, elle rend `(Max−y, x)` : les coordonnées restent positives et le champ de
  profondeur garde ses dix-huit bits non signés.
- **Échelle.** Facteur entier obligatoire vers la fenêtre, bandes autour quand
  la taille ne tombe pas juste. Jamais de mise à l'échelle fractionnaire.

La projection étant une matrice paramétrable, une grille carrée vue de dessus
est un constructeur de plus et non un second moteur. Le dimétrique reste le
sujet ; le reste vient avec.

## Déterminisme entre architectures

La même scène rend la même image, au pixel près, du poste au Raspberry Pi et au
navigateur. Sans cela, la suite de non-régression — qui compare pixel à pixel —
ne vaudrait que sur la machine qui a produit les références.

**La virgule fixe commence à la projection.** La matrice est en entiers, le
produit en entiers, et plus un flottant ne subsiste dans le chemin de rendu.
L'entrée reste flottante, comme le repère ci-dessus le veut : un jeu pose une
entité en `x = 3.5`, et la conversion vers le fixe se fait à l'entrée de la
projection — un arrondi, déterministe, et non une expression fusionnable.

Le piège est propre à Go, dont la spécification autorise la fusion de `x*y + z`
en une opération FMA que `arm64`, `riscv64`, `ppc64` et `s390x` pratiquent là où
`amd64` ne la pratique pas. Une matrice affine n'est faite que de `x*y + z`. Le
contournement par `float64(x*y) + z` existe, mais il demande d'y penser à chaque
expression, pour toujours, sans que rien ne le vérifie : la virgule fixe, elle,
tient par construction.

Trois conséquences, à ne pas redécouvrir :

- **Seize bits de fraction sur un `int32`** laissent ±32 768 à la partie entière,
  ce qui tombe exactement sur la limite de cases de côté que la clé de tri
  documente. **L'accord porte sur les coordonnées de monde, et sur elles seules**
  — en pixels d'écran absolus, la même limite demanderait 524 272, soit seize fois
  le plafond. C'est ce qui oblige la caméra à retrancher la vue.
- **Le produit passe par `int64`** avant décalage : un produit de coordonnée par
  coefficient déborde un `int32` bien avant les bornes utiles.
- **La conversion depuis le flottant sature**, et sans elle tout le reste est
  vain : hors plage, la conversion d'un flottant vers un `int32` est laissée à
  l'implémentation, `amd64` rendant le plus petit entier là où `arm64` sature au
  plus grand. Deux comparaisons par appel, et une entité posée hors plage cesse de
  rendre des pixels différents selon la machine.
- **L'aller-retour admet un quantum.** L'inverse d'une matrice en virgule fixe
  n'est pas exact ; la tolérance s'écrit dans la godoc plutôt que de se supposer
  nulle.

**Le format n'est pas qu'un choix de plage : c'est ce qui rend l'entrée flottante
gratuite.** Un consommateur qui tient déjà ses positions en virgule fixe au même
format — seize bits de fraction sur un `int32` — les passe en `float64` et les
retrouve exactes de l'autre côté, un `float64` ayant 53 bits de mantisse contre 32.
Mesuré sur 4,2 millions de valeurs couvrant tout le domaine : aucun écart. Vingt
bits de fraction, ou un `int64`, et ce même consommateur perdrait un cran à chaque
appel. C'est pourquoi le moteur n'expose aucun type de coordonnée fixe : il serait
définitif, tous les consommateurs devraient le connaître, et il n'achèterait rien
que le format n'offre déjà.

Ce que le déterminisme ne demande pas ici : réécrire des fonctions
transcendantes. Une projection isométrique affine n'en appelle aucune — ses
constructeurs prennent des longueurs, jamais un angle — et la division comme la
racine carrée d'IEEE-754 sont à arrondi correct.

## Structure et interopérabilité

- **Primitive unique.** Tout converge vers le quad trié. Les fonctions de
  confort sont du sucre au-dessus, jamais un second chemin de rendu. Une couche
  d'interface est une couche haute de la clé de tri, pas une exception.
- **Options en structure.** Valeur zéro utilisable. Tout ce qui n'est pas
  renseigné a un défaut raisonnable.
- **Disposition compatible C.** `Quad` et `DrawList` n'emploient que des champs de
  taille fixe, dans un tableau contigu. Pas de chaîne, pas d'interface, pas de
  slice imbriquée, pas de pointeur vers de la mémoire gérée.
- **Aucun rappel dans le chemin de rendu.** L'apparence est résolue en amont,
  jamais par une fonction appelée pendant le rendu.

Les deux dernières viennent de l'exposition du noyau aux langages parlant l'ABI
C — la convention par laquelle un langage en appelle un autre —, et elles
s'appliquent bien avant que cette exposition existe. Une chaîne dans `Quad`
imposerait une conversion par quad et par image, ce qui la rendrait
inutilisable ; un appel entrant depuis un autre langage coûte des centaines de
nanosecondes, ce qu'un chemin parcouru des milliers de fois par image ne peut
pas absorber.

Ce qui sera exposé est le noyau, jamais la fenêtre ni la boucle : un jeu hôte
possède déjà les siennes et ne cédera pas son thread principal. Deux contrats,
au choix de l'hôte — rastérisation dans un tampon qu'il alloue, ou sortie d'un
tableau contigu de quads qu'il pousse dans son propre pipeline, y compris un
pipeline graphique que le moteur n'a pas. Les objets vivants traversent la
frontière par handles opaques, jamais par pointeur Go.

## Paramètres retenus

- Résolution interne : 480×270 **au minimum**. Tuile de sol 32×16, personnage debout
  autour de 24 pixels. C'est un plancher et la taille d'ouverture de la fenêtre, pas
  une taille fixe : au-delà, la vue s'élargit avec la fenêtre jusqu'au plafond que le
  jeu pose, ce qui évite les bandes sans toucher au facteur entier.
- Directions d'apparence : déclarées par le fournisseur, jamais codées en dur.
  Huit pour l'implémentation de référence en planches PNG, seize pour la
  fabrique volumique. Le littéral n'apparaît que dans le fournisseur.
- Clé de tri : un entier de 64 bits composé, du bit fort au bit faible — couche
  4 bits, étage 6, profondeur `x+y` 18 en quarts de cellule, `z` 16, planche 10,
  index stable 10.

Le tri se fait sur cette clé, et ce qu'elle laisse à égalité se départage à
l'intérieur d'un groupe de clés identiques. Dix-huit bits donnent 262 144 quarts,
soit 65 536 cellules de profondeur `x+y`, et une carte carrée de côté N a `x+y`
au plus égal à 2N : la limite documentée est donc de 32 768 cases de côté, exacte
et non arrondie.

Les deux bits qui séparent cette répartition d'une profondeur de vingt vont à
l'index stable, seul champ sans recours : avec 256 valeurs il départagerait mal des
quads partageant déjà couche, étage, profondeur au quart, `z` et planche. Cent
trente mille cases de côté ne servent personne quand la cible de contrôle est
500×500.

**L'axe transverse `x−y` n'est pas un champ de la clé, et il n'a pas besoin de
l'être.** Un moteur qui trie par comparateur le porte naturellement comme critère de
départage, après la profondeur ; celui-ci trie par paquets et n'a pas de comparateur
où le mettre. Lui donner des bits les prendrait à l'index stable, et rouvrirait
l'arbitrage que les deux bits ci-dessus viennent de clore. Le départage emploie donc
l'abscisse écran, qui lui est proportionnelle — par un facteur négatif, puisqu'elle
croît avec `y − x` —, et qui est déjà calculée : la recalculer ferait un second
arrondi, qui peut tomber de l'autre côté d'un quart et réordonner deux choses sans
que rien ne le signale.

## Frontières de paquets

Le sens des dépendances ne s'inverse jamais.

```
lozengine             surface publique, boucle, assemblage      tous les internes
geometry              projection, matrices, conversions         aucune
scene                 grille, entités, couches, caméra          geometry, render
internal/sorting      retrait hors image, clé, départage        geometry, scene, render
render                structures Quad et DrawList, publiques    aucune
raster                planches, atlas, rastériseur, PNG         render
appearance            interface fournisseur, animation          geometry, render, raster, scene
input                 touches, état du clavier                  aucune
internal/backend/…    contrat de fenêtre, familles, présentation  input, render
internal/audio        mixeur et sorties système                 aucune
internal/viewer       scènes scriptées, rejeu, images           tous les précédents
cmd/viewer            commande de rejeu                         viewer
examples/…            ce qu'un jeu écrit, en entier             lozengine
```

Les exemples sont dans ce tableau parce qu'ils sont la mesure du critère de fin d'un
jalon : ils n'importent que le paquet racine, et le jour où l'un d'eux doit atteindre un
paquet interne, c'est que la racine ne donne pas ce qu'elle promet.

`scene` et `sorting` atteignent `render` parce que l'un décrit des entités qui portent
une teinte et des drapeaux, et que l'autre produit la liste de quads : sans cela, le
noyau ne pourrait pas rendre ce que son critère de fin lui demande. Cela ne leur
ouvre rien d'autre — `render` n'a lui-même aucune dépendance.

La visionneuse n'est pas le moteur : c'est l'outillage qui le rend jugeable sans
écran. Rien ne dépend d'elle, et c'est pourquoi elle atteint tout le reste — un
scénario de non-régression construit une scène, la fait trier, puis la rastérise.

Le paquet racine est le seul à tout connaître, et rien ne dépend de lui.

**Un paquet sort d'`internal/` quand un consommateur doit lire la documentation de ses
types, et c'est une décision, pas un rangement.** Le tableau ci-dessus dit lesquels, et
c'est le seul endroit qui les énumère : une seconde liste a divergé deux fois. Le reste
— le tri, les backends, la visionneuse — n'a aucune raison d'être lu par qui écrit un
jeu, et le compilateur Go interdit de l'importer depuis un autre module.

**`raster` est sorti en entier, faute de découpe possible.** Un jeu construit ses
planches et son atlas, donc leurs champs doivent se lire ; mais le rastériseur emploie
deux méthodes privées de `Sheet` — l'encodage des lignes qui lui donne des segments
à traiter en bloc —, et laisser le dessin derrière aurait obligé à publier le détail de
cette optimisation dans un type que l'ABI exposera. Les cinq fonctions de dessin sont
donc publiques ; ce sont celles que le jalon 11 exposera de toute façon, et le paquet
racine ne les republie pas, un jeu n'ayant pas à les appeler.

**Le critère est mécanique, et il s'est vérifié deux fois.** Un type publié par alias
ne montre rien quand il désigne un paquet `internal` : un consommateur lisait
`type Quad = rendu.Quad` et la godoc de l'alias, sans un seul champ — ni coordonnées,
ni teinte, ni planche —, ce qui a sorti `render` au jalon 0. La même chose vaut pour les
méthodes, et personne ne l'avait vu : le 25 octobre 2026, `go doc -all` ne montrait
aucune des quatre méthodes de `Matrix`, alors que la `0.2.0` l'annonçait comme la
surface du jalon 1 et que sa propre godoc disait « s'emploie par ses méthodes ».
`geometry` et `scene` sont sortis de là.

**Ce que cela retire, et qu'il faut savoir.** La règle précédente voulait que la
surface publique se réduise à ce que le paquet racine republie, publier demandant de
l'écrire. Elle supposait qu'un alias publie quelque chose, et la mesure l'a défaite :
ce qui sort d'`internal/` est désormais surveillé à l'export. Le paquet racine
republie toujours par alias, pour qu'un jeu n'ait qu'un seul import à écrire.

Conséquence assumée : un paquet public publie tout ce qu'il exporte, donc les deux
portes par lesquelles le tri lit la scène le sont aussi. Il n'y a pas de voie Go pour
l'éviter, et le dégât est borné — la clé sature, donc une couche écrite hors borne
range la chose au plafond sans rien casser d'autre. Leur documentation nomme la voie
normale.

Deux acquis restent entiers. Réorganiser ce qui est interne ne casse aucun
consommateur. Et un hôte ne peut pas importer un backend pour se lier directement à
X11 : ce n'est pas un contrôle qui l'interdit, c'est le compilateur.

Les types du noyau sont republiés par **alias** et non par enveloppe. Un alias
désigne le même type, sans conversion ni copie — ce qui est indispensable pour des
structures qu'un hôte reçoit par milliers à chaque image, et qui doivent garder
leur disposition compatible C jusqu'à la frontière.

`scene` et `sorting` ne connaissent aucun backend. `raster` et `backend` ne
connaissent pas `scene` : ils consomment une `DrawList` déjà triée. Si un backend a
besoin de trier, de projeter ou de consulter la scène, la frontière est mal
placée, et le correctif est dans le noyau.

C'est ce qui permet au rastériseur logiciel et aux backends système de consommer
exactement la même chose, et donc à la suite de non-régression de valider le
noyau sans écran.

## Les deux portes d'entrée

Le paquet racine sert qui écrit un jeu en Go : une fenêtre, une boucle à pas
fixe, des entrées, et une seule méthode à écrire. Pas de méthode de dessin,
parce que l'ordre vient de la clé de tri ; pas de méthode de mise en page, parce
que l'échelle vient du facteur entier.

Le noyau sert qui intègre le moteur dans un jeu qui a déjà sa boucle. Il ne
réclame ni fenêtre ni thread.

Le paquet racine ajoute du confort, jamais de capacité : tout ce qu'il permet se
fait aussi par le noyau, et il n'abrite aucune logique de rendu. Dès qu'une
décision d'ordre, de projection ou d'échelle y apparaît, elle est mal placée.

## Universalité

Le moteur est destiné à d'autres projets que celui qui l'a fait naître. Deux
conséquences :

- Une décision interne se change au prix d'une refonte ; une décision d'API se
  change au prix de la confiance de ceux qui ont adopté le moteur. Toute
  nouvelle signature publique est traitée comme définitive.
- Ne jamais coder en dur ce qui relève du jeu : nombre de directions, taille de
  tuile, ratio de projection, nombre de couches **employées**. Tout vient du
  paramétrage ou du fournisseur d'apparence.

  Les **plafonds de structure** du moteur n'en relèvent pas, et la distinction
  compte : une clé de tri sur 64 bits est par nature un jeu de plafonds — seize
  couches, soixante-quatre étages, mille vingt-quatre planches —, et les annoncer
  dit ce que le format porte au lieu de décider à la place du jeu. La règle
  interdit de fixer combien de couches un jeu emploie, pas de borner ce que la clé
  sait distinguer.

Critère permanent : le hello world tient en moins de 30 lignes. S'il gonfle,
c'est qu'un défaut manque quelque part.

## Dépendances

**Le cœur du moteur n'en a aucune.** Bibliothèque standard seule, sans cgo, tests
compris, pour les paquets que le `Makefile` liste sous `SANS_DEPENDANCE` — c'est
cette liste que le contrôle applique, donc la seule qui ne puisse pas se périmer.
C'est là que la promesse se joue, et elle se vérifie paquet par paquet plutôt que
globalement.

Deux paquets périphériques font exception, sous licence permissive et par
décision inscrite dans le `Makefile` : le rendu de polices dans `appearance`, la
sortie audio dans `audio`. Les réécrire coûterait des semaines pour un résultat
qui ne distinguerait en rien le moteur.

Ce qui relève du sujet, en revanche, s'écrit quel qu'en soit le coût — le
protocole X11 demande aussi des semaines, et elles sont assumées : le déléguer
reviendrait à annuler un jalon. `docs/go.md` porte la liste et ce qu'une
candidature doit passer pour y entrer.

Conséquences assumées, qui ne se rediscutent pas :

- Accès système par les modules que le système fournit lui-même : `syscall` sous
  Windows, protocole X11 écrit sur socket sous Linux. Le critère n'est pas qu'aucune
  bibliothèque ne se charge — `os.Open` charge déjà `kernel32` sur toute machine
  Windows, donc le moteur le fait depuis qu'il lit un PNG —, c'est que **le moteur
  n'embarque aucune bibliothèque C tierce, n'exige aucun compilateur C et ne
  distribue rien à côté de son binaire**. `go.mod` vide, `CGO_ENABLED=0` : appeler
  `user32` est du même ordre qu'ouvrir un fichier.
- Pas de carte graphique, donc rendu logiciel en résolution interne basse.
- macOS, iOS et Android natifs fermés par l'ABI, et non par arbitrage de coût :
  Cocoa ne s'atteint que par l'ABI Objective-C, dont le point d'entrée s'obtient
  par `dlsym` ; Go refuse de lier pour iOS sans cgo ; le NDK et JNI sont des
  bibliothèques C. Tout chemin y demande donc cgo, et avec lui un compilateur C,
  ou bien une dépendance de plus pour s'en passer — et le critère ci-dessus refuse
  les deux. C'est là qu'il mord, là où une interdiction de charger quoi que ce soit
  aurait fermé Windows avec eux. Le backend web y donne accès dans un navigateur,
  et la frontière backend les garde atteignables si la contrainte change.

Si une fonctionnalité du moteur paraît exiger cgo ou un paquet hors de la liste
close, c'est la fonctionnalité qui est hors périmètre.

Une seule exception, et elle ne concerne pas le moteur : produire une
bibliothèque native en `c-shared` ou `c-archive` réclame cgo et un compilateur C
sur la machine de compilation, imposés par le runtime Go pour s'accrocher à
l'ABI C. Cette cible est optionnelle et séparée. `go.mod` reste vide, le code
reste identique, et les binaires des deux plateformes continuent de se
construire sans cgo. La règle porte sur ce que le moteur embarque, pas sur ce
qu'un artefact annexe exige au moment de le produire.

## Les backends s'écrivent depuis la spécification

Aucun backend ne se dérive du code d'un autre moteur. La source est la
spécification, jamais une implémentation :

| Backend | Ce qu'on lit |
| --- | --- |
| Windows | documentation Microsoft de `user32` et `gdi32` |
| X11 | spécification du protocole X.Org, `xproto.xml` comme description |
| Wayland | `wayland.xml`, qui *est* la description du protocole |
| DRM/KMS, `evdev` | en-têtes UAPI du noyau, sous `GPL-2.0 WITH Linux-syscall-note` |
| Web | spécifications W3C du canvas et de WebAudio |

Un numéro d'appel système, une constante d'`ioctl`, un opcode de protocole sont
des faits d'interface : les relever dans une documentation est légitime, et la
`Linux-syscall-note` existe précisément pour que du code non-GPL emploie les
en-têtes UAPI. Ce qui ne se reprend pas, c'est l'implémentation — découpage des
fonctions, structures internes, contournements, commentaires.

**Si un passage ne s'explique que par « c'est comme ça qu'un autre moteur fait »,
il est à réécrire.** Devant une difficulté de protocole, la réponse est dans la
spécification, pas dans un dépôt voisin.
