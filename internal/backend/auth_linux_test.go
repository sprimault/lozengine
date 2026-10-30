// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0
// Le choix de l'entrée d'autorisation dans `.Xauthority` : celle de cette machine
// d'abord, le repli quand elle a été renommée, et la lecture gros-boutien des longueurs
// que le format impose.

package backend

import "testing"

// authEntryFields décrit une entrée du fichier d'autorisation telle qu'un cas de test
// la pose.
type authEntryFields struct {
	family                        uint16
	address, number, name, cookie string
}

// authFileOf assemble un fichier d'autorisation à partir des entrées décrites.
//
// Les longueurs y sont écrites en gros-boutien, parce que c'est ce que le format
// impose quelle que soit la machine. Les écrire autrement ici ne ferait pas échouer
// les cas : cela les ferait passer sur un lecteur également faux, ce qui est
// exactement le défaut que ce format a de particulier.
func authFileOf(entries ...authEntryFields) []byte {
	var file []byte

	appendField := func(s string) {
		file = append(file, byte(len(s)>>8), byte(len(s)))
		file = append(file, s...)
	}

	for _, e := range entries {
		file = append(file, byte(e.family>>8), byte(e.family))
		appendField(e.address)
		appendField(e.number)
		appendField(e.name)
		appendField(e.cookie)
	}
	return file
}

// TestPickAuthPrefersTheEntryOfThisMachine vérifie que l'entrée nommant la machine
// l'emporte sur une entrée valable pour toutes.
//
// C'est l'ordre naturel et ce n'est pas un détail de confort : un fichier fusionné par
// un gestionnaire de session porte les deux, et le secret de la session locale est
// celui de la première.
func TestPickAuthPrefersTheEntryOfThisMachine(t *testing.T) {
	file := authFileOf(
		authEntryFields{familyWild, "", "0", authMagicCookie, "large"},
		authEntryFields{familyLocal, "machine", "0", authMagicCookie, "precise"},
	)

	name, data := pickAuth(file, "machine", 0)

	if string(name) != authMagicCookie {
		t.Errorf("protocole %q, attendu %q", name, authMagicCookie)
	}
	if string(data) != "precise" {
		t.Errorf("cookie %q, attendu celui de l'entrée nommant la machine", data)
	}
}

// TestPickAuthFallsBackWhenTheMachineWasRenamed vérifie le repli sur une entrée locale
// dont l'adresse ne correspond plus.
//
// Le fichier garde le nom que la machine portait quand la session s'est ouverte.
// Refuser cette entrée reviendrait à refuser la seule utilisable, et le serveur
// renverrait un refus d'autorisation qui ne ressemblerait pas du tout à un
// changement de nom de machine.
func TestPickAuthFallsBackWhenTheMachineWasRenamed(t *testing.T) {
	file := authFileOf(authEntryFields{familyLocal, "ancien-nom", "0", authMagicCookie, "secret"})

	_, data := pickAuth(file, "nouveau-nom", 0)

	if string(data) != "secret" {
		t.Errorf("cookie %q, attendu celui de l'entrée locale malgré l'adresse", data)
	}
}

// TestPickAuthSkipsWhatItCannotPresent vérifie que les entrées inutilisables sont
// sautées au lieu de faire échouer la lecture.
//
// Un fichier réel en porte : d'autres sessions, et des protocoles qui ne servent
// qu'aux connexions par le réseau. S'arrêter à la première ferait manquer le secret
// qui se trouve derrière.
func TestPickAuthSkipsWhatItCannotPresent(t *testing.T) {
	cases := []struct {
		name string
		file []byte
	}{
		{
			"un protocole que le backend ne sait pas présenter",
			authFileOf(
				authEntryFields{familyLocal, "machine", "0", "XDM-AUTHORIZATION-1", "autre"},
				authEntryFields{familyLocal, "machine", "0", authMagicCookie, "bon"},
			),
		},
		{
			"une autre session de la même machine",
			authFileOf(
				authEntryFields{familyLocal, "machine", "7", authMagicCookie, "autre"},
				authEntryFields{familyLocal, "machine", "0", authMagicCookie, "bon"},
			),
		},
		{
			"une famille qui ne vaut pas pour une session locale",
			authFileOf(
				authEntryFields{0, "10.0.0.1", "0", authMagicCookie, "autre"},
				authEntryFields{familyLocal, "machine", "0", authMagicCookie, "bon"},
			),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, data := pickAuth(c.file, "machine", 0); string(data) != "bon" {
				t.Errorf("cookie %q, attendu celui de l'entrée utilisable", data)
			}
		})
	}
}

// TestPickAuthAcceptsAnEntryWithoutSessionNumber vérifie qu'un numéro de session vide
// vaut pour toute session.
//
// C'est ce que portent les entrées qu'un gestionnaire d'affichage fusionne, et les
// exiger numérotées rendrait le fichier inutilisable là où il est pourtant juste.
func TestPickAuthAcceptsAnEntryWithoutSessionNumber(t *testing.T) {
	file := authFileOf(authEntryFields{familyLocal, "machine", "", authMagicCookie, "secret"})

	if _, data := pickAuth(file, "machine", 99); string(data) != "secret" {
		t.Errorf("cookie %q, attendu celui de l'entrée sans numéro", data)
	}
}

// TestPickAuthKeepsWhatPrecedesATruncation vérifie qu'un fichier coupé rend les
// entrées complètes qui le précèdent.
//
// Le fichier s'écrit en ajoutant : une dernière entrée incomplète est le régime d'une
// lecture faite pendant une écriture, et les précédentes restent bonnes. Tout refuser
// ferait échouer une ouverture pour une raison qui disparaît d'elle-même.
func TestPickAuthKeepsWhatPrecedesATruncation(t *testing.T) {
	file := authFileOf(authEntryFields{familyLocal, "machine", "0", authMagicCookie, "secret"})
	file = append(file, 1, 0, 0, 9) // une entrée qui s'arrête au milieu de sa première longueur

	if _, data := pickAuth(file, "machine", 0); string(data) != "secret" {
		t.Errorf("cookie %q, attendu celui de l'entrée complète", data)
	}
}

// TestPickAuthReadsLengthsBigEndian verrouille le seul fait du backend qui ne soit
// écrit dans aucune spécification.
//
// Les longueurs du fichier sont sur deux octets gros-boutiens **quelle que soit la
// machine**, ce qui est l'inverse de tout le reste de la connexion. Un lecteur qui
// suivrait l'ordre de la machine marcherait sur un gros-boutien et lirait des
// longueurs de plusieurs milliers d'octets sur le poste : le cas le montre en
// fabriquant le fichier à l'envers, ce qui ne doit rien rendre.
func TestPickAuthReadsLengthsBigEndian(t *testing.T) {
	file := authFileOf(authEntryFields{familyLocal, "machine", "0", authMagicCookie, "secret"})

	reversed := make([]byte, len(file))
	copy(reversed, file)
	for at := 0; at+1 < len(reversed); at += 2 {
		reversed[at], reversed[at+1] = reversed[at+1], reversed[at]
	}

	if _, data := pickAuth(reversed, "machine", 0); data != nil {
		t.Errorf("un fichier aux longueurs inversées a rendu le cookie %q", data)
	}
}

// TestAuthRequestPadsEveryString vérifie que le bloc d'ouverture porte les longueurs
// annoncées et le bourrage qui va avec.
//
// Le serveur compte les deux. Un bourrage omis décale les données d'autorisation, que
// le serveur compare alors à autre chose que le cookie — et le refus qu'il rend ne
// parle ni de longueur ni de bourrage.
func TestAuthRequestPadsEveryString(t *testing.T) {
	name, data := []byte("MIT-MAGIC-COOKIE-1"), make([]byte, 16)

	b := authRequest(name, data)

	if got := b[0]; got != handshakeOrder() {
		t.Errorf("octet d'ordre %q, attendu %q", got, handshakeOrder())
	}
	if got := order.Uint16(b[2:]); got != protocolMajor {
		t.Errorf("version majeure %d, attendu %d", got, protocolMajor)
	}
	if got := int(order.Uint16(b[6:])); got != len(name) {
		t.Errorf("longueur de protocole annoncée %d, attendu %d", got, len(name))
	}
	if got := int(order.Uint16(b[8:])); got != len(data) {
		t.Errorf("longueur de données annoncée %d, attendu %d", got, len(data))
	}

	// Dix-huit octets de nom, donc deux de bourrage : les données commencent à un
	// multiple de quatre, et le bloc entier aussi.
	want := 12 + len(name) + pad(len(name)) + len(data) + pad(len(data))
	if len(b) != want {
		t.Errorf("bloc de %d octets, attendu %d", len(b), want)
	}
	if len(b)%4 != 0 {
		t.Errorf("bloc de %d octets, qui n'est pas un multiple de quatre", len(b))
	}
	if at := 12 + len(name) + pad(len(name)); string(b[at:at+len(data)]) != string(data) {
		t.Error("les données d'autorisation ne commencent pas après le bourrage du nom")
	}
}
