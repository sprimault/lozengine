// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la présentation Wayland coûte, et ce que cette famille ne paie pas.
//
// **Les deux bancs se lisent ensemble** : l'un mesure l'écriture dans la mémoire que le
// compositeur lit, l'autre la présentation entière autour d'elle. Leur rapport dit si le
// chemin a autre chose qu'une copie, et c'est ce qu'on relit — les durées, elles, sont des
// faits de la machine qui les a produites.
//
// **Aucun des deux ne demande de compositeur réel**, et c'est voulu : l'écriture n'en a
// pas besoin, et la présentation est servie par le compositeur de pacotille, qui parle le
// protocole et rend les tampons. Ce qu'ils mesurent est donc le coût **client**, le seul
// qui entre dans le budget d'une image du moteur — ce qu'un compositeur dépense ensuite
// vit dans un autre processus. Les deux tournent partout sur Linux, `make check` et
// `make remote` compris.

package backend

import (
	"syscall"
	"testing"

	"github.com/sprimault/lozengine/internal/bench"
	"github.com/sprimault/lozengine/render"
)

// wlBenchSizes sont les trois surfaces que les bancs de cette famille mesurent.
//
// Les mêmes qu'en X11, et pour la même raison : ce ne sont pas trois points d'un
// échantillon mais trois questions — la résolution interne du moteur, un écran courant, et
// un écran complet. L'écriture grandissant avec la surface, c'est la plus grande qui
// décide.
func wlBenchSizes() []struct {
	name          string
	width, height uint16
} {
	return []struct {
		name          string
		width, height uint16
	}{
		{"480×270, la résolution interne du moteur", 480, 270},
		{"1280×720", 1280, 720},
		{"1920×1080, un écran entier", 1920, 1080},
	}
}

// wlBenchFormats sont les deux formats qu'un compositeur peut accorder, et les deux
// chemins d'écriture qui en découlent.
//
// **Les deux, parce que le format décide de la nature du chemin et non de sa vitesse** :
// sous `xbgr8888` la disposition du moteur est déjà celle de la destination, donc
// l'écriture est une copie en bloc ; sous `xrgb8888` il faut permuter deux composantes par
// pixel. Ne mesurer que le premier annoncerait un coût que la moitié des compositeurs ne
// paie pas, et ne mesurer que le second l'inverse.
func wlBenchFormats() []struct {
	name   string
	code   uint32
	advert uint32
} {
	return []struct {
		name   string
		code   uint32
		advert uint32
	}{
		{"xbgr8888, l'ordre du moteur", formatXBGR8888, formatXBGR8888},
		{"xrgb8888, le rouge et le bleu échangés", formatXRGB8888, wlShmXRGB8888},
	}
}

// wlBenchPool monte une mémoire partagée de la géométrie donnée, sans compositeur.
//
// **Elle refait les trois gestes de [wlNewPool] qui ne parlent pas au protocole**, et
// c'est la seule voie : le reste de cette fonction crée des objets chez un compositeur, et
// ce qu'on mesure ici est l'écriture dans la projection. La duplication est de trois
// lignes et son pendant est mesuré par le second banc, qui passe par le vrai chemin.
//
// La projection est bien celle d'un fichier partagé et non une tranche du tas : une
// écriture y paie ses défauts de page au premier accès, et c'est précisément ce qu'une
// mesure doit sortir de sa boucle.
func wlBenchPool(tb testing.TB, width, height int) *wlPool {
	tb.Helper()

	stride := width * 4
	size := stride * height * wlBuffers

	file, err := wlMemoryFile(size)
	if err != nil {
		tb.Fatalf("fichier de mémoire partagée : %v", err)
	}

	mem, err := syscall.Mmap(int(file.Fd()), 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		tb.Fatalf("projection de %d octets : %v", size, err)
	}

	p := &wlPool{file: file, mem: mem, width: width, height: height, stride: stride}
	tb.Cleanup(p.unmap)
	return p
}

// wlBenchFrame rend une image dont chaque pixel diffère de ses voisins.
//
// Un tampon uni laisserait un compilateur ou un cache en tirer parti, et surtout il ne
// distinguerait pas une écriture par mot d'une écriture par composante.
func wlBenchFrame(width, height uint16) *render.Buffer {
	buffer := render.NewBuffer(width, height)
	for i := range buffer.Pixels {
		buffer.Pixels[i] = render.Color{
			R: uint8(i), G: uint8(i >> 8), B: uint8(i >> 16), A: 255,
		}
	}
	return buffer
}

// BenchmarkWaylandWrite mesure l'écriture d'une image dans la mémoire partagée.
//
// **C'est la seule copie du chemin de présentation de cette famille**, et c'est ce qui la
// distingue des deux autres : les pixels sont écrits là où le compositeur les lira, donc
// il n'y a ni transfert sur socket comme en X11, ni remise au système comme sous Windows.
//
// **Ce que ce banc existe pour trancher est la destination et non la conversion.** La
// conversion elle-même est déjà mesurée vers un tampon du tas par [BenchmarkAppendPixels],
// et la godoc de [pixelOrder.appendPixels] annonce depuis le jalon 3 que ses chiffres ne
// valent que pour cette destination — « une mémoire partagée projetée ou un tampon dumb du
// noyau sont une autre destination, et le lot qui présente les mesure ». C'est ce lot, et
// le rapport entre les deux bancs est la réponse.
//
// **Le premier tour est hors de la boucle**, et il ne paie pas la même chose qu'en X11 :
// là-bas c'est l'agrandissement du tampon d'émission, ici les défauts de page de la
// projection. Les deux sont du coût de première image, qu'une mesure de la soixantième ne
// doit pas porter.
//
// # Ce que le lot a mesuré
//
// Six séries sur la machine de contrôle le 4 novembre 2026, à 1920×1080, en part brute du
// budget d'une image à soixante par seconde :
//
//   - `xbgr8888` 142 µs, soit 58,4 Go/s et **0,85 %** ;
//   - `xrgb8888` 1,89 ms, soit 4,38 Go/s et **11,4 %**.
//
// **Le format accordé décide donc d'un facteur treize sur le chemin le plus chaud de cette
// famille**, et c'est le fait que le lot cherchait. Il ne se choisit pas : le compositeur
// annonce ce qu'il accepte, et le backend demande `XB24` d'abord.
//
// **La destination, elle, ne change rien, et c'est la réserve que ce lot lève.** La godoc
// de [pixelOrder.appendPixels] annonce depuis le jalon 3 que ses chiffres ne valent que
// pour un tampon du tas, « une mémoire partagée projetée ou un tampon dumb du noyau sont
// une autre destination, et le lot qui présente les mesure ». Mesurée dans la même série,
// la conversion échangée rend le même chiffre vers la projection partagée que vers le tas :
// deux millièmes d'écart, dont **le signe s'inverse entre deux séries** — donc rien que le
// banc puisse distinguer. Ce qui compte est la permutation, pas où elle écrit.
//
// **Ce qui porte le facteur treize est donc la forme et non la destination** : sous
// `xbgr8888` l'écriture est un [bytesOf] suivi d'une copie en bloc, là que la même
// disposition directe passée par la boucle de [pixelOrder.appendPixels] rend 1,73 ms dans
// la même série — un facteur 12,2, qui recoupe le facteur douze que la godoc de [bytesOf]
// annonce pour le navigateur.
//
// **Et les deux régimes ne se classent pas pareil**, ce que la division des octets par le
// temps dit : la copie en bloc tient 58,4 Go/s sur une machine qui en porte environ 250 de
// crête, soit le quart — elle est limitée par la mémoire, comme l'agrandissement, donc sa
// part croît sur une machine plus étroite. La conversion échangée tient 4,38 Go/s, soit un
// cinquantième du crête : elle est limitée par ses opérations, donc sa part brute est déjà
// sa part. Ce qu'on n'a pas est une mesure sur une machine modeste, cette famille ne
// tournant pas sur le poste.
func BenchmarkWaylandWrite(b *testing.B) {
	for _, format := range wlBenchFormats() {
		o, err := pixelOrderOfFourCC(format.code)
		if err != nil {
			b.Fatalf("format %s refusé : %v", format.name, err)
		}

		for _, size := range wlBenchSizes() {
			b.Run(format.name+"/"+size.name, func(b *testing.B) {
				p := wlBenchPool(b, int(size.width), int(size.height))
				buffer := wlBenchFrame(size.width, size.height)

				p.write(0, o, buffer.Pixels)

				b.SetBytes(int64(4 * len(buffer.Pixels)))
				b.ResetTimer()

				for range b.N {
					p.write(0, o, buffer.Pixels)
				}
				bench.Part(b)
			})
		}
	}
}

// BenchmarkWaylandPresent mesure la présentation entière, écriture comprise.
//
// **Ce qu'il ajoute au banc d'écriture est tout ce qui n'est pas la copie** : une lecture
// non bloquante de la socket, le choix d'un tampon libre, trois requêtes encodées et un
// envoi. Son écart avec [BenchmarkWaylandWrite] est donc ce que le protocole coûte par
// image, et c'est le seul chiffre qui dise si cette famille a autre chose à payer que ses
// pixels.
//
// **Le compositeur de pacotille sert ici de pair et non de juge**, et c'est le seul banc du
// dépôt dans ce régime. Deux conséquences à ne pas perdre. Il rend le tampon au commit,
// comme la description le dit « typically », donc la rotation ne s'arrête pas — sans quoi le
// troisième tour sauterait sa présentation et la mesure moyennerait des tours qui ne font
// rien. Et il tourne dans une goroutine du même processus, donc il dispute ses cycles au
// banc : la durée mesurée est une **borne haute** de ce que le même client paierait devant
// un compositeur sur un autre cœur.
//
// **La part des tours qui ont vraiment présenté est rapportée, et elle se lit avant la
// durée** : un tour qui trouve les deux tampons détenus revient aussitôt, donc une moyenne
// prise sans ce témoin vaudrait d'autant moins qu'elle paraîtrait meilleure. Elle se lit
// sans sonde parce que la rotation la dit : une présentation effective change toujours le
// dernier tampon employé, [wlPool.free] partant de celui qui le suit.
//
// **Et le témoin a mordu, ce qui borne le dispositif plutôt que le moteur.** Mesuré le
// 4 novembre 2026 sur la machine de contrôle : il vaut 1 à 1920×1080 et 0,95 à 1280×720
// sous `xbgr8888`, mais 0,73 à 480×270 — où le tour du client tombe sous six microsecondes,
// donc du même ordre que celui que le compositeur de pacotille met à rendre le tampon. Il
// vaut 0,98 et plus à toutes les tailles sous `xrgb8888`, dont la conversion laisse au pair
// le temps qu'il lui faut. **Les durées à lire sont donc celles dont le témoin vaut 1** : aux
// autres, le pair est le goulot et la moyenne compte des tours vides.
//
// **Un témoin nul ne fait pas échouer le banc, et il l'a fait d'abord à tort** : `go test`
// calibre par un tour unique, donc un seul tour tombant sur deux tampons détenus suffit à
// l'obtenir — le banc rougissait une série sur deux à 480×270. Ce que le filet devait
// garder est gardé ailleurs et par un cas déterministe, la rotation de
// [TestWLPoolTurnsItsBuffers].
//
// # Ce que le lot a mesuré
//
// Même série que [BenchmarkWaylandWrite], à 1920×1080, où le témoin vaut 1 : 144 µs sous
// `xbgr8888` et 1,894 ms sous `xrgb8888`, soit 0,86 % et 11,4 % du budget d'une image.
//
// **L'écart avec l'écriture seule est de deux microsecondes, soit un centième de pour cent
// du budget — la présentation Wayland *est* son écriture.** Le drain, le choix du tampon,
// les trois requêtes et l'envoi tiennent ensemble dans ce qu'un seul pixel sur mille coûte,
// et il n'y a donc aucun maillon à chercher ailleurs dans cette famille.
//
// **Ce qui la situe parmi les trois familles livrées** : sous `XB24`, elle a le chemin de
// présentation le moins cher du dépôt — 0,86 % contre 3,7 % pour le navigateur et 11,5 %
// pour Windows. Sous le repli `xrgb8888`, elle vaut la présentation Windows. La différence
// entre les deux n'est pas une conception mais un format que le compositeur accorde.
func BenchmarkWaylandPresent(b *testing.B) {
	for _, format := range wlBenchFormats() {
		for _, size := range wlBenchSizes() {
			b.Run(format.name+"/"+size.name, func(b *testing.B) {
				f := wlNewFake(wlDefaultGlobals(), []uint32{format.advert})
				w, err := wlMount(b, f, Options{
					Title: "lozengine", Width: size.width, Height: size.height,
				})
				if err != nil {
					b.Fatalf("montage de la fenêtre : %v", err)
				}
				b.Cleanup(w.Close)

				buffer := wlBenchFrame(size.width, size.height)

				// Trois tours hors de la boucle : le premier monte la mémoire
				// partagée, et les deux suivants touchent chacun des deux tampons,
				// dont les défauts de page sont du coût de première image.
				for range 3 {
					w.Present(buffer)
				}
				if err := w.Err(); err != nil {
					b.Fatalf("présentation initiale : %v", err)
				}
				if w.pool == nil {
					b.Fatal("la présentation n'a monté aucune mémoire partagée")
				}

				presented := 0

				b.SetBytes(int64(4 * len(buffer.Pixels)))
				b.ResetTimer()

				for range b.N {
					before := w.pool.last
					w.Present(buffer)
					if w.pool.last != before {
						presented++
					}
				}
				bench.Part(b)
				b.ReportMetric(float64(presented)/float64(b.N), "présentées/op")

				// Le relevé se lit après la boucle et jamais pendant : une écriture
				// dans la mesure fabrique le symptôme qu'on cherche.
				b.StopTimer()
				if err := w.Err(); err != nil {
					b.Fatalf("présentation : %v", err)
				}
			})
		}
	}
}
