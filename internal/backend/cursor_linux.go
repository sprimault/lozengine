// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le curseur vide : quatre requêtes pour obtenir ce qu'aucune n'offre directement, le
// protocole ne sachant pas cacher un pointeur mais sachant en dessiner un qui ne
// couvre aucun pixel.

package backend

// hideCursor pose sur la fenêtre un curseur qui ne dessine rien.
//
// **Le protocole n'a pas de requête pour cacher un pointeur**, et c'est pourquoi le
// chemin paraît détourné. Un curseur y est fait d'une source et d'un masque, et le
// masque dit quels pixels sont peints : un masque entièrement à zéro donne un curseur
// qui existe, suit la souris et ne couvre rien. L'extension qui ferait mieux n'est pas
// garantie présente, et celle-ci tient dans le cœur du protocole.
//
// La pixmap est effacée explicitement plutôt que supposée vide : le protocole ne dit
// rien du contenu d'une pixmap fraîche, et un serveur qui la rendrait pleine de uns
// peindrait un carré d'un pixel à la place du curseur.
//
// Les ressources ne sont pas libérées : elles vivent autant que la connexion, qui se
// ferme avec la fenêtre. Les rendre demanderait deux requêtes de plus pour neuf octets,
// et poserait la question de l'ordre entre la destruction du curseur et celle de la
// fenêtre qui s'en sert.
func (c *conn) hideCursor(window uint32) {
	pixmap, gc, cursor := c.id(), c.id(), c.id()

	// Une pixmap de profondeur un : un curseur se décrit en noir et blanc, et sa
	// source comme son masque doivent la porter.
	at := len(c.out)
	c.out = append(c.out, opCreatePixmap, 1, 0, 0)
	order.PutUint16(c.out[at+2:], 4)
	c.out = order.AppendUint32(c.out, pixmap)
	c.out = order.AppendUint32(c.out, window)
	c.out = order.AppendUint16(c.out, 1)
	c.out = order.AppendUint16(c.out, 1)
	c.next()

	at = len(c.out)
	c.out = append(c.out, opCreateGC, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 5)
	c.out = order.AppendUint32(c.out, gc)
	c.out = order.AppendUint32(c.out, pixmap)
	c.out = order.AppendUint32(c.out, gcForeground)
	c.out = order.AppendUint32(c.out, 0)
	c.next()

	at = len(c.out)
	c.out = append(c.out, opFillRectangle, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 5)
	c.out = order.AppendUint32(c.out, pixmap)
	c.out = order.AppendUint32(c.out, gc)
	c.out = order.AppendUint16(c.out, 0)
	c.out = order.AppendUint16(c.out, 0)
	c.out = order.AppendUint16(c.out, 1)
	c.out = order.AppendUint16(c.out, 1)
	c.next()

	// La même pixmap sert de source et de masque. Le masque étant nul, la source
	// n'est jamais consultée et les deux couleurs n'ont aucun effet.
	at = len(c.out)
	c.out = append(c.out, opCreateCursor, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 8)
	c.out = order.AppendUint32(c.out, cursor)
	c.out = order.AppendUint32(c.out, pixmap)
	c.out = order.AppendUint32(c.out, pixmap)
	for range 6 {
		c.out = order.AppendUint16(c.out, 0)
	}
	c.next()

	at = len(c.out)
	c.out = append(c.out, opChangeWindow, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 4)
	c.out = order.AppendUint32(c.out, window)
	c.out = order.AppendUint32(c.out, cwCursor)
	c.out = order.AppendUint32(c.out, cursor)
	c.next()
}
