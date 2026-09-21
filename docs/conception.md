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
lui-même une clé de tri, des étages, des objets à cheval sur deux cellules et le
démêlage des chevauchements cycliques.

Lozengine prend ce travail à sa charge. On lui décrit ce qui existe et où ; il
décide de l'ordre. Tout le reste de la conception en découle.

Conséquence directe sur la forme de l'API : une scène se **décrit**, elle ne se
dessine pas. Il n'existe pas de fonction de dessin publique prenant une position
d'écran, parce qu'elle court-circuiterait la seule chose que le moteur apporte.

## Repère et projection

- **Monde distinct de l'écran.** Cellules flottantes, `x` et `y` dans la grille,
  `z` en hauteur au-dessus du sol, `y` croissant vers le bas-droite de l'écran.
- **Ancre.** Point de contact au sol, centre de case par défaut. Surchargeable
  par apparence, jamais par appel.
- **Hauteur.** `z` reste un champ séparé, jamais fondu dans la matrice de
  projection. L'écran est la projection de `(x, y)` moins `z` fois un facteur ;
  la clé de tri se calcule sur `(x+y)` puis `z`. Fondre les deux perd
  l'information à l'entrée du pipeline et rend le tri irréparable.
- **Projection.** Matrice affine avec son inverse calculée une fois et gardée.
  Constructeurs pour le dimétrique et pour l'isométrie vraie, plus la matrice
  brute pour les cas exotiques. Aucune constante de ratio ailleurs que dans un
  constructeur de matrice : pas de division par deux disséminée dans le code.
- **Caméra.** Quatre orientations discrètes à 90°. Elles touchent la matrice, la
  permutation des axes dans la clé de tri, et le décalage de direction demandé
  au fournisseur d'apparence.
- **Échelle.** Facteur entier obligatoire vers la fenêtre, bandes autour quand
  la taille ne tombe pas juste. Jamais de mise à l'échelle fractionnaire.

La projection étant une matrice paramétrable, une grille carrée vue de dessus
est un constructeur de plus et non un second moteur. Le dimétrique reste le
sujet ; le reste vient avec.

## Structure et interopérabilité

- **Primitive unique.** Tout converge vers le quad trié. Les fonctions de
  confort sont du sucre au-dessus, jamais un second chemin de rendu. Une couche
  d'interface est une couche haute de la clé de tri, pas une exception.
- **Options en structure.** Valeur zéro utilisable. Tout ce qui n'est pas
  renseigné a un défaut raisonnable.
- **Disposition compatible C.** `Quad` et `Rendu` n'emploient que des champs de
  taille fixe, dans un tableau contigu. Pas de chaîne, pas d'interface, pas de
  slice imbriquée, pas de pointeur vers de la mémoire gérée.
- **Aucun rappel dans le chemin de rendu.** L'apparence est résolue en amont,
  jamais par une fonction appelée pendant le rendu.

Les deux dernières viennent de l'exposition du noyau aux langages parlant l'ABI
C, et elles s'appliquent bien avant que cette exposition existe. Une chaîne dans
`Quad` imposerait une conversion par quad et par image, ce qui la rendrait
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

- Résolution interne : 480×270. Tuile de sol 32×16, personnage debout autour de
  24 pixels.
- Directions d'apparence : déclarées par le fournisseur, jamais codées en dur.
  Huit pour l'implémentation de référence en planches PNG, seize pour la
  fabrique volumique. Le littéral n'apparaît que dans le fournisseur.
- Clé de tri : un entier de 64 bits composé, du bit fort au bit faible — couche
  4 bits, étage 6, profondeur `x+y` 20 en quarts de cellule, `z` 16, planche 10,
  index stable 8.

Le tri topologique exprime son ordre partiel en ajustant l'index stable ; la clé
reste la seule vérité de l'ordre de dessin. Limite documentée qui en découle :
32 000 cases de côté.

## Frontières de paquets

Le sens des dépendances ne s'inverse jamais.

```
lozengine             surface publique, boucle, assemblage      tous les internes
internal/geometrie    projection, matrices, conversions         aucune
internal/scene        grille, entités, couches, caméra          geometrie
internal/tri          culling, clé, tri topologique             geometrie, scene
internal/rendu        structures Quad et Rendu                  geometrie
internal/raster       rastériseur logiciel, écriture PNG        rendu
internal/apparence    interface fournisseur, atlas, animation   geometrie, rendu
internal/backend/…    fenêtre, entrées, présentation            rendu
internal/audio        mixeur et sorties système                 aucune
```

Le paquet racine est le seul à tout connaître, et rien ne dépend de lui.

**Tout le moteur vit sous `internal/`, et c'est une décision, pas un rangement.**
Le compilateur Go interdit d'importer un tel paquet depuis un autre module : la
surface publique se réduit donc à ce que le paquet racine republie, et publier
quelque chose demande de l'écrire, au lieu de l'être par défaut. C'est ce qui rend
tenable la règle voulant que toute signature publique soit définitive.

Deux conséquences qui comptent davantage que la discipline. Réorganiser l'intérieur
ne casse aucun consommateur, tant que ce que republie le paquet racine ne bouge pas.
Et un hôte ne peut plus importer un backend pour se lier directement à X11 : ce
n'est plus un contrôle qui l'interdit, c'est le compilateur.

Les types du noyau sont republiés par **alias** et non par enveloppe. Un alias
désigne le même type, sans conversion ni copie — ce qui est indispensable pour des
structures qu'un hôte reçoit par milliers à chaque image, et qui doivent garder
leur disposition compatible C jusqu'à la frontière.

`scene` et `tri` ne connaissent aucun backend. `raster` et `backend` ne
connaissent pas `scene` : ils consomment un `Rendu` déjà trié. Si un backend a
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
  tuile, ratio de projection, nombre de couches. Tout vient du paramétrage ou du
  fournisseur d'apparence.

Critère permanent : le hello world tient en moins de 30 lignes. S'il gonfle,
c'est qu'un défaut manque quelque part.

## Zéro dépendance

`go.mod` ne contient aucune ligne `require`. Bibliothèque standard uniquement,
sans cgo, et cela vaut pour les tests, les exemples et l'outillage.

Conséquences assumées, qui ne se rediscutent pas :

- Accès système par appel direct : appels système documentés sous Windows,
  protocole X11 écrit sur socket sous Linux. Jamais de chargement de
  bibliothèque C, sur aucune plateforme.
- Pas de carte graphique, donc rendu logiciel en résolution interne basse.
- macOS impossible sans compromis, donc écarté.

Si une fonctionnalité paraît exiger une liaison native, c'est la fonctionnalité
qui est hors périmètre.

Une seule exception, et elle ne concerne pas le moteur : produire une
bibliothèque native en `c-shared` ou `c-archive` réclame cgo et un compilateur C
sur la machine de compilation, imposés par le runtime Go pour s'accrocher à
l'ABI C. Cette cible est optionnelle et séparée. `go.mod` reste vide, le code
reste identique, et les binaires des deux plateformes continuent de se
construire sans cgo. La règle porte sur ce que le moteur embarque, pas sur ce
qu'un artefact annexe exige au moment de le produire.
