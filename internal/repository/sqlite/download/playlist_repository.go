package download

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"uuid"

	"github.com/Masterminds/squirrel"
	ddownload "github.com/neosy/elengrab/internal/domain/download"
	dtypes "github.com/neosy/elengrab/internal/domain/types"
	ierrors "github.com/neosy/elengrab/internal/errors"
	"github.com/neosy/elengrab/internal/ports/persistence"
	"github.com/neosy/elengrab/internal/repository/sqlite/dbexec"
)

const (
	playlistsTable      = "playlists"
	playlistItemsTable  = "playlist_items"
	mediaDownloadsTable = "media_downloads"
)

// firstMediaInfoExpr selects the media_info JSON of the first (by position)
// non-deleted media item in the playlist. It is used to derive the playlist
// preview thumbnail.
const firstMediaInfoExpr = `(SELECT f.media_info
	FROM ` + playlistItemsTable + ` pi
	JOIN ` + mediaDownloadsTable + ` f ON f.download_id = pi.download_id AND f.deleted_at IS NULL
	WHERE pi.playlist_id = p.playlist_id
	ORDER BY pi.position ASC, pi.added_at ASC
	LIMIT 1) AS first_media_info`

type PlaylistRepository struct {
	dbEntry persistence.DBEntry

	retryOptions dbexec.RetryOptions
}

// NewPlaylistRepository returns a factory for creating playlist repositories.
func NewPlaylistRepository(dbEntry persistence.DBEntry) persistence.PlaylistRepositoryFactory {
	return func() persistence.PlaylistRepository {
		return &PlaylistRepository{
			dbEntry: dbEntry,

			retryOptions: dbexec.RetryOptions{
				MaxRetries: maxRetriesDefault,
				Delay:      retryDelayDefault,
			},
		}
	}
}

func (r *PlaylistRepository) Insert(ctx context.Context, playlist *ddownload.Playlist) error {
	return r.save(ctx, playlist)
}

func (r *PlaylistRepository) Update(ctx context.Context, playlist *ddownload.Playlist) error {
	return r.save(ctx, playlist)
}

func (r *PlaylistRepository) save(ctx context.Context, playlist *ddownload.Playlist) error {
	if playlist == nil {
		return ierrors.ErrFuncParamNullPointer
	}

	var userID any
	if playlist.UserID != nil {
		userID = playlist.UserID.String()
	}

	var description any
	if playlist.Description != nil {
		description = *playlist.Description
	}

	sqlQuery, args, err := squirrel.
		Insert(playlistsTable).
		Columns("playlist_id", "user_id", "title", "description", "created_at", "updated_at").
		Values(
			playlist.PlaylistID.String(),
			userID,
			playlist.Title,
			description,
			playlist.CreatedAt.UTC(),
			playlist.UpdatedAt.UTC(),
		).
		Suffix(`ON CONFLICT(playlist_id) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			user_id = excluded.user_id,
			updated_at = excluded.updated_at`).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	return dbexec.ExecContext(ctx, r.dbEntry, sqlQuery, args, r.retryOptions)
}

// mergeOrder возвращает сквозной порядок элементов: сначала переданные downloadIDs
// (в заданном порядке), затем остальные элементы плейлиста в прежнем порядке.
func (r *PlaylistRepository) mergeOrder(
	ctx context.Context,
	playlistID uuid.UUID,
	downloadIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	sqlQuery, args, err := squirrel.
		Select("download_id").
		From(playlistItemsTable).
		Where(squirrel.Eq{"playlist_id": playlistID.String()}).
		OrderBy("position ASC", "download_id ASC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	db := dbexec.Resolve(ctx, r.dbEntry)
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	existing := make(map[uuid.UUID]bool)
	var allIDs []uuid.UUID

	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			return nil, err
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			continue
		}

		existing[id] = true
		allIDs = append(allIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ordered := make([]uuid.UUID, 0, len(allIDs))
	used := make(map[uuid.UUID]bool, len(downloadIDs))

	for _, id := range downloadIDs {
		if !existing[id] || used[id] {
			continue
		}

		ordered = append(ordered, id)
		used[id] = true
	}

	for _, id := range allIDs {
		if used[id] {
			continue
		}

		ordered = append(ordered, id)
	}

	return ordered, nil
}

func (r *PlaylistRepository) SoftDelete(ctx context.Context, playlistID uuid.UUID) error {
	return dbexec.Tx(ctx, r.dbEntry, func(ctx context.Context) error {
		sqlQuery, args, err := squirrel.
			Update(playlistsTable).
			Set("deleted_at", time.Now().UTC()).
			Set("updated_at", time.Now().UTC()).
			Where(squirrel.Eq{"playlist_id": playlistID.String()}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}

		if err := dbexec.ExecContext(ctx, r.dbEntry, sqlQuery, args, r.retryOptions); err != nil {
			return err
		}

		sqlQuery, args, err = squirrel.
			Delete(playlistItemsTable).
			Where(squirrel.Eq{"playlist_id": playlistID.String()}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}

		return dbexec.ExecContext(ctx, r.dbEntry, sqlQuery, args, r.retryOptions)
	})
}

func (r *PlaylistRepository) FindByID(ctx context.Context, playlistID uuid.UUID) (*ddownload.Playlist, error) {
	sqlQuery, args, err := squirrel.
		Select(
			"p.playlist_id", "p.user_id", "p.title", "p.description",
			"p.created_at", "p.updated_at", "p.deleted_at",
			"(SELECT COUNT(*) FROM playlist_items pi WHERE pi.playlist_id = p.playlist_id) AS items_count",
			firstMediaInfoExpr,
		).
		From(playlistsTable + " AS p").
		Where(squirrel.Eq{"p.playlist_id": playlistID.String(), "p.deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	db := dbexec.Resolve(ctx, r.dbEntry)
	row := db.QueryRowContext(ctx, sqlQuery, args...)

	playlist, err := scanPlaylist(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return playlist, nil
}

func (r *PlaylistRepository) List(ctx context.Context, userID *uuid.UUID) ([]*ddownload.Playlist, error) {
	var condition squirrel.Sqlizer = squirrel.Eq{"p.deleted_at": nil}
	if userID != nil && *userID != uuid.Nil() {
		condition = squirrel.And{
			squirrel.Eq{"p.deleted_at": nil},
			squirrel.Or{
				squirrel.Eq{"p.user_id": nil},
				squirrel.Eq{"p.user_id": userID.String()},
			},
		}
	} else {
		condition = squirrel.And{
			squirrel.Eq{"p.deleted_at": nil},
			squirrel.Eq{"p.user_id": nil},
		}
	}

	sqlQuery, args, err := squirrel.
		Select(
			"p.playlist_id", "p.user_id", "p.title", "p.description",
			"p.created_at", "p.updated_at", "p.deleted_at",
			"(SELECT COUNT(*) FROM playlist_items pi WHERE pi.playlist_id = p.playlist_id) AS items_count",
			firstMediaInfoExpr,
		).
		From(playlistsTable + " AS p").
		Where(condition).
		OrderBy("p.title COLLATE NOCASE ASC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	db := dbexec.Resolve(ctx, r.dbEntry)
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*ddownload.Playlist

	for rows.Next() {
		playlist, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, playlist)
	}

	return playlists, rows.Err()
}

func (r *PlaylistRepository) AddItem(ctx context.Context, playlistID, downloadID uuid.UUID) error {
	// Новый элемент встаёт в конец: position = max(position) + 1.
	sqlQuery, args, err := squirrel.
		Insert(playlistItemsTable).
		Columns("playlist_id", "download_id", "position", "added_at").
		Values(
			playlistID.String(),
			downloadID.String(),
			squirrel.Expr("(SELECT COALESCE(MAX(position) + 1, 0) FROM "+playlistItemsTable+" WHERE playlist_id = ?)", playlistID.String()),
			time.Now().UTC(),
		).
		Suffix("ON CONFLICT(playlist_id, download_id) DO NOTHING").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	return dbexec.ExecContext(ctx, r.dbEntry, sqlQuery, args, r.retryOptions)
}

// UpdatePositions rewrites the order of playlist items: position = index in downloadIDs.
// Элементы, которых нет в downloadIDs (например, ещё не загруженные на клиенте),
// идут следом в прежнем относительном порядке — так нумерация остаётся сквозной.
func (r *PlaylistRepository) UpdatePositions(ctx context.Context, playlistID uuid.UUID, downloadIDs []uuid.UUID) error {
	if len(downloadIDs) == 0 {
		return nil
	}

	return dbexec.Tx(ctx, r.dbEntry, func(ctx context.Context) error {
		finalOrder, err := r.mergeOrder(ctx, playlistID, downloadIDs)
		if err != nil {
			return err
		}

		for position, downloadID := range finalOrder {
			sqlQuery, args, err := squirrel.
				Update(playlistItemsTable).
				Set("position", position).
				Where(squirrel.Eq{
					"playlist_id": playlistID.String(),
					"download_id": downloadID.String(),
				}).
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if err != nil {
				return fmt.Errorf("failed to build query: %w", err)
			}

			if err := dbexec.ExecContext(ctx, r.dbEntry, sqlQuery, args, r.retryOptions); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *PlaylistRepository) RemoveItem(ctx context.Context, playlistID, downloadID uuid.UUID) error {
	sqlQuery, args, err := squirrel.
		Delete(playlistItemsTable).
		Where(squirrel.Eq{
			"playlist_id": playlistID.String(),
			"download_id": downloadID.String(),
		}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	return dbexec.ExecContext(ctx, r.dbEntry, sqlQuery, args, r.retryOptions)
}

func (r *PlaylistRepository) ItemExists(ctx context.Context, playlistID, downloadID uuid.UUID) (bool, error) {
	sqlQuery, args, err := squirrel.
		Select("1").
		From(playlistItemsTable).
		Where(squirrel.Eq{
			"playlist_id": playlistID.String(),
			"download_id": downloadID.String(),
		}).
		PlaceholderFormat(squirrel.Dollar).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build query: %w", err)
	}

	db := dbexec.Resolve(ctx, r.dbEntry)

	var exists int
	err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *PlaylistRepository) ListPlaylistIDsByDownload(
	ctx context.Context,
	downloadID uuid.UUID,
	userID *uuid.UUID,
) ([]uuid.UUID, error) {
	condition := squirrel.And{
		squirrel.Eq{"pi.download_id": downloadID.String()},
		squirrel.Eq{"p.deleted_at": nil},
	}
	if userID != nil && *userID != uuid.Nil() {
		condition = append(condition, squirrel.Or{
			squirrel.Eq{"p.user_id": nil},
			squirrel.Eq{"p.user_id": userID.String()},
		})
	} else {
		condition = append(condition, squirrel.Eq{"p.user_id": nil})
	}

	sqlQuery, args, err := squirrel.
		Select("pi.playlist_id").
		From(playlistItemsTable + " AS pi").
		Join(playlistsTable + " AS p ON p.playlist_id = pi.playlist_id").
		Where(condition).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	db := dbexec.Resolve(ctx, r.dbEntry)
	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlistIDs []uuid.UUID

	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			return nil, err
		}

		playlistID, err := uuid.Parse(rawID)
		if err != nil {
			return nil, err
		}

		playlistIDs = append(playlistIDs, playlistID)
	}

	return playlistIDs, rows.Err()
}

func (r *PlaylistRepository) CountItems(ctx context.Context, playlistID uuid.UUID) (int64, error) {
	sqlQuery, args, err := squirrel.
		Select("COUNT(*)").
		From(playlistItemsTable + " AS pi").
		Join("media_downloads AS f ON f.download_id = pi.download_id AND f.deleted_at IS NULL").
		Where(squirrel.Eq{"pi.playlist_id": playlistID.String()}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	db := dbexec.Resolve(ctx, r.dbEntry)

	var count int64
	if err := db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

type playlistScanner interface {
	Scan(dest ...any) error
}

func scanPlaylist(row playlistScanner) (*ddownload.Playlist, error) {
	var (
		playlist       ddownload.Playlist
		rawPlaylistID  string
		rawUserID      *string
		description    *string
		deletedAt      *time.Time
		firstMediaInfo *string
	)

	err := row.Scan(
		&rawPlaylistID,
		&rawUserID,
		&playlist.Title,
		&description,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
		&deletedAt,
		&playlist.ItemsCount,
		&firstMediaInfo,
	)
	if err != nil {
		return nil, err
	}

	if firstMediaInfo != nil && *firstMediaInfo != "" {
		var mediaInfo dtypes.MediaInfo
		if err := json.Unmarshal([]byte(*firstMediaInfo), &mediaInfo); err == nil {
			if thumbID := mediaInfo.PreferredThumbnailID(); thumbID != uuid.Nil() {
				playlist.ThumbnailID = &thumbID
			}
		}
	}

	playlistID, err := uuid.Parse(rawPlaylistID)
	if err != nil {
		return nil, err
	}
	playlist.PlaylistID = playlistID

	if rawUserID != nil && *rawUserID != "" {
		userID, err := uuid.Parse(*rawUserID)
		if err != nil {
			return nil, err
		}
		playlist.UserID = &userID
	}

	playlist.Description = description
	playlist.DeletedAt = deletedAt

	return &playlist, nil
}
