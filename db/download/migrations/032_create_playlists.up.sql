CREATE TABLE IF NOT EXISTS playlists (
    -- Unique playlist identifier (UUID)
    playlist_id TEXT PRIMARY KEY,

    -- Owner identifier (UUID). NULL means a shared playlist
    user_id TEXT NULL,

    -- Playlist title
    title TEXT NOT NULL,

    -- Optional description
    description TEXT NULL,

    -- Record creation timestamp, set automatically
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Record update timestamp, set automatically
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Record delete timestamp
    deleted_at DATETIME NULL
);

CREATE INDEX IF NOT EXISTS playlists_user_id_idx
ON playlists(user_id);

CREATE INDEX IF NOT EXISTS playlists_deleted_at_null_idx
ON playlists(deleted_at)
WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS playlist_items (
    -- Playlist identifier
    playlist_id TEXT NOT NULL,

    -- Media download identifier
    download_id TEXT NOT NULL,

    -- Sort position inside the playlist
    position INTEGER NOT NULL DEFAULT 0,

    -- Timestamp when the item was added
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (playlist_id, download_id)
);

CREATE INDEX IF NOT EXISTS playlist_items_download_id_idx
ON playlist_items(download_id);
