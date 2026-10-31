// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package raster porte les images dans lesquelles le moteur puise — les planches et
// l'atlas qui les range — et dessine une liste de quads dans un tampon de pixels, sans
// carte graphique et sans fenêtre.
//
// C'est le rastériseur du moteur, et le mot ne dit rien de plus que cela : peindre
// une forme décrite en recopiant ses pixels un à un, là où une carte graphique le
// ferait pour nous. Le dessin d'un quad s'appelle un blit dans le même esprit — une
// recopie de pixels d'une planche vers le tampon, sans rotation ni étirement.
//
// **Les deux moitiés vivent ensemble parce qu'elles ne se séparent pas.** Un jeu
// construit ses planches, donc il lit leurs champs ; mais le dessin emploie l'encodage
// des lignes qu'[Atlas.Add] calcule, lequel reste privé. Les mettre dans deux
// paquets aurait obligé à publier ce détail d'optimisation dans [Sheet], que l'ABI
// exposera un jour.
//
// Pour écrire un jeu en Go, le paquet racine republie [Sheet], [Atlas], [Rectangle] et
// trois façons d'obtenir une planche — `NewSheet`, `LoadSheet` et
// `NewSheetFromStraight` —, auxquelles elle ajoute `OpenSheet`, qui lit un fichier et
// n'a pas d'équivalent ici. Elle republie aussi [ScaleFactor], [Upscale] et
// [WritePNG] : un hôte qui garde sa propre boucle reçoit l'image du moteur et doit la
// présenter, et la lui faire chercher dans un second paquet n'achetait rien.
//
// Ce qu'elle ne republie pas est le **dessin** — [Clear] et [Draw] —, dont la boucle du
// moteur se charge dans les deux cas. Un hôte qui voudrait dessiner lui-même les
// appelle depuis ce paquet.
//
// Le tampon lui-même vit dans [github.com/sprimault/lozengine/render], parce qu'un
// hôte en remplit les champs. Les opérations de ce paquet le prennent donc en premier
// argument au lieu d'être ses méthodes : Go veut qu'une méthode soit déclarée dans le
// paquet de son type.
//
// Il consomme un rendu déjà placé et rangé dans l'ordre de dessin, dont les quads
// hors tampon ont été retirés : il ne décide d'aucun ordre.
// S'il devait trier, projeter ou consulter la scène, la frontière serait mal
// placée et le correctif appartiendrait au noyau, pas à lui.
//
// Ce n'est pas un échafaudage en attendant les backends système : c'est le moteur
// définitif, et il reste la référence de vérité quand ils arriveront. C'est lui
// que la suite de non-régression interroge, ce qui la rend exécutable sans écran.
package raster
