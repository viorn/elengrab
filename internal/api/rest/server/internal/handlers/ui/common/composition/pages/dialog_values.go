package pages

import "html/template"

type AboutDialogValues struct {
	LogoIcon   template.HTML
	DonateIcon template.HTML

	VersionLinkEnabled bool

	ShowDeveloperInAbout    bool
	ShowDonationLinkInAbout bool

	Version       string
	GoVersion     string
	YtDlpVersion  string
	FFmpegVersion string
	DenoVersion   string
}

type PlaylistDialogValues struct {
	LogoIcon template.HTML

	DownloadID string

	Playlists    []PlaylistDialogItem
	HasPlaylists bool
	CanCreate    bool

	ToggleURL string
	CreateURL string
}

type PlaylistDialogItem struct {
	EncodedPlaylistID string
	Title             string
	ItemsCount        int64
	Checked           bool
	CanEdit           bool
}

type UploadFileDialogValues struct {
	LogoIcon template.HTML

	VisibilityList []VisibilityOption
	Visibility     string
}
