package mediadownload

import (
	"context"

	ddownload "github.com/neosy/elengrab/internal/domain/download"
)

// FindByMediaURLPatterns returns non-deleted downloads whose source URL matches
// any of the given SQL LIKE patterns.
func (uc *MediaDownload) FindByMediaURLPatterns(
	ctx context.Context,
	patterns []string,
) ([]*ddownload.MediaDownload, error) {
	downloads, err := uc.downloadRepo().FindByMediaURLPatterns(ctx, patterns)
	if err != nil {
		uc.logger.Warn("Failed to find downloads by media URL patterns", "error", err)
		return nil, err
	}

	return downloads, nil
}
