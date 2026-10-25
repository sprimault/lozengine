// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les places du clavier que le moteur nomme, avec la légende d'un clavier américain.
//
// Les valeurs sont les codes de balayage du jeu 1, celui que les claviers rapportent
// depuis toujours et que Windows transmet dans ses messages de touches. Elles ont été
// vérifiées une à une en demandant au système comment la disposition active nomme
// chaque place, plutôt que d'être recopiées d'une table : sur un clavier français,
// [KeyW] porte un Z, [KeyA] porte un Q et [KeyM] porte une virgule.

package input

// Les lettres, nommées par la légende américaine de leur place.
//
// **C'est une place, pas un caractère**, et c'est ce qui rend le nom utile : un jeu
// qui fait avancer son personnage sur [KeyW] le fait avancer sur la touche au-dessus
// de [KeyS], quelle que soit la disposition du joueur. Nommer le caractère produit
// aurait dispersé la croix de déplacement sur un clavier français.
const (
	KeyA Key = 0x1E
	KeyB Key = 0x30
	KeyC Key = 0x2E
	KeyD Key = 0x20
	KeyE Key = 0x12
	KeyF Key = 0x21
	KeyG Key = 0x22
	KeyH Key = 0x23
	KeyI Key = 0x17
	KeyJ Key = 0x24
	KeyK Key = 0x25
	KeyL Key = 0x26
	KeyM Key = 0x32
	KeyN Key = 0x31
	KeyO Key = 0x18
	KeyP Key = 0x19
	KeyQ Key = 0x10
	KeyR Key = 0x13
	KeyS Key = 0x1F
	KeyT Key = 0x14
	KeyU Key = 0x16
	KeyV Key = 0x2F
	KeyW Key = 0x11
	KeyX Key = 0x2D
	KeyY Key = 0x15
	KeyZ Key = 0x2C
)

// Les chiffres de la rangée du haut, qui n'est pas le pavé numérique.
const (
	Key1 Key = 0x02
	Key2 Key = 0x03
	Key3 Key = 0x04
	Key4 Key = 0x05
	Key5 Key = 0x06
	Key6 Key = 0x07
	Key7 Key = 0x08
	Key8 Key = 0x09
	Key9 Key = 0x0A
	Key0 Key = 0x0B
)

// extended est le bit que le moteur ajoute au code de balayage d'une touche étendue.
//
// Le clavier annonce ces touches par un préfixe, et deux places portent alors le même
// code de base : la flèche gauche et le 4 du pavé numérique sont tous deux 0x4B. Le
// bit les sépare, et c'est lui qui fait tenir les places sur neuf bits.
const extended Key = 0x100

// Les quatre flèches, qui sont des touches étendues.
//
// Le bloc directionnel et le pavé numérique partagent leurs codes de base : sans le
// bit étendu, un jeu qui lit les flèches répondrait au pavé numérique, et les deux
// n'ont pas le même usage.
const (
	KeyUp    = extended | 0x48
	KeyLeft  = extended | 0x4B
	KeyRight = extended | 0x4D
	KeyDown  = extended | 0x50
)

// Les touches d'édition et de validation.
const (
	KeyEscape    Key = 0x01
	KeyBackspace Key = 0x0E
	KeyTab       Key = 0x0F
	KeyEnter     Key = 0x1C
	KeySpace     Key = 0x39
)

// Les modificateurs, distingués à gauche et à droite.
//
// Un jeu qui ne veut pas les distinguer lit les deux ; l'inverse serait impossible si
// le moteur les avait confondus, et c'est la raison de ce choix. Les trois de droite
// sont des touches étendues, sauf la majuscule, que le clavier distingue par son
// propre code.
const (
	KeyLeftShift    Key = 0x2A
	KeyRightShift   Key = 0x36
	KeyLeftControl  Key = 0x1D
	KeyRightControl     = extended | 0x1D
	KeyLeftAlt      Key = 0x38
	KeyRightAlt         = extended | 0x38
)

// Les douze touches de fonction.
//
// F11 et F12 ne suivent pas les dix premières : leurs codes ont été ajoutés après, à
// la fin de la plage libre, et c'est un fait du jeu de codes et non une erreur.
const (
	KeyF1  Key = 0x3B
	KeyF2  Key = 0x3C
	KeyF3  Key = 0x3D
	KeyF4  Key = 0x3E
	KeyF5  Key = 0x3F
	KeyF6  Key = 0x40
	KeyF7  Key = 0x41
	KeyF8  Key = 0x42
	KeyF9  Key = 0x43
	KeyF10 Key = 0x44
	KeyF11 Key = 0x57
	KeyF12 Key = 0x58
)

// Les touches de navigation, toutes étendues.
const (
	KeyHome     = extended | 0x47
	KeyEnd      = extended | 0x4F
	KeyPageUp   = extended | 0x49
	KeyPageDown = extended | 0x51
	KeyInsert   = extended | 0x52
	KeyDelete   = extended | 0x53
)
