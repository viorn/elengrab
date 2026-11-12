package downloader

import (
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/components"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/icons"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/pages"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/paths"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/policy"
	httppaths "github.com/neosy/elengrab/internal/api/rest/server/internal/paths"
	nfasthttp "github.com/neosy/elengrab/internal/pkg/fasthttpx"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
	"github.com/valyala/fasthttp"
)

type playlistDialogPageData struct {
	pages.PageFragmentData

	Values struct {
		PlaylistDialog pages.PlaylistDialogValues
	}
}

// renderMediaItemPlaylistsDialog renders the dialog with playlists containing the media item.
func (h *DownloaderHandlers) renderMediaItemPlaylistsDialog(ctx *fasthttp.RequestCtx) {
	authCtx := policy.ResolveUserOrAnonym(ctx)

	downloadID, err := h.extractDownloadID(ctx)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	items, err := h.downloader.ListPlaylistsForItem(ctx, authCtx, downloadID)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	list := make([]pages.PlaylistDialogItem, 0, len(items))
	for _, item := range items {
		if item == nil || item.Playlist == nil {
			continue
		}

		list = append(list, pages.PlaylistDialogItem{
			EncodedPlaylistID: idcodec.EncodeUUIDBase64URL(item.Playlist.PlaylistID),
			Title:             item.Playlist.Title,
			ItemsCount:        item.Playlist.ItemsCount,
			Checked:           item.Checked,
			CanEdit:           h.canEditPlaylist(authCtx, item.Playlist),
		})
	}

	pageData := playlistDialogPageData{
		PageFragmentData: pages.PageFragmentData{
			BasePaths: paths.NewHttpPaths(),
		},
	}

	pageData.Values.PlaylistDialog = pages.PlaylistDialogValues{
		LogoIcon:     icons.UploadMenuMyDownloadsIcon.FileRaw(),
		DownloadID:   idcodec.EncodeUUIDBase64URL(downloadID),
		Playlists:    list,
		HasPlaylists: len(list) > 0,
		CanCreate:    h.downloader.CanCreateMediaDownload(authCtx),
		ToggleURL:    httppaths.BuildMediaItemPlaylistsPath(downloadID),
		CreateURL:    httppaths.DownloaderPlaylistsPath,
	}

	if err := h.templates.Base.ExecuteTemplate(ctx, components.PlaylistDialogContentKey, pageData); err != nil {
		nfasthttp.WriteErrorx(ctx, errInternal(err))
		return
	}
}
