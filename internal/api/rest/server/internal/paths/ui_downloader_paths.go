package httppaths

import (
	"fmt"
	"strings"

	"uuid"

	dtypes "github.com/neosy/elengrab/internal/domain/types"
	"github.com/neosy/elengrab/internal/pkg/idcodec"
)

// UI
const (
	// Groups
	DownloaderGroup = "/downloader"

	// Paths Downloader
	AccountMenuPath  = "/account-menu"
	SettingsMenuPath = "/settings-menu"
	GrabPath         = "/grab"
	ShareTargetPath  = "/share-target"
	DownloadFilePath = "/download"
	SearchPath       = "/search"
	EventsPath       = "/events"
	MediaItemsPath   = "/items"
	ChannelsPath     = "/channels"
	PlaylistsPath    = "/playlists"
	UploadPath       = "/upload"

	// Paths items Downloader
	MediaItemPath                = MediaItemsPath + "/{itemId}"
	MediaItemRowPath             = MediaItemsPath + "/{itemId}/row"
	MediaItemDownloadRepeatPath  = MediaItemsPath + "/{itemId}/repeat"
	MediaItemImagePath           = MediaItemsPath + "/{itemId}/image"
	MediaItemMenuPath            = MediaItemsPath + "/{itemId}/menu"
	MediaItemShortLinkPath       = MediaItemsPath + "/{itemId}/short-link"
	MediaItemStreamPath          = MediaItemsPath + "/{itemId}/stream"
	MediaItemEditPath            = MediaItemsPath + "/{itemId}/edit"
	MediaItemRefreshPath         = MediaItemsPath + "/{itemId}/refresh"
	MediaItemReWatchTrackingPath = MediaItemsPath + "/{itemId}/watch-tracking"
	MediaItemWatchPositionPath   = MediaItemsPath + "/{itemId}/watch-position"
	MediaItemPlaylistsPath       = MediaItemsPath + "/{itemId}/playlists"

	// Paths Uploader
	UploadFilePath = UploadPath + "/{fileId}"

	// Paths channels
	ChannelPath = ChannelsPath + "/{channelId}"

	// Paths playlists
	PlaylistPath      = PlaylistsPath + "/{playlistId}"
	PlaylistItemsPath = PlaylistPath + "/items/{itemId}"

	// Paths users
	UserLoginPath     = UsersPath + "/{login}"
	UserDownloadsPath = UserLoginPath + "/downloads"

	// Downloader Paths
	DownloaderAccountMenuPath   = DownloaderGroup + AccountMenuPath
	DownloaderSettingsMenuPath  = DownloaderGroup + SettingsMenuPath
	DownloaderGrabPath          = DownloaderGroup + GrabPath
	DownloaderShareTargetPath   = DownloaderGroup + ShareTargetPath
	DownloaderItemsPath         = DownloaderGroup + MediaItemsPath
	DownloaderDownloadFilePath  = DownloaderGroup + DownloadFilePath
	DownloaderSearchPath        = DownloaderGroup + SearchPath
	DownloaderEventsPath        = DownloaderGroup + EventsPath
	DownloaderChannelsPath      = DownloaderGroup + ChannelsPath
	DownloaderPlaylistsPath     = DownloaderGroup + PlaylistsPath
	DownloaderPlaylistPath      = DownloaderGroup + PlaylistPath
	DownloaderPlaylistItemPath  = DownloaderGroup + PlaylistItemsPath
	DownloaderItemPlaylistsPath = DownloaderGroup + MediaItemPlaylistsPath
	DownloaderUsersPath         = DownloaderGroup + ChannelPath
	DownloaderUploadPath        = DownloaderGroup + UploadPath
	DownloaderUploadFilePath    = DownloaderGroup + UploadFilePath
)

func buildMediaItemPath(path string, downloadID uuid.UUID) string {
	id := idcodec.EncodeUUIDBase64URL(downloadID)
	return DownloaderGroup + strings.Replace(path, "{itemId}", id, 1)
}

func BuildMediaItemPath(downloadID uuid.UUID) string {
	return buildMediaItemPath(MediaItemPath, downloadID)
}

func BuildMediaItemRowPath(downloadID uuid.UUID) string {
	return buildMediaItemPath(MediaItemRowPath, downloadID)
}

func BuildMediaItemDownloadRepeatPath(downloadID uuid.UUID) string {
	return buildMediaItemPath(MediaItemDownloadRepeatPath, downloadID)
}

func BuildMediaItemStreamPath(downloadID uuid.UUID) string {
	return buildMediaItemPath(MediaItemStreamPath, downloadID)
}

func BuildMediaItemEditPath(downloadID uuid.UUID) string {
	return buildMediaItemPath(MediaItemEditPath, downloadID)
}

func BuildMediaItemPlaylistsPath(downloadID uuid.UUID) string {
	return buildMediaItemPath(MediaItemPlaylistsPath, downloadID)
}

func BuildMediaItemDownloadPath(downloadID uuid.UUID) string {
	id := idcodec.EncodeUUIDBase64URL(downloadID)
	return fmt.Sprintf("%s?itemId=%s", DownloaderGroup+DownloadFilePath, id)
}

func BuildMediaItemImagePath(downloadID uuid.UUID, verHash string, sources []dtypes.ImageSource) string {
	urlSufix := buildImageSufix(verHash, sources)
	return buildMediaItemPath(MediaItemImagePath, downloadID) + urlSufix
}

func buildUploadFilePath(path string, fileID uuid.UUID) string {
	id := idcodec.EncodeUUIDBase64URL(fileID)
	return DownloaderGroup + strings.Replace(path, "{fileId}", id, 1)
}

func BuildUploadFilePath(path string, fileID uuid.UUID) string {
	return buildUploadFilePath(UploadFilePath, fileID)
}

func buildChannelPath(path string, channelID uuid.UUID) string {
	id := idcodec.EncodeUUIDBase64URL(channelID)
	return DownloaderGroup + strings.Replace(path, "{channelId}", id, 1)
}

func BuildChannelPath(channelID uuid.UUID) string {
	return buildChannelPath(ChannelPath, channelID)
}

func buildPlaylistPath(path string, playlistID uuid.UUID) string {
	id := idcodec.EncodeUUIDBase64URL(playlistID)
	return DownloaderGroup + strings.Replace(path, "{playlistId}", id, 1)
}

func BuildPlaylistPath(playlistID uuid.UUID) string {
	return buildPlaylistPath(PlaylistPath, playlistID)
}

func BuildPlaylistItemPath(playlistID, downloadID uuid.UUID) string {
	path := strings.Replace(PlaylistItemsPath, "{playlistId}", idcodec.EncodeUUIDBase64URL(playlistID), 1)
	path = strings.Replace(path, "{itemId}", idcodec.EncodeUUIDBase64URL(downloadID), 1)
	return DownloaderGroup + path
}

func buildUserLoginPath(path string, login string) string {
	return DownloaderGroup + strings.Replace(path, "{login}", login, 1)
}

func BuildUserDownloadsPath(login string) string {
	return buildUserLoginPath(UserDownloadsPath, login)
}
