package persistence

import (
	"context"

	"uuid"

	ddownload "github.com/neosy/elengrab/internal/domain/download"
)

type PlaylistRepositoryFactory func() PlaylistRepository

type PlaylistRepository interface {
	// Insert inserts a playlist record.
	Insert(ctx context.Context, playlist *ddownload.Playlist) error
	// Update updates a playlist record.
	Update(ctx context.Context, playlist *ddownload.Playlist) error
	// SoftDelete soft deletes a playlist and detaches all its items.
	SoftDelete(ctx context.Context, playlistID uuid.UUID) error

	// FindByID returns a playlist by identifier.
	FindByID(ctx context.Context, playlistID uuid.UUID) (*ddownload.Playlist, error)
	// List returns playlists visible to the given user (own + shared).
	List(ctx context.Context, userID *uuid.UUID) ([]*ddownload.Playlist, error)

	// AddItem adds a media download to a playlist.
	AddItem(ctx context.Context, playlistID, downloadID uuid.UUID) error
	// RemoveItem removes a media download from a playlist.
	RemoveItem(ctx context.Context, playlistID, downloadID uuid.UUID) error
	// ItemExists reports whether the media download belongs to the playlist.
	ItemExists(ctx context.Context, playlistID, downloadID uuid.UUID) (bool, error)
	// ListPlaylistIDsByDownload returns identifiers of playlists containing the download.
	ListPlaylistIDsByDownload(ctx context.Context, downloadID uuid.UUID, userID *uuid.UUID) ([]uuid.UUID, error)
	// CountItems returns the number of existing media items in the playlist.
	CountItems(ctx context.Context, playlistID uuid.UUID) (int64, error)
}
