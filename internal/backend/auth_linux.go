// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le secret que le serveur exige, et où le lire.
//
// **Le format de ce fichier n'est écrit dans aucune spécification X.Org**, et c'est le
// seul fait de tout le backend dans ce cas : il a été relevé dans la documentation
// d'interface de `xauth`, qui le décrit comme le contrat de son propre fichier. Un
// format de fichier est un fait d'interface au même titre qu'un opcode, et c'est à ce
// titre qu'il est ici ; aucune ligne n'est reprise d'une implémentation.

package backend

import (
	"os"
	"strconv"
)

// authMagicCookie est le seul protocole d'autorisation que le backend sait présenter.
//
// Les autres — `XDM-AUTHORIZATION-1` et ses successeurs — chiffrent un défi avec une
// clé partagée, ne servent que les connexions par le réseau, et ne sont posés par
// aucune session locale. Une entrée qui en porte un est donc passée plutôt que de faire
// échouer la lecture.
const authMagicCookie = "MIT-MAGIC-COOKIE-1"

// Les deux familles d'adresse qu'une entrée locale peut porter.
//
// **Une session locale s'enregistre en [familyLocal] avec le nom de la machine pour
// adresse**, jamais sous la forme `:0` qu'on a sous les yeux dans `DISPLAY`. Chercher
// cette forme littérale ne trouve rien et fait conclure que le serveur n'exige pas
// d'autorisation — après quoi le handshake est refusé pour une raison qui ne ressemble
// plus à celle-là. [familyWild] vaut pour toute adresse, et `xauth` en pose dans les
// fichiers qu'il fusionne.
const (
	familyLocal = 256
	familyWild  = 65535
)

// authority rend le nom et les données d'autorisation à présenter pour une session.
//
// **L'absence n'est pas une erreur**, et les deux rendus vides sont un cas ordinaire :
// un serveur peut n'exiger aucune autorisation, et c'est le régime d'un serveur sans
// écran lancé sans `-auth`. Le handshake dira ce qu'il en pense, et son refus porte
// une raison plus utile que tout ce qu'on devinerait ici.
func authority(number int) (name, data []byte) {
	// La variable prime sur le fichier par défaut, et ce n'est pas une politesse :
	// les sessions récentes ne créent plus `.Xauthority` dans le répertoire
	// personnel, elles posent le fichier ailleurs et nomment l'endroit.
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return nil, nil
		}
		path = home + "/.Xauthority"
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	return pickAuth(file, host, number)
}

// pickAuth choisit, parmi les entrées du fichier, celle qui vaut pour cette machine et
// cette session.
//
// Trois degrés de correspondance, du plus précis au plus large, et le meilleur gagne :
// la machine nommée par son nom, puis l'entrée qui vaut pour toute adresse, puis une
// entrée locale dont l'adresse ne correspond pas. **Ce dernier degré est le repli d'une
// machine renommée** : le fichier garde le nom qu'elle portait quand la session s'est
// ouverte, et refuser serait refuser la seule entrée utilisable.
//
// Un fichier tronqué ou illisible rend ce qui a pu être lu avant la coupure, sans
// erreur. Ce n'est pas de la complaisance : `xauth` écrit en ajoutant, donc une
// dernière entrée incomplète est le régime d'un fichier surpris en cours d'écriture,
// et les précédentes restent bonnes.
func pickAuth(file []byte, host string, number int) (name, data []byte) {
	session := strconv.Itoa(number)

	best := 0
	for at := 0; at < len(file); {
		family, fields, next, ok := authEntry(file, at)
		if !ok {
			break
		}
		at = next

		if string(fields[2]) != authMagicCookie {
			continue
		}
		// Un numéro vide vaut pour toute session, et c'est ce que portent les
		// entrées qu'un gestionnaire d'affichage fusionne.
		if n := string(fields[1]); n != "" && n != session {
			continue
		}

		rank := 0
		switch {
		case family == familyLocal && host != "" && string(fields[0]) == host:
			rank = 3
		case family == familyWild:
			rank = 2
		case family == familyLocal:
			rank = 1
		}
		if rank > best {
			best, name, data = rank, fields[2], fields[3]
		}
	}
	return name, data
}

// authEntry découpe une entrée du fichier d'autorisation et dit où la suivante
// commence.
//
// Les quatre tranches rendues sont l'adresse, le numéro de session, le nom du
// protocole et les données, dans l'ordre du fichier. Elles pointent dans le tampon
// reçu, qui n'est ni gardé ni modifié ensuite.
//
// **Les longueurs sont sur deux octets gros-boutiens quelle que soit la machine**, et
// c'est la seule place du backend où l'ordre ne suit pas celui de la connexion : le
// fichier est écrit une fois et relu par tout ce qui passe, donc il ne pouvait pas
// dépendre de qui l'écrit.
func authEntry(file []byte, at int) (family uint16, fields [4][]byte, next int, ok bool) {
	if at+2 > len(file) {
		return 0, fields, 0, false
	}
	family = uint16(file[at])<<8 | uint16(file[at+1])
	at += 2

	for i := range fields {
		if at+2 > len(file) {
			return 0, fields, 0, false
		}
		length := int(file[at])<<8 | int(file[at+1])
		at += 2
		if at+length > len(file) {
			return 0, fields, 0, false
		}
		fields[i] = file[at : at+length]
		at += length
	}
	return family, fields, at, true
}

// authRequest écrit le bloc d'ouverture de connexion : l'ordre des octets, la version
// du protocole, et l'autorisation qu'on présente.
//
// C'est la seule chose que le client envoie avant d'avoir lu quoi que ce soit, et elle
// n'a pas de numéro de requête : le compte commence à un après le handshake.
func authRequest(name, data []byte) []byte {
	var zeros [3]byte

	b := make([]byte, 12, 12+len(name)+pad(len(name))+len(data)+pad(len(data)))
	b[0] = handshakeOrder()
	order.PutUint16(b[2:], protocolMajor)
	order.PutUint16(b[4:], protocolMinor)
	order.PutUint16(b[6:], uint16(len(name)))
	order.PutUint16(b[8:], uint16(len(data)))

	// Chaque chaîne est suivie de son bourrage à un multiple de quatre, et le
	// serveur compte les deux : le sauter décalerait les données d'autorisation, que
	// le serveur comparerait alors à autre chose que le cookie.
	b = append(b, name...)
	b = append(b, zeros[:pad(len(name))]...)
	b = append(b, data...)
	b = append(b, zeros[:pad(len(data))]...)
	return b
}
