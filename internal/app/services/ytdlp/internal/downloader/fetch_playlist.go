package downloader

import (
	"context"

	idto "github.com/neosy/elengrab/internal/app/services/ytdlp/internal/downloader/dto"
	"github.com/neosy/elengrab/internal/app/services/ytdlp/internal/downloader/helper"
)

// FetchPlaylist returns the flat list of playlist entries.
func (d *Downloader) FetchPlaylist(
	ctx context.Context,
	url string,
	limit int,
	options idto.RequestOptions,
) (*idto.PlaylistInfo, error) {
	var cookieFilePath string
	if options.AllowCookies {
		cookieFilePath, _ = helper.CookieFilePathFromURL(url, d.serviceOptions.CookiesDir)
	}

	return d.executor.FetchPlaylistInfo(ctx, url, limit, idto.WithUseCookies(cookieFilePath))
}
