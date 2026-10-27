# Contribuer

English: [CONTRIBUTING.md](CONTRIBUTING.md)

## Avant d'écrire du code

Ouvrir une issue d'abord, pour tout ce qui dépasse une correction. La conception
est écrite dans [`docs/conception.md`](docs/conception.md) et elle fait foi : un
désaccord entre le document et le code est un défaut du code. En changer une
ligne est une discussion, pas un correctif.

Le moteur est destiné à d'autres projets que celui qui l'a fait naître. Une
décision interne se change au prix d'une refonte ; une décision d'API se change
au prix de la confiance de ceux qui ont adopté le moteur. **Toute nouvelle
signature publique est traitée comme définitive**, et c'est ce qui rend la
discussion préalable utile plutôt que bureaucratique.

## La règle sur les dépendances

**Le cœur du moteur n'en a aucune.** `geometry`, `scene`, `sorting`, `render` et
`raster` n'emploient que la bibliothèque standard, sans cgo, tests compris. Une
pull request qui y ajoute quoi que ce soit est refusée, quelle que soit sa
qualité : « juste pour les tests » n'est pas une exception, et une bibliothèque
d'assertions encore moins.

Ailleurs, une **liste close** tenue dans le `Makefile`, qui compte aujourd'hui
deux entrées — le rendu de polices et la sortie audio. Une candidature s'y ajoute
par une décision relue en pull request, jamais par un `require` glissé dans un
lot qui parle d'autre chose. [`docs/go.md`](docs/go.md) énonce ce qu'elle doit
passer : la licence d'abord, qui est rédhibitoire ; puis le sujet, car ce que le
moteur revendique s'écrit quel qu'en soit le coût ; puis l'arbitrage du coût pour
le reste.

Ouvrir une issue **avant** d'écrire du code qui suppose un paquet extérieur. Ce
qui se refuse, ce n'est pas la dépendance, c'est celle qu'on découvre à la
relecture.

Le critère porte sur ce que le moteur embarque, pas sur ce qui se charge en
mémoire : aucune bibliothèque C tierce, aucun compilateur C exigé, rien à
distribuer à côté du binaire. Appeler `user32` par `syscall` reste dedans — la
bibliothèque standard charge déjà `kernel32` dès qu'on ouvre un fichier —, et si
une fonctionnalité du moteur paraît exiger cgo ou un paquet hors de la liste
close, c'est la fonctionnalité qui est hors périmètre.

Une seule cible y échappe, et elle ne met rien dans le moteur : la bibliothèque
native du dernier jalon réclame cgo et un compilateur C sur la machine de
compilation, parce que le runtime Go l'impose pour s'accrocher à la convention d'appel
du C, qu'on nomme son ABI.

## Ce qui se discute avant d'être écrit

Une pull request qui touche l'un de ces points sans discussion préalable sera
renvoyée à une issue, quelle que soit sa qualité — non par principe, mais parce
que ce sont les endroits où une modification en fait basculer d'autres.

- **[`docs/conception.md`](docs/conception.md)** fait foi. Le code s'y conforme,
  donc en changer une ligne change ce que le code doit faire.
- **Le sens des dépendances entre paquets.** Il ne s'inverse jamais. Si un
  backend a besoin de trier, de projeter ou de consulter la scène, la frontière
  est mal placée et le correctif est dans le noyau, pas dans le backend.
- **La disposition de `Quad` et `DrawList`.** Champs de taille fixe, tableau
  contigu, ni chaîne ni interface ni slice imbriquée. Un seul champ ajouté au
  mauvais type rend le moteur inexploitable depuis un autre langage.
- **La composition de la clé de tri.** Elle est la seule vérité de l'ordre de
  dessin, et sa répartition de bits fixe les limites annoncées du moteur.
- **`testdata/references/`.** Les images font foi sur le rendu ; en régénérer
  une sans raison énoncée fait entrer un défaut au vert.

Le reste — code, tests, documentation d'accompagnement — se propose directement.

## Ce sur quoi une contribution est jugée

Les conventions de code et la doctrine de test sont dans
[`docs/go.md`](docs/go.md). Ce qui suit en est le résumé exigible.

- `make check` passe.
- Tout fichier source porte l'en-tête de copyright et l'identifiant SPDX. La
  liste close des dispensés est dans [`docs/go.md`](docs/go.md), et
  `make entetes` la vérifie.
- Toute déclaration a son godoc, en français, commençant par le nom de ce
  qu'elle déclare — exportée ou non. Les commentaires disent *pourquoi* ; ils ne
  paraphrasent jamais la ligne suivante.
- Pas de bannière, pas d'emoji, ni dans le code, ni dans les messages de commit.
- **Aucune allocation dans le chemin de rendu.** Tampons réutilisés, remis à
  longueur nulle plutôt que réalloués. Pas d'`interface{}`, pas de réflexion,
  pas de fermeture appelée par élément.
- **Les fonctions du chemin de rendu ne renvoient pas d'erreur.** Les erreurs se
  produisent au chargement, pas à l'image.
- **Rien de ce qui relève du jeu n'est codé en dur** : nombre de directions,
  taille de tuile, ratio de projection, nombre de couches **employées**. Tout vient
  du paramétrage ou du fournisseur d'apparence. Les plafonds du format n'en relèvent
  pas : borner ce que la clé de tri distingue dit ce qu'elle porte, et ne décide pas
  combien de couches un jeu emploie.
- **Aucune constante de ratio hors d'un constructeur de matrice.** Pas de
  division par deux disséminée dans le code.
- Toute modification du tri, du rastériseur, de la projection ou de l'occlusion
  s'accompagne d'un scénario de référence couvrant le cas traité — avant la
  correction quand c'est un défaut.

## Livraison

**Un lot, une branche, une pull request.** La branche part de `master` à jour et
porte un préfixe de nature suivi de son sujet en français — `feat/`, `fix/`,
`doc/`, `refactor/`, `test/`, `chore/`, `perf/` —, ce qui rend la liste des
branches lisible sans les ouvrir. Ne pas enchaîner deux lots sur la même branche :
chacun doit rester lisible et annulable seul.

**Vérifier avant de pousser, pas après :**

```
make check
```

**La liste se passe entière**, jamais réduite à ce qui touche au changement
qu'on vient d'écrire. Composer sa liste revient à ne vérifier que ce qu'on a
déjà en tête, et le défaut est ailleurs par construction : s'il avait été là où
l'on regardait, on l'aurait vu en écrivant.

Une compilation croisée prouve que le code compile pour l'autre plateforme,
jamais qu'il y fonctionne. **Un changement qui touche un backend se vérifie sur
la plateforme concernée**, nativement, avant d'être livré.

**Et l'intégration continue mesure le plancher de Go**, que la version installée
sur une machine de développement n'est presque jamais. Deux cibles rejouent ce
régime avant la poussée :

```
make floor
make container
```

La première relance le contrôle sous la toolchain du plancher, dont le numéro se
lit dans `go.mod` ; Go la télécharge au premier appel. La seconde le rejoue dans
un conteneur jetable sur l'image officielle de cette version, où seul le contenu
versionné entre — c'est ce que fait le runner, et elle demande `docker`, le seul
outil que le dépôt n'exige nulle part ailleurs.

Le conteneur est détruit à chaque fois, mais son **cache de compilation** est gardé
dans un volume nommé : le contrôle passe ainsi de dix-huit secondes à trois. Le
système reste donc neuf à chaque appel, et ce qui est gardé ne peut pas faire passer
un contrôle au vert à tort — Go retrouve une entrée de cache par le contenu qu'elle
décrit, donc une entrée qui ne correspond plus est ignorée. Les tests, eux, sont
**réexécutés à chaque appel** malgré ce cache, qui porterait aussi leurs résultats :
c'est gratuit, et sans cela la cible cesserait d'éprouver quoi que ce soit lorsque
rien n'a changé. `make container-clean` vide ce cache, et rien n'en dépend.

Les deux restent hors de `make check`, qui doit continuer à passer sur un clone
neuf sans rien installer.

**La section du `CHANGELOG` part avec le lot**, pas au moment de la publication :
elle est relue en pull request, donc au moment où elle compte. Chaque entrée est
écrite en français et en anglais sous la même version, dans le même commit —
une entrée présente dans une seule langue est un oubli, pas une traduction à
faire plus tard — et `make journal`, que `make check` enchaîne, refuse une section
dont les deux blocs ne portent pas autant d'entrées. N'y entre que ce qui se voit
depuis l'extérieur du moteur : signature publique, comportement de rendu, format de
ressource, plateforme prise en charge. Ni les remaniements internes, ni les tests, ni
l'outillage.

**La documentation part avec le changement.** Avant de commiter, vérifier ce que
le changement rend faux ailleurs : l'état annoncé dans le README, une décision
de [`docs/conception.md`](docs/conception.md), un critère de fin de
[`ROADMAP.md`](ROADMAP.md).

**Un message dit ce qui change et pourquoi**, sous la forme `nature(portée):
sujet à l'impératif présent`, la nature étant celle de la branche. Première
ligne courte ; un corps n'existe que s'il porte quelque chose que le titre ne dit
pas et que le diff ne montre pas. Pas de trailer `Co-Authored-By`.

**Le message est bilingue** : le bloc français puis l'anglais, séparés par une
ligne de trois astérisques. Le préfixe ne se traduit pas — c'est ce que lit
l'outillage, et une portée traduite ferait deux noms pour un même paquet.

```
doc(jalons): avancer le backend web avant l'audio

C'est lui qui donne Android et iOS par le navigateur.

***

doc(jalons): move the web backend ahead of audio

It is the one that brings Android and iOS through the browser.
```

Ne jamais séparer les deux blocs par `---` : `git am` le traite comme le début
d'un patch et tronque tout ce qui suit.

**La pull request reprend ce message**, titre et corps, à l'identique. Un lot
tient en un commit : il n'y a rien à dire dans la PR que le commit ne dise déjà,
et deux textes à tenir d'accord finiraient par diverger.

## Langue

**Les identifiants sont en anglais** — répertoires, fichiers, paquets, types,
fonctions, champs, locales, noms de tests. Les messages d'erreur aussi : un message se
colle dans un rapport de bug, là qu'un commentaire se lit avec le code.

**La documentation est en français** : godoc, commentaires, et les libellés de cas de
test, qui sont des phrases et non des identifiants.

Sont bilingues les documents qui s'adressent à quelqu'un ne connaissant pas
encore le projet — le README, ce guide, la politique de sécurité et le journal
des modifications —, ainsi que les messages de commit et les descriptions de
pull request. Chaque paire est tenue d'accord dans le même commit : une version
qui avance seule est un oubli, pas une traduction à faire plus tard. Les mentions
de licence, elles, sont en anglais.

La feuille de route et `docs/` restent en français seulement : ce sont les deux
documents dont on n'attend pas de lectorat anglophone, là où le README et ce guide
en ont un.

Les contributions rédigées en anglais sont les bienvenues et ne sont pas
soumises à la règle bilingue.

## Versions

Le dépôt suit SemVer avec la clause du zéro, définie dans
[`CHANGELOG.md`](CHANGELOG.md) : **en `0.x`, rien n'est garanti.** Les signatures
publiques peuvent changer à chaque version mineure.

**Le journal des modifications suit la feuille de route.** Le mineur marque un
jalon franchi, pas une rupture d'API ; tout le reste s'accumule en correctif. Une
section de version s'écrit donc quand un jalon se termine, et le tag qui la
publie porte le même numéro.

Le gel de l'API et le passage en `1.0.0` interviennent après validation du
moteur par un second jeu, au jalon 9 de [`ROADMAP.md`](ROADMAP.md). Une API
validée par un seul jeu n'est pas universelle, et une ABI C se révise encore
moins bien qu'une API Go.

Conséquence directe : **le numéro ne prévient de rien** tant qu'on est en `0.x`,
et ce sont les notes de version qui disent ce qu'un projet consommateur doit
reprendre.
