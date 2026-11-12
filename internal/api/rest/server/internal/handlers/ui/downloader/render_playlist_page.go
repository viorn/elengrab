package downloader

import (
	"bytes"
	"fmt"
	"html/template"
	"mime"

	"github.com/neosy/elengrab/internal/api/rest/server/internal/clientcap"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/icons"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/pages"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/paths"
	pagesdata "github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/pages_data"
	udto "github.com/neosy/elengrab/internal/app/usecases/dto"
	dauth "github.com/neosy/elengrab/internal/domain/auth"
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	nfasthttp "github.com/neosy/elengrab/internal/pkg/fasthttpx"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
	"github.com/valyala/fasthttp"
)

func (h *DownloaderHandlers) renderPlaylistPage(
	ctx *fasthttp.RequestCtx,
	authCtx dauth.AuthContext,
	playlist *ddownload.Playlist,
	query udto.MediaDownloadQuery,
) {
	var rowsBuf bytes.Buffer
	if err := h.renderDownloadItemList(ctx, &rowsBuf, authCtx, query); err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	cssPaths, err := h.assetPaths.PlaylistPageCssPaths()
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	caps := clientcap.Detect(string(ctx.UserAgent()))

	jsScripts, err := h.assetPaths.PlaylistPageJsPaths(caps.IsLegacyWebKit)
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
	baseValues.Title = fmt.Sprintf("%s - %s", playlist.Title, pages.AppName)
	baseValues.MetaOgItems = pagesdata.BuildMetaOgItems(h.baseURL)

	pagesListValues := pagesdata.BuildPagesListValues(authCtx, query, queryFilters, h.downloader)
	pagesListValues.HasCreateAccess = false
	// В плейлисте порядок ручной — сортировка new/popular/old не имеет смысла.
	pagesListValues.HideViewModeTabs = true
	pagesListValues.ResultNoRows = rowsBuf.Len() == 0
	pagesListValues.ResultRowsHTML = template.HTML(rowsBuf.String())
	pagesListValues.AboutDialog = h.mappers.MapAboutDialogValues(systemInfo)
	pagesListValues.UploadFileDialog = h.mappers.MapUploadFileDialogValues()
	pagesListValues.VideoPlayerMinimizeIcon = icons.VideoPlayerMinimizeIcon.FileRaw()
	pagesListValues.AudioPlayingIcon = icons.MediaAudioPlayingIcon.FileRaw()
	pagesListValues.AudioNoteIcon = icons.MediaAudioNoteIcon.FileRaw()
	pagesListValues.AudioPlayIcon = icons.MediaAudioPlayIcon.FileRaw()
	pagesListValues.AudioPauseIcon = icons.MediaAudioPauseIcon.FileRaw()

	playlistHeader := pages.PlaylistHeader{
		EncodedPlaylistID: idcodec.EncodeUUIDBase64URL(playlist.PlaylistID),
		Title:             playlist.Title,
		ItemsCount:        playlist.ItemsCount,
		CanEdit:           h.canEditPlaylist(authCtx, playlist),
		IsShared:          playlist.IsShared(),
		Show:              true,
	}
	if playlist.Description != nil {
		playlistHeader.Description = *playlist.Description
	}

	pageData := pages.PlaylistPageData{
		BasePaths:  paths.NewHttpPaths(),
		BaseValues: baseValues,
		Paths: pages.PagePaths{
			Css:         cssPaths,
			JsScripts:   jsScripts,
			PwaManifest: pwaManifestPath,
		},
		Values: pages.PlaylistPageValues{
			PagesListValues: pagesListValues,
			PlaylistHeader:  playlistHeader,
		},
	}

	ctx.SetContentType(mime.TypeByExtension(".html"))

	if err := h.templates.Pages[pages.PlaylistPage.Key()].ExecuteTemplate(ctx, pages.PlaylistPage.Key(), pageData); err != nil {
		h.logger.Warn("Failed to render playlist page", "error", err)
		nfasthttp.WriteErrorx(ctx, errInternal(err))
		return
	}
}
