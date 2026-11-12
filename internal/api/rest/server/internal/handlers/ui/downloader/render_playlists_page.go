package downloader

import (
	"fmt"
	"mime"

	"github.com/neosy/elengrab/internal/api/rest/server/internal/clientcap"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/icons"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/pages"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/paths"
	pagesdata "github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/pages_data"
	httppaths "github.com/neosy/elengrab/internal/api/rest/server/internal/paths"
	udto "github.com/neosy/elengrab/internal/app/usecases/dto"
	dauth "github.com/neosy/elengrab/internal/domain/auth"
	nfasthttp "github.com/neosy/elengrab/internal/pkg/fasthttpx"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
	"github.com/valyala/fasthttp"
)

func (h *DownloaderHandlers) renderPlaylistsPage(
	ctx *fasthttp.RequestCtx,
	authCtx dauth.AuthContext,
	query udto.MediaDownloadQuery,
) {
	playlists, err := h.downloader.ListPlaylists(ctx, authCtx)
	if err != nil {
		h.logger.Warn("Failed to list playlists", "error", err)
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	cssPaths, err := h.assetPaths.PlaylistsPageCssPaths()
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	caps := clientcap.Detect(string(ctx.UserAgent()))

	jsScripts, err := h.assetPaths.PlaylistsPageJsPaths(caps.IsLegacyWebKit)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	pwaManifestPath, err := h.assetPaths.PwaManifestPath()
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	queryFilters := h.mappers.MapQueryFiltersDomainToFilters(query.Filters)

	systemInfo := h.downloader.SystemInfo()

	baseValues := pages.NewBaseValues()
	baseValues.Title = fmt.Sprintf("Playlists - %s", pages.AppName)
	baseValues.MetaOgItems = pagesdata.BuildMetaOgItems(h.baseURL)

	pagesListValues := pagesdata.BuildPagesListValues(authCtx, query, queryFilters, h.downloader)
	pagesListValues.AboutDialog = h.mappers.MapAboutDialogValues(systemInfo)
	pagesListValues.UploadFileDialog = h.mappers.MapUploadFileDialogValues()
	pagesListValues.AudioNoteIcon = icons.MediaAudioNoteIcon.FileRaw()
	pagesListValues.AudioPlayIcon = icons.MediaAudioPlayIcon.FileRaw()
	pagesListValues.AudioPauseIcon = icons.MediaAudioPauseIcon.FileRaw()
	pagesListValues.AudioPlayingIcon = icons.MediaAudioPlayingIcon.FileRaw()
	pagesListValues.VideoPlayerMinimizeIcon = icons.VideoPlayerMinimizeIcon.FileRaw()

	cards := make([]pages.PlaylistCard, 0, len(playlists))
	for _, playlist := range playlists {
		if playlist == nil {
			continue
		}

		card := pages.PlaylistCard{
			EncodedPlaylistID: idcodec.EncodeUUIDBase64URL(playlist.PlaylistID),
			Title:             playlist.Title,
			ItemsCount:        playlist.ItemsCount,
			IsShared:          playlist.IsShared(),
			CanEdit:           h.canEditPlaylist(authCtx, playlist),
			URL:               httppaths.BuildPlaylistPath(playlist.PlaylistID),
		}
		if playlist.Description != nil {
			card.Description = *playlist.Description
		}

		cards = append(cards, card)
	}

	canCreate := h.downloader.CanCreateMediaDownload(authCtx)

	pageData := pages.PlaylistsPageData{
		BasePaths:  paths.NewHttpPaths(),
		BaseValues: baseValues,
		Paths: pages.PagePaths{
			Css:         cssPaths,
			JsScripts:   jsScripts,
			PwaManifest: pwaManifestPath,
		},
		Values: pages.PlaylistsPageValues{
			PagesListValues: pagesListValues,
			Playlists:       cards,
			HasPlaylists:    len(cards) > 0,
			CanCreate:       canCreate,
		},
	}

	ctx.SetContentType(mime.TypeByExtension(".html"))

	if err := h.templates.Pages[pages.PlaylistsPage.Key()].ExecuteTemplate(ctx, pages.PlaylistsPage.Key(), pageData); err != nil {
		h.logger.Warn("Failed to render playlists page", "error", err)
		nfasthttp.WriteErrorx(ctx, errInternal(err))
		return
	}
}
