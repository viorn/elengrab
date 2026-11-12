package downloader

import (
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/policy"
	qkeys "github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/query_keys.go"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/types"
	"github.com/neosy/elengrab/internal/pkg/errorx"
	"github.com/neosy/elengrab/internal/pkg/errorx/exceptionx"
	"github.com/neosy/elengrab/internal/pkg/fasthttpx"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
	"github.com/valyala/fasthttp"
)

// PlaylistsPageHandler renders the list of playlists available to the current user.
func (h *DownloaderHandlers) PlaylistsPageHandler(ctx *fasthttp.RequestCtx) {
	if ctx.IsHead() {
		fasthttpx.WriteHTMLHeadResponse(ctx)
		return
	}

	authCtx := policy.ResolveUserOrAnonym(ctx)

	searchValues, err := h.parseSearchGetRequest(ctx)
	if err != nil {
		fasthttpx.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	query, err := h.mappers.MapSearchValuesToUsecaseQuery(searchValues)
	if err != nil {
		fasthttpx.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	h.renderPlaylistsPage(ctx, authCtx, query)
}

// PlaylistPageHandler renders a single playlist page with its media items.
func (h *DownloaderHandlers) PlaylistPageHandler(ctx *fasthttp.RequestCtx) {
	if ctx.IsHead() {
		fasthttpx.WriteHTMLHeadResponse(ctx)
		return
	}

	authCtx := policy.ResolveUserOrAnonym(ctx)

	playlistID, err := h.extractPlaylistID(ctx)
	if err != nil {
		fasthttpx.WriteErrorx(ctx, err)
		return
	}

	playlist, err := h.downloader.GetPlaylist(ctx, authCtx, playlistID)
	if err != nil {
		fasthttpx.WriteErrorx(ctx, err)
		return
	}

	searchValues, err := h.parseSearchGetRequest(ctx)
	if err != nil {
		fasthttpx.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	if searchValues.Parameters.Filters == nil {
		searchValues.Parameters.Filters = types.NewQueryFilters()
	}

	searchValues.Parameters.Filters.Add(qkeys.PlaylistIDKey, idcodec.EncodeUUIDBase64URL(playlistID))

	query, err := h.mappers.MapSearchValuesToUsecaseQuery(searchValues)
	if err != nil {
		fasthttpx.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	h.renderPlaylistPage(ctx, authCtx, playlist, query)
}
