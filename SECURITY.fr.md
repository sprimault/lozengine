# Sécurité

English: [SECURITY.md](SECURITY.md)

## Versions prises en charge

La dernière version publiée. En `0.x`, il n'y a pas de branche de maintenance :
un correctif part dans la version suivante.

## Signaler une faille

Par un avis de sécurité privé sur le dépôt GitHub, jamais par une issue
publique. Réponse sous quelques jours.

## Dans le périmètre

Le moteur décode des octets que l'hôte lui confie sans les avoir écrits :
planches, atlas, descriptions de scène. C'est la seule vraie surface d'attaque,
et la seule qui compte — d'autant qu'il tourne dans le processus de l'hôte, pas
dans le sien.

- Une planche ou un atlas qui fait planter le chargement, boucler
  indéfiniment, ou épuiser la mémoire au décodage.
- Une écriture hors bornes, une lecture hors bornes ou un débordement d'entier
  atteignable depuis une ressource malformée, ou depuis une scène dont les
  coordonnées dépassent la limite documentée de la clé de tri, 32 000 cases de
  côté.
- Le rastériseur qui écrit en dehors du tampon qu'on lui a donné, malgré le
  découpage qu'il fait sur les bords.
- Toute fuite du cookie `MIT-MAGIC-COOKIE-1` lu dans `.Xauthority` par le
  backend Linux : journalisé, glissé dans un message d'erreur, ou envoyé
  ailleurs qu'au serveur X désigné par `DISPLAY`.
- Un écart entre ce que la documentation publique garantit et ce que le code
  fait, y compris une panique qui s'échappe vers l'appelant à travers la
  frontière C une fois que celle-ci existera.
- Tout ce qui exécuterait du code venu d'une ressource chargée — **rien dans
  les formats ne le permet, et c'est un invariant** : une ressource porte des
  pixels, des rectangles et des nombres, aucun binaire, aucun script, aucun
  chemin de fichier.

## Hors périmètre

**Un hôte qui viole les préconditions de l'API.** Fournir un tampon plus petit
que les dimensions qu'il annonce, un pas de ligne inférieur à la largeur, un
handle déjà libéré : ces conditions sont documentées, et les tenir relève de
l'appelant. C'est la nature d'une frontière destinée à être franchie depuis
d'autres langages, pas un défaut du moteur.

**L'environnement choisi par l'hôte.** Le backend Linux se connecte au serveur X
désigné par `DISPLAY` et à rien d'autre ; ce que fait ce serveur dépasse ce dont
le moteur peut répondre.

Modifier ses propres ressources pour obtenir une autre image. Il n'y a rien ici
à protéger de son propriétaire.
