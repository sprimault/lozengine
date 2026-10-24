// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package raster dessine une liste de quads dans un tampon de pixels, sans carte
// graphique et sans fenêtre.
//
// C'est le rastériseur du moteur, et le mot ne dit rien de plus que cela : peindre
// une forme décrite en recopiant ses pixels un à un, là où une carte graphique le
// ferait pour nous. Le dessin d'un quad s'appelle un blit dans le même esprit — une
// recopie de pixels d'une planche vers le tampon, sans rotation ni étirement.
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
