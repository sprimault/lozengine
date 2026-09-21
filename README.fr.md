# Lozengine

English: [README.md](README.md)

Un moteur de rendu isométrique 2D en Go, sans aucune dépendance externe. Rendu
logiciel, sans GPU : on lui décrit une scène, il en sort une liste de quads
projetés et triés, qu'il rastérise lui-même ou qu'un hôte pousse dans son propre
pipeline.

MIT ou Apache-2.0, au choix — voir [`LICENSE-MIT`](LICENSE-MIT) et
[`LICENSE-APACHE`](LICENSE-APACHE). Sauf mention contraire de son auteur, toute
contribution proposée à l'inclusion est placée sous ces deux mêmes licences, sans
condition supplémentaire.

Le nom vient de *lozenge*, le losange en anglais, fusionné avec *engine* : la
cellule projetée est la forme que le moteur manipule de bout en bout.

La projection est une matrice affine paramétrable, gardée avec son inverse. Le
dimétrique est le sujet du moteur, mais l'isométrie vraie et la grille carrée vue
de dessus sont des constructeurs de plus, pas un second chemin de rendu.

La règle de dépendance se prend au mot. `go.mod` ne contient aucune ligne
`require`, et cela vaut pour les tests, les exemples et l'outillage. La fenêtre,
les entrées et le son passent par `syscall` sous Windows et par les protocoles
X11 et PulseAudio écrits sur socket sous Linux, jamais par le chargement d'une
bibliothèque C. Le rendu est donc logiciel, en résolution interne basse remontée
vers la fenêtre par un facteur entier.

## La frontière qui porte tout

Le noyau produit un `Rendu` : une liste de quads déjà projetés, cullés, triés et
groupés par planche. Rien dans cette structure ne connaît Windows, X11 ou un
format d'image.

Un backend traduit cette liste et rien d'autre. S'il doit trier, projeter ou
consulter la scène, la frontière est mal placée, et le correctif est dans le
noyau. C'est ce qui permet au rastériseur logiciel et aux backends système de
consommer exactement la même chose — et donc à la suite de non-régression de
juger le rendu sans jamais ouvrir de fenêtre.

`Quad` et `Rendu` restent en disposition compatible C : champs de taille fixe,
tableau contigu, ni chaîne ni interface, et aucune fonction de rappel dans le
chemin de rendu. Le noyau est destiné à être consommé depuis Rust, C++ ou tout
langage parlant l'ABI C ; un `string` dans `Quad` imposerait une conversion par
quad et par image, ce qui rendrait cette exposition inutilisable.

## Deux portes d'entrée

Un seul paquet à importer, et deux façons de s'en servir.

Pour écrire un jeu en Go, le moteur tient la fenêtre, la boucle à pas fixe et les
entrées : il ne reste qu'une méthode à écrire. Pas de méthode de dessin, parce que
l'ordre vient de la clé de tri ; pas de méthode de mise en page, parce que
l'échelle vient du facteur entier. C'est ce que rapporte une scène décrite plutôt
que dessinée.

Pour intégrer le moteur dans un jeu qui a déjà sa boucle, les mêmes opérations
s'appellent une par une, sans que rien ne réclame de fenêtre ni de thread. La
boucle fournie ajoute du confort, jamais de capacité : tout ce qu'elle permet se
fait aussi en pilotant le moteur à la main.

Tout le reste vit sous `internal/`, que le compilateur interdit d'importer depuis
un autre module. Ce qui est publié l'a donc été délibérément — et un hôte ne peut
pas se lier à un backend en contournant la frontière.

## Ce qu'il ne fait pas

Le moteur ne connaît aucune notion de jeu — ni joueur, ni inventaire, ni score —
et ne code en dur rien qui en relève : le nombre de directions d'une apparence,
la taille d'une tuile, le ratio de projection et le nombre de couches viennent
tous du paramétrage ou du fournisseur d'apparence.

Trois exclusions suivent de la règle de dépendance, et elles sont assumées :

- **Pas d'accélération GPU.** Le rendu est logiciel, et la résolution interne
  basse est ce qui le rend tenable.
- **Pas de macOS.** Impossible sans dépendance ni compilateur C. La frontière
  backend le garde atteignable si la contrainte change.
- **Pas de Wayland natif** pour l'instant. XWayland couvre le besoin.

Ce qui sera exposé aux autres langages est le noyau, jamais la fenêtre ni la
boucle : un jeu hôte possède déjà les siennes et ne cédera pas son thread
principal.

## État

**Jalon 0 en cours. Le moteur ne rend rien d'utilisable pour l'instant**, et
rien n'est publié : les signatures publiques ne gèlent qu'une fois le moteur
validé par un second jeu, au jalon 7.

La feuille de route compte dix jalons, chacun avec un critère de fin vérifiable.

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
make test      # la suite de non-régression, qui tourne sans écran
make check     # ce qui doit passer avant un commit
make frontiere # le noyau ne touche ni au système ni à un backend
make cover     # couverture, ouverte dans le navigateur
make bench     # bancs d'essai, un profil par paquet
make clean
```

`make check` enchaîne le formatage, `go vet`, le contrôle d'absence de
dépendance externe, la mention de licence, l'étanchéité de la frontière, la
compilation croisée vers Windows et Linux, puis les tests.

Le contrôle d'étanchéité est celui qui garde l'architecture : aucun paquet du
noyau ne doit importer `syscall`, `net` ou un backend, ni directement ni à
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
