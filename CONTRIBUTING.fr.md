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

## La règle qui n'a pas d'exception

`go.mod` ne contient aucune ligne `require`. Bibliothèque standard uniquement,
sans cgo, y compris pour les tests, les exemples et l'outillage.

**Une pull request qui ajoute une dépendance est refusée quelle que soit sa
qualité.** « Juste pour les tests » n'est pas une exception, et une bibliothèque
d'assertions encore moins. Avant de proposer quelque chose qui suppose un paquet
extérieur : ne pas le proposer, écrire le code. [`docs/go.md`](docs/go.md) porte
la table de ce qu'on fait à la place, besoin par besoin.

Si une fonctionnalité paraît exiger une liaison native, c'est la fonctionnalité
qui est hors périmètre.

Une seule cible y échappe, et elle ne met rien dans le moteur : la bibliothèque
native du dernier jalon réclame cgo et un compilateur C sur la machine de
compilation, parce que le runtime Go l'impose pour s'accrocher à l'ABI C.

## Ce qui se discute avant d'être écrit

Une pull request qui touche l'un de ces points sans discussion préalable sera
renvoyée à une issue, quelle que soit sa qualité — non par principe, mais parce
que ce sont les endroits où une modification en fait basculer d'autres.

- **[`docs/conception.md`](docs/conception.md)** fait foi. Le code s'y conforme,
  donc en changer une ligne change ce que le code doit faire.
- **Le sens des dépendances entre paquets.** Il ne s'inverse jamais. Si un
  backend a besoin de trier, de projeter ou de consulter la scène, la frontière
  est mal placée et le correctif est dans le noyau, pas dans le backend.
- **La disposition de `Quad` et `Rendu`.** Champs de taille fixe, tableau
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
  taille de tuile, ratio de projection, nombre de couches. Tout vient du
  paramétrage ou du fournisseur d'apparence.
- **Aucune constante de ratio hors d'un constructeur de matrice.** Pas de
  division par deux disséminée dans le code.
- Toute modification du tri, du rastériseur, de la projection ou de l'occlusion
  s'accompagne d'un scénario de référence couvrant le cas traité — avant la
  correction quand c'est un défaut.

## Livraison

**Un lot, une branche, une pull request.** La branche part de `master` à jour et
porte le nom de son sujet, en français et sans préfixe de convention : le dépôt
n'en utilise pas, ni dans les branches ni dans les commits. Ne pas enchaîner
deux lots sur la même branche — chacun doit rester lisible et annulable seul.

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

**La section du `CHANGELOG` part avec le lot**, pas au moment de la publication :
elle est relue en pull request, donc au moment où elle compte. Chaque entrée est
écrite en français et en anglais sous la même version, dans le même commit —
une entrée présente dans une seule langue est un oubli, pas une traduction à
faire plus tard. N'y entre que ce qui se voit depuis l'extérieur du moteur :
signature publique, comportement de rendu, format de ressource, plateforme prise
en charge. Ni les remaniements internes, ni les tests, ni l'outillage.

**La documentation part avec le changement.** Avant de commiter, vérifier ce que
le changement rend faux ailleurs : l'état annoncé dans le README, une décision
de [`docs/conception.md`](docs/conception.md), un critère de fin de
[`ROADMAP.md`](ROADMAP.md).

**Un message dit ce qui change et pourquoi**, en français, à l'impératif
présent, sans préfixe de convention. Première ligne courte ; un corps n'existe
que s'il porte quelque chose que le titre ne dit pas et que le diff ne montre
pas. Pas de trailer `Co-Authored-By`.

**La pull request reprend ce message**, titre et corps, à l'identique. Un lot
tient en un commit : il n'y a rien à dire dans la PR que le commit ne dise déjà,
et deux textes à tenir d'accord finiraient par diverger.

## Langue

**Les identifiants sont en français par défaut** — répertoires, fichiers,
paquets, types, fonctions, champs. L'anglais est admis quand c'est le terme
technique naturel et sans équivalent usuel courant : `atlas`, `sprite`,
`buffer`. Ne pas franciser de force ce que personne n'appelle autrement.

**La documentation est en français** : godoc, commentaires, messages d'erreur.

Sont bilingues les documents qui s'adressent à quelqu'un ne connaissant pas
encore le projet — le README, ce guide, la politique de sécurité et le journal
des modifications. Chaque paire est tenue d'accord dans le même commit : une
version qui avance seule est un oubli, pas une traduction à faire plus tard.
Les mentions de licence, elles, sont en anglais.

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
moteur par un second jeu, au jalon 7 de [`ROADMAP.md`](ROADMAP.md). Une API
validée par un seul jeu n'est pas universelle, et une ABI C se révise encore
moins bien qu'une API Go.

Conséquence directe : **le numéro ne prévient de rien** tant qu'on est en `0.x`,
et ce sont les notes de version qui disent ce qu'un projet consommateur doit
reprendre.
