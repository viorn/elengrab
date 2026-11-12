package download

import (
	"uuid"

	dtypes "github.com/neosy/elengrab/internal/domain/types"
	"github.com/neosy/elengrab/internal/pkg/dbutils"
	"github.com/neosy/elengrab/internal/ports/persistence"
	edownload "github.com/neosy/elengrab/internal/repository/sqlite/download/entity"
	"github.com/neosy/elengrab/internal/repository/sqlite/types"
)

type mediaDownloadQueryOptions struct {
	types.QueryMediaOptions

	includeDeleted bool
	statuses       []dtypes.MediaDownloadStatus
	downloadIDs    []uuid.UUID

	partialHash **string

	// playlistID limits the result to media belonging to the playlist.
	playlistID *uuid.UUID

	// searchText performs a case-insensitive search over title and description.
	searchText *string

	// mediaURLPatterns limits the result to source URLs matching these LIKE patterns.
	mediaURLPatterns []string
}

type queryArgs struct {
	Placeholder string
	Values      []any
}

func newMediaDownloadQueryOptions() mediaDownloadQueryOptions {
	return mediaDownloadQueryOptions{
		QueryMediaOptions: types.NewQueryMediaOptions(),
	}
}

func (r *MediaDownloadRepository) WithOptions(options dtypes.QueryMediaOptions) persistence.MediaDownloadRepository {
	r.queryOptions.Offset = options.Offset
	r.queryOptions.Limit = options.Limit

	r.queryOptions.ViewMode = options.ViewMode
	r.queryOptions.Visibility = options.Visibility
	r.queryOptions.IsGuestRequest = options.IsGuestRequest

	if len(options.OrderBys) > 0 {
		r.WithOrderBy(options.OrderBys...)
	}

	if options.Filters != nil && options.Filters.Len() > 0 {
		r.WithFilters(options.Filters.List()...)
	}

	if options.PlaylistID != nil && *options.PlaylistID != uuid.Nil() {
		playlistID := *options.PlaylistID
		r.queryOptions.playlistID = &playlistID
	}

	return r
}

func (r *MediaDownloadRepository) WithStatus(statuses ...dtypes.MediaDownloadStatus) persistence.MediaDownloadRepository {
	if len(statuses) == 0 {
		return r
	}

	r.queryOptions.statuses = statuses

	return r
}

func (r *MediaDownloadRepository) WithDeleted() persistence.MediaDownloadRepository {
	r.queryOptions.includeDeleted = true
	return r
}

func (r *MediaDownloadRepository) WithUser(userID uuid.UUID) persistence.MediaDownloadRepository {
	var eDownload edownload.MediaDownload

	if userID == uuid.Nil() {
		r.queryOptions.Visibility = new(dtypes.QueryMediaVisibilityPublic)
	} else {
		r.queryOptions.Filters.Append(dbutils.NewFieldFilterEq(eDownload.FieldName(&eDownload.UserID), userID))
	}

	return r
}

func (r *MediaDownloadRepository) WithFilters(filters ...dtypes.QueryFilter) persistence.MediaDownloadRepository {
	var (
		eDownload edownload.MediaDownload

		fieldNameByAllowedFilter = map[dtypes.QueryFilterName]string{
			dtypes.QueryFilterNameUserID:    eDownload.FieldName(&eDownload.UserID),
			dtypes.QueryFilterNameChannelID: eDownload.FieldName(&eDownload.ChannelID),
		}
	)

	for _, filter := range filters {
		fieldName, exists := fieldNameByAllowedFilter[filter.Name]
		if exists {
			switch fieldName {
			case eDownload.FieldName(&eDownload.UserID):
				id, ok := filter.Value().(uuid.UUID)
				if !ok {
					continue
				}
				if id == uuid.Nil() {
					r.queryOptions.Visibility = new(dtypes.QueryMediaVisibilityPublic)
					continue
				}
				r.queryOptions.Filters.Append(dbutils.NewFieldFilterEq(fieldName, filter.Value()))
			default:
				r.queryOptions.Filters.Append(dbutils.NewFieldFilterWithCondition(fieldName, filter.Condition()))
			}
			continue
		}

		switch filter.Name {
		case dtypes.QueryFilterNamePlaylistID:
			if playlistID, ok := filter.Value().(uuid.UUID); ok && playlistID != uuid.Nil() {
				r.queryOptions.playlistID = &playlistID
			}
		case dtypes.QueryFilterNameSearchQuery:
			if text, ok := filter.Value().(string); ok && text != "" {
				r.queryOptions.searchText = &text
			}
		case dtypes.QueryFilterNameDownloadIDs:
			ids, ok := filter.Value().([]uuid.UUID)
			if ok {
				r.queryOptions.downloadIDs = ids
			}
		}
	}

	return r
}

func (r *MediaDownloadRepository) WithOrderBy(list ...dtypes.QueryOrderBy) persistence.MediaDownloadRepository {
	var (
		eDownload edownload.MediaDownload

		fieldNameByAllowedFilter = eDownload.PaginationFieldNames()
	)

	for _, orderBy := range list {
		fieldName, exists := fieldNameByAllowedFilter[orderBy.Field]
		if !exists {
			continue
		}

		r.queryOptions.OrderBys.Add(fieldName, orderBy.Direction)
	}

	return r
}
