package pages

import "github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/common/composition/paths"

// Playlists page
type PlaylistsPageData struct {
	BasePaths  paths.HttpPaths
	BaseValues baseValues
	Paths      PagePaths
	Values     PlaylistsPageValues
	Extra      map[string]any
}

type PlaylistsPageValues struct {
	PagesListValues

	Playlists    []PlaylistCard
	HasPlaylists bool
	CanCreate    bool
}

// PlaylistCard is a single playlist entry rendered on the playlists page.
type PlaylistCard struct {
	EncodedPlaylistID string
	Title             string
	Description       string

	ItemsCount int64
	IsShared   bool
	CanEdit    bool

	URL string
}
