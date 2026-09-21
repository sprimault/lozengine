# Construction

## En local

```
make build        # compile tous les paquets
make test         # la suite de non-régression, sans écran
make check        # ce qui doit passer avant un commit
make fmt          # échoue si un fichier n'est pas au format
make vet
make deps         # échoue si une dépendance externe est entrée
make entetes      # la mention de licence, sur ce que le dépôt publie
make frontiere    # le noyau ne touche ni au système ni à un backend
make cross        # compile pour Windows et pour Linux
make cover        # couverture, ouverte dans le navigateur
make bench        # bancs d'essai, un profil par paquet
make references   # régénère les images de référence
make clean
```

`PKG` restreint la portée de `test`, `bench` et `cover` à un paquet :
`make test PKG=./tri`. `RUN` restreint `test` à un motif de nom.

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

Formatage, `go vet`, absence de dépendance externe, mention de licence,
étanchéité de la frontière, compilation croisée vers les deux plateformes, puis
la suite de tests. **La liste se passe entière**, jamais
réduite à ce qui touche au changement qu'on vient d'écrire : composer sa liste
revient à ne vérifier que ce qu'on a déjà en tête, et le défaut est ailleurs par
construction.

Deux de ces étapes méritent un mot, parce qu'elles vérifient autre chose que ce
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
tentante, et elle interroge les deux plateformes, parce qu'un import placé sous
contrainte de construction ne se voit pas depuis l'autre.

**`make frontiere`** vérifie que le noyau ne touche ni au système ni à un
backend. Aucun paquet parmi `geometrie`, `scene`, `tri`, `rendu`, `raster` et
`apparence` ne doit importer `syscall`, `net`, `os/exec` ou un backend — ni
directement, ni à travers un autre paquet du moteur.

C'est le contrôle que `make cross` ne fait pas, contrairement à ce qu'on croit
spontanément. Compiler pour les deux plateformes ne dit rien de l'étanchéité :
le backend Linux est du protocole écrit sur socket, donc il n'emploie que `net`
et `os`, qui existent des deux côtés. Un `net.Dial` placé dans `geometrie`
compile pour Windows comme pour Linux, sans un mot.

Seuls les imports écrits comptent, suivis de proche en proche à travers les
paquets du moteur. Prendre les dépendances transitives complètes ne marcherait
pas : `fmt` tire `os`, qui tire `syscall`, si bien que le moindre paquet
deviendrait une fuite.

**`make cross`** n'est pas là pour produire un binaire ; les artefacts sont
jetés. Elle vérifie que le code compile pour les deux plateformes sans
compilateur C, ce qui attrape l'usage d'une API absente d'un des deux systèmes.

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

## Les images de référence

La suite de non-régression compare le rendu à des images versionnées dans
`testdata/references/`. Elles ne se régénèrent jamais toutes seules :

```
make references
```

Une régénération se justifie dans le message de commit. Une image qui change
sans raison énoncée est un défaut accepté par erreur, et il faudra l'avoir vu
pour le corriger.
