package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neosy/elengrab/internal/app/services/internal/utils"
	"github.com/neosy/elengrab/internal/app/services/ytdlp/internal/consts"
	idto "github.com/neosy/elengrab/internal/app/services/ytdlp/internal/downloader/dto"
)

// playlistJSON is the subset of `yt-dlp -J --flat-playlist` output we need.
type playlistJSON struct {
	Type          string          `json:"_type"`
	Title         string          `json:"title"`
	PlaylistCount int             `json:"playlist_count"`
	Entries       []playlistEntry `json:"entries"`
}

type playlistEntry struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	Duration *float64 `json:"duration"`
	WebURL   string   `json:"webpage_url"`
}

// FetchPlaylistInfo returns the flat list of playlist entries without downloading
// any media. limit <= 0 means "all entries".
func (e *Executor) FetchPlaylistInfo(
	ctx context.Context,
	url string,
	limit int,
	opts ...idto.ExecutorOption,
) (*idto.PlaylistInfo, error) {
	args := []string{"--flat-playlist", "--no-warnings", "--quiet"}

	if limit > 0 {
		args = append(args, "--playlist-end", fmt.Sprintf("%d", limit))
	}

	args = append(args, "--dump-single-json")

	args = addCookiesToArgs(e.logger, args, opts...)

	args = append(args, url)

	out, err := utils.ExecCommandContext(ctx, e.ytDlpPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s command: %w", consts.YtDlpName, err)
	}

	outStr := string(out)

	start := strings.Index(outStr, "{")
	if start == -1 {
		return nil, fmt.Errorf("no JSON found in %s output", consts.YtDlpName)
	}

	var raw playlistJSON
	if err := json.NewDecoder(bytes.NewReader([]byte(outStr[start:]))).Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid JSON output from %s: %w", consts.YtDlpName, err)
	}

	info := &idto.PlaylistInfo{
		Title: strings.TrimSpace(raw.Title),
		Total: raw.PlaylistCount,
	}

	for index, entry := range raw.Entries {
		mediaURL := strings.TrimSpace(entry.WebURL)
		if mediaURL == "" {
			mediaURL = strings.TrimSpace(entry.URL)
		}
		if mediaURL == "" && entry.ID != "" {
			mediaURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", entry.ID)
		}
		if mediaURL == "" {
			continue
		}

		item := idto.PlaylistEntry{
			Index:    index + 1,
			MediaURL: mediaURL,
			Title:    strings.TrimSpace(entry.Title),
		}

		if entry.Duration != nil && *entry.Duration > 0 {
			item.Duration = int64(*entry.Duration)
		}

		info.Entries = append(info.Entries, item)
	}

	if info.Total == 0 {
		info.Total = len(info.Entries)
	}

	return info, nil
}
