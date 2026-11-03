// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la connexion doit tenir face à un compositeur de pacotille : ne perdre aucun
// descripteur d'où qu'il arrive, garder un message coupé en deux, nommer ce qui l'a fait
// tomber, et réemployer les identifiants qu'on lui rend.

package backend

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

// wlConnPair monte une connexion sur une paire de sockets, et rend le bout du
// compositeur.
//
// Aucun compositeur n'est en jeu : le test écrit lui-même les octets que le protocole
// décrit, ce qui est la seule façon d'éprouver un codage sans dépendre de ce qu'une
// implémentation tierce envoie. Le bout rendu sert à pousser des événements et à joindre
// des descripteurs.
func wlConnPair(t testing.TB) (*wlConn, *os.File) {
	t.Helper()

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("montage de la paire de sockets : %v", err)
	}

	sock := unixConnOf(t, fds[0])
	raw, err := sock.SyscallConn()
	if err != nil {
		t.Fatalf("accès brut à la socket : %v", err)
	}

	c := &wlConn{
		sock:   sock,
		raw:    raw,
		oob:    make([]byte, syscall.CmsgSpace(4*wlFDsPerRead)),
		in:     make([]byte, 0, 4096),
		nextID: wlFirstID,
	}
	t.Cleanup(c.Close)

	server := os.NewFile(uintptr(fds[1]), "compositeur")
	t.Cleanup(func() { server.Close() })
	return c, server
}

// wlEvent compose un événement, comme le compositeur l'enverrait.
func wlEvent(sender uint32, opcode uint16, args ...uint32) []byte {
	var w wlWriter
	w.begin(sender, opcode)
	for _, a := range args {
		w.uint(a)
	}
	w.end()
	return w.buf
}

// TestWLConnTakesADescriptorFromAnyByteOfTheStream garde ce qui distingue cette famille
// d'X11.
//
// **La spécification avertit que « any byte of the stream, even the message header, may
// carry the ancillary data with file descriptors »**, et ce cas pose exactement ce
// régime : le descripteur voyage avec l'en-tête du message, pas avec l'argument qui s'en
// sert. Une lecture ordinaire le ferait fermer par le noyau sans erreur ni trace, et la
// carte du clavier ou le tampon partagé n'arriverait jamais — un gel sans message.
//
// Il rougit si la lecture cesse de porter un tampon de contrôle, ou si la file des
// descripteurs est remplacée par un champ : le descripteur arrive avant le message qui le
// réclame, donc il faut l'avoir rangé.
func TestWLConnTakesADescriptorFromAnyByteOfTheStream(t *testing.T) {
	c, server := wlConnPair(t)

	// Un descripteur quelconque mais identifiable : le compositeur enverrait une
	// carte de clavier, un test se contente d'un tube dont il connaît le contenu.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("montage du tube : %v", err)
	}
	defer r.Close()
	defer w.Close()
	if _, err := w.WriteString("carte"); err != nil {
		t.Fatalf("écriture dans le tube : %v", err)
	}

	msg := wlEvent(wlDisplayID, wlDisplayDeleteID, 7)
	rights := syscall.UnixRights(int(r.Fd()))
	if err := syscall.Sendmsg(int(server.Fd()), msg, rights, nil, 0); err != nil {
		t.Fatalf("envoi du message avec son descripteur : %v", err)
	}

	if err := c.fill(); err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if err := c.dispatch(); err != nil {
		t.Fatalf("découpage : %v", err)
	}

	fd, ok := c.takeFD()
	if !ok {
		t.Fatal("aucun descripteur dans la file, attendu celui que l'en-tête portait")
	}
	defer syscall.Close(fd)

	got := make([]byte, 5)
	if _, err := syscall.Read(fd, got); err != nil {
		t.Fatalf("lecture du descripteur reçu : %v", err)
	}
	if string(got) != "carte" {
		t.Errorf("le descripteur reçu porte %q, attendu %q", got, "carte")
	}
}

// TestWLConnRefusesTruncatedAncillaryData garde le seul mode de panne irréparable.
//
// Le noyau ferme les descripteurs qui ne tiennent pas dans le tampon de contrôle et pose
// `MSG_CTRUNC`. Continuer voudrait dire attendre indéfiniment quelque chose que plus
// personne ne peut livrer, donc la connexion doit tomber en le nommant.
//
// Le cas envoie un descripteur de plus que le tampon n'en porte, ce qui est le seul moyen
// de provoquer la troncature sans toucher au noyau.
func TestWLConnRefusesTruncatedAncillaryData(t *testing.T) {
	c, server := wlConnPair(t)

	opened := make([]int, 0, wlFDsPerRead+1)
	for i := 0; i <= wlFDsPerRead; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("montage du tube %d : %v", i, err)
		}
		defer r.Close()
		defer w.Close()
		opened = append(opened, int(r.Fd()))
	}

	msg := wlEvent(wlDisplayID, wlDisplayDeleteID, 7)
	if err := syscall.Sendmsg(int(server.Fd()), msg, syscall.UnixRights(opened...), nil, 0); err != nil {
		t.Fatalf("envoi des %d descripteurs : %v", len(opened), err)
	}

	err := c.fill()
	if err == nil {
		t.Fatalf("la lecture a réussi, attendu un refus : %d descripteurs pour un tampon de %d",
			len(opened), wlFDsPerRead)
	}
	if !strings.Contains(err.Error(), "file descriptors") {
		t.Errorf("le refus dit %q, attendu qu'il nomme les descripteurs perdus", err)
	}
}

// TestWLConnKeepsAMessageSplitAcrossReads garde le découpage du flux.
//
// Une socket de flux ne respecte aucune frontière de message : un message peut arriver en
// deux morceaux, et deux messages en une seule lecture. Le mode de panne qu'on évite est
// un en-tête lu sur des octets incomplets, donc une taille absurde et une connexion qui
// tombe sur un message parfaitement valide.
func TestWLConnKeepsAMessageSplitAcrossReads(t *testing.T) {
	c, server := wlConnPair(t)

	var seen []uint32
	c.handler = func(sender uint32, opcode uint16, r *wlReader) error {
		seen = append(seen, sender, uint32(opcode), r.uint())
		return nil
	}

	msg := wlEvent(42, 3, 0xcafe)
	if _, err := server.Write(msg[:5]); err != nil {
		t.Fatalf("envoi du premier morceau : %v", err)
	}
	if err := c.fill(); err != nil {
		t.Fatalf("première lecture : %v", err)
	}
	if err := c.dispatch(); err != nil {
		t.Fatalf("premier découpage : %v", err)
	}
	if len(seen) != 0 {
		t.Fatalf("le message a été routé sur %d octets, attendu qu'il attende la suite", 5)
	}

	if _, err := server.Write(msg[5:]); err != nil {
		t.Fatalf("envoi du second morceau : %v", err)
	}
	if err := c.fill(); err != nil {
		t.Fatalf("seconde lecture : %v", err)
	}
	if err := c.dispatch(); err != nil {
		t.Fatalf("second découpage : %v", err)
	}

	if want := []uint32{42, 3, 0xcafe}; len(seen) != 3 || seen[0] != want[0] || seen[1] != want[1] || seen[2] != want[2] {
		t.Errorf("le message routé est %v, attendu %v", seen, want)
	}
}

// TestWLConnNamesTheCompositorError garde ce qu'un échec de protocole rapporte.
//
// C'est la seule information qu'on aura : le protocole est asynchrone, donc l'erreur
// n'arrive pas en réponse à la requête qui l'a causée, et le message du compositeur est ce
// qui dit laquelle. Le reformuler ou le taire laisserait un backend muet devant son propre
// défaut — et c'est le régime courant quand on écrit un backend.
func TestWLConnNamesTheCompositorError(t *testing.T) {
	c, server := wlConnPair(t)

	var w wlWriter
	w.begin(wlDisplayID, wlDisplayError)
	w.uint(12)
	w.uint(wlErrInvalidMethod)
	w.string("surface has no role")
	w.end()

	if _, err := server.Write(w.buf); err != nil {
		t.Fatalf("envoi de l'erreur : %v", err)
	}
	if err := c.fill(); err != nil {
		t.Fatalf("lecture : %v", err)
	}

	err := c.dispatch()
	if err == nil {
		t.Fatal("le découpage a réussi, attendu l'erreur du compositeur")
	}
	for _, want := range []string{"12", "invalid_method", "surface has no role"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("l'erreur dit %q, attendu qu'elle porte %q", err, want)
		}
	}
	if c.err == nil {
		t.Error("la connexion n'a pas retenu l'erreur, donc la boucle ne la rendra pas par Err")
	}
}

// TestWLConnRecyclesIDsAfterDeleteID garde le réemploi des identifiants.
//
// **Ce que ce cas protège n'est pas chez nous, c'est chez le compositeur** : il tient une
// entrée par identifiant jamais libéré, donc un client qui numérote toujours en croissant
// le fait grossir sans borne. À un rappel d'image par tour, cela fait soixante
// identifiants par seconde.
//
// Il garde aussi l'ordre : le dernier rendu repart le premier, ce qui tient l'exigence
// d'allocation dense — tous les identifiants inférieurs ont déjà servi.
func TestWLConnRecyclesIDsAfterDeleteID(t *testing.T) {
	c, server := wlConnPair(t)

	first, err := c.newID()
	if err != nil {
		t.Fatalf("premier identifiant : %v", err)
	}
	second, err := c.newID()
	if err != nil {
		t.Fatalf("second identifiant : %v", err)
	}
	if second != first+1 {
		t.Fatalf("les identifiants neufs valent %d puis %d, attendu une suite dense", first, second)
	}

	if _, err := server.Write(wlEvent(wlDisplayID, wlDisplayDeleteID, first)); err != nil {
		t.Fatalf("envoi de delete_id : %v", err)
	}
	if err := c.fill(); err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if err := c.dispatch(); err != nil {
		t.Fatalf("découpage : %v", err)
	}

	again, err := c.newID()
	if err != nil {
		t.Fatalf("identifiant après delete_id : %v", err)
	}
	if again != first {
		t.Errorf("l'identifiant suivant vaut %d, attendu %d : celui que le compositeur a rendu", again, first)
	}
}

// TestWLConnRefusesAMessageBelowItsHeader garde la borne basse de la taille annoncée.
//
// Une taille sous huit ne laisse pas de place à l'en-tête qui la porte, donc le découpage
// n'avancerait jamais : il relirait indéfiniment les mêmes octets. C'est une boucle
// infinie et non une erreur, ce qui en fait le pire mode de panne du découpage.
func TestWLConnRefusesAMessageBelowItsHeader(t *testing.T) {
	c, server := wlConnPair(t)

	msg := make([]byte, wlHeaderSize)
	wlOrder.PutUint32(msg, wlDisplayID)
	wlOrder.PutUint32(msg[4:], uint32(4)<<16)
	if _, err := server.Write(msg); err != nil {
		t.Fatalf("envoi du message : %v", err)
	}
	if err := c.fill(); err != nil {
		t.Fatalf("lecture : %v", err)
	}

	err := c.dispatch()
	if err == nil {
		t.Fatal("le découpage a réussi, attendu un refus sur une taille de 4 octets")
	}
	if !strings.Contains(err.Error(), "header") {
		t.Errorf("le refus dit %q, attendu qu'il nomme l'en-tête", err)
	}
}

// TestWLDisplayPathResolvesTheEnvironment garde les trois régimes de résolution.
//
// Ils viennent de la documentation de référence : un nom absolu est employé tel quel
// depuis Wayland 1.15, un nom relatif se résout dans `XDG_RUNTIME_DIR`, et l'absence de
// `WAYLAND_DISPLAY` vaut `wayland-0`, qui est relatif comme le reste.
//
// Le quatrième cas est celui qui compte pour un intégrateur : sans `XDG_RUNTIME_DIR` et
// sans nom absolu, il n'y a aucun répertoire où chercher, et le refus doit nommer la
// variable plutôt que d'échouer sur un chemin construit à moitié.
func TestWLDisplayPathResolvesTheEnvironment(t *testing.T) {
	cases := []struct {
		label   string
		display string
		runtime string
		want    string
		refuses bool
	}{
		{"un nom relatif, dans le répertoire d'exécution", "wayland-1", "/run/user/1000", "/run/user/1000/wayland-1", false},
		{"un nom absolu, tel quel", "/tmp/lozengine-wayland", "/run/user/1000", "/tmp/lozengine-wayland", false},
		{"rien, donc le nom par défaut", "", "/run/user/1000", "/run/user/1000/" + wlDefaultDisplay, false},
		{"un nom relatif sans répertoire d'exécution", "wayland-1", "", "", true},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			t.Setenv("WAYLAND_DISPLAY", c.display)
			t.Setenv("XDG_RUNTIME_DIR", c.runtime)

			got, err := wlDisplayPath()
			if c.refuses {
				if err == nil {
					t.Fatalf("le chemin rendu est %q, attendu un refus", got)
				}
				if !strings.Contains(err.Error(), "XDG_RUNTIME_DIR") {
					t.Errorf("le refus dit %q, attendu qu'il nomme la variable manquante", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("résolution : %v", err)
			}
			if got != c.want {
				t.Errorf("le chemin rendu est %q, attendu %q", got, c.want)
			}
		})
	}
}
