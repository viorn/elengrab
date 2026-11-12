package downloader

import (
	"encoding/json"
	"net/http"
	"strconv"

	"uuid"

	apierrors "github.com/neosy/elengrab/internal/api/errors"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/policy"
	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/dto"
	httppaths "github.com/neosy/elengrab/internal/api/rest/server/internal/paths"
	udownloader "github.com/neosy/elengrab/internal/app/usecases/downloader"
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	"github.com/neosy/elengrab/internal/pkg/errorx"
	"github.com/neosy/elengrab/internal/pkg/errorx/exceptionx"
	nfasthttp "github.com/neosy/elengrab/internal/pkg/fasthttpx"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
	"github.com/valyala/fasthttp"
)

// PlaylistPreviewHandler inspects an external playlist (e.g. YouTube) before import.
func (h *DownloaderHandlers) PlaylistPreviewHandler(ctx *fasthttp.RequestCtx) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	mediaURL := string(ctx.QueryArgs().Peek("url"))
	if mediaURL == "" {
		nfasthttp.WriteErrorx(ctx, apierrors.ErrURLIsRequired)
		return
	}

	if err := h.validators.Validate.Var(mediaURL, "url"); err != nil {
		nfasthttp.WriteErrorx(ctx, apierrors.ErrInvalidURL.Wrap(err))
		return
	}

	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))

	preview, err := h.downloader.FetchPlaylistPreview(ctx, authCtx, mediaURL, limit)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	resp := dto.PlaylistPreviewResponse{
		Title:             preview.Title,
		Total:             preview.Total,
		Limit:             preview.Limit,
		AlreadyDownloaded: preview.AlreadyDownloaded,
		Items:             make([]dto.PlaylistPreviewItem, 0, len(preview.Entries)),
		Playlists:         make([]dto.PlaylistPreviewTarget, 0, len(preview.Playlists)),
	}

	for _, entry := range preview.Entries {
		item := dto.PlaylistPreviewItem{
			MediaURL:        entry.MediaURL,
			Title:           entry.Title,
			DurationSeconds: entry.DurationSeconds,
			Existing:        entry.Existing,
		}
		if entry.Existing {
			item.DownloadID = idcodec.EncodeUUIDBase64URL(entry.DownloadID)
		}

		resp.Items = append(resp.Items, item)
	}

	for _, playlist := range preview.Playlists {
		resp.Playlists = append(resp.Playlists, dto.PlaylistPreviewTarget{
			PlaylistID: idcodec.EncodeUUIDBase64URL(playlist.PlaylistID),
			Title:      playlist.Title,
		})
	}

	if err := nfasthttp.WriteResponse(ctx, resp); err != nil {
		return
	}
}

// PlaylistImportHandler queues the items of an external playlist into a playlist.
func (h *DownloaderHandlers) PlaylistImportHandler(ctx *fasthttp.RequestCtx) {
	authCtx, err := policy.ResolveUserOrFallback(ctx, h.appMode)
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	var receivedReq dto.PlaylistImportRequest
	if err := json.Unmarshal(ctx.PostBody(), &receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewHTTPMessage("Invalid JSON request body", http.StatusBadRequest, err))
		return
	}

	if err := h.validators.Validate.Struct(receivedReq); err != nil {
		nfasthttp.WriteErrorx(ctx, errorx.NewFromError(err, exceptionx.VALIDATE))
		return
	}

	var playlistID *uuid.UUID
	if receivedReq.PlaylistID != "" {
		id, err := idcodec.DecodeUUIDBase64URL(receivedReq.PlaylistID)
		if err != nil {
			nfasthttp.WriteErrorx(ctx, apierrors.ErrPlaylistIDIsIncorrect)
			return
		}
		playlistID = &id
	}

	result, err := h.downloader.ImportPlaylist(ctx, authCtx, udownloader.PlaylistImportParams{
		URL:              receivedReq.MediaURL,
		PlaylistID:       playlistID,
		NewPlaylistTitle: receivedReq.NewPlaylistTitle,
		Limit:            receivedReq.Limit,
		Options: &ddownload.DownloadOptions{
			FormatType:      h.mappers.MapFormatType(receivedReq.QualityCodec, receivedReq.Format),
			VideoFormat:     h.mappers.MapVideoFormat(receivedReq.QualityCodec, receivedReq.Format),
			VideoCodec:      h.mappers.MapVideoCodec(receivedReq.QualityCodec),
			VideoResolution: h.mappers.MapVideoResolution(receivedReq.QualityResolution),
			AudioFormat:     h.mappers.MapAudioFormat(receivedReq.Format),
		},
	})
	if err != nil {
		nfasthttp.WriteErrorx(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)

	if err := nfasthttp.WriteResponse(ctx, dto.PlaylistImportResponse{
		PlaylistID:    idcodec.EncodeUUIDBase64URL(result.PlaylistID),
		PlaylistTitle: result.PlaylistTitle,
		URL:           httppaths.BuildPlaylistPath(result.PlaylistID),
		Total:         result.Total,
		Queued:        result.Queued,
		Reused:        result.Reused,
		Skipped:       result.Skipped,
	}); err != nil {
		return
	}
}
