package dto

// PlaylistRequest is the request body for creating or renaming a playlist.
type PlaylistRequest struct {
	Title string `json:"title" validate:"required,max=100"`
}

// PlaylistItemRequest is the request body for toggling a media item in a playlist.
type PlaylistItemRequest struct {
	PlaylistID string `json:"playlistId" validate:"required"`
	Checked    bool   `json:"checked"`
}
