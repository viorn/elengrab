package dto

// PlaylistResponse is the response payload for a playlist.
type PlaylistResponse struct {
	PlaylistID string `json:"playlistId"`
	Title      string `json:"title"`

	ItemsCount int64 `json:"itemsCount"`
	IsShared   bool  `json:"isShared"`
	CanEdit    bool  `json:"canEdit"`

	URL string `json:"url,omitempty"`
}

// PlaylistItemResponse is the response payload for a playlist membership toggle.
type PlaylistItemResponse struct {
	PlaylistID string `json:"playlistId"`
	DownloadID string `json:"downloadId"`

	Checked bool `json:"checked"`
}
