package downloader

import (
	"context"
	"regexp"
	"strings"

	"uuid"

	dauth "github.com/neosy/elengrab/internal/domain/auth"
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	dtypes "github.com/neosy/elengrab/internal/domain/types"
	ierrors "github.com/neosy/elengrab/internal/errors"
	pservices "github.com/neosy/elengrab/internal/ports/services"
)

const (
	// PlaylistImportDefaultLimit is the default number of items imported from an external playlist.
	PlaylistImportDefaultLimit = 50
	// PlaylistImportMaxLimit is the hard cap for a single import.
	PlaylistImportMaxLimit = 500

	playlistImportFallbackTitle = "Imported playlist"
)

// PlaylistImportEntry is a single item of an external playlist prepared for import.
type PlaylistImportEntry struct {
	Index           int
	MediaURL        string
	Title           string
	DurationSeconds int64
	// Existing is true when the media is already downloaded (or being downloaded).
	Existing   bool
	DownloadID uuid.UUID
}

// PlaylistImportPreview describes an external playlist before importing.
type PlaylistImportPreview struct {
	Title             string
	Total             int
	Limit             int
	AlreadyDownloaded int
	Entries           []PlaylistImportEntry
	// Playlists are the playlists the current user may write into.
	Playlists []*ddownload.Playlist
}

// PlaylistImportParams are the parameters of the import.
type PlaylistImportParams struct {
	URL string
	// PlaylistID is the target playlist; when nil a new one is created.
	PlaylistID *uuid.UUID
	// NewPlaylistTitle overrides the title of the created playlist.
	NewPlaylistTitle string
	Limit            int
	Options          *ddownload.DownloadOptions
}

// PlaylistImportResult reports what happened during the import.
type PlaylistImportResult struct {
	PlaylistID    uuid.UUID
	PlaylistTitle string
	Queued        int
	Reused        int
	Skipped       int
	Total         int
}

var youTubeIDRe = regexp.MustCompile(`(?:youtu\.be/|[?&]v=|/shorts/|/embed/|/live/)([A-Za-z0-9_-]{11})`)

// mediaURLKey builds a comparable key for a media URL so that different forms of
// the same link (watch?v=ID, youtu.be/ID, …&list=…) match each other.
func mediaURLKey(mediaURL string) string {
	if id := youTubeIDRe.FindStringSubmatch(mediaURL); len(id) == 2 {
		return "yt:" + id[1]
	}

	return strings.TrimRight(strings.TrimSpace(mediaURL), "/")
}

// mediaURLPatterns builds SQL LIKE patterns used to look up already downloaded media.
func mediaURLPatterns(mediaURLs []string) []string {
	patterns := make([]string, 0, len(mediaURLs)*2)
	seen := make(map[string]bool, len(mediaURLs)*2)

	add := func(pattern string) {
		if seen[pattern] {
			return
		}
		seen[pattern] = true
		patterns = append(patterns, pattern)
	}

	for _, mediaURL := range mediaURLs {
		if id := youTubeIDRe.FindStringSubmatch(mediaURL); len(id) == 2 {
			add("%v=" + id[1] + "%")
			add("%youtu.be/" + id[1] + "%")
			continue
		}

		if mediaURL = strings.TrimSpace(mediaURL); mediaURL != "" {
			add("%" + mediaURL + "%")
		}
	}

	return patterns
}

func clampPlaylistImportLimit(limit int) int {
	if limit <= 0 {
		return PlaylistImportDefaultLimit
	}
	if limit > PlaylistImportMaxLimit {
		return PlaylistImportMaxLimit
	}

	return limit
}

// findExistingDownloads maps media URL keys to already downloaded media.
func (uc *downloader) findExistingDownloads(
	ctx context.Context,
	mediaURLs []string,
) (map[string]*ddownload.MediaDownload, error) {
	patterns := mediaURLPatterns(mediaURLs)
	if len(patterns) == 0 {
		return map[string]*ddownload.MediaDownload{}, nil
	}

	downloads, err := uc.download.FindByMediaURLPatterns(ctx, patterns)
	if err != nil {
		return nil, err
	}

	existing := make(map[string]*ddownload.MediaDownload, len(downloads))
	for _, download := range downloads {
		if download == nil || download.MediaURL == nil {
			continue
		}

		key := mediaURLKey(*download.MediaURL)
		if _, exists := existing[key]; exists {
			continue
		}

		existing[key] = download
	}

	return existing, nil
}

// FetchPlaylistPreview returns the items of an external playlist and marks the ones
// that are already downloaded.
func (uc *downloader) FetchPlaylistPreview(
	ctx context.Context,
	authCtx dauth.AuthContext,
	mediaURL string,
	limit int,
) (*PlaylistImportPreview, error) {
	if !uc.CanCreateMediaDownload(authCtx) {
		return nil, ierrors.ErrAccessDenied
	}

	limit = clampPlaylistImportLimit(limit)

	info, err := uc.downloaderSrv.FetchPlaylist(ctx, mediaURL, limit, pservices.WithDownloaderCookies())
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, ierrors.ErrFuncParamNullPointer
	}

	entries := info.Entries
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	mediaURLs := make([]string, 0, len(entries))
	for _, entry := range entries {
		mediaURLs = append(mediaURLs, entry.MediaURL)
	}

	existing, err := uc.findExistingDownloads(ctx, mediaURLs)
	if err != nil {
		return nil, err
	}

	preview := &PlaylistImportPreview{
		Title:   info.Title,
		Total:   info.Total,
		Limit:   limit,
		Entries: make([]PlaylistImportEntry, 0, len(entries)),
	}

	if preview.Title == "" {
		preview.Title = playlistImportFallbackTitle
	}

	for _, entry := range entries {
		item := PlaylistImportEntry{
			Index:           entry.Index,
			MediaURL:        entry.MediaURL,
			Title:           entry.Title,
			DurationSeconds: int64(entry.Duration.Seconds()),
		}

		if download, ok := existing[mediaURLKey(entry.MediaURL)]; ok {
			item.Existing = true
			item.DownloadID = download.DownloadID
			preview.AlreadyDownloaded++
		}

		preview.Entries = append(preview.Entries, item)
	}

	playlists, err := uc.ListPlaylists(ctx, authCtx)
	if err != nil {
		return nil, err
	}

	for _, playlist := range playlists {
		if uc.canEditPlaylist(authCtx, playlist) {
			preview.Playlists = append(preview.Playlists, playlist)
		}
	}

	return preview, nil
}

// ImportPlaylist queues every downloadable item of an external playlist into a
// playlist (existing or newly created). Already downloaded media is reused.
func (uc *downloader) ImportPlaylist(
	ctx context.Context,
	authCtx dauth.AuthContext,
	params PlaylistImportParams,
) (*PlaylistImportResult, error) {
	if !uc.CanCreateMediaDownload(authCtx) {
		return nil, ierrors.ErrAccessDenied
	}

	limit := clampPlaylistImportLimit(params.Limit)

	info, err := uc.downloaderSrv.FetchPlaylist(ctx, params.URL, limit, pservices.WithDownloaderCookies())
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, ierrors.ErrFuncParamNullPointer
	}

	entries := info.Entries
	if len(entries) > limit {
		entries = entries[:limit]
	}

	var playlist *ddownload.Playlist
	switch {
	case params.PlaylistID != nil && *params.PlaylistID != uuid.Nil():
		playlist, err = uc.getPlaylistForUser(ctx, authCtx, *params.PlaylistID)
		if err != nil {
			return nil, err
		}
		if !uc.canEditPlaylist(authCtx, playlist) {
			return nil, ierrors.ErrAccessDenied
		}
	default:
		title := strings.TrimSpace(params.NewPlaylistTitle)
		if title == "" {
			title = info.Title
		}
		if title == "" {
			title = playlistImportFallbackTitle
		}

		playlist, err = uc.CreatePlaylist(ctx, authCtx, title)
		if err != nil {
			return nil, err
		}
	}

	mediaURLs := make([]string, 0, len(entries))
	for _, entry := range entries {
		mediaURLs = append(mediaURLs, entry.MediaURL)
	}

	existing, err := uc.findExistingDownloads(ctx, mediaURLs)
	if err != nil {
		return nil, err
	}

	options := params.Options
	if options == nil {
		options = &ddownload.DownloadOptions{FormatType: dtypes.FormatTypeVideoAudio}
	}

	result := &PlaylistImportResult{
		PlaylistID:    playlist.PlaylistID,
		PlaylistTitle: playlist.Title,
		Total:         len(entries),
	}

	for _, entry := range entries {
		download, exists := existing[mediaURLKey(entry.MediaURL)]

		var downloadID uuid.UUID
		if exists && download != nil {
			downloadID = download.DownloadID
			result.Reused++
		} else {
			scheduleResp, err := uc.ScheduleDownload(ctx, authCtx, entry.MediaURL, options)
			if err != nil || scheduleResp == nil {
				uc.logger.Warn("Failed to schedule playlist item",
					"url", entry.MediaURL, "error", err)
				result.Skipped++
				continue
			}

			downloadID = scheduleResp.DownloadID
			result.Queued++
		}

		if err := uc.AddPlaylistItem(ctx, authCtx, playlist.PlaylistID, downloadID); err != nil {
			uc.logger.Warn("Failed to add playlist item",
				"playlistID", playlist.PlaylistID, "downloadID", downloadID, "error", err)
			result.Skipped++
		}

		existing[mediaURLKey(entry.MediaURL)] = &ddownload.MediaDownload{DownloadID: downloadID}
	}

	result.Total = result.Queued + result.Reused + result.Skipped

	return result, nil
}
