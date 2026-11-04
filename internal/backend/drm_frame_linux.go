// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le tampon dans lequel le moteur écrit et que le contrôleur balaie : sa fabrication, sa
// projection, et l'écriture d'une image dedans.
//
// **Ce que cette famille a de particulier, et c'est tout le sujet du fichier : le pas de
// ligne n'est pas celui du moteur.** Le pilote aligne la largeur à sa convenance et rend
// le pas qu'il a choisi, donc une image ne se recopie pas d'un bloc — mesuré le 4 novembre
// 2026 sur la machine de contrôle, une largeur de 1366 donne un pas de 5632 octets au lieu
// de 5464, et 1000 donne 4096 au lieu de 4000. Les largeurs de 1920 et 2560 tombent juste,
// ce qui est précisément ce qui rend le défaut difficile à voir.

package backend

import (
	"fmt"
	"syscall"

	"github.com/sprimault/lozengine/render"
)

// drmFrames est le nombre de tampons que la carte fait tourner.
//
// **Deux, pour la raison qui vaut sous Wayland et qui est ici plus dure** : le contrôleur
// balaie le tampon qu'on lui a donné, en continu et sans rien dire. Écrire dedans pendant
// qu'il le lit produit du déchirement qu'aucune erreur ne signale, donc l'image suivante
// s'écrit dans l'autre et la bascule les échange au retour de balayage.
//
// Trois n'achèteraient rien : avec deux, un tour qui trouve une bascule déjà en vol saute
// sa présentation, ce que le contrat de fenêtre autorise, et le contrôleur garde la
// dernière image.
const drmFrames = 2

// drmFrame est un tampon simple, ce que le pilote en a dit, et sa projection.
type drmFrame struct {
	// handle désigne le tampon chez le pilote, fb l'objet qu'`ADDFB2` en a fait. Les
	// deux se détruisent séparément et dans cet ordre inverse.
	handle uint32
	fb     uint32

	// pitch est le nombre d'octets d'une ligne, tel que le pilote l'a choisi. Il vaut
	// au moins quatre fois la largeur et souvent plus.
	pitch uint32

	// mem est la projection du tampon. Les pixels d'une image s'y écrivent
	// directement, ce qui est la seule copie du chemin de présentation.
	mem []byte

	// pending dit qu'une bascule vers ce tampon n'a pas encore été honorée. Écrire
	// dedans alors que le contrôleur s'y prépare ferait du déchirement.
	pending bool
}

// drmNewFrame fabrique un tampon de la géométrie donnée et le rend prêt à écrire.
//
// **Le format se choisit en l'essayant, et non en l'interrogeant.** Il n'y a pas
// d'énumération des formats qu'un contrôleur accepte sans passer par les plans universels,
// donc la seule voie sûre est de demander [formatXBGR8888] — la disposition du moteur — et
// de retomber sur [formatXRGB8888] si le pilote refuse. C'est la même cascade que `wl_shm`,
// pour la même raison, et elle évite une capacité client que le moteur n'a pas à demander.
//
// **`ADDFB2` et non l'`ADDFB` historique** : mesuré par le brief, en trente-deux bits par
// pixel sur une profondeur de vingt-quatre, l'ancienne requête ne peut exprimer que `XR24`
// — l'ordre que la présentation Windows fabrique par conversion. La nouvelle prend un code
// de format explicite, donc elle sait demander celui du moteur.
//
// L'ordre des gestes n'est pas réversible : le tampon existe, puis il devient un objet
// d'affichage, puis il se projette. Un échec en route défait ce qui a été fait.
func drmNewFrame(fd int, width, height uint16) (drmFrame, pixelOrder, error) {
	var f drmFrame

	// `height` précède `width`, et les permuter ne lève aucune erreur : le tampon
	// aurait la bonne taille et la mauvaise forme.
	create := drmModeCreateDumb{height: uint32(height), width: uint32(width), bpp: 32}
	if err := drmIoctl(fd, drmModeCreateDumbReq, &create); err != nil {
		return f, pixelOrder{}, fmt.Errorf("creating a %d×%d dumb buffer: %w", width, height, err)
	}
	f.handle, f.pitch = create.handle, create.pitch

	order, err := drmAddFrameBuffer(fd, &f, width, height)
	if err != nil {
		f.drop(fd)
		return drmFrame{}, pixelOrder{}, err
	}

	mapping := drmModeMapDumb{handle: f.handle}
	if err := drmIoctl(fd, drmModeMapDumbReq, &mapping); err != nil {
		f.drop(fd)
		return drmFrame{}, pixelOrder{}, fmt.Errorf("asking where to map the dumb buffer: %w", err)
	}

	size := int(create.size)
	f.mem, err = syscall.Mmap(fd, int64(mapping.offset), size,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.drop(fd)
		return drmFrame{}, pixelOrder{}, fmt.Errorf("mapping the %d-byte dumb buffer: %w", size, err)
	}
	return f, order, nil
}

// drmAddFrameBuffer fait du tampon un objet que le contrôleur sait balayer, et rend la
// disposition que le pilote a accordée.
//
// Les deux formats sont essayés dans l'ordre où le moteur les préfère, et l'erreur rendue
// est celle du second : un pilote qui refuse les deux refuse quelque chose de plus
// fondamental qu'un ordre d'octets, et c'est son message qui le dira.
func drmAddFrameBuffer(fd int, f *drmFrame, width, height uint16) (pixelOrder, error) {
	var last error

	for _, code := range [2]uint32{formatXBGR8888, formatXRGB8888} {
		order, err := pixelOrderOfFourCC(code)
		if err != nil {
			return order, err
		}

		cmd := drmModeFBCmd2{
			width:       uint32(width),
			height:      uint32(height),
			pixelFormat: code,
		}
		cmd.handles[0] = f.handle
		cmd.pitches[0] = f.pitch

		if err := drmIoctl(fd, drmModeAddFB2, &cmd); err != nil {
			last = fmt.Errorf("asking the driver for format %s: %w", fourccName(code), err)
			continue
		}
		f.fb = cmd.fbID
		return order, nil
	}
	return pixelOrder{}, last
}

// write pose les pixels d'une image dans le tampon.
//
// **L'écriture est par ligne et non d'un bloc, et c'est le pas du pilote qui l'impose.**
// Une image du moteur a des lignes jointives de quatre octets par pixel ; le tampon a des
// lignes de `pitch` octets, dont la fin peut n'appartenir à personne. Recopier d'un bloc
// décalerait chaque ligne d'un peu plus que la précédente, et l'image sortirait en biais —
// un défaut qui ne se voit pas aux largeurs où le pas tombe juste, c'est-à-dire sur les
// deux tailles qu'on essaie d'abord.
//
// **Le cas où le pas tombe juste est reconnu à part, et ce que cela achète est mesuré** :
// c'est alors une copie unique de toute l'image, là que la boucle en fait une par ligne.
// Et le pas allongé coûte plus que la fragmentation seule — mesuré le 4 novembre 2026,
// écrire 1366×768 dans le tampon du pilote tient 57,5 Go/s contre 74,5 dans une tranche du
// tas de même pas, soit **trente pour cent de plus pour la même arithmétique**. La mémoire
// qu'un pilote prête n'a pas les mêmes propriétés d'écriture que le tas, et c'est en
// écritures fragmentées que cela se voit : à pas aligné, les deux destinations rendent le
// même chiffre. Voir [BenchmarkDRMWrite].
//
// La disposition du moteur est reconnue à part de la même façon, et pour la raison que
// [bytesOf] porte : quand le pilote a accordé `XB24`, les octets sont déjà dans l'ordre et
// il n'y a rien à convertir.
func (f *drmFrame) write(o pixelOrder, t *render.Buffer) {
	width, height := int(t.Width), int(t.Height)
	line := width * 4
	pitch := int(f.pitch)

	if o.kind == pixelDirect {
		src := bytesOf(t.Pixels)
		if pitch == line {
			copy(f.mem, src)
			return
		}
		for y := 0; y < height; y++ {
			copy(f.mem[y*pitch:y*pitch+line], src[y*line:(y+1)*line])
		}
		return
	}

	for y := 0; y < height; y++ {
		row := t.Pixels[y*width : (y+1)*width]
		o.appendPixels(f.mem[y*pitch:y*pitch:y*pitch+line], row)
	}
}

// drop rend au pilote tout ce que le tampon lui a demandé.
//
// L'ordre est l'inverse de la fabrication, et il n'est pas indifférent : l'objet
// d'affichage référence le tampon, donc le détruire d'abord évite que le pilote garde une
// référence sur une poignée qui n'existe plus. La fonction est idempotente, parce que les
// deux chemins qui l'appellent se recouvrent — un montage qui échoue à mi-course, et la
// fermeture ordinaire.
func (f *drmFrame) drop(fd int) {
	if f.mem != nil {
		syscall.Munmap(f.mem)
		f.mem = nil
	}
	if f.fb != 0 {
		id := f.fb
		_ = drmIoctl(fd, drmModeRmFB, &id)
		f.fb = 0
	}
	if f.handle != 0 {
		destroy := drmModeDestroyDumb{handle: f.handle}
		_ = drmIoctl(fd, drmModeDestroyDumbReq, &destroy)
		f.handle = 0
	}
}
