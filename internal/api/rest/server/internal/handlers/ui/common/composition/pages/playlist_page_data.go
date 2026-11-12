package pages

import (
	"encoding/json"

	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/paths"
)

// Playlist page
type PlaylistPageData struct {
	BasePaths  paths.HttpPaths
	BaseValues baseValues
	Paths      PagePaths
	Values     PlaylistPageValues
	Extra      map[string]any
}

type PlaylistPageValues struct {
	PagesListValues

	PlaylistHeader PlaylistHeader
}

type PlaylistHeader struct {
	EncodedPlaylistID string `json:"playlistId"`

	Title       string `json:"title"`
	Description string `json:"description"`

	ItemsCount int64 `json:"itemsCount"`
	CanEdit    bool  `json:"canEdit"`
	IsShared   bool  `json:"isShared"`

	Show bool `json:"show"`
}

func (p PlaylistHeader) JSON() []byte {
	data, err := json.Marshal(p)
	if err != nil {
		return nil
	}

	return data
}
