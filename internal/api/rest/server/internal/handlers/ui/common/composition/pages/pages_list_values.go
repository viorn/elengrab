package pages

import "html/template"

type PagesListValues struct {
	RowsFragmentValues

	HeaderActionsSearchButtonIcon   template.HTML
	HeaderActionsDownloadButtonIcon template.HTML
	HeaderActionsAvatarTitle        string
	HeaderActionsPlaylistsIcon      template.HTML

	// DownloaderPlaylistsPath is the link target of the playlists header button.
	DownloaderPlaylistsPath string

	UserAvatar UserAvatar

	ShowHistorySearch   bool
	SearchBackArrowIcon template.HTML

	SearchQuery string

	UploadMenuContent template.HTML

	ActiveViewMode string
	ViewModeTabs   []ViewModeTab
	// HideViewModeTabs скрывает сортировку new/popular/old: на странице плейлиста
	// порядок задаётся вручную и сортировки по дате/просмотрам там нет.
	HideViewModeTabs bool

	HasCreateAccess         bool
	HasWriteOperationAccess bool

	VideoPreview VideoPreview

	AboutDialog      AboutDialogValues
	UploadFileDialog UploadFileDialogValues
	PlaylistDialog   PlaylistDialogValues
}

type UserAvatar struct {
	Icon       template.HTML
	ActionMode string
}

type VideoPreview struct {
	SoundOnIcon  template.HTML
	SoundOffIcon template.HTML
	PlayIcon     template.HTML
	PauseIcon    template.HTML
}
