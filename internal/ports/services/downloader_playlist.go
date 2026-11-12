package pservices

import "time"

// PlaylistEntry is a single media item of an external playlist.
type PlaylistEntry struct {
	Index    int
	MediaURL string
	Title    string
	Duration time.Duration
}

// PlaylistInfo is the metadata of an external playlist.
type PlaylistInfo struct {
	Title   string
	Total   int
	Entries []PlaylistEntry
}
