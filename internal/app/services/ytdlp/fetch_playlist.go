package ytdlpsrv

import (
	"context"
	"time"

	pservices "github.com/neosy/elengrab/internal/ports/services"
)

// FetchPlaylist retrieves the flat list of playlist items for the given URL.
func (srv *YtDlpService) FetchPlaylist(
	ctx context.Context,
	url string,
	limit int,
	opts ...pservices.DownloaderOption,
) (*pservices.PlaylistInfo, error) {
	info, err := srv.downloader.FetchPlaylist(ctx, url, limit, internalRequestOptions(opts...))
	if err != nil {
		return nil, err
	}

	if info == nil {
		return nil, nil
	}

	result := &pservices.PlaylistInfo{
		Title:   info.Title,
		Total:   info.Total,
		Entries: make([]pservices.PlaylistEntry, 0, len(info.Entries)),
	}

	for _, entry := range info.Entries {
		result.Entries = append(result.Entries, pservices.PlaylistEntry{
			Index:    entry.Index,
			MediaURL: entry.MediaURL,
			Title:    entry.Title,
			Duration: time.Duration(entry.Duration) * time.Second,
		})
	}

	return result, nil
}
