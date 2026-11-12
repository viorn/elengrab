package mediadownload

import (
	"context"

	"uuid"

	ddownload "github.com/neosy/elengrab/internal/domain/download"
	dtypes "github.com/neosy/elengrab/internal/domain/types"
)

// ListByPlaylist returns media downloads that belong to the playlist.
func (uc *MediaDownload) ListByPlaylist(
	ctx context.Context,
	playlistID uuid.UUID,
	options dtypes.QueryMediaOptions,
) ([]*ddownload.MediaDownload, error) {
	downloads, err := uc.downloadRepo().ListByPlaylist(ctx, playlistID, options)
	if err != nil {
		uc.logger.Warn("Failed to get playlist downloads", "error", err)
		return nil, err
	}

	return downloads, nil
}
