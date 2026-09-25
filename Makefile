# Cibles en anglais, commentaires en français : convention des autres projets.

SORTIE ?= .tmp

# Décision de projet, pas réglage de poste : le moteur se compile sans cgo sur
# les deux cibles. Seul le jalon 9 surcharge, et lui seul.
export CGO_ENABLED := 0

# Fixé avant l'inclusion : GNU make retient comme but par défaut la première cible
# qu'il rencontre, et un réglage local qui en déclarerait une ferait de la sienne le
# but de `make` nu. L'écart ne se voit pas — on l'attribue à autre chose.
.DEFAULT_GOAL := build

-include makefile.local

.PHONY: build test bench cover depot fmt vet deps entetes frontiere cross check \
        references notes titre clean

build:
	go build ./...

# PKG et RUN restreignent la portée sans sortir du Makefile : une commande go
# tapée directement perd les réglages du poste, et l'oubli ne se voit pas dans
# la sortie.
PKG ?= ./...
RUN ?=

# Aucun test n'ouvre de fenêtre. La suite de non-régression est visuelle mais
# passe par le rastériseur logiciel, donc elle tourne sans écran — c'est ce qui
# permet de la lancer en intégration continue et de juger le noyau sans backend.
test:
	go test $(if $(RUN),-run '$(RUN)') $(PKG)

# Rien ne crée le répertoire de sortie : il est ignoré par git, donc absent d'un
# clone neuf, et `make clean` le supprime. Les cibles qui y écrivent en dépendent.
$(SORTIE):
	@mkdir -p $@

# Les chiffres de référence d'un banc d'essai vivent dans le commit qui
# l'introduit ; les profils sont jetables.
#
# Un paquet à la fois, et non `$(PKG)` tel quel : la chaîne Go refuse
# `-cpuprofile` dès qu'il y a plus d'un paquet, si bien que la valeur par défaut
# `./...` rendait la cible inutilisable au deuxième paquet du moteur. Les profils
# sont donc nommés d'après le paquet, ce qui les rend comparables entre deux
# exécutions.
#
# `-o` place aussi le binaire de test dans la sortie, au lieu de l'abandonner à la
# racine du dépôt.
bench: | $(SORTIE)
	@for p in $$(go list $(PKG)); do \
	  nom=$$(basename $$p); \
	  echo "--- $$nom"; \
	  go test -run '^$$' -bench . -benchmem \
	    -cpuprofile $(SORTIE)/$$nom-cpu.prof \
	    -memprofile $(SORTIE)/$$nom-mem.prof \
	    -o $(SORTIE)/$$nom.test $$p || exit 1; \
	done

# `-o` n'est pas cosmétique : sans lui, `go tool cover` écrit son rapport dans le
# temporaire du système — que `GOTMPDIR` ne couvre pas, celui-ci suivant `TMP` — et
# ouvre le navigateur par défaut. Une cible de lecture ne prend pas la main sur la
# session de qui la lance, et tout ce que la chaîne Go produit reste dans la sortie.
cover: | $(SORTIE)
	go test -coverprofile=$(SORTIE)/couverture.out $(PKG)
	go tool cover -html=$(SORTIE)/couverture.out -o $(SORTIE)/couverture.html

# Plusieurs contrôles tirent leur périmètre de git. Hors d'un dépôt, la liste
# revient vide et la cible passe au vert sans avoir rien lu : un contrôle qui ne
# peut pas échouer n'en est pas un.
depot:
	@git rev-parse --git-dir >/dev/null 2>&1 || { \
	  echo "pas un dépôt git : le périmètre des contrôles ne peut pas être établi"; \
	  exit 1; \
	}

# Le périmètre de tous les contrôles qui portent sur des fichiers.
#
# `--others --exclude-standard` en plus de l'index : un fichier qu'on vient
# d'écrire et pas encore ajouté est précisément celui qu'on veut voir contrôlé, et
# `git ls-files` seul ne le liste pas. `--exclude-standard` applique `.gitignore`,
# ce qui écarte du même coup le répertoire de sortie.
#
# `core.quotepath=false` parce que git échappe en octal tout chemin non ASCII et
# que le dépôt impose des noms de fichiers en français.
SUIVIS = git -c core.quotepath=false ls-files --cached --others --exclude-standard

# Une cible qui échoue plutôt qu'une qui réécrit : avant un commit, on veut savoir,
# pas subir.
#
# Le périmètre vient de git et non d'un `gofmt -l .` sur tout l'arbre : celui-ci
# descend dans le répertoire de sortie du Makefile lui-même, où un test interrompu
# laisse un `_testmain.go` que personne n'a écrit et que rien ne formatera.
#
# La liste vide est traitée à part : `gofmt -l` sans argument lit l'entrée standard
# et attend indéfiniment.
fmt: depot
	@fichiers=$$($(SUIVIS) -- '*.go'); \
	[ -n "$$fichiers" ] || exit 0; \
	restants=$$(printf '%s\n' "$$fichiers" | xargs gofmt -l); \
	if [ -n "$$restants" ]; then \
	  echo "fichiers non formatés :"; \
	  echo "$$restants"; \
	  exit 1; \
	fi

vet:
	go vet ./...

# Les dépendances admises, et elles seules. La liste est close : docs/go.md dit ce
# que chacune doit passer pour y entrer, et pourquoi ces deux-là y sont.
#
# Une liste blanche plutôt qu'une interdiction : une interdiction absolue finit par
# être contournée en silence, alors qu'une entrée dans cette ligne se relit en pull
# request. On n'ajoute donc pas une dépendance par accident, seulement par décision.
AUTORISEES = golang\.org/x/image(/.*)?|github\.com/ebitengine/oto/v3(/.*)?

# Les paquets qui n'ont droit à rien, pas même à ce que la liste autorise. C'est
# le cœur, et c'est lui qui porte la promesse.
#
# `apparence` n'y figure pas alors qu'il appartient au noyau au sens de la
# frontière système : c'est lui qui rastérise les polices, donc lui qui pourra
# puiser dans la liste. Deux notions voisines, deux listes — les confondre
# interdirait à `apparence` ce que la liste existe précisément pour lui permettre.
SANS_DEPENDANCE = internal/geometrie internal/scene internal/tri internal/rendu \
                  internal/raster

# La règle du dépôt, rendue exécutable.
#
# `.Standard` et non un filtre sur le point dans le chemin : la bibliothèque
# standard embarque des paquets vendored qui en portent un, à commencer par
# `vendor/golang.org/x/net/dns/dnsmessage`, que `net` tire — et le backend Linux
# repose sur `net`. Les prendre pour des dépendances externes rendrait le contrôle
# rouge sur du code conforme, et la seule issue serait de l'affaiblir.
#
# `-test` parce que la règle vise nommément les tests. Sans lui, un import externe
# dans un fichier `_test.go` passe inaperçu : c'est la seule entorse que quelqu'un
# aurait une raison de tenter, et c'est précisément celle qui échappait.
#
# Les deux plateformes, parce qu'un import placé sous contrainte de construction ne
# se voit pas depuis l'autre.
#
# Le filtre du module n'est pas ancré : `-test` fait remonter des pseudo-paquets de
# la forme `paquet [paquet.test]`, dont le chemin du module ne commence pas la ligne.
deps:
	@externes=""; \
	for os in windows linux; do \
	  liste=$$(GOOS=$$os go list -deps -test \
	    -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./...) \
	    || { echo "go list a échoué pour $$os"; exit 1; }; \
	  externes="$$externes$$liste "; \
	done; \
	externes=$$(printf '%s\n' $$externes \
	  | grep -v 'github.com/sprimault/lozengine' | sort -u); \
	hors_liste=$$(printf '%s\n' $$externes | grep -Ev '^($(AUTORISEES))$$'); \
	if [ -n "$$hors_liste" ]; then \
	  echo "dépendance hors de la liste close de docs/go.md :"; \
	  echo "$$hors_liste"; \
	  exit 1; \
	fi; \
	for p in $(SANS_DEPENDANCE); do \
	  [ -d "$$p" ] || continue; \
	  atteint=$$(go list -deps -test \
	    -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./$$p \
	    | grep -v 'github.com/sprimault/lozengine'); \
	  if [ -n "$$atteint" ]; then \
	    echo "le cœur du moteur n'a droit à aucune dépendance, et $$p atteint :"; \
	    echo "$$atteint"; \
	    exit 1; \
	  fi; \
	done

# Les paquets qui consomment ou produisent un Rendu, et rien d'autre. Le paquet
# racine en est absent : c'est lui qui assemble, donc il connaît les backends.
NOYAU = internal/geometrie internal/scene internal/tri internal/rendu \
        internal/raster internal/apparence

# Ce qu'aucun d'eux ne doit importer. Les appels directs et les sockets vivent dans
# un backend, l'audio parle son protocole depuis le sien.
#
# `os` n'y est pas, et c'est délibéré : charger une planche ou écrire une image
# l'emploie légitimement. `net` suffit à attraper la fuite qui compte, puisque le
# backend Linux ouvre un socket Unix, et qu'`os` ne sait pas le faire.
INTERDIT = syscall|net|os/exec|github\.com/sprimault/lozengine/internal/backend(/.*)?

# Le contrôle que `cross` ne fait pas.
#
# Compiler pour les deux plateformes ne dit rien de l'étanchéité : le backend Linux
# est du protocole écrit sur socket, donc il n'emploie que `net` et `os`, qui
# existent des deux côtés. Un `net.Dial` placé dans `geometrie` compile pour Windows
# comme pour Linux, sans un mot.
#
# On suit les imports directs, de proche en proche, mais seulement à travers les
# paquets du module. Prendre les dépendances transitives complètes ne marche pas :
# `fmt` tire `os`, qui tire `syscall`, si bien que le moindre paquet du monde
# deviendrait une fuite. Ce qu'on cherche est l'arête qu'un humain a écrite.
#
# Le sens des dépendances entre paquets internes n'est pas contrôlé ici. Il est dans
# docs/conception.md, et l'y recopier ferait deux définitions qui divergeraient.
frontiere:
	@fuites=""; \
	for os in windows linux; do \
	  for p in $(NOYAU); do \
	    [ -d "$$p" ] || continue; \
	    internes=$$(GOOS=$$os go list -deps ./$$p \
	      | grep '^github\.com/sprimault/lozengine') \
	      || { echo "go list a échoué pour $$p"; exit 1; }; \
	    arcs=$$(GOOS=$$os go list \
	      -f '{{range .Imports}}{{$$.ImportPath}} {{.}}{{"\n"}}{{end}}' $$internes); \
	    trouve=$$(printf '%s\n' "$$arcs" | grep -E " ($(INTERDIT))$$"); \
	    if [ -n "$$trouve" ]; then \
	      fuites="$$fuites$$(printf '%s\n' "$$trouve" | sed "s/\$$/ ($$os)/")\n"; \
	    fi; \
	  done; \
	done; \
	if [ -n "$$fuites" ]; then \
	  echo "le noyau importe un paquet système ou un backend :"; \
	  printf '%b' "$$fuites" | grep -v '^$$' | sort -u; \
	  exit 1; \
	fi

# Produit deux binaires qu'on jette : ce qui compte est que le code compile pour les
# deux cibles sans compilateur C.
cross:
	GOOS=windows GOARCH=amd64 go build ./...
	GOOS=linux GOARCH=amd64 go build ./...

# Le périmètre vient de git : ce que le dépôt ne publie pas n'a rien à déclarer.
# La liste close des dispensés est dans docs/go.md — la reprendre ici en dur
# ferait deux définitions qui divergeraient sans que rien ne le signale, donc
# elle tient en un seul motif, relu avec la table.
#
# Les alternatives sont ancrées une par une et non par un `^(…)$` global : sans le
# préfixe optionnel, `testdata/references/` ne serait reconnu qu'à la racine du
# dépôt, alors que la chaîne Go veut ce répertoire à côté du paquet testé.
DISPENSES = ^(\.editorconfig|\.gitattributes|\.gitignore|Makefile|go\.mod|LICENSE-(MIT|APACHE)|THIRD-PARTY-NOTICES|.*\.md|(.*/)?testdata/references/.*\.png)$$

# `while IFS= read -r` et non un `for` : un nom qui porte une espace serait sinon
# coupé en deux. Pas de `read -d ''`, qui est une extension bash absente du shell
# POSIX qui exécute ces recettes sur la machine Linux.
#
# Les deux lignes sont vérifiées, pas seulement la ligne SPDX : docs/go.md impose
# aussi la ligne de copyright, et un contrôle qui n'en regarde qu'une laisse passer
# la moitié de ce qu'il annonce. L'année n'est pas figée.
entetes: depot
	@manquants=$$($(SUIVIS) \
	  | grep -Ev '$(DISPENSES)' \
	  | while IFS= read -r f; do \
	      tete=$$(head -5 "$$f"); \
	      absents=""; \
	      printf '%s\n' "$$tete" \
	        | grep -qE '^.{0,4}Copyright [0-9]{4} ' || absents="copyright"; \
	      printf '%s\n' "$$tete" \
	        | grep -q 'SPDX-License-Identifier: MIT OR Apache-2.0' \
	        || absents="$$absents$${absents:+ et }SPDX"; \
	      [ -z "$$absents" ] || echo "$$f : $$absents"; \
	    done); \
	if [ -n "$$manquants" ]; then \
	  echo "en-tête de licence incomplet :"; \
	  echo "$$manquants"; \
	  exit 1; \
	fi

check: fmt vet deps entetes frontiere cross test

# Le paquet qui porte la suite de non-régression visuelle, et lui seul : un
# drapeau de test n'existe que pour le paquet qui le déclare, donc `./...` ferait
# échouer tous les autres sur « flag provided but not defined ».
PKG_REFERENCES ?= ./internal/visionneuse

# Jamais automatique, et toujours justifiée dans le message de commit : une
# image de référence qui change sans raison énoncée est un bug entériné.
references:
	go test $(PKG_REFERENCES) -regenerer

# Les notes d'une version sont la section du CHANGELOG, reprise telle quelle. Une
# section absente arrête la publication : c'est le seul moment où quelqu'un relit
# ce qui change, et une version qui n'en porte pas ne le dit à personne.
#
# Une cible plutôt qu'un bout de script dans le workflow : celui-ci ne peut pas
# s'essayer avant d'être poussé, et une extraction fausse ne se découvrirait qu'au
# moment de publier.
#
# JOURNAL se surcharge pour éprouver la cible sur une copie, sans jamais écrire
# dans le fichier versionné. Le journal suit la feuille de route : y ajouter une
# section d'essai reviendrait à inventer une étape.
VERSION ?=
JOURNAL ?= CHANGELOG.md

notes:
	@[ -n "$(VERSION)" ] || { \
	  echo "VERSION attendue, par exemple : make notes VERSION=0.1.0"; exit 1; }
	@section=$$(awk -v v="$(VERSION)" \
	  'index($$0, "## [" v "]") == 1 {p=1; next} p && /^## / {exit} p {print}' \
	  $(JOURNAL)); \
	if [ -z "$$(printf '%s' "$$section" | tr -d '[:space:]')" ]; then \
	  echo "aucune section [$(VERSION)] dans $(JOURNAL)"; \
	  exit 1; \
	fi; \
	printf '%s\n' "$$section"

# Le titre d'une version est celui que porte la page des versions. Le numéro ne
# suffit pas : le mineur marque un jalon franchi sans en porter le numéro, donc
# `0.1.0` ne dit pas de quel jalon il vient. La date, elle, n'y entre pas — GitHub
# affiche déjà la sienne, et deux dates qui diffèrent d'un fuseau font douter de
# la bonne.
#
# Un correctif ne franchit aucun jalon et sa section n'en nomme donc aucun : le
# titre se réduit alors au numéro, ce qui est exact plutôt que vide.
#
# Le tiret du séparateur s'écrit en octal et le motif reste en ASCII : make
# réencode les octets non ASCII d'une ligne de recette avant de les passer au
# shell, et un tiret long tapé ici ressortirait en mojibake dans le titre. Ce qui
# vient du journal par `sed`, lui, traverse intact.
titre:
	@[ -n "$(VERSION)" ] || { \
	  echo "VERSION attendue, par exemple : make titre VERSION=0.1.0"; exit 1; }
	@ligne=$$(awk -v v="$(VERSION)" \
	  'index($$0, "## [" v "]") == 1 {print; exit}' $(JOURNAL)); \
	if [ -z "$$ligne" ]; then \
	  echo "aucune section [$(VERSION)] dans $(JOURNAL)"; \
	  exit 1; \
	fi; \
	jalon=$$(printf '%s' "$$ligne" | \
	  sed -E 's/^## \[[^]]+\][^0-9]*[0-9]{4}-[0-9]{2}-[0-9]{2}[^[:alnum:]]*//'); \
	case "$$jalon" in \
	  '## '*) echo "titre de section non conforme : $$ligne"; exit 1;; \
	esac; \
	if [ -n "$$jalon" ]; then \
	  printf '%s \342\200\224 %s\n' "$(VERSION)" "$$jalon"; \
	else \
	  printf '%s\n' "$(VERSION)"; \
	fi

# Les binaires de test que `bench` produit vont dans la sortie, mais un `go test -c`
# lancé à la main les dépose à la racine : on les retire aussi.
clean:
	rm -rf $(SORTIE) dist
	rm -f *.test *.test.exe
