package idto

// PlaylistEntry is a single item of an external playlist.
type PlaylistEntry struct {
	Index    int
	MediaURL string
	Title    string
	Duration int64 // seconds, 0 if unknown
}

// PlaylistInfo is the metadata of an external playlist.
type PlaylistInfo struct {
	Title   string
	Total   int
	Entries []PlaylistEntry
}
