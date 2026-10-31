# Lozengine

English: [README.md](README.md)

Un moteur de rendu isométrique 2D en Go, dont le cœur n'a aucune dépendance.
Rendu logiciel, sans GPU : on lui décrit une scène, il en sort la liste des
rectangles à peindre — des *quads* —, déjà placés à l'écran et rangés dans l'ordre
de dessin. Il les peint lui-même, ou un hôte les pousse dans son propre pipeline.

MIT ou Apache-2.0, **au choix de qui emploie le moteur** : l'une des deux suffit,
elles ne se cumulent pas — voir [`LICENSE-MIT`](LICENSE-MIT) et
[`LICENSE-APACHE`](LICENSE-APACHE). Les annuaires de paquets les affichent
souvent côte à côte sans le « ou », ce qui est une façon de les indexer et non une
exigence double. Sauf mention contraire de son auteur, toute contribution proposée
à l'inclusion est placée sous ces deux mêmes licences, sans condition
supplémentaire.

Le nom vient de *lozenge*, le losange en anglais, fusionné avec *engine* : la
cellule projetée est la forme que le moteur manipule de bout en bout.

La projection est une matrice affine paramétrable, gardée avec son inverse. Le
dimétrique est le sujet du moteur, mais l'isométrie vraie et la grille carrée vue
de dessus sont des constructeurs de plus, pas un second chemin de rendu.

La règle de dépendance se prend au mot là où elle compte. **Géométrie, scène, tri,
structures et rastériseur — celui qui peint les pixels un à un, là où une carte
graphique le ferait — n'emploient que la bibliothèque standard**, tests compris, et
un contrôle le vérifie paquet par paquet. La fenêtre, les entrées et
le son passent par `syscall` sous Windows et par les protocoles X11 et
PulseAudio écrits sur socket sous Linux : **aucune bibliothèque C tierce, aucun
compilateur C, un `go.mod` vide**. Le rendu est donc logiciel, en résolution
interne basse remontée vers la fenêtre par un facteur entier.

**Les plateformes, et ce que chacune promet.** Compile, tourne, ou reste vert en
intégration continue : trois promesses différentes, et seule celle qui est tenue
est annoncée. Windows et Linux en `amd64`, et Linux en `arm64` — la suite de tests
y passe en intégration continue, à chaque lot, et sur `arm64` elle vérifie en plus
que la même scène rend exactement les mêmes pixels qu'ailleurs, ce qui est la
promesse du calcul en nombres entiers. `windows/arm64`, `windows/386`,
`linux/arm`, `linux/386` et `js/wasm` — ils **compilent**, ce que `make check`
vérifie, et rien de plus tant que le parc ne contient ni Raspberry Pi, ni
téléphone. C'est le zéro-dépendance qui ouvre cette liste, et une dépendance cgo
la refermerait au lieu de l'élargir : `go build` ne suffirait plus.

Le navigateur est la troisième cible, par WebAssembly, et c'est lui qui donnera
Android et iOS. Le moteur n'ouvre pas de fenêtre **native** sur macOS ni sur iOS, et
ce n'est pas la même chose qu'y être inaccessible : la bibliothèque native prévue par
la feuille de route le laisse consommer par une application qui fournit la sienne.
[`ROADMAP.md`](ROADMAP.md) en donne le détail, jalon par jalon.

Deux paquets périphériques font exception, sous licence permissive et par
décision inscrite : le rendu de polices et la sortie audio. Les réécrire
coûterait des semaines pour un résultat qui ne distinguerait en rien le moteur.
Le détail est dans [`docs/go.md`](docs/go.md).

## La frontière qui porte tout

Le noyau produit une `DrawList` : la liste des quads déjà placés, rangés dans l'ordre
de dessin, ceux qui sortent de l'écran en ayant été retirés. Rien dans cette structure
ne connaît Windows, X11 ou un format d'image.

Un backend traduit cette liste et rien d'autre. S'il doit trier, projeter ou
consulter la scène, la frontière est mal placée, et le correctif est dans le
noyau. C'est ce qui permet au rastériseur logiciel et aux backends système de
consommer exactement la même chose — et donc à la suite de non-régression de
juger le rendu sans jamais ouvrir de fenêtre.

`Quad` et `DrawList` restent en disposition compatible C : champs de taille fixe,
tableau contigu, ni chaîne ni interface, et aucune fonction de rappel dans le
chemin de rendu. Le noyau est destiné à être consommé depuis Rust, C++ ou tout
langage parlant l'ABI C ; un `string` dans `Quad` imposerait une conversion par
quad et par image, ce qui rendrait cette exposition inutilisable.

## Deux portes d'entrée

Un seul paquet à importer, et deux façons de s'en servir.

Pour écrire un jeu en Go, `Run` tient la fenêtre, la boucle à pas fixe et les
entrées : il ne reste qu'une méthode à écrire, `Game.Step`. Pas de méthode de dessin,
parce que l'ordre vient de la clé de tri ; pas de méthode de mise en page, parce que
l'échelle vient du facteur entier. C'est ce que rapporte une scène décrite plutôt
que dessinée.

Pour intégrer le moteur dans un jeu qui a déjà sa boucle, `NewEngine` rend la même
capacité sans fenêtre ni horloge, et l'hôte appelle un pas par image. La première
porte ajoute du confort, jamais de capacité : tout ce qu'elle permet se fait aussi en
pilotant le moteur à la main.

**La fenêtre du moteur ne se pilote pas depuis l'extérieur** : rien ne permet de la
redimensionner, de la déplacer ni de la passer en plein écran, et rien ne le
permettra. Un jeu qui veut sa propre fenêtre a déjà la seconde porte, où le moteur
n'en ouvre aucune.

Six paquets sont publics parce qu'on doit pouvoir lire la documentation de leurs
types : `render`, `geometry`, `scene`, `raster`, `input` et `appearance`. Tout le reste
vit sous `internal/`, que le compilateur interdit d'importer depuis un autre module —
un hôte ne peut donc pas se lier à un backend en contournant la frontière. Le paquet
racine republie l'essentiel, pour qu'un jeu n'ait qu'un seul import à écrire.

## Ce qu'il ne fait pas

Le moteur ne connaît aucune notion de jeu — ni joueur, ni inventaire, ni score —
et ne code en dur rien qui en relève : le nombre de directions d'une apparence,
la taille d'une tuile et le ratio de la projection viennent du paramétrage ou du
fournisseur d'apparence.

Deux exclusions suivent de la règle de dépendance, et elles sont assumées :

- **Pas d'accélération GPU.** Le rendu est logiciel, et la résolution interne
  basse est ce qui le rend tenable.
- **Pas de fenêtre sous macOS.** Le cœur y compile et y calcule ses images, donc
  un hôte qui tient sa propre boucle s'en sert déjà ; c'est la fenêtre qui
  manque, Cocoa ne s'atteignant que par l'ABI Objective-C, dont le point d'entrée
  demande un compilateur C ou une dépendance de plus. iOS va plus loin : Go
  lui-même refuse de lier sans cgo. Le jalon 5 couvre les deux par le navigateur,
  et la frontière backend les garde atteignables si la contrainte change.
Wayland natif n'en est pas une, et c'était une erreur de le ranger là : c'est un
protocole qu'on écrit sur une socket comme X11, donc il vient à son jalon et non
jamais. XWayland couvre le besoin jusque-là.

Ce qui sera exposé aux autres langages est le noyau, jamais la fenêtre ni la
boucle : un jeu hôte possède déjà les siennes et ne cédera pas son thread
principal.

## État

Un jeu s'écrit. Le moteur ouvre une fenêtre, lit le clavier et la souris — boutons,
molette, et la case que le pointeur désigne —, tient une boucle à pas fixe et n'attend
du jeu qu'une seule méthode : décrire sa scène, lui se chargeant de l'ordre de dessin
et de l'échelle. Sa suite de non-régression compare chaque image à une référence, sans
écran.

Le backend Linux est éprouvé à chaque lot sous deux serveurs sans écran : le protocole
est juste et les pixels exacts, mais personne ne l'a encore vu sur un écran réel.
`make example` ouvre le plus petit des trois exemples.

**Le jalon courant et ce que chaque version a apporté** se lisent dans la
[feuille de route](ROADMAP.md) et le [journal](CHANGELOG.md) — les répéter ici ferait un
troisième texte à tenir d'accord.

Une version marque chaque jalon franchi, mais **aucune signature n'est figée** :
elles peuvent changer à chaque mineure jusqu'au jalon 9, où la validation du
moteur par un second jeu les gèle.

La feuille de route compte douze jalons, chacun avec un critère de fin vérifiable.

- [`ROADMAP.md`](ROADMAP.md) — les jalons, leurs critères de fin, et ce qui est
  hors périmètre
- [`CHANGELOG.md`](CHANGELOG.md) — ce que chaque version a apporté
- [`docs/conception.md`](docs/conception.md) — les décisions d'architecture, qui
  font foi
- [`docs/go.md`](docs/go.md) — conventions de code et doctrine de test
- [`docs/construction.md`](docs/construction.md) — les cibles et ce qu'elles
  vérifient
- [`CONTRIBUTING.fr.md`](CONTRIBUTING.fr.md) — ce qui se discute avant d'être
  écrit, et ce sur quoi une contribution est jugée

## Construction

```
make build     # compile tous les paquets
make example   # lance le plus petit exemple, dans une fenêtre
make test      # la suite de non-régression, qui tourne sans écran
make check     # ce qui doit passer avant un commit
make frontiere # le noyau ne touche ni au système ni à un backend
make cover     # couverture, écrite dans .tmp/couverture.html
make bench     # bancs d'essai, un profil par paquet
make clean
```

`make check` enchaîne le formatage, `go vet`, le contrôle d'absence de
dépendance externe, la mention de licence, l'étanchéité de la frontière,
l'équilibre des deux langues du journal, la surface publique face au journal, la
compilation croisée vers les neuf couples annoncés, puis les tests.

Le contrôle d'étanchéité est celui qui garde l'architecture : aucun paquet du
noyau ne doit importer `syscall`, `net`, `os/exec` ou un backend, ni directement ni à
travers un autre paquet du moteur. La compilation croisée ne le remplace pas —
le backend Linux n'emploie que des paquets qui existent aussi sous Windows, si
bien qu'une fuite compilerait des deux côtés sans un mot.

**Le moteur se construit avec une chaîne Go seule** : `go build ./...` et
`go test ./...` suffisent, sur les deux plateformes. Pas de compilateur C, pas
de bibliothèque de développement, rien à poser avant de commencer.

Les cibles ci-dessus demandent en plus GNU make, git et un shell POSIX — sous
Windows, celui de Git pour Windows ou WSL. Elles ne font rien que `go` ne sache
faire ; elles tiennent les réglages et les périmètres à un seul endroit.

[`docs/construction.md`](docs/construction.md) détaille les cibles et ce que
chacune vérifie réellement.
