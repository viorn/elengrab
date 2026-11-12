package mappers

import (
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	"github.com/neosy/elengrab/internal/pkg/idcodec"

	"github.com/neosy/elengrab/internal/api/rest/server/internal/handlers/ui/downloader/dto"
)

// MapPlaylistToResponse maps a playlist domain entity to the API response.
func (m *Mappers) MapPlaylistToResponse(
	playlist *ddownload.Playlist,
	canEdit bool,
) *dto.PlaylistResponse {
	if playlist == nil {
		return nil
	}

	return &dto.PlaylistResponse{
		PlaylistID: idcodec.EncodeUUIDBase64URL(playlist.PlaylistID),
		Title:      playlist.Title,
		ItemsCount: playlist.ItemsCount,
		IsShared:   playlist.IsShared(),
		CanEdit:    canEdit,
	}
}
