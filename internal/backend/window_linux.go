// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La fenêtre X11 : ce qu'elle retient des événements du serveur, et les six réponses
// qu'elle doit à la boucle.
//
// La différence de fond avec Windows tient en une phrase : là-bas le système rappelle
// une procédure pendant qu'on pompe, ici on lit un flux et on en décide soi-même.
// Aucun verrou donc, et aucune table de poignées — rien ne nous appelle. Mais aussi :
// tout ce qui arrive peut être un échec de transport, ce qu'une file de messages
// locale ne connaît pas.

package backend

import (
	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
)

// window est une fenêtre ouverte et ce que le serveur en a dit depuis la dernière
// image.
type window struct {
	c  *conn
	id uint32

	// gc est le contexte graphique par lequel l'image entre dans la fenêtre. Créé
	// une fois à l'ouverture : il ne porte que des défauts, donc rien n'oblige à le
	// refaire quand la fenêtre change de taille.
	gc uint32

	// protocols et deleteWindow sont les deux atomes par lesquels se reconnaît la
	// demande de fermeture : le type du message pour le premier, son contenu pour
	// le second.
	protocols    uint32
	deleteWindow uint32

	// state accumule ce que le clavier a fait. Le décodage des touches vient avec le
	// lot qui lit le clavier ; ce qui est déjà câblé est l'oubli à la perte du
	// focus, parce que sans lui une touche tenue le reste pour toujours.
	state input.State

	// closed retient que la fenêtre ne présentera plus rien. Il ne se relève jamais.
	closed bool

	// err porte ce qui a mis fin à la fenêtre autrement que par la volonté du
	// joueur. Seule la première cause est gardée : ce qui suit une connexion perdue
	// n'en est que la conséquence.
	err error

	// width et height sont la taille que le serveur a annoncée en dernier.
	//
	// Elles n'ont pas besoin du traitement que Win32 réclame pour une fenêtre
	// réduite : X11 ne redimensionne pas une fenêtre qu'on masque, il cesse
	// seulement de l'afficher. La taille reste donc celle qu'elle avait, et la
	// boucle garde un tampon valide.
	width  uint16
	height uint16

	// mapped dit si la fenêtre est affichée. Elle ne sert pas encore à décider quoi
	// que ce soit — le transfert d'image sur une fenêtre masquée est permis et sans
	// effet — mais elle est ce qui distinguera plus tard « rien à peindre » de
	// « plus rien à peindre jamais ».
	mapped bool
}

// Input lit les événements qui attendent puis rend ce que le clavier dit au pas qui
// commence.
//
// C'est ici que le flux se vide, parce que c'est le premier appel d'un pas : la boucle
// lit les entrées, met à jour, puis présente. Lire ailleurs ferait répondre la fenêtre
// en retard d'une image, et la taille servie serait celle du pas précédent.
func (w *window) Input() input.Input {
	// Les événements mis de côté pendant l'ouverture passent d'abord, et dans
	// l'ordre où ils sont arrivés : c'est là que vient la première taille réelle,
	// celle qu'un gestionnaire de fenêtres a imposée à la fenêtre en la plaçant.
	for _, m := range w.c.pending {
		w.handle(m)
	}
	w.c.pending = w.c.pending[:0]

	for !w.closed {
		m, err := w.c.poll()
		if err != nil {
			w.fail(err)
			break
		}
		if m == nil {
			break
		}
		w.handle(m)
	}

	return w.state.Frame()
}

// handle met à jour la fenêtre d'après un message du serveur.
//
// Elle prend le message tel qu'il est arrivé, et c'est ce qui la rend vérifiable sans
// serveur : un événement est trente-deux octets, donc un cas de test en écrit un et
// regarde ce que la fenêtre en retient.
func (w *window) handle(m []byte) {
	// Le bit de poids fort dit qu'un autre client a fabriqué l'événement plutôt que
	// le serveur. Il se masque avant de comparer, et c'est indispensable ici : la
	// demande de fermeture arrive toujours ainsi.
	switch m[0] & ^byte(eventSentBit) {
	case msgError:
		// Une erreur à ce moment accuse une requête qui n'avait pas de réponse à
		// elle. Elle est terminale : ce qui la suivait s'est appuyé sur un objet
		// que le serveur a refusé de créer.
		w.fail(protocolError(m))

	case msgReply:
		// Aucune requête à réponse ne part pendant une image. Une réponse ici
		// voudrait dire que le comptage des séquences a dérivé, et la jeter
		// silencieusement laisserait chercher le décalage ailleurs.
		w.fail(protocolError(m))

	case eventConfigureNotify:
		// La taille vient du serveur et non d'une question qu'on lui pose : c'est
		// ce que le contrat appelle un accesseur pur, et ce qui épargne un
		// aller-retour par image pour une valeur déjà annoncée.
		if width, height := order.Uint16(m[20:]), order.Uint16(m[22:]); width != 0 && height != 0 {
			w.width, w.height = width, height
		}

	case eventMapNotify:
		w.mapped = true

	case eventUnmapNotify:
		// Masquée, pas fermée. C'est ce que fait un gestionnaire de fenêtres quand
		// on réduit la fenêtre, et le prendre pour une fermeture arrêterait le jeu
		// à la première réduction.
		w.mapped = false

	case eventDestroyNotify:
		// Là, c'est fini : la fenêtre n'existe plus côté serveur. Cela arrive sans
		// qu'on l'ait demandé quand le gestionnaire de fenêtres ferme de force.
		w.closed = true

	case eventClientMessage:
		// La croix cliquée. Le type du message dit de quel protocole il relève, et
		// son premier mot de données lequel : vérifier les deux évite de prendre
		// pour une fermeture un message d'un autre protocole que nous n'avons pas
		// demandé.
		if order.Uint32(m[8:]) == w.protocols && order.Uint32(m[12:]) == w.deleteWindow {
			w.closed = true
		}

	case eventFocusOut:
		// Le relâchement d'une touche tenue est perdu, pas différé : la fenêtre ne
		// reçoit plus rien après, et le serveur dirait encore « enfoncée » si on
		// le lui demandait. Oublier est la seule parade, sans quoi le personnage
		// marcherait tout seul au retour.
		w.state.Forget()
	}
}

// fail retient la première cause d'une fin qui n'est pas celle du joueur.
//
// La fenêtre est déclarée fermée du même geste : il n'y a rien à présenter sur une
// connexion perdue, et la boucle a besoin d'une seule question pour s'arrêter.
func (w *window) fail(err error) {
	if w.err == nil {
		w.err = err
	}
	w.closed = true
}

// Present pose le tampon à l'écran, converti dans l'ordre que le serveur attend.
//
// **Les pixels sont recopiés, et c'est le prix de cette plateforme.** Windows n'en
// paie aucun : il décrit ses masques au système, qui lit le tampon là où il est. X11
// impose les siens, donc la conversion est à nous — c'est ce que la frontière du
// noyau prévoit, chaque backend posant le tampon comme sa plateforme l'exige et
// mesurant son coût chez lui. La copie n'a pas lieu en plus du transfert : elle
// *est* le remplissage du tampon d'émission.
//
// **Le découpage en bandes n'est pas une précaution, c'est le protocole.** Une
// requête est bornée, et une image de la taille d'un écran dépasse largement la
// borne du handshake. L'extension qui la lève est demandée à l'ouverture, et quand
// elle manque le découpage reste la seule voie.
//
// Sans erreur, comme le contrat l'exige : une présentation qui ne peut pas avoir lieu
// ne fait rien, et il n'y a personne à qui se plaindre soixante fois par seconde. Un
// lien coupé pendant l'envoi met fin à la fenêtre, et c'est [window.Err] qui le dira.
func (w *window) Present(t *render.Buffer) {
	if w.closed || w.gc == 0 || t == nil || len(t.Pixels) == 0 {
		return
	}

	lines := w.c.linesPerRequest(t.Width)
	if lines == 0 {
		// Une seule ligne ne tient pas dans une requête, ce qui demanderait une
		// fenêtre de plus de seize mille pixels de large sur le plafond le plus
		// bas. Ne rien envoyer vaut mieux qu'envoyer une requête que le serveur
		// refusera en coupant la connexion.
		return
	}

	for y := 0; y < int(t.Height); y += lines {
		height := lines
		if y+height > int(t.Height) {
			height = int(t.Height) - y
		}
		from, to := y*int(t.Width), (y+height)*int(t.Width)

		w.c.putImage(w.id, w.gc, w.c.setup.depth, int16(y), t.Width, uint16(height), t.Pixels[from:to])
	}

	if err := w.c.flush(); err != nil {
		w.fail(err)
	}
}

// Size rend les dimensions de la zone utile, en pixels.
//
// C'est la dernière taille que le serveur a annoncée, et il n'y a ici ni bordure ni
// barre de titre à retrancher : celles-ci appartiennent à une fenêtre que le
// gestionnaire met autour de la nôtre, et la nôtre ne mesure que ce qu'on y peint.
func (w *window) Size() (width, height uint16) {
	return w.width, w.height
}

// Closed dit que la fenêtre ne présentera plus rien.
func (w *window) Closed() bool {
	return w.closed
}

// Err dit ce qui a mis fin à la fenêtre, et nil quand c'est le joueur.
//
// Contrairement à Windows, le cas non nul est ici courant et attendu : tout passe par
// une socket, donc un serveur qui s'arrête, une session qui se termine ou un lien
// coupé mettent fin à la fenêtre sans que personne n'ait rien fermé. C'est ce membre
// du contrat qui sépare cela d'une croix cliquée.
func (w *window) Err() error {
	return w.err
}

// Close rend au serveur ce qu'Open a pris. Appeler deux fois est sans effet.
func (w *window) Close() {
	// L'état se pose d'abord : une fenêtre fermée ne présentera plus rien, et c'est
	// vrai au second appel comme au premier.
	w.closed = true

	if w.c == nil {
		return
	}

	// La destruction est demandée puis envoyée, et l'échec de l'envoi est ignoré :
	// on est en train de partir, et la fermeture de la socket détruit de toute façon
	// tout ce que ce client avait créé.
	w.c.destroyWindow(w.id)
	_ = w.c.flush()
	w.c.Close()
	w.c = nil
}
