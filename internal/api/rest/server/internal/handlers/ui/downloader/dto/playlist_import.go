package dto

// PlaylistImportRequest is the request body for importing an external playlist.
type PlaylistImportRequest struct {
	MediaURL string `json:"url" validate:"required,url,max=2048"`
	// PlaylistID is the target playlist; empty means "create a new one".
	PlaylistID       string `json:"playlistId"`
	NewPlaylistTitle string `json:"newPlaylistTitle" validate:"max=100"`
	Limit            int    `json:"limit" validate:"omitempty,min=1,max=500"`

	QualityCodec      string `json:"qualityCodec"`
	QualityResolution string `json:"qualityResolution"`
	Format            string `json:"format"`
}

// PlaylistPreviewItem is a single item of an external playlist.
type PlaylistPreviewItem struct {
	MediaURL        string `json:"url"`
	Title           string `json:"title"`
	DurationSeconds int64  `json:"duration"`
	Existing        bool   `json:"existing"`
	DownloadID      string `json:"downloadId,omitempty"`
}

// PlaylistPreviewTarget is a playlist the items may be imported into.
type PlaylistPreviewTarget struct {
	PlaylistID string `json:"playlistId"`
	Title      string `json:"title"`
}

// PlaylistPreviewResponse describes an external playlist before import.
type PlaylistPreviewResponse struct {
	Title             string                  `json:"title"`
	Total             int                     `json:"total"`
	Limit             int                     `json:"limit"`
	AlreadyDownloaded int                     `json:"alreadyDownloaded"`
	Items             []PlaylistPreviewItem   `json:"items"`
	Playlists         []PlaylistPreviewTarget `json:"playlists"`
}

// PlaylistImportResponse reports the import result.
type PlaylistImportResponse struct {
	PlaylistID    string `json:"playlistId"`
	PlaylistTitle string `json:"title"`
	URL           string `json:"url"`
	Total         int    `json:"total"`
	Queued        int    `json:"queued"`
	Reused        int    `json:"reused"`
	Skipped       int    `json:"skipped"`
}
