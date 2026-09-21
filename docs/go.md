# Conventions Go

Ce que le code doit respecter pour entrer dans le dépôt. La conception, elle, est
dans [`conception.md`](conception.md), et c'est elle qui fait foi sur les
décisions d'architecture.

## Langue

- Commentaires, godoc et messages d'erreur : français.
- Identifiants, noms de fichiers et de répertoires : français par défaut.
  L'anglais est admis quand c'est le terme technique naturel et sans équivalent
  usuel courant — `atlas`, `sprite`, `buffer`. Ne pas franciser de force ce que
  personne n'appelle autrement.
- Chaque type et chaque fonction exportés portent un godoc en français,
  commençant par le nom de ce qu'ils déclarent.
- **Bilingue, français et anglais** : ce qui s'adresse à quelqu'un qui ne connaît
  pas encore le projet — README, guide de contribution, politique de sécurité,
  journal des modifications. Chaque paire est tenue d'accord dans le même commit.
- **Français** : le reste de la documentation, dont ce document, la conception et
  la feuille de route. Un renvoi depuis un document anglais le signale.
- **Anglais** : les mentions de licence, qui suivent l'usage de leur domaine.

Les contributions rédigées en anglais sont les bienvenues et ne sont pas soumises
à la règle bilingue.

## Style

- Le code se lit comme du code écrit par quelqu'un d'expérimenté.
- Pas de commentaire qui paraphrase la ligne suivante. Un commentaire explique un
  pourquoi, un invariant, un piège. Jamais un quoi.
- Pas de bannière, pas de séparateur décoratif, pas d'emoji, ni dans le code, ni
  dans les messages de commit.
- Pas de défensive inutile : ne pas vérifier ce que le type garantit déjà.
- Règle de trois avant toute factorisation, sauf pour les utilitaires de
  formatage.
- `gofmt` et `go vet` propres en permanence.

## En-tête de fichier

Tout fichier source commence par :

```go
// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0
```

**Une ligne vide sépare l'en-tête de ce qui suit**, faute de quoi le copyright
devient la godoc du paquet.

La règle vaut hors du Go, avec la syntaxe de commentaire du format concerné.
Le périmètre est ce que le dépôt publie : un fichier que git ne suit pas n'a
rien à déclarer.

**La liste des dispensés est close** :

| Dispensé | Pourquoi |
|---|---|
| `.editorconfig`, `.gitattributes`, `.gitignore`, `Makefile` | outillage que le dépôt ne redistribue dans aucune archive |
| `go.mod` | réécrit par la chaîne Go, qui n'y préserverait rien |
| `LICENSE-MIT`, `LICENSE-APACHE`, `THIRD-PARTY-NOTICES` | ce sont les mentions de licence, elles ne s'en préfixent pas |
| les documents Markdown | ils s'adressent à un lecteur, pas à un compilateur ; la licence est déclarée en tête du README et dans les deux fichiers de licence |
| les images de `testdata/references/` | un PNG ne porte pas de commentaire, et ce sont des sorties du moteur, pas des sources |

Tout le reste en porte un, y compris un fichier d'une ligne. `make entetes` le
vérifie, et la cible entre dans `make check`.

Elle contrôle la présence des **deux** lignes dans les cinq premières du fichier,
sans figer l'année. La ligne vide qui suit n'est pas contrôlée : elle dépend du
format — un script porte son shebang avant l'en-tête — et une règle mécanique y
ferait plus de faux positifs que de prises.

## Documentation des déclarations

**Toute déclaration a sa documentation**, sans exception ni passe-droit. Types,
fonctions, méthodes, constantes, variables de paquet — exportées ou non — et les
fonctions de test comme le reste. Une méthode d'une ligne en a une aussi.

C'est la longueur qui s'ajuste, jamais la présence.

- **Une ligne quand la déclaration est évidente.** `// Fermer libère l'atlas.`
  suffit, et c'est la bonne réponse la plupart du temps.
- **Un paragraphe quand il y a une décision à retrouver dans six mois** : un
  arbitrage, une contrainte de protocole, un piège.
- **Pas de patron uniforme.** Une suite de commentaires bâtis sur le même moule
  est aussi lassante que leur absence.

La première ligne commence par le nom de ce qu'elle déclare et dit ce que c'est.
Elle ne paraphrase pas la signature : `// Projeter projette un point` n'apprend
rien ; dire ce que la fonction impose, ou ce qu'elle refuse, oui.

Ce qui vient après, quand il y a lieu, répond à quatre questions — jamais les
quatre à la fois :

- **Quelle alternative a été écartée, et ce qu'elle coûterait.** C'est la forme
  la plus utile, parce que c'est celle qu'on rouvrira.
- **Ce que la déclaration n'attrape pas.** Une garantie mal comprise est pire
  qu'une garantie absente.
- **Où s'arrête sa frontière**, quand elle en a une.
- **Des chiffres, pas des adjectifs.** « Deux ordres de grandeur au-dessus du
  plus grand lieu concevable » se vérifie ; « largement suffisant » non.

```go
// Cle est l'ordre de dessin d'un quad, réduit à un entier comparable.
//
// Composée du bit fort au bit faible : couche 4 bits, étage 6, profondeur en
// quarts de cellule 20, hauteur 16, planche 10, index stable 8. L'ordre des
// champs est l'ordre de priorité, et c'est tout le procédé — comparer deux
// clés revient à comparer les couches, puis les étages à couche égale, et
// ainsi de suite, sans une seule branche.
//
// Pas deux champs séparés pour la profondeur et la hauteur. Un comparateur
// les départagerait aussi bien, mais le tri se fait par paquets sur des
// entiers de 64 bits : une fonction appelée des milliers de fois par image
// coûte plus que tout le reste du tri réuni.
//
// Les 20 bits de profondeur fixent la limite annoncée du moteur, 32 000 cases
// de côté. Au-delà, deux cellules éloignées se replient sur la même valeur et
// s'ordonnent par leur index stable, c'est-à-dire par leur ordre d'insertion :
// le rendu resterait déterministe, et faux.
type Cle uint64
```

Un fichier peut commencer par un commentaire qui dit son rôle, séparé du
commentaire de paquet par une ligne vide — sans quoi il en deviendrait la godoc.

## Erreurs

- Erreurs enveloppées avec `%w` et un contexte utile, en français.
- Pas de `panic` dans le code de bibliothèque, sauf violation d'un invariant de
  programmation documenté comme tel.
- **Les fonctions du chemin de rendu ne renvoient pas d'erreur.** Les erreurs se
  produisent au chargement, pas à l'image.

## Performance

Le chemin de rendu tourne soixante fois par seconde sur des milliers de quads.
Dans `tri`, `raster` et les backends :

- Pas d'allocation par image. Les tampons sont réutilisés et remis à zéro par
  une remise à longueur nulle, jamais réalloués.
- Pas d'`interface{}`, pas de réflexion, pas de fermeture appelée par élément.
- Les tris se font sur des entiers de 64 bits, par paquets, pas par comparateur.
- `Quad` et `Rendu` gardent une disposition compatible C. La contrainte vient de
  l'export vers d'autres langages, et elle vaut avant même que la bibliothèque
  native existe.

Ailleurs, la lisibilité prime. Ne pas optimiser un chemin de chargement.

## Tests

### Non-régression visuelle

C'est le seul juge du rendu, et elle tourne sans écran : une scène scriptée, un
rendu par le rastériseur logiciel, une comparaison pixel à pixel avec une image
de référence versionnée.

- Les références vivent dans `testdata/references/`.
- Régénération explicite par `make references`, jamais automatique.
- **Toute régénération est justifiée dans le message de commit.** Une image de
  référence qui change sans raison énoncée est un défaut accepté par erreur.

Les scénarios couvrent en priorité ce qui casse en silence : chevauchements
cycliques, objets multi-cellules, étages, occlusion, entités à cheval sur deux
cellules, caméra dans les quatre orientations.

### Quand ajouter un scénario

Toute modification du tri, du rastériseur, de la projection ou de l'occlusion
s'accompagne d'un scénario de référence couvrant le cas traité — avant la
correction quand c'est un défaut.

Un défaut de rendu qui n'a pas laissé de scénario derrière lui reviendra.

### Tests unitaires

- `geometrie` : aller-retour monde vers écran et retour, sur le dimétrique,
  l'isométrie vraie et un ratio arbitraire. La même suite doit passer sur les
  trois.
- `tri` : ordre attendu sur des scènes construites à la main, et détection des
  cycles.
- Tables de cas, sans bibliothèque d'assertions — la règle de dépendance vaut
  aussi pour les tests.

### Bancs d'essai

Sur le tri et sur le rastériseur. Les chiffres de référence vivent dans le
commit qui les introduit, avec la machine sur laquelle ils ont été relevés :
sans elle, deux mesures ne se comparent pas.

Cible de contrôle : 5000 entités sur une carte de 500×500, et 800 entités de six
types changeant de direction sur la même image sans pic.

## Dépendances

**Le cœur du moteur n'a aucune dépendance.** `geometrie`, `scene`, `tri`, `rendu`
et `raster` n'emploient que la bibliothèque standard, sans cgo — tests compris.
C'est là que la promesse se joue, et `make deps` la vérifie paquet par paquet.

Ailleurs, une **liste close de deux entrées**, et aucune autre :

| Dépendance | Licence | Pour | À partir du |
| --- | --- | --- | --- |
| `golang.org/x/image` | BSD-3 | rastériser une police dans `apparence` | jalon 4 |
| `github.com/ebitengine/oto` | Apache-2.0 | sortie audio dans `audio` | jalon 5 |

### Ce qu'une dépendance doit passer pour entrer

**La licence d'abord, et elle est rédhibitoire.** BSD-2/3, MIT, Apache-2.0, ISC,
Zlib passent. GPL et AGPL, jamais : incompatibles avec du permissif. LGPL non
plus, Go liant statiquement — le relinking qu'elle exige devient impraticable.

**Ensuite, le sujet.** Si ce qu'elle fait appartient à ce que le moteur
revendique, on l'écrit, quel qu'en soit le coût. Le protocole X11 demande des
semaines et on le fera : c'est précisément le sujet du projet, et le déléguer
reviendrait à annuler un jalon.

**Sinon, c'est l'arbitrage du coût.** Des semaines de travail pour retrouver ce
qu'un paquet fait déjà, et qui ne distinguera jamais le moteur, c'est du temps
pris au moteur lui-même. Dans ce cas on prend le paquet, sans regret : un
rastériseur TrueType écrit à la main ne rendra pas le rendu isométrique meilleur.
Quelques heures de travail, en revanche, ne valent pas une ligne de `require`.

C'est cet arbitrage qui a retenu les deux entrées ci-dessus, et lui seul. La
liste n'est pas fermée par principe : elle s'ouvre à ce qui passe les mêmes
tests, par une décision inscrite dans le `Makefile` et relue en pull request.
C'est pour cela que c'est une liste blanche et non une interdiction — une
interdiction absolue finit par être contournée en silence.

Une entrée porte sa notice dans `THIRD-PARTY-NOTICES` dès le commit qui
l'importe.

### Ce qui reste écarté, et pourquoi

- **`x/sys`, `x/sync`, `x/text`** — chacun a son équivalent standard dans la table
  ci-dessous, et l'écart se comble en quelques heures. L'arbitrage du coût tombe
  du mauvais côté.
- **`purego`** — il supprime cgo, pas la dépendance à une bibliothèque C. Il
  déplace le problème au lieu de le résoudre.
- **`jezek/xgb`** — un client X11 tout fait. Il échoue sur le sujet : c'est le
  travail que le moteur revendique, et les semaines qu'il coûte sont assumées.

## Version minimale de Go

Le plancher de `go.mod` est le minimum réellement exigé par le code, jamais la
version installée sur le poste de développement. Depuis Go 1.21 cette ligne est
une exigence impérative : un consommateur dont la chaîne est plus ancienne doit
en télécharger une autre, ce qui suppose le réseau et l'accès au proxy de
modules, et échoue net en `GOTOOLCHAIN=local`.

**On ne le remonte que lorsqu'une fonctionnalité précise l'impose**, en le
justifiant dans le message de commit. Sans cette règle il dérive vers la version
du poste, et chaque dérive exclut des projets pour rien.

N'ayant aucune dépendance, le moteur choisit ce plancher librement — là où une
bibliothèque qui en a subit celui du plus exigeant de ses modules. C'est l'un des
bénéfices concrets de la règle ci-dessus, et il vaut mieux ne pas le dilapider.

L'intégration continue éprouve les deux bouts de la plage annoncée : le plancher
tel que `go.mod` le déclare, et la dernière version publiée.

| Besoin | Ce qu'on fait à la place |
| --- | --- |
| Fenêtre et entrées Windows | appels système directs sur `user32` et `gdi32` |
| Présentation Windows | `StretchDIBits`, qui prend en charge la mise à l'échelle entière, donc sans passe supplémentaire côté moteur |
| Fenêtre et entrées Linux | protocole X11 écrit sur socket |
| Authentification X11 | lecture du fichier d'autorisation, cookie magique |
| Transfert d'image X11 | `PutImage`, mémoire partagée quand elle est là |
| Audio Windows | `winmm`, puis l'interface audio système |
| Audio Linux | protocole du serveur de son sur socket, jamais ALSA |
| Décodage PNG | `image/png` de la bibliothèque standard |
| Comparaison d'images en test | code du dépôt, pas de bibliothèque d'assertions |

Si une fonctionnalité paraît exiger une liaison native, c'est la fonctionnalité
qui est hors périmètre.
