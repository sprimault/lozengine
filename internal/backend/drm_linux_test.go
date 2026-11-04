// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la transposition des structures du noyau doit tenir, et ce que ces cas ne
// peuvent pas attraper.

package backend

import (
	"testing"
	"unsafe"
)

// TestDRMLayoutMatchesTheKernelHeaders fige la taille et les décalages que l'UAPI
// déclare.
//
// **Il garde plus qu'une convention d'appel : il garde les requêtes elles-mêmes.** Un mot
// d'`ioctl` porte la taille de son argument sur quatorze bits, donc une structure élargie
// d'un champ change le numéro de la requête — en un autre numéro parfaitement formé, que
// le noyau refuse par un `EINVAL` sans dire lequel de ses champs il n'a pas reconnu. Le
// cas déplace cet échec d'une machine qui a un écran vers `make check`.
//
// **Les valeurs attendues viennent du compilateur C et non d'un calcul**, relevées le
// 4 novembre 2026 par un programme qui inclut les en-têtes et imprime `sizeof` et
// `offsetof`. C'est la leçon du cas X11 reprise ici : un test qui éprouve une
// transposition ne peut pas se servir de cette transposition pour dire ce qu'il attend.
//
// **Ce qu'il ne peut pas attraper, et qui demande donc un autre garde** : deux champs
// voisins du même type et de même taille, dont la permutation ne déplace aucun décalage.
// C'est exactement le défaut qui a laissé `Overlap` doubler de taille sans qu'un contrôle
// le voie, et il y a ici un cas réel — `height` précède `width` dans
// [drmModeCreateDumb], et les inverser rend un tampon de la bonne taille et de la
// mauvaise forme. Ce qui l'attrapera est le lot qui crée le tampon, sur une largeur et
// une hauteur différentes.
func TestDRMLayoutMatchesTheKernelHeaders(t *testing.T) {
	var (
		res       drmModeCardRes
		mode      drmModeInfo
		connector drmModeGetConnectorArg
		crtc      drmModeCrtc
		dumb      drmModeCreateDumb
		fb        drmModeFBCmd2
		flip      drmModeCrtcPageFlip
		vblank    drmEventVblank
	)

	for _, c := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"drm_set_client_cap", unsafe.Sizeof(drmClientCap{}), 16},
		{"drm_mode_card_res", unsafe.Sizeof(res), 64},
		{"drm_mode_modeinfo", unsafe.Sizeof(mode), 68},
		{"drm_mode_get_connector", unsafe.Sizeof(connector), 80},
		{"drm_mode_get_encoder", unsafe.Sizeof(drmModeGetEncoderArg{}), 20},
		{"drm_mode_crtc", unsafe.Sizeof(crtc), 104},
		{"drm_mode_create_dumb", unsafe.Sizeof(dumb), 32},
		{"drm_mode_map_dumb", unsafe.Sizeof(drmModeMapDumb{}), 16},
		{"drm_mode_destroy_dumb", unsafe.Sizeof(drmModeDestroyDumb{}), 4},
		{"drm_mode_fb_cmd2", unsafe.Sizeof(fb), 104},
		{"drm_mode_crtc_page_flip", unsafe.Sizeof(flip), 24},
		{"drm_event", unsafe.Sizeof(drmEventHeader{}), 8},
		{"drm_event_vblank", unsafe.Sizeof(vblank), 32},
	} {
		if c.got != c.want {
			t.Errorf("taille de %s : %d, l'en-tête déclare %d", c.name, c.got, c.want)
		}
	}

	// Les décalages relevés sont ceux des champs qu'un ajout déplacerait sans changer
	// la taille de la structure : le dernier de chacune, et ceux qui suivent un
	// groupe de pointeurs ou un tableau. Les énumérer tous n'apprendrait rien de plus,
	// des champs consécutifs de même type se déduisant l'un de l'autre.
	for _, c := range []struct {
		field string
		got   uintptr
		want  uintptr
	}{
		{"drm_mode_card_res.count_fbs", unsafe.Offsetof(res.countFBs), 32},
		{"drm_mode_card_res.max_height", unsafe.Offsetof(res.maxHeight), 60},

		{"drm_mode_modeinfo.hdisplay", unsafe.Offsetof(mode.hdisplay), 4},
		{"drm_mode_modeinfo.vrefresh", unsafe.Offsetof(mode.vrefresh), 24},
		{"drm_mode_modeinfo.name", unsafe.Offsetof(mode.name), 36},

		{"drm_mode_get_connector.count_modes", unsafe.Offsetof(connector.countModes), 32},
		{"drm_mode_get_connector.connection", unsafe.Offsetof(connector.connection), 60},
		{"drm_mode_get_connector.pad", unsafe.Offsetof(connector.pad), 76},

		{"drm_mode_crtc.mode_valid", unsafe.Offsetof(crtc.modeValid), 32},
		{"drm_mode_crtc.mode", unsafe.Offsetof(crtc.mode), 36},

		{"drm_mode_create_dumb.size", unsafe.Offsetof(dumb.size), 24},

		{"drm_mode_fb_cmd2.handles", unsafe.Offsetof(fb.handles), 20},
		{"drm_mode_fb_cmd2.pitches", unsafe.Offsetof(fb.pitches), 36},
		{"drm_mode_fb_cmd2.offsets", unsafe.Offsetof(fb.offsets), 52},
		{"drm_mode_fb_cmd2.modifier", unsafe.Offsetof(fb.modifier), 72},

		{"drm_mode_crtc_page_flip.user_data", unsafe.Offsetof(flip.userData), 16},

		{"drm_event_vblank.user_data", unsafe.Offsetof(vblank.userData), 8},
		{"drm_event_vblank.crtc_id", unsafe.Offsetof(vblank.crtcID), 28},
	} {
		if c.got != c.want {
			t.Errorf("position de %s : %d, l'en-tête déclare %d", c.field, c.got, c.want)
		}
	}
}

// TestDRMRequestsMatchTheHeaderMacros fige les mots que les macros du noyau composent.
//
// **Il est le pendant du cas de disposition et non un doublon**, et la différence est le
// sens de la vérification : là-bas on garde la structure contre l'en-tête, ici on garde la
// **composition** du mot — direction, décalages, type d'interface — contre ce que le
// préprocesseur rend. Une erreur de décalage dans [drmRequestSizeShift] passerait le
// premier cas et tomberait sur celui-ci.
//
// Les valeurs sont celles qu'un programme C a imprimées le 4 novembre 2026 en développant
// `DRM_IOCTL_*`, sur la même machine et dans le même relevé que les tailles.
//
// **Les trois directions y sont toutes les trois**, et c'est ce qui donne sa portée au
// cas : `_IOWR` pour le réglage de mode, `_IOW` pour la capacité client, et `_IO` nu pour
// la prise et la remise de la maîtrise — dont le mot ne porte aucune taille, donc une
// composition qui en ajouterait une s'y verrait.
func TestDRMRequestsMatchTheHeaderMacros(t *testing.T) {
	for _, c := range []struct {
		macro string
		got   uintptr
		want  uintptr
	}{
		{"DRM_IOCTL_SET_CLIENT_CAP", drmSetClientCap, 0x4010640d},
		{"DRM_IOCTL_SET_MASTER", drmSetMaster, 0x641e},
		{"DRM_IOCTL_DROP_MASTER", drmDropMaster, 0x641f},

		{"DRM_IOCTL_MODE_GETRESOURCES", drmModeGetResources, 0xc04064a0},
		{"DRM_IOCTL_MODE_GETCRTC", drmModeGetCrtc, 0xc06864a1},
		{"DRM_IOCTL_MODE_SETCRTC", drmModeSetCrtc, 0xc06864a2},
		{"DRM_IOCTL_MODE_GETENCODER", drmModeGetEncoder, 0xc01464a6},
		{"DRM_IOCTL_MODE_GETCONNECTOR", drmModeGetConnector, 0xc05064a7},
		{"DRM_IOCTL_MODE_RMFB", drmModeRmFB, 0xc00464af},
		{"DRM_IOCTL_MODE_PAGE_FLIP", drmModePageFlip, 0xc01864b0},
		{"DRM_IOCTL_MODE_CREATE_DUMB", drmModeCreateDumbReq, 0xc02064b2},
		{"DRM_IOCTL_MODE_MAP_DUMB", drmModeMapDumbReq, 0xc01064b3},
		{"DRM_IOCTL_MODE_DESTROY_DUMB", drmModeDestroyDumbReq, 0xc00464b4},
		{"DRM_IOCTL_MODE_ADDFB2", drmModeAddFB2, 0xc06864b8},
	} {
		if c.got != c.want {
			t.Errorf("%s vaut %#x, la macro développe %#x", c.macro, c.got, c.want)
		}
	}
}

// TestDRMFormatIsTheOneTheEngineAlreadyCarries garde que les deux moitiés du jalon
// nomment la même valeur.
//
// `DRM_FORMAT_XBGR8888` et le format que `wl_shm` annonce sont le même code à quatre
// caractères, donc le relevé du réglage de mode ne le déclare pas une seconde fois. Ce cas
// est ce qui rend ce silence sûr : si quelqu'un ajoutait une constante propre à cette
// famille, ou changeait celle de [formatXBGR8888], il faudrait qu'il le voie ici.
//
// **La valeur attendue est écrite en clair et non dérivée de `fourcc`**, pour la raison
// qui vaut pour tout le fichier : un cas qui éprouve une composition ne peut pas s'en
// servir pour dire ce qu'il attend.
func TestDRMFormatIsTheOneTheEngineAlreadyCarries(t *testing.T) {
	// 'X' | 'B'<<8 | '2'<<16 | '4'<<24, tel que `drm_fourcc.h` compose
	// `DRM_FORMAT_XBGR8888`, soit « [31:0] x:B:G:R 8:8:8:8 little endian ».
	const want uint32 = 0x34324258

	if formatXBGR8888 != want {
		t.Errorf("le format du moteur vaut %#x, drm_fourcc.h compose %#x", formatXBGR8888, want)
	}
}
