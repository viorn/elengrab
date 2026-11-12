package ddownload

import (
	"time"

	"uuid"
)

// Playlist groups media downloads into a user-defined collection.
// A media download can belong to any number of playlists.
type Playlist struct {
	// Unique playlist identifier (UUID)
	PlaylistID uuid.UUID

	// Owner identifier (UUID). Nil means a shared playlist.
	UserID *uuid.UUID

	// Playlist title
	Title string

	// Optional description
	Description *string

	// Number of media items in the playlist
	ItemsCount int64

	// ThumbnailID is the thumbnail of the first media item in the playlist.
	// Nil when the playlist is empty or the item has no thumbnail.
	ThumbnailID *uuid.UUID

	// Timestamp when the record was created
	CreatedAt time.Time

	// Timestamp when the record was last updated
	UpdatedAt time.Time

	// Timestamp when the record was soft deleted
	DeletedAt *time.Time
}

// IsShared reports whether the playlist has no owner and is visible to everyone.
func (p *Playlist) IsShared() bool {
	return p != nil && p.UserID == nil
}

// IsOwnedBy reports whether the playlist belongs to the given user.
func (p *Playlist) IsOwnedBy(userID uuid.UUID) bool {
	return p != nil && p.UserID != nil && *p.UserID == userID
}

// Clone returns a deep copy of the playlist.
func (src *Playlist) Clone() *Playlist {
	if src == nil {
		return nil
	}

	copy := *src

	if src.UserID != nil {
		userID := *src.UserID
		copy.UserID = &userID
	}

	if src.Description != nil {
		description := *src.Description
		copy.Description = &description
	}

	if src.DeletedAt != nil {
		deletedAt := *src.DeletedAt
		copy.DeletedAt = &deletedAt
	}

	if src.ThumbnailID != nil {
		thumbnailID := *src.ThumbnailID
		copy.ThumbnailID = &thumbnailID
	}

	return &copy
}
