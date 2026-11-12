package downloader

import (
	"context"
	"strings"
	"time"

	"uuid"

	apperrors "github.com/neosy/elengrab/internal/app/errors"
	dauth "github.com/neosy/elengrab/internal/domain/auth"
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	ierrors "github.com/neosy/elengrab/internal/errors"
	"github.com/neosy/elengrab/internal/pkg/errorx"
	"github.com/valyala/fasthttp"
)

const (
	playlistTitleMaxLen       = 100
	playlistDescriptionMaxLen = 1000
)

var (
	ErrPlaylistNotFound = errorx.New(
		"playlist not found",
		errorx.WithErrorMessage("Playlist not found"),
		errorx.WithHttpStatus(fasthttp.StatusNotFound),
	)
	ErrPlaylistTitleIsEmpty = errorx.New(
		"playlist title is empty",
		errorx.WithErrorMessage("Playlist title is required"),
		errorx.WithHttpStatus(fasthttp.StatusBadRequest),
	)
)

// PlaylistWithMembership describes a playlist and whether the media item belongs to it.
type PlaylistWithMembership struct {
	Playlist *ddownload.Playlist
	Checked  bool
}

// playlistOwnerID returns the owner ID for new playlists.
// Anonymous users create shared playlists (user_id = NULL).
func playlistOwnerID(authCtx dauth.AuthContext) *uuid.UUID {
	if authCtx.IsAnonymous() || authCtx.UserID == uuid.Nil() {
		return nil
	}

	userID := authCtx.UserID
	return &userID
}

func (uc *downloader) validatePlaylistTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", ErrPlaylistTitleIsEmpty
	}
	if len([]rune(title)) > playlistTitleMaxLen {
		title = string([]rune(title)[:playlistTitleMaxLen])
	}

	return title, nil
}

func (uc *downloader) getPlaylistForUser(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
) (*ddownload.Playlist, error) {
	playlist, err := uc.playlistRepo().FindByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, ErrPlaylistNotFound
	}

	if playlist.UserID != nil && *playlist.UserID != authCtx.UserID && !authCtx.IsAdmin() {
		return nil, ierrors.ErrAccessDenied
	}

	return playlist, nil
}

func (uc *downloader) canEditPlaylist(authCtx dauth.AuthContext, playlist *ddownload.Playlist) bool {
	if playlist == nil {
		return false
	}

	if playlist.UserID == nil {
		return uc.HasWriteAccess(authCtx)
	}

	return *playlist.UserID == authCtx.UserID || authCtx.IsAdmin()
}

// CreatePlaylist creates a new playlist.
func (uc *downloader) CreatePlaylist(
	ctx context.Context,
	authCtx dauth.AuthContext,
	title string,
) (*ddownload.Playlist, error) {
	if !uc.CanCreateMediaDownload(authCtx) {
		return nil, ierrors.ErrAccessDenied
	}

	title, err := uc.validatePlaylistTitle(title)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	playlist := &ddownload.Playlist{
		PlaylistID: uuid.New(),
		UserID:     playlistOwnerID(authCtx),
		Title:      title,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := uc.playlistRepo().Insert(ctx, playlist); err != nil {
		return nil, err
	}

	return playlist, nil
}

// ListPlaylists returns playlists visible to the current user.
func (uc *downloader) ListPlaylists(
	ctx context.Context,
	authCtx dauth.AuthContext,
) ([]*ddownload.Playlist, error) {
	return uc.playlistRepo().List(ctx, playlistOwnerID(authCtx))
}

// GetPlaylist returns a playlist by identifier.
func (uc *downloader) GetPlaylist(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
) (*ddownload.Playlist, error) {
	return uc.getPlaylistForUser(ctx, authCtx, playlistID)
}

// RenamePlaylist updates the playlist title.
func (uc *downloader) RenamePlaylist(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
	title string,
) error {
	playlist, err := uc.getPlaylistForUser(ctx, authCtx, playlistID)
	if err != nil {
		return err
	}
	if !uc.canEditPlaylist(authCtx, playlist) {
		return ierrors.ErrAccessDenied
	}

	title, err = uc.validatePlaylistTitle(title)
	if err != nil {
		return err
	}

	playlist.Title = title
	playlist.UpdatedAt = time.Now().UTC()

	return uc.playlistRepo().Update(ctx, playlist)
}

// DeletePlaylist soft deletes the playlist.
func (uc *downloader) DeletePlaylist(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
) error {
	playlist, err := uc.getPlaylistForUser(ctx, authCtx, playlistID)
	if err != nil {
		return err
	}
	if !uc.canEditPlaylist(authCtx, playlist) {
		return ierrors.ErrAccessDenied
	}

	return uc.playlistRepo().SoftDelete(ctx, playlistID)
}

// AddPlaylistItem adds a media download to the playlist.
func (uc *downloader) AddPlaylistItem(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
	downloadID uuid.UUID,
) error {
	playlist, err := uc.getPlaylistForUser(ctx, authCtx, playlistID)
	if err != nil {
		return err
	}
	if !uc.canEditPlaylist(authCtx, playlist) {
		return ierrors.ErrAccessDenied
	}

	download, err := uc.GetDownloadInfo(ctx, authCtx, downloadID)
	if err != nil {
		return err
	}
	if download == nil || download.MediaDownload == nil {
		return apperrors.ErrDownloadNotFound
	}

	return uc.playlistRepo().AddItem(ctx, playlistID, downloadID)
}

// RemovePlaylistItem removes a media download from the playlist.
func (uc *downloader) RemovePlaylistItem(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
	downloadID uuid.UUID,
) error {
	playlist, err := uc.getPlaylistForUser(ctx, authCtx, playlistID)
	if err != nil {
		return err
	}
	if !uc.canEditPlaylist(authCtx, playlist) {
		return ierrors.ErrAccessDenied
	}

	return uc.playlistRepo().RemoveItem(ctx, playlistID, downloadID)
}

// ReorderPlaylistItems rewrites the order of the playlist items.
func (uc *downloader) ReorderPlaylistItems(
	ctx context.Context,
	authCtx dauth.AuthContext,
	playlistID uuid.UUID,
	downloadIDs []uuid.UUID,
) error {
	playlist, err := uc.getPlaylistForUser(ctx, authCtx, playlistID)
	if err != nil {
		return err
	}
	if !uc.canEditPlaylist(authCtx, playlist) {
		return ierrors.ErrAccessDenied
	}

	if len(downloadIDs) == 0 {
		return nil
	}

	return uc.playlistRepo().UpdatePositions(ctx, playlistID, downloadIDs)
}

// ListPlaylistsForItem returns visible playlists with membership flags for the download.
func (uc *downloader) ListPlaylistsForItem(
	ctx context.Context,
	authCtx dauth.AuthContext,
	downloadID uuid.UUID,
) ([]*PlaylistWithMembership, error) {
	playlists, err := uc.ListPlaylists(ctx, authCtx)
	if err != nil {
		return nil, err
	}

	memberIDs, err := uc.playlistRepo().ListPlaylistIDsByDownload(ctx, downloadID, playlistOwnerID(authCtx))
	if err != nil {
		return nil, err
	}

	members := make(map[uuid.UUID]bool, len(memberIDs))
	for _, id := range memberIDs {
		members[id] = true
	}

	result := make([]*PlaylistWithMembership, 0, len(playlists))
	for _, playlist := range playlists {
		result = append(result, &PlaylistWithMembership{
			Playlist: playlist,
			Checked:  members[playlist.PlaylistID],
		})
	}

	return result, nil
}
