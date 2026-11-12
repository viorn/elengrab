package dto

// PlaylistRequest is the request body for creating or renaming a playlist.
type PlaylistRequest struct {
	Title string `json:"title" validate:"required,max=100"`
}

// PlaylistReorderRequest is the request body for reordering playlist items.
// IDs are base64url-encoded download identifiers in the desired order.
type PlaylistReorderRequest struct {
	IDs []string `json:"ids" validate:"required,min=1,dive,required"`
}

// PlaylistItemRequest is the request body for toggling a media item in a playlist.
type PlaylistItemRequest struct {
	PlaylistID string `json:"playlistId" validate:"required"`
	Checked    bool   `json:"checked"`
}
