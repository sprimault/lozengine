// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Commande qui sert un jeu compilé pour le navigateur : la page que le moteur attend,
// le module, et le fichier de liaison de la chaîne Go.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"go/build"
	"net/http"
	"os"
	"path/filepath"
)

// page est la page que le moteur attend d'un hôte, embarquée plutôt que copiée.
//
// **Embarquée pour qu'il n'y ait rien à recopier** : une page qu'on duplique dans chaque
// jeu diverge du moteur dès que celui-ci change ce qu'il attend d'elle, et c'est arrivé
// à tous les fichiers que ce dépôt a laissé recopier. Un jeu qui veut la sienne la sert
// lui-même, et son contenu dit ce qu'elle doit porter.
//
//go:embed page.html
var page []byte

// bridgeFile est le nom du fichier que la chaîne Go fournit pour démarrer un module.
const bridgeFile = "wasm_exec.js"

// main sert le module donné, la page et le fichier de liaison, sur la boucle locale.
func main() {
	module := flag.String("wasm", "", "le module compilé pour js/wasm, à servir")
	addr := flag.String("addr", "127.0.0.1:8080", "où écouter")
	flag.Parse()

	if *module == "" {
		fmt.Fprintln(os.Stderr, "serve: -wasm is required: point it at a js/wasm build")
		os.Exit(2)
	}
	if _, err := os.Stat(*module); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}

	bridgePath, err := bridge()
	if err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	http.HandleFunc("/main.wasm", func(w http.ResponseWriter, r *http.Request) {
		// Le type compte : sans lui, `instantiateStreaming` refuse le flux et la
		// page retombe sur un chemin plus lent, quand elle ne tombe pas tout court.
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, *module)
	})
	http.HandleFunc("/"+bridgeFile, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		http.ServeFile(w, r, bridgePath)
	})

	fmt.Printf("lozengine on http://%s/\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

// bridge rend le chemin du fichier de liaison de la chaîne Go résolue.
//
// **Il ne se versionne pas**, et c'est une décision : il appartient à la chaîne, il
// change avec elle, et un exemplaire figé dans le dépôt démarrerait un module compilé
// par une autre version — avec une panne dont le message ne dit rien de la cause.
//
// Deux emplacements, parce que la chaîne l'a déplacé : `lib/wasm` depuis Go 1.24,
// `misc/wasm` avant. L'erreur nomme les deux et la racine où ils ont été cherchés, faute
// de quoi un hôte sans liaison ne saurait pas quoi installer.
func bridge() (string, error) {
	root := build.Default.GOROOT
	for _, dir := range []string{"lib/wasm", "misc/wasm"} {
		path := filepath.Join(root, filepath.FromSlash(dir), bridgeFile)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no %s in the resolved Go toolchain: looked in %s under lib/wasm and misc/wasm",
		bridgeFile, root)
}
