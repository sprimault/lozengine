# Construction

## En local

```
make build        # compile tous les paquets
make example      # lance le plus petit exemple, dans une fenêtre
make test         # la suite de non-régression, sans écran
make check        # ce qui doit passer avant un commit
make fmt          # échoue si un fichier n'est pas au format
make vet
make deps         # échoue si une dépendance externe est entrée
make entetes      # la mention de licence, sur ce que le dépôt publie
make frontiere    # le noyau ne touche ni au système ni à un backend
make procedures   # aucune procédure système appelée sans garde de présence
make liens        # aucun lien de godoc qui ne se résoudra pas
make journal      # chaque section du CHANGELOG dit la même chose dans ses deux langues
make surface      # refuse un ajout à la surface publique sans entrée de journal
make hello        # le hello world tient sous trente lignes de code
make cross        # compile pour chacun des couples de CIBLES
make floor        # le contrôle complet sous le plancher du module
make container    # le même, dans un conteneur, au plus près du runner
make container-clean   # vide le cache de compilation du conteneur
make cover        # couverture, écrite dans .tmp/couverture.html
make bench        # bancs d'essai, un profil par paquet
make bancs        # ce qu'aucun banc ne traverse, sur le chemin par image
make references   # régénère les images de référence
make notes VERSION=0.1.0   # les notes d'une version, tirées du CHANGELOG
make titre VERSION=0.1.0   # le titre d'une version, que la publication réclame
make clean
```

`PKG` restreint la portée de `test`, `bench` et `cover` à un paquet :
`make test PKG=./raster`. `RUN` restreint `test` à un motif de nom. `EXAMPLE` choisit
l'exemple que `example` lance : `make example EXAMPLE=walk`.

Les exemples ouvrent une fenêtre, donc aucune cible de contrôle ne les lance. Ils
entrent en revanche dans `./...`, si bien que `build` et `cross` les compilent : un
exemple qui cesse de compiler fait échouer `check` comme le reste.

`bench` mesure un paquet à la fois et nomme ses profils d'après lui, parce que la
chaîne Go refuse `-cpuprofile` dès qu'il y a plus d'un paquet. `references` ne
s'adresse qu'au paquet qui porte la suite visuelle : un drapeau de test n'existe
que pour le paquet qui le déclare, et les autres échoueraient sur un drapeau
inconnu.

## Prérequis

**Le moteur se construit avec une chaîne Go seule.** `go build ./...` et
`go test ./...` suffisent, sur les deux plateformes. Pas de compilateur C, pas
de bibliothèque de développement, pas d'outil à poser avant de commencer.

**Les cibles `make` demandent en plus GNU make, git et un shell POSIX** — sous
Windows, celui de Git pour Windows ou WSL. Elles ne font rien que `go` ne sache
faire, mais elles tiennent les réglages et les périmètres à un seul endroit :
une commande tapée directement les perd, et l'oubli ne se voit pas dans la
sortie.

Git n'est pas là par confort. Plusieurs contrôles tirent leur périmètre de ce
que le dépôt publie, et `make check` refuse de s'exécuter hors d'un dépôt plutôt
que de passer au vert sur une liste vide.

## Ce que `make check` vérifie, et pourquoi

```
make check
```

Les étapes sont celles que le bloc de commandes ci-dessus énumère, et c'est le
seul endroit du dépôt qui les liste : la phrase récapitulative qui vivait ici les
comptait **neuf** quand il y en avait onze, et trois autres textes portaient leur
propre version du même décompte. **La liste se passe entière**, jamais réduite à
ce qui touche au changement qu'on vient d'écrire : composer sa liste revient à ne
vérifier que ce qu'on a déjà en tête, et le défaut est ailleurs par construction.

Cinq de ces étapes méritent un mot, parce qu'elles vérifient autre chose que ce
que leur nom laisse croire.

**`make deps`** est la règle du dépôt rendue exécutable. Elle liste les
dépendances transitives et échoue dès que l'une d'elles n'appartient pas à la
bibliothèque standard, en interrogeant la chaîne Go plutôt qu'en devinant
d'après la forme du chemin.

La distinction compte, parce que la bibliothèque standard embarque des paquets
dont le chemin ressemble à celui d'une dépendance externe :
`vendor/golang.org/x/net/dns/dnsmessage`, que `net` tire — et le backend Linux
repose sur `net`. Un contrôle qui jugerait sur la forme du chemin virerait au
rouge sur du code parfaitement conforme, et la seule issue serait de
l'affaiblir.

Elle inclut les dépendances de test, parce que c'est là que l'entorse est
tentante, et elle interroge Windows, Linux et le navigateur, parce qu'un import
placé sous contrainte de construction ne se voit pas depuis un autre système.

**`make frontiere`** vérifie que le noyau ne touche ni au système ni à un
backend. Aucun paquet parmi `geometry`, `scene`, `sorting`, `render`, `raster`,
`input` et `appearance` ne doit importer `syscall`, `net`, `os/exec` ou un
backend — ni directement, ni à travers un autre paquet du moteur, et leurs
sous-paquets comptent, `syscall/js` comme `net/http`.

C'est le contrôle que `make cross` ne fait pas, contrairement à ce qu'on croit
spontanément. Compiler pour une plateforme ne dit rien de l'étanchéité :
le backend Linux est du protocole écrit sur socket, donc il n'emploie que `net`
et `os`, qui existent des deux côtés. Un `net.Dial` placé dans `geometry`
compile pour Windows comme pour Linux, sans un mot.

Seuls les imports écrits comptent, suivis de proche en proche à travers les
paquets du moteur. Prendre les dépendances transitives complètes ne marcherait
pas : `fmt` tire `os`, qui tire `syscall`, si bien que le moindre paquet
deviendrait une fuite.

**`make journal`** vérifie que chaque section du `CHANGELOG` dit la même chose dans
ses deux langues : les mêmes catégories de part et d'autre du séparateur, et le même
nombre d'entrées dans chacune. Une entrée écrite dans une seule langue est un oubli,
et les notes de publication reprenant la section telle quelle, l'oubli partirait avec
elles.

Ce qu'elle compare est le **nombre** d'entrées par catégorie et non leur nom : une
section de trois entrées puis une donne la même signature des deux côtés, quelle que
soit la langue. Elle attrape donc une catégorie oubliée et une entrée oubliée, mais
elle ne dit pas qu'une traduction est juste — cela se relit.

**`make procedures`** garde ce qui décide si le moteur démarre sur un Windows
ancien. Une procédure système appelée sans vérifier qu'elle existe fait paniquer
le programme au lancement : le défaut a vécu trois jalons sur
`SetProcessDpiAwarenessContext`, et il rendait le moteur inutilisable sur tout
Windows antérieur à la mise à jour d'avril 2017 de Windows 10. La cible exige que
les procédures de sa liste passent par la garde, et elle compte les usages au lieu
de constater une absence — ce qui attrape celle qu'on ajoutera plus tard sans y
penser.

**`make cross`** n'est pas là pour produire un binaire ; les artefacts sont
jetés. Elle vérifie que le code compile sans compilateur C pour chaque couple de
la variable `CIBLES`, ce qui attrape l'usage d'une API absente d'un système ou
d'une architecture.

**`CIBLES` est la liste des promesses tenues, pas celle du possible**, et c'est
elle qui fait foi : les deux `README` la reflètent, et un contrôle le vérifie dans
les deux sens. L'énumération ne se recopie donc pas ici.

« Compile » est tout ce qu'elle prouve : « tourne » et « reste vert en intégration
continue » sont deux autres promesses. **Elles valent pour `windows/amd64`,
`linux/amd64` et `linux/arm64`** — cette dernière depuis la `0.5.8`, et c'est
l'architecture qui compte, puisqu'elle fusionne les multiplications-additions là
où `amd64` ne le fait pas : elle est la seule à pouvoir rendre une autre image, et
la suite y compare les pixels. D'autres couples compilent aussi —
`darwin/amd64` et `darwin/arm64` ; ils n'entrent ici que si un texte du dépôt les
annonce, faute de quoi la cible vérifierait plus que ce que le dépôt promet et
personne ne saurait lequel des deux croire.

**Une seule cible échoue, et le motif est à connaître avant de publier** : `ios/arm64`
rend « requires external (cgo) linking, but cgo is not enabled ». Ce n'est pas une
limite du moteur mais un refus de Go à la liaison, et aucune ligne de code du dépôt ne
le lèverait — d'où le hors-périmètre de la feuille de route. Mesuré le 26 octobre
2026 : le module entier se construit pour les deux `darwin`, et seul `ios/arm64` tombe.

## Compiler sans cgo

Le moteur se construit avec cgo désactivé sur les deux cibles, et le Makefile
l'impose plutôt que de s'en remettre à l'environnement.

```
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build ./...
```

Une compilation croisée prouve que le code compile pour l'autre plateforme,
jamais qu'il y fonctionne. Les appels système, le protocole X11 et le mixeur
audio ne se jugent que nativement : un changement qui touche un backend se
vérifie sur la plateforme concernée avant d'être livré.

## La cible d'interopérabilité

Elle vient au dernier jalon et reste à part. Produire une bibliothèque native en
`c-shared` ou `c-archive` réclame cgo et un compilateur C sur la machine de
compilation, imposés par le runtime Go pour s'accrocher à l'ABI C.

C'est la seule exception à la règle de dépendance, et elle ne met rien dans le
moteur : `go.mod` reste vide, le code reste identique, et les binaires des deux
plateformes continuent de se construire sans cgo. La règle porte sur ce que le
moteur embarque, pas sur ce qu'un artefact annexe exige au moment de le produire.

## Publier une version

Une bibliothèque Go ne publie aucun binaire : `go get` lit le dépôt au tag
demandé. Publier, c'est donc poser un tag `vX.Y.Z` sur `master` — le reste est
automatique.

Le numéro ne se choisit pas : **le journal des modifications suit la feuille de
route**, et le mineur marque un jalon franchi. Une version se publie quand un
jalon se termine, pas quand on en a envie.

Le workflow rejoue les contrôles sur le commit taggé plutôt que de les supposer
passés, puis tire les notes de la version de `CHANGELOG.md` :

```
make notes VERSION=0.1.0
```

**Une section absente arrête la publication.** C'est le seul moment où quelqu'un
relit ce qui change, et une version qui n'en porte pas ne le dit à personne. La
cible étant la même en local et dans le workflow, elle s'essaie avant de poser le
tag.

Un déclenchement manuel fait le même travail sans rien publier : il affiche les
notes et s'arrête là.

## Les images de référence

La suite de non-régression compare le rendu à des images versionnées dans
`testdata/references/`. Elles ne se régénèrent jamais toutes seules :

```
make references
```

Une régénération se justifie dans le message de commit. Une image qui change
sans raison énoncée est un défaut accepté par erreur, et il faudra l'avoir vu
pour le corriger.
