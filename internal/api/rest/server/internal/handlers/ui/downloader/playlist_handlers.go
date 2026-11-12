package downloader

import (
	"encoding/json"
	"net/http"

	"uuid"

	apierrors "github.com/neosy/elengrab/internal/api/errors"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/policy"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/dto"
	httppaths "github.com/neosy/elengrab/internal/api/rest/server/internal/paths"
	dauth "github.com/neosy/elengrab/internal/domain/auth"
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	"github.com/neosy/elengrab/internal/pkg/errorx"
	"github.com/neosy/elengrab/internal/pkg/errorx/exceptionx"
	nfasthttp "github.com/neosy/elengrab/internal/pkg/fasthttpx"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
	"github.com/valyala/fasthttp"
)

// canEditPlaylist reports whether the current auth context may modify the playlist.
func (h *DownloaderHandlers) canEditPlaylist(authCtx dauth.AuthContext, playlist *ddownload.Playlist) bool {
	if playlist == nil {
		return false
	}

	if playlist.UserID == nil {
		return h.downloader.HasWriteAccess(authCtx)
	}

	return *playlist.UserID == authCtx.UserID || authCtx.IsAdmin()
}

func (h *DownloaderHandlers) playlistResponse(
	authCtx dauth.AuthContext,
	playlist *ddownload.Playlist,
) *dto.PlaylistResponse {
	resp := h.mappers.MapPlaylistToResponse(playlist, h.canEditPlaylist(authCtx, playlist))
	if resp == nil {
		return nil
	}

	resp.URL = httppaths.BuildPlaylistPath(playlist.PlaylistID)

	return resp
}

// CreatePlaylistHandler creates a new playlist.
func (h *DownloaderHandlers) CreatePlaylistHandler(ctx *fasthttp.RequestCtx) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	var receivedReq dto.PlaylistRequest
	if err := json.Unmarshal(ctx.PostBody(), &receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewHTTPMessage("Invalid JSON request body", http.StatusBadRequest, err))
		return
	}

	if err := h.validators.Validate.Struct(receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	playlist, err := h.downloader.CreatePlaylist(ctx, authCtx, receivedReq.Title)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	if err := nfasthttp.WriteResponse(ctx, h.playlistResponse(authCtx, playlist)); err != nil {
		return
	}
}

// RenamePlaylistHandler updates the playlist title.
func (h *DownloaderHandlers) RenamePlaylistHandler(ctx *fasthttp.RequestCtx) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	playlistID, err := h.extractPlaylistID(ctx)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	var receivedReq dto.PlaylistRequest
	if err := json.Unmarshal(ctx.PostBody(), &receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewHTTPMessage("Invalid JSON request body", http.StatusBadRequest, err))
		return
	}

	if err := h.validators.Validate.Struct(receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	if err := h.downloader.RenamePlaylist(ctx, authCtx, playlistID, receivedReq.Title); err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	playlist, err := h.downloader.GetPlaylist(ctx, authCtx, playlistID)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	if err := nfasthttp.WriteResponse(ctx, h.playlistResponse(authCtx, playlist)); err != nil {
		return
	}
}

// DeletePlaylistHandler soft deletes the playlist.
func (h *DownloaderHandlers) DeletePlaylistHandler(ctx *fasthttp.RequestCtx) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	playlistID, err := h.extractPlaylistID(ctx)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	if err := h.downloader.DeletePlaylist(ctx, authCtx, playlistID); err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// AddPlaylistItemHandler adds a media item to the playlist.
func (h *DownloaderHandlers) AddPlaylistItemHandler(ctx *fasthttp.RequestCtx) {
	h.togglePlaylistItem(ctx, true)
}

// RemovePlaylistItemHandler removes a media item from the playlist.
func (h *DownloaderHandlers) RemovePlaylistItemHandler(ctx *fasthttp.RequestCtx) {
	h.togglePlaylistItem(ctx, false)
}

func (h *DownloaderHandlers) togglePlaylistItem(ctx *fasthttp.RequestCtx, add bool) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	playlistID, err := h.extractPlaylistID(ctx)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	downloadID, err := h.extractDownloadID(ctx)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	if add {
		err = h.downloader.AddPlaylistItem(ctx, authCtx, playlistID, downloadID)
	} else {
		err = h.downloader.RemovePlaylistItem(ctx, authCtx, playlistID, downloadID)
	}
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	resp := dto.PlaylistItemResponse{
		PlaylistID: idcodec.EncodeUUIDBase64URL(playlistID),
		DownloadID: idcodec.EncodeUUIDBase64URL(downloadID),
		Checked:    add,
	}

	if err := nfasthttp.WriteResponse(ctx, resp); err != nil {
		return
	}
}

// MediaItemPlaylistsHandler renders the "playlists of the item" dialog (GET)
// or toggles item membership (POST).
func (h *DownloaderHandlers) MediaItemPlaylistsHandler(ctx *fasthttp.RequestCtx) {
	if ctx.IsGet() || ctx.IsHead() {
		h.renderMediaItemPlaylistsDialog(ctx)
		return
	}

	if ctx.IsPost() {
		h.toggleMediaItemPlaylist(ctx)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
}

func (h *DownloaderHandlers) toggleMediaItemPlaylist(ctx *fasthttp.RequestCtx) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	downloadID, err := h.extractDownloadID(ctx)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	var receivedReq dto.PlaylistItemRequest
	if err := json.Unmarshal(ctx.PostBody(), &receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewHTTPMessage("Invalid JSON request body", http.StatusBadRequest, err))
		return
	}

	if err := h.validators.Validate.Struct(receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	playlistID, err := idcodec.DecodeUUIDBase64URL(receivedReq.PlaylistID)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, apierrors.ErrPlaylistIDIsIncorrect.Wrap(err))
		return
	}
	if playlistID == uuid.Nil() {
		nfasthttp.WriteErrorx(ctx, apierrors.ErrPlaylistIDIsRequired)
		return
	}

	if receivedReq.Checked {
		err = h.downloader.AddPlaylistItem(ctx, authCtx, playlistID, downloadID)
	} else {
		err = h.downloader.RemovePlaylistItem(ctx, authCtx, playlistID, downloadID)
	}
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	resp := dto.PlaylistItemResponse{
		PlaylistID: receivedReq.PlaylistID,
		DownloadID: idcodec.EncodeUUIDBase64URL(downloadID),
		Checked:    receivedReq.Checked,
	}

	if err := nfasthttp.WriteResponse(ctx, resp); err != nil {
		return
	}
}
