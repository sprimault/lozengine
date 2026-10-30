# Cibles en anglais, commentaires en français : convention des autres projets.

SORTIE ?= .tmp

# Décision de projet, pas réglage de poste : le moteur se compile sans cgo sur
# les deux cibles. Seul le jalon 11 surcharge, et lui seul.
export CGO_ENABLED := 0

# Fixé avant l'inclusion : GNU make retient comme but par défaut la première cible
# qu'il rencontre, et un réglage local qui en déclarerait une ferait de la sienne le
# but de `make` nu. L'écart ne se voit pas — on l'attribue à autre chose.
.DEFAULT_GOAL := build

-include makefile.local

.PHONY: build example test bench cover depot fmt vet deps entetes frontiere cross check \
        floor container container-clean references journal notes titre clean

build:
	go build ./...

# Les exemples ouvrent une fenêtre, donc rien ne les lance en intégration continue.
# Ils entrent en revanche dans `./...`, si bien que `build` et `cross` les compilent :
# un exemple qui ne compile plus fait rougir `check`.
#
# `make example` lance le plus petit, celui qui tient le critère des trente lignes ;
# `make example EXAMPLE=walk` en choisit un autre. Passer par là plutôt que par un
# `go run` tapé directement, pour la raison qui vaut pour PKG et RUN ci-dessous : une
# commande go hors du Makefile perd ses réglages, et l'oubli ne se voit pas dans la
# sortie.
EXAMPLE ?= hello

example:
	go run ./examples/$(EXAMPLE)

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

# Les sites où `unsafeptr` signale une conversion voulue, par fichier et par compte.
#
# Le contrôle attrape la lecture d'une structure que Win32 passe par adresse dans un
# `lParam` : c'est la convention de la procédure de fenêtre, dont il n'existe pas
# d'autre voie, et `go vet` ne sait pas la distinguer d'un `uintptr` gardé puis
# reconverti — le défaut qu'il cherche vraiment.
#
# La liste est close et on **exige** ce qu'elle décrit : un site de plus échoue, un
# site qui disparaît aussi. C'est la forme du job `Plancher`, qui ne garde le plancher
# de Go qu'en exigeant un échec — un contrôle qui ne peut pas échouer n'en est pas un,
# et une liste qui survit à ce qu'elle décrit ne garde rien. Désactiver `unsafeptr`
# pour le paquet aurait couvert d'avance les backends qu'il va accueillir.
#
# Par fichier et par compte, jamais par ligne : un numéro de ligne se décale à la
# première édition au-dessus du site, et le contrôle rougirait sans défaut.
UNSAFEPTR = internal/backend/window_windows.go=1

# Les deux plateformes, pour la raison de `deps` : un fichier sous contrainte de
# construction ne se voit pas depuis l'autre. Les comptes portent donc sur l'union des
# deux passes, les fichiers étant disjoints par leur suffixe.
#
# Les séparateurs sont normalisés avant comparaison, la sortie portant des antislashs
# sous Windows. L'antislash se donne à `tr` en octal : écrit tel quel, il traverse la
# recette et le shell, et `tr` avertit qu'il en reste un seul en fin de chaîne.
#
# **Les lignes de titre de paquet sont retirées, et c'est la version de Go qui les
# décide.** Sous 1.22, le plancher du module, `go vet` annonce chaque paquet par une
# ligne `# chemin` — et une seconde entre crochets pour un résultat qui remonte de son
# cache. Sous 1.27 il n'en émet aucune. Rien n'est perdu à les jeter : un diagnostic
# est une ligne `fichier:ligne:colonne:`, et une erreur de compilation garde la sienne.
#
# Le défaut a traversé le poste et la machine de contrôle sans mordre, les deux
# résolvant une toolchain récente là que l'intégration continue mesure le plancher.
# Pour l'éprouver ici : `GOTOOLCHAIN=go1.22.0 make vet`.
#
# Et le code de sortie de `go vet` est ignoré au profit de sa sortie : ce qui n'est pas
# dans la liste reste affiché et fait échouer la cible, y compris une erreur de
# compilation.
vet:
	@sortie=$$(for os in windows linux; do GOOS=$$os go vet ./... 2>&1; done \
	  | tr '\134' '/' | grep -v '^#' || true); \
	reste="$$sortie"; \
	for site in $(UNSAFEPTR); do \
	  fichier=$${site%=*}; attendus=$${site#*=}; \
	  motif="^$$fichier:[0-9][0-9]*:[0-9][0-9]*: possible misuse of unsafe\.Pointer$$"; \
	  vus=$$(printf '%s\n' "$$sortie" | grep -c "$$motif" || true); \
	  if [ "$$vus" != "$$attendus" ]; then \
	    echo "unsafeptr : $$vus site(s) dans $$fichier, $$attendus attendu(s)"; \
	    echo "une conversion de plus se décide en pull request, elle ne s'ajoute pas"; \
	    if [ -n "$$sortie" ]; then echo "--- ce que go vet a dit ---"; echo "$$sortie"; fi; \
	    exit 1; \
	  fi; \
	  reste=$$(printf '%s\n' "$$reste" | grep -v "$$motif" || true); \
	done; \
	reste=$$(printf '%s\n' "$$reste" | grep -v '^$$' || true); \
	if [ -n "$$reste" ]; then \
	  echo "$$reste"; \
	  exit 1; \
	fi

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
# `appearance` y est entré au jalon 4 avec lui-même. La liste close l'avait prévu
# ouvert, au motif qu'il rastériserait les polices ; l'atlas de glyphes étant une
# donnée produite hors du dépôt, il n'en tire rien et reste sans dépendance comme
# le reste du cœur. Ce qui lui est réservé dans la liste attend donc un besoin
# qu'aucun jalon n'a encore.
#
# Tous vivent hors d'`internal/`, parce qu'un consommateur doit pouvoir lire la
# documentation de leurs champs et de leurs méthodes — un alias vers un paquet
# interne n'en montre aucun. Être public ne relâche rien : ils restent le cœur, donc
# ils n'ont droit à aucune dépendance.
SANS_DEPENDANCE = render geometry scene raster input appearance internal/sorting

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
# se voit pas depuis l'autre — et les deux boucles, pas seulement la première. Le trou
# que cela ferme est étroit mais c'est le pire possible : une dépendance **de la liste
# close** importée dans le cœur sous contrainte de construction. La première boucle la
# laisse passer puisqu'elle est autorisée ailleurs, et la seconde ne la voyait pas.
# Éprouvé sur une copie dans `.tmp/`, par un `golang.org/x/image` substitué et importé
# depuis un fichier `_linux.go` de `raster` : le contrôle passait au vert.
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
	for os in windows linux; do \
	  for p in $(SANS_DEPENDANCE); do \
	    [ -d "$$p" ] || continue; \
	    atteint=$$(GOOS=$$os go list -deps -test \
	      -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./$$p \
	      | grep -v 'github.com/sprimault/lozengine'); \
	    if [ -n "$$atteint" ]; then \
	      echo "le cœur du moteur n'a droit à aucune dépendance, et $$p atteint sous $$os :"; \
	      echo "$$atteint"; \
	      exit 1; \
	    fi; \
	  done; \
	done

# Les paquets qui consomment ou produisent une DrawList, et rien d'autre. Le paquet
# racine en est absent : c'est lui qui assemble, donc il connaît les backends.
NOYAU = render geometry scene raster input appearance internal/sorting

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
# existent des deux côtés. Un `net.Dial` placé dans `geometry` compile pour Windows
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

# Produit un binaire par couple, qu'on jette : ce qui compte est que le code compile
# pour chacun sans compilateur C.
# Les couples que les README annoncent, et rien de plus. « Compile » est tout ce que
# cette cible prouve : ni « tourne », ni « reste vert en intégration continue », qui
# sont deux autres promesses et ne valent que pour `amd64`.
#
# Ils passent tous sans rien changer au code, et c'est le zéro-dépendance qui le
# permet — une dépendance cgo retirerait des cibles au lieu d'en ajouter, puisque
# `go build` ne suffirait plus. `js/wasm` y figure parce qu'il porte le backend du
# jalon 5 et que `syscall/js` est dans la bibliothèque standard, donc la cible n'ouvre
# pas la liste close. `windows/386` parce que la famille Windows est annoncée sur les
# trois architectures, et qu'une annonce sans cible compilée n'est gardée par rien.
#
# `linux/riscv64`, `freebsd/amd64` et les cibles Apple compilent aussi, mesuré le
# 5 octobre 2026. Elles ne sont pas ici parce qu'aucun README ne les annonce : cette
# liste est celle des promesses tenues, pas celle du possible.
CIBLES = windows/amd64 windows/arm64 windows/386 linux/amd64 linux/arm64 \
         linux/arm linux/386 js/wasm

cross:
	@for cible in $(CIBLES); do \
	  os=$${cible%/*}; arch=$${cible#*/}; \
	  echo "GOOS=$$os GOARCH=$$arch go build ./..."; \
	  GOOS=$$os GOARCH=$$arch go build ./... || exit 1; \
	done

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

# Un ajout à la surface publique sans entrée de journal ne se rattrape plus après le
# tag : une section publiée ne se retouche pas, donc le trou est définitif. C'est
# arrivé deux fois dans la même série, et seule la chance l'a rendu réparable — la
# section n'était pas encore taguée.
#
# Ce qu'elle compare : les identifiants exportés des paquets publics, entre la base et
# l'arbre courant, par `go doc -all`. C'est la surface telle qu'un consommateur la lit,
# méthodes et champs compris : sans `-all`, `go doc` ne montre aucune méthode, mesuré.
# Les lignes d'une déclaration se reconnaissent à leur indentation — colonne zéro pour
# `func`, `type`, `const` et `var`, une tabulation pour un champ ou une constante de
# bloc, la godoc étant indentée de quatre espaces.
#
# La liste des paquets se dérive de `go list`, elle ne se tient pas : ce dépôt a déjà
# vu huit listes recopiées diverger toutes les huit.
#
# **Ce qu'elle ne voit pas, et il faut le dire** : sans base de comparaison, elle
# passe. Le contrôle distant reconstitue un index sans historique, et l'intégration
# continue ne récupère que la branche de la pull request — il y faudrait un
# `fetch-depth: 0`, qui touche le workflow et non cette cible. Elle ne protège donc que
# le poste, c'est-à-dire l'endroit où `/verif` tourne avant chaque poussée, qui est le
# moment où elle sert.
#
# SANS_JOURNAL=1 la contourne, pour un alias mécanique qui n'a rien à dire. Le
# contournement s'écrit, il ne se devine pas.
surface:
	@if [ -n "$(SANS_JOURNAL)" ]; then exit 0; fi; \
	base=$$(git rev-parse --verify --quiet origin/master \
	  || git rev-parse --verify --quiet master || true); \
	if [ -z "$$base" ]; then \
	  echo "surface : pas de base de comparaison, rien à vérifier"; \
	  exit 0; \
	fi; \
	if ! git diff --quiet "$$base" -- $(JOURNAL); then exit 0; fi; \
	paquets=$$(go list ./... | grep -vE '/(internal|examples|cmd)(/|$$)' \
	  | sed "s|^$$(go list -m)|.|"); \
	arbre=$(CURDIR)/.tmp/surface-base; \
	rm -rf "$$arbre"; \
	git worktree add --detach --quiet "$$arbre" "$$base" \
	  || { echo "surface : l'arbre de comparaison n'a pas pu être monté"; exit 1; }; \
	releve() { \
	  for p in $$paquets; do \
	    go doc -all "$$p" 2>/dev/null \
	      | awk -v p="$$p" '/^(func|type|const|var) / || /^\t[A-Z]/ { print p ": " $$0 }'; \
	  done; \
	}; \
	( cd "$$arbre" && releve ) | sort > $(CURDIR)/.tmp/surface-base.txt; \
	releve | sort > $(CURDIR)/.tmp/surface-tete.txt; \
	git worktree remove --force "$$arbre" >/dev/null 2>&1; \
	ajouts=$$(comm -13 $(CURDIR)/.tmp/surface-base.txt $(CURDIR)/.tmp/surface-tete.txt); \
	if [ -n "$$ajouts" ]; then \
	  echo "surface publique élargie sans entrée dans $(JOURNAL) :"; \
	  printf '%s\n' "$$ajouts" | sed 's/^/  /'; \
	  echo "écrire l'entrée, ou contourner par SANS_JOURNAL=1 en disant pourquoi"; \
	  exit 1; \
	fi

check: fmt vet deps entetes frontiere journal surface cross test

# Le plancher du module, lu dans `go.mod` et nulle part ailleurs.
#
# Par `awk` et non par `go list -m`, pour une raison de circularité : déterminer avec
# quelle toolchain appeler `go` ne peut pas demander d'appeler `go`. C'est la commande
# qu'emploie déjà le job `Plancher` du workflow, donc une seule façon de lire ce
# numéro dans tout le dépôt.
PLANCHER = $(shell awk '/^go /{print $$2; exit}' go.mod)

# `check` rejoué sous la toolchain du plancher, qui est celle que l'intégration
# continue mesure.
#
# **Le poste et la machine de contrôle ne la voient jamais** : le premier tourne la
# version qu'il a installée, la seconde résout `GOTOOLCHAIN=auto`. Les deux passent
# donc au vert sur un Go que le runner n'emploie pas. Le 27 octobre 2026, la cible
# `vet` y est passée deux fois avant d'échouer en intégration continue, `go vet` du
# plancher préfixant ses diagnostics d'une ligne que les versions récentes n'émettent
# pas.
#
# Hors de `check`, et pas seulement par prudence : `check` l'appelant relancerait
# `check`.
floor:
	@echo "contrôle complet sous Go $(PLANCHER), le plancher du module"
	@GOTOOLCHAIN=go$(PLANCHER) $(MAKE) check

# Le volume qui porte le cache de compilation du conteneur, et le chemin où il est
# monté. Le nom est préfixé par le dépôt : un démon partagé doit dire à qui appartient
# ce qu'il garde, et c'est ce qui permet de le retrouver sans le chercher.
CACHE_VOLUME = lozengine-gocache
CACHE_MONTE = /gocache

# `check` rejoué dans un conteneur jetable, sur l'image officielle du plancher.
#
# Ce qu'il apporte en plus de `floor` : un système nu. Ni la toolchain résolue du
# poste, ni ce que la machine de contrôle porte en propre, ni `makefile.local`.
#
# **Le périmètre vient de `git ls-files`, donc de git et non d'une liste tenue à la
# main.** C'est ce qui épargne une troisième liste d'exclusion à tenir d'accord avec
# les deux autres, et le piège est celui que `poste.md` relève déjà pour `entetes` :
# un fichier créé dans le lot et non encore indexé n'est pas transporté, donc
# `git add` avant d'éprouver.
#
# Les fichiers suivis **tels qu'ils sont**, et non `git archive HEAD` : celui-ci rend
# le dernier commit, si bien que la cible éprouverait l'état précédent alors qu'on
# l'appelle justement avant de commiter.
#
# Ce qu'il n'apporte pas, et il faut le dire : l'image est une Debian, le runner une
# Ubuntu. On reproduit la version de Go et la nature jetable, pas la distribution.
#
# `GOTOOLCHAIN=local` et `CGO_ENABLED=0` comme le job `Contrôles`. Un index git est
# reconstitué sur place pour la raison qui vaut déjà à distance : sans lui, `entetes`
# tire son périmètre d'un `git ls-files` qui ne rend rien, et la cible passe au vert
# sans avoir lu un seul fichier.
#
# **Le démon peut être ailleurs**, et c'est le cas ici : `DOCKER_HOST` vit dans
# `makefile.local` avec le reste de ce qui est propre au poste. Rien dans cette cible
# ne nomme de machine. **Les sources entrent donc par l'entrée standard** : un montage
# serait résolu du côté du démon, et ne verrait pas ce dépôt.
#
# **Le cache de compilation, lui, est exactement ce qu'on veut du côté du démon**, et
# c'est la nuance à ne pas confondre avec la précédente. Il vit dans un volume nommé,
# ce qui fait passer la cible de 18 à 3 secondes — mesuré le 27 octobre 2026, et les
# chiffres encadrés par deux mesures sans cache qui donnent le même temps.
#
# Il ne rend pas le contrôle complaisant, et la raison est précise : **un cache Go est
# indexé par hachage de contenu**, donc une entrée qui ne correspond plus est ignorée
# et non réutilisée. C'est ce qui le distingue d'un conteneur réutilisé, qui garderait
# des sources, un index git et les fichiers du lot précédent — et c'est cette phrase
# qu'il faut relire avant d'être tenté d'y persister autre chose.
#
# **`-count=1` force la réexécution des tests malgré ce cache**, qui porte aussi leurs
# résultats. Sans lui, un passage sans changement les sauterait tous — et la cible
# cesserait de rejouer le régime du runner, qui part d'un cache vide et les exécute.
# Ce qui a décidé est la mesure : la réexécution est **gratuite**, 3 secondes avec ou
# sans, parce que le temps était dans la compilation et non dans des tests qui durent
# des millisecondes. Un test non déterministe reste donc attrapé ici.
#
# **La machine qui porte le démon est partagée.** Tout ce que cette cible crée porte
# le préfixe du dépôt : un conteneur `--rm` nommé par le dépôt et l'instant, et ce
# volume. Aucune commande de nettoyage global n'a sa place ici — ni `prune`, ni une
# suppression d'image, l'image du plancher pouvant servir à d'autres. Le volume se
# vide par `container-clean`, et rien n'en dépend.
#
# `MSYS_NO_PATHCONV` empêche le shell de Git Bash de convertir les chemins destinés au
# conteneur en chemins Windows avant que `docker` les voie. Sans elle, Go rapporte
# « GOCACHE is not an absolute path » et la cible échoue sur un contrôle sans rapport.
# La variable n'a de sens que sur ce shell, et Linux l'ignore.
container:
	@command -v docker >/dev/null || { echo "docker est absent, et cette cible seule en a besoin"; exit 1; }
	@echo "contrôle complet dans un conteneur, Go $(PLANCHER)"
	@git ls-files -z | tar -cf - --null -T - | MSYS_NO_PATHCONV=1 docker run --rm -i \
	  --name lozengine-check-$$$$ \
	  --label projet=lozengine \
	  -e CGO_ENABLED=0 \
	  -e GOTOOLCHAIN=local \
	  -e GOCACHE=$(CACHE_MONTE) \
	  -e GOFLAGS=-count=1 \
	  -v $(CACHE_VOLUME):$(CACHE_MONTE) \
	  golang:$(PLANCHER) \
	  sh -c 'mkdir -p /src && cd /src && tar -x && git init -q . && git add -A && make check'

# Vide le cache du conteneur, et c'est le pendant obligé de sa persistance : ce qui
# survit à une commande doit avoir un moyen de partir qui ne demande pas de connaître
# la machine. Rien n'en dépend — le contrôle suivant le remplit en 18 secondes.
container-clean:
	@command -v docker >/dev/null || { echo "docker est absent, et cette cible seule en a besoin"; exit 1; }
	@docker volume rm -f $(CACHE_VOLUME) >/dev/null
	@echo "cache du conteneur vidé"

# Le paquet qui porte la suite de non-régression visuelle, et lui seul : un
# drapeau de test n'existe que pour le paquet qui le déclare, donc `./...` ferait
# échouer tous les autres sur « flag provided but not defined ».
PKG_REFERENCES ?= ./internal/viewer

# Jamais automatique, et toujours justifiée dans le message de commit : une
# image de référence qui change sans raison énoncée est un bug entériné.
references:
	go test $(PKG_REFERENCES) -regenerate

# Chaque section du journal doit dire la même chose dans ses deux langues : les
# mêmes catégories de part et d'autre du séparateur, et le même nombre d'entrées
# dans chacune. Une entrée présente dans une seule langue est un oubli, pas une
# traduction à faire plus tard.
#
# Elle existe parce que l'oubli est arrivé et qu'il est passé en pull request
# fusionnée : une section avait reçu « Ajouté » sans son « Added », et ni le
# formatage, ni les tests, ni la relecture du diff ne l'ont vu. `notes` extrait la
# section telle quelle, donc la publication aurait servi des notes à moitié
# traduites.
#
# Ce qui est comparé est le **nombre** de catégories et le nombre d'entrées dans
# chacune, pas leur nom : une section qui porte trois entrées puis une donne la
# signature « 3 1 » des deux côtés, quelle que soit la langue. Apparier les noms
# demanderait une table accentuée dans le Makefile, et elle ne survit pas au
# passage par `make` et `awk` de cette machine — un « Ajouté » du journal n'y
# retrouvait pas celui de la table, ce qui rendait le contrôle faux sur toutes les
# sections.
#
# Ce que la signature attrape : une catégorie oubliée dans une langue, et une
# entrée oubliée dans une catégorie. Ce qu'elle laisse passer : un « Corrigé » en
# face d'un « Added » portant autant d'entrées. Ce contrôle dit que les deux blocs
# se correspondent, pas que la traduction est juste — celle-là se relit.
#
# Le comptage prend les puces de premier niveau, celles qui commencent la ligne :
# une continuation est indentée, donc elle ne compte pas deux fois.
journal:
	@ecarts=$$(awk ' \
	  function solde() { \
	    if (entrees == 0) return; \
	    if (cote == 0) gauche = gauche " " entrees; else droite = droite " " entrees; \
	    entrees = 0; \
	  } \
	  function verdict() { \
	    solde(); \
	    if (section != "" && gauche != droite) \
	      printf "%s : %s en français, %s en anglais\n", \
	        section, gauche ? gauche : "rien", droite ? droite : "rien"; \
	  } \
	  /^## / { \
	    verdict(); \
	    section = match($$0, /\[[^]]*\]/) ? substr($$0, RSTART, RLENGTH) : $$0; \
	    gauche = droite = ""; cote = 0; next \
	  } \
	  /^\*\*\*[[:space:]]*$$/ { solde(); cote = 1; next } \
	  /^### / { solde(); next } \
	  /^- / { entrees++ } \
	  END { verdict() } \
	' $(JOURNAL)); \
	if [ -n "$$ecarts" ]; then \
	  echo "$(JOURNAL) ne dit pas la même chose dans ses deux langues :"; \
	  printf '%s\n' "$$ecarts" | sed 's/^/  /'; \
	  exit 1; \
	fi

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
