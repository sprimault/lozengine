// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La mémoire que le client et le compositeur se partagent : un fichier sans nom, sa
// projection, et les tampons que la fenêtre fait tourner dedans.
//
// **Le client écrit dans la mémoire que le compositeur lit, et c'est tout le dispositif**
// — il n'y a pas de transfert, donc pas de copie vers une socket comme sous X11. Ce que
// cela demande en échange est de savoir **quand** le compositeur a fini de lire, sans quoi
// l'image suivante s'écrirait sous ses yeux. C'est ce que `wl_buffer.release` dit, et c'est
// la raison des deux tampons.

package backend

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sprimault/lozengine/render"
)

// wlBuffers est le nombre de tampons que la fenêtre fait tourner.
//
// **Deux, et la description dit pourquoi un pourrait suffire sans le garantir** : « if a
// client receives a release event before the frame callback requested in the same
// wl_surface.commit […] then the client is immediately free to reuse the buffer […] and
// does not need a second buffer ». Elle ajoute aussitôt que c'est le cas « typically », et
// seulement quand le compositeur garde une copie de la surface. Un client à un seul tampon
// dépend donc d'une optimisation du compositeur, et le mode de panne est une image écrite
// pendant qu'elle est lue — du déchirement qu'aucune erreur ne signale.
//
// Trois n'achèteraient rien de plus : avec deux, un tour qui ne trouve rien de libre saute
// sa présentation, ce que le contrat de fenêtre autorise, et le compositeur garde la
// dernière image.
const wlBuffers = 2

// wlPoolBuffer est un tampon dans le pool, et ce que le compositeur en fait.
type wlPoolBuffer struct {
	id uint32

	// at est le décalage du tampon dans la projection.
	at int

	// held dit que le compositeur ne l'a pas encore rendu. Écrire dedans alors
	// qu'il le lit ferait du déchirement sans qu'aucune erreur ne soit levée.
	held bool
}

// wlPool est la mémoire partagée d'une géométrie donnée.
//
// **Elle ne s'agrandit pas : elle se remplace.** `wl_shm_pool.resize` existe et saurait
// agrandir, mais il garde la projection du client sur la même mémoire — donc réécrire aux
// mêmes décalages obligerait à attendre que le compositeur ait rendu ses tampons, ce qu'une
// présentation ne peut pas faire sans bloquer. Un pool neuf laisse l'ancienne mémoire au
// compositeur seul, qui la lit par sa propre projection et son propre descripteur, et la
// rend quand il a fini sans que personne ne l'attende.
type wlPool struct {
	// file porte la mémoire et n'a plus de nom : il est retiré de son répertoire
	// aussitôt créé.
	file *os.File

	// mem est la projection partagée. Les pixels d'une image s'y écrivent
	// directement, ce qui est la seule copie du chemin de présentation.
	mem []byte

	id uint32

	// width, height et stride décrivent la géométrie des tampons, qui est figée : un
	// `wl_buffer` porte ses dimensions, donc une fenêtre qui change de taille demande
	// un pool neuf.
	width, height int
	stride        int

	buffers [wlBuffers]wlPoolBuffer

	// last est le tampon employé en dernier, pour que le suivant soit l'autre. Sans
	// cela, un compositeur qui rend le tampon avant la présentation suivante ferait
	// toujours réemployer le même, et le second ne servirait jamais — donc le défaut
	// qu'il garde ne se verrait qu'une fois en production.
	last int
}

// wlShmDirs énumère les répertoires où le fichier de mémoire partagée peut se créer, dans
// l'ordre où on les essaie.
//
// `XDG_RUNTIME_DIR` d'abord, qui est l'endroit prévu pour ce qu'une session partage ;
// `/dev/shm` ensuite, qui est de la mémoire et existe presque partout ; le répertoire
// temporaire en dernier, qui peut être sur un disque et n'est donc pas un bon endroit pour
// une image par tour — mais qui vaut mieux que de ne pas ouvrir de fenêtre.
//
// **Le repli vaut ici et ne vaut pas pour la socket**, dont la godoc dit l'inverse : une
// socket est posée par le compositeur, donc la chercher ailleurs ne la trouverait pas, là
// qu'un fichier qu'on crée soi-même se crée où l'on peut.
func wlShmDirs() []string {
	dirs := make([]string, 0, 3)
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		dirs = append(dirs, dir)
	}
	return append(dirs, "/dev/shm", os.TempDir())
}

// wlMemoryFile crée le fichier qui porte la mémoire partagée, de la taille demandée.
//
// **Il est retiré de son répertoire aussitôt créé**, et le descripteur suffit à tout ce
// qui suit — le compositeur en reçoit un duplicata par les données auxiliaires, et plus
// personne ne peut l'ouvrir par son nom. C'est ce qui remplace `memfd_create`, dont le
// numéro d'appel système manque à `syscall` pour trois des cinq architectures Linux
// visées.
func wlMemoryFile(size int) (*os.File, error) {
	var last error

	for _, dir := range wlShmDirs() {
		file, err := os.CreateTemp(dir, "lozengine-*")
		if err != nil {
			last = err
			continue
		}
		if err := os.Remove(file.Name()); err != nil {
			file.Close()
			last = err
			continue
		}
		if err := file.Truncate(int64(size)); err != nil {
			file.Close()
			last = err
			continue
		}
		return file, nil
	}
	return nil, fmt.Errorf("creating a %d-byte shared memory file for the compositor: %w", size, last)
}

// wlNewPool monte un pool et ses tampons pour la géométrie donnée.
//
// L'envoi a lieu ici et non chez l'appelant, parce que le descripteur doit accompagner le
// message qui le déclare : le laisser attendre un envoi ultérieur marcherait — les
// descripteurs forment une file — mais ferait dépendre la justesse d'un ordre écrit
// ailleurs.
func wlNewPool(c *wlConn, g wlGlobals, width, height int) (*wlPool, error) {
	stride := width * 4
	frame := stride * height
	size := frame * wlBuffers

	file, err := wlMemoryFile(size)
	if err != nil {
		return nil, err
	}

	mem, err := syscall.Mmap(int(file.Fd()), 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("mapping the %d-byte shared memory file: %w", size, err)
	}

	p := &wlPool{file: file, mem: mem, width: width, height: height, stride: stride}

	if p.id, err = c.newID(); err != nil {
		p.unmap()
		return nil, err
	}
	c.out.begin(g.shm, wlShmCreatePool)
	c.out.uint(p.id)
	c.out.uint(uint32(size))
	c.out.end()
	c.sendFD(int(file.Fd()))

	for i := range p.buffers {
		id, err := c.newID()
		if err != nil {
			p.unmap()
			return nil, err
		}
		p.buffers[i] = wlPoolBuffer{id: id, at: i * frame}

		c.out.begin(p.id, wlShmPoolCreateBuffer)
		c.out.uint(id)
		c.out.uint(uint32(i * frame))
		c.out.uint(uint32(width))
		c.out.uint(uint32(height))
		c.out.uint(uint32(stride))
		c.out.uint(g.format)
		c.out.end()
	}

	// Le pool se détruit dès ses tampons créés, ce que la description autorise
	// explicitement : « a buffer will keep a reference to the pool it was created from
	// so it is valid to destroy the pool immediately after creating a buffer from it ».
	// Ce qui disparaît est l'objet du protocole, pas la mémoire.
	c.out.begin(p.id, wlShmPoolDestroy)
	c.out.end()

	if err := c.flush(); err != nil {
		p.unmap()
		return nil, err
	}
	p.last = wlBuffers - 1
	return p, nil
}

// write pose les pixels dans le tampon qui commence au décalage donné.
//
// **C'est la seule copie du chemin de présentation**, et il n'y en a pas de seconde : le
// tampon de sortie du moteur est lu une fois et écrit dans la mémoire que le compositeur
// lira. X11 paie la même conversion plus un transfert sur socket.
//
// La disposition du moteur est reconnue à part, et ce n'est pas une optimisation
// opportuniste : [bytesOf] dit déjà que Wayland est dans ce cas dès qu'il accorde `XB24`.
// L'octet d'alpha tombe alors dans l'octet que le format déclare ignoré, et la boucle
// qu'on évite est la plus parcourue du backend.
//
// **Ce que ce cas à part achète est mesuré, et c'est un facteur treize** : 142 µs contre
// 1,89 ms à 1920×1080 sur la machine de contrôle, soit 0,85 % du budget d'une image contre
// 11,4 %. Le format accordé décide donc à lui seul si cette famille a le chemin de
// présentation le moins cher du dépôt ou l'un des plus chers, et un jeu n'y peut rien :
// c'est le compositeur qui annonce ce qu'il accepte. Voir [BenchmarkWaylandWrite].
//
// La longueur écrite vient du nombre de pixels et non de la géométrie du pool, parce que
// l'appelant vient d'ajuster l'une à l'autre. Le découpage à trois indices borne quand même
// la capacité au tampon : sous une incohérence, l'extension irait au tas et l'image ne
// paraîtrait pas, là qu'une capacité plus large écrirait dans le tampon voisin.
func (p *wlPool) write(at int, o pixelOrder, pixels []render.Color) {
	frame := p.stride * p.height

	if o.kind == pixelDirect {
		copy(p.mem[at:at+frame], bytesOf(pixels))
		return
	}
	o.appendPixels(p.mem[at:at:at+frame], pixels)
}

// free rend l'index d'un tampon libre, ou -1 quand le compositeur détient tout.
//
// La recherche part de celui qui suit le dernier employé, ce qui fait tourner les tampons
// au lieu de réemployer le même tant qu'il est libre.
func (p *wlPool) free() int {
	for n := 1; n <= wlBuffers; n++ {
		i := (p.last + n) % wlBuffers
		if !p.buffers[i].held {
			return i
		}
	}
	return -1
}

// returned marque un tampon comme rendu par le compositeur, et dit si l'identifiant était
// l'un des siens.
func (p *wlPool) returned(id uint32) bool {
	for i := range p.buffers {
		if p.buffers[i].id == id {
			p.buffers[i].held = false
			return true
		}
	}
	return false
}

// drop rend la mémoire et détruit ce que le pool a créé chez le compositeur.
//
// **Les tampons se détruisent même détenus**, et c'est sans danger ici : ce qui est détruit
// est l'objet du protocole, et le compositeur continue de lire la mémoire par sa propre
// projection — qu'un `munmap` de notre côté ne touche pas. C'est ce qui permet à une
// fenêtre redimensionnée de ne rien attendre.
func (p *wlPool) drop(c *wlConn) {
	for _, b := range p.buffers {
		if b.id == 0 {
			continue
		}
		c.out.begin(b.id, wlBufferDestroy)
		c.out.end()
	}
	p.unmap()
}

// unmap rend la mémoire du côté client, sans rien dire au compositeur.
//
// Elle est idempotente, parce que les deux chemins qui l'appellent se recouvrent : un
// montage qui échoue à mi-course, et la fermeture ordinaire.
func (p *wlPool) unmap() {
	if p.mem != nil {
		syscall.Munmap(p.mem)
		p.mem = nil
	}
	if p.file != nil {
		p.file.Close()
		p.file = nil
	}
}
