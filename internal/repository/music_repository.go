package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MusicRepository interface {
	CreateSong(ctx context.Context, song *Song) error
	GetSong(ctx context.Context, id string) (*Song, error)
	ListSongs(ctx context.Context) ([]Song, error)
	ListAdminSongs(ctx context.Context, limit, offset int) ([]Song, error)
	RecordPlay(ctx context.Context, songID, userID string, payload map[string]interface{}) error
	ListPlayCounts(ctx context.Context) (map[string]int, error)
	GetAnalytics(ctx context.Context) (*MusicAnalytics, error)
	ListLikedSongs(ctx context.Context, userID string) ([]LikedSong, error)
	LikeSong(ctx context.Context, userID, songID string, song LikedSong) error
	UnlikeSong(ctx context.Context, userID, songID string) error

	GetPlayerState(ctx context.Context, userID string) (*PlayerState, error)
}

type Song struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	Album       string    `json:"album"`
	Genre       string    `json:"genre"`
	ReleaseYear string    `json:"release_year"`
	ProviderID  string    `json:"provider_id"`
	UploadedAt  time.Time `json:"uploaded_at"`
	PlayCount   int       `json:"plays"`
	LikeCount   int       `json:"likes"`
	Duration    string    `json:"duration,omitempty"`
}

type MusicAnalytics struct {
	TotalSongs    int            `json:"total_songs"`
	TotalPlays    int            `json:"total_plays"`
	TopSongs      []Song         `json:"top_songs"`
	PlaysByGenre  map[string]int `json:"plays_by_genre"`
	RecentUploads []Song         `json:"recent_uploads"`
}

type LikedSong struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Artwork  string `json:"artwork"`
}



type PlayerState struct {
	CurrentSong *Song  `json:"current_song"`
	IsPlaying   bool   `json:"is_playing"`
	Position    int    `json:"position"`
	Queue       []Song `json:"queue"`
}

type PostgresMusicRepository struct {
	pool *pgxpool.Pool
}

func NewMusicRepository(pool *pgxpool.Pool) MusicRepository {
	return &PostgresMusicRepository{pool: pool}
}

func (r *PostgresMusicRepository) CreateSong(ctx context.Context, song *Song) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO provider_songs (provider_id, title, artist, album, genre, release_year)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, uploaded_at`,
		song.ProviderID, song.Title, song.Artist, song.Album, song.Genre, song.ReleaseYear).Scan(&song.ID, &song.UploadedAt)
	return err
}

func (r *PostgresMusicRepository) GetSong(ctx context.Context, id string) (*Song, error) {
	var song Song
	var playCount, likeCount int
	err := r.pool.QueryRow(ctx, `
		SELECT ps.id::text, ps.title, ps.artist, COALESCE(ps.album, ''), COALESCE(ps.genre, ''),
			COALESCE(ps.release_year, ''), ps.provider_id::text, ps.uploaded_at,
			COALESCE(spc.play_count, 0)::int,
			COALESCE(l.like_count, 0)::int
		FROM provider_songs ps
		LEFT JOIN song_play_counts spc ON spc.song_id = ps.id::text
		LEFT JOIN (
			SELECT song_id, COUNT(*) like_count FROM liked_songs GROUP BY song_id
		) l ON l.song_id = ps.id::text
		WHERE ps.id = $1`, id).Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.ReleaseYear, &song.ProviderID, &song.UploadedAt, &playCount, &likeCount)
	if err != nil {
		return nil, err
	}
	song.PlayCount = playCount
	song.LikeCount = likeCount
	return &song, nil
}

func (r *PostgresMusicRepository) ListSongs(ctx context.Context) ([]Song, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ps.id::text, ps.title, ps.artist, COALESCE(ps.album, ''), COALESCE(ps.genre, ''),
			COALESCE(ps.release_year, ''), ps.provider_id::text, ps.uploaded_at,
			COALESCE(spc.play_count, 0)::int,
			COALESCE(l.like_count, 0)::int
		FROM provider_songs ps
		LEFT JOIN song_play_counts spc ON spc.song_id = ps.id::text
		LEFT JOIN (
			SELECT song_id, COUNT(*) like_count FROM liked_songs GROUP BY song_id
		) l ON l.song_id = ps.id::text
		ORDER BY ps.uploaded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.ReleaseYear, &song.ProviderID, &song.UploadedAt, &song.PlayCount, &song.LikeCount); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

func (r *PostgresMusicRepository) RecordPlay(ctx context.Context, songID, userID string, payload map[string]interface{}) error {
	title, _ := payload["title"].(string)
	artist, _ := payload["artist"].(string)
	album, _ := payload["album"].(string)
	duration, _ := payload["duration"].(string)
	artwork, _ := payload["artwork"].(string)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO song_play_counts (song_id, play_count, title, artist, album, duration, artwork, updated_at)
		VALUES ($1, 1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (song_id)
		DO UPDATE SET 
			play_count = song_play_counts.play_count + 1, 
			updated_at = NOW(),
			title = COALESCE(EXCLUDED.title, song_play_counts.title),
			artist = COALESCE(EXCLUDED.artist, song_play_counts.artist),
			album = COALESCE(EXCLUDED.album, song_play_counts.album),
			duration = COALESCE(EXCLUDED.duration, song_play_counts.duration),
			artwork = COALESCE(EXCLUDED.artwork, song_play_counts.artwork)`,
		songID, title, artist, album, duration, artwork)

	if err != nil {
		return err
	}

	// Also record the play event
	_, err = r.pool.Exec(ctx, `
		INSERT INTO user_activity_events (user_id, event_type, resource_type, resource_id, metadata)
		VALUES ($1, 'song_played', 'song', $2, '{"timestamp": "'||NOW()||'"}')`,
		userID, songID)
	return err
}

func (r *PostgresMusicRepository) ListPlayCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT song_id, play_count FROM song_play_counts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var songID string
		var count int
		if err := rows.Scan(&songID, &count); err != nil {
			return nil, err
		}
		counts[songID] = count
	}
	return counts, rows.Err()
}

func (r *PostgresMusicRepository) GetAnalytics(ctx context.Context) (*MusicAnalytics, error) {
	analytics := &MusicAnalytics{
		PlaysByGenre: make(map[string]int),
	}

	// Get total songs and plays
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int, COALESCE(SUM(play_count), 0)::int
		FROM provider_songs ps
		LEFT JOIN song_play_counts spc ON spc.song_id = ps.id::text`).Scan(&analytics.TotalSongs, &analytics.TotalPlays)
	if err != nil {
		return nil, err
	}

	// Get top songs
	rows, err := r.pool.Query(ctx, `
		SELECT ps.id::text, ps.title, ps.artist, COALESCE(ps.album, ''), COALESCE(ps.genre, ''),
			COALESCE(ps.release_year, ''), ps.provider_id::text, ps.uploaded_at,
			COALESCE(spc.play_count, 0)::int,
			COALESCE(l.like_count, 0)::int
		FROM provider_songs ps
		LEFT JOIN song_play_counts spc ON spc.song_id = ps.id::text
		LEFT JOIN (
			SELECT song_id, COUNT(*) like_count FROM liked_songs GROUP BY song_id
		) l ON l.song_id = ps.id::text
		ORDER BY COALESCE(spc.play_count, 0) DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var song Song
		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.ReleaseYear, &song.ProviderID, &song.UploadedAt, &song.PlayCount, &song.LikeCount); err != nil {
			return nil, err
		}
		analytics.TopSongs = append(analytics.TopSongs, song)
	}

	// Get plays by genre
	genreRows, err := r.pool.Query(ctx, `
		SELECT COALESCE(ps.genre, 'Unknown'), SUM(COALESCE(spc.play_count, 0))::int
		FROM provider_songs ps
		LEFT JOIN song_play_counts spc ON spc.song_id = ps.id::text
		GROUP BY ps.genre`)
	if err != nil {
		return nil, err
	}
	defer genreRows.Close()

	for genreRows.Next() {
		var genre string
		var plays int
		if err := genreRows.Scan(&genre, &plays); err != nil {
			return nil, err
		}
		analytics.PlaysByGenre[genre] = plays
	}

	// Get recent uploads
	recentRows, err := r.pool.Query(ctx, `
		SELECT id::text, title, artist, COALESCE(album, ''), COALESCE(genre, ''),
			COALESCE(release_year, ''), provider_id::text, uploaded_at
		FROM provider_songs
		ORDER BY uploaded_at DESC
		LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer recentRows.Close()

	for recentRows.Next() {
		var song Song
		if err := recentRows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.ReleaseYear, &song.ProviderID, &song.UploadedAt); err != nil {
			return nil, err
		}
		analytics.RecentUploads = append(analytics.RecentUploads, song)
	}

	return analytics, nil
}

func (r *PostgresMusicRepository) ListLikedSongs(ctx context.Context, userID string) ([]LikedSong, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT song_id, COALESCE(title, ''), COALESCE(artist, ''), COALESCE(album, ''), 
			COALESCE(duration, ''), COALESCE(artwork, '')
		FROM liked_songs
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []LikedSong
	for rows.Next() {
		var song LikedSong
		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration, &song.Artwork); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

func (r *PostgresMusicRepository) LikeSong(ctx context.Context, userID, songID string, song LikedSong) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO liked_songs (user_id, song_id, title, artist, album, duration, artwork, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id, song_id) DO UPDATE SET 
			title = EXCLUDED.title,
			artist = EXCLUDED.artist,
			album = EXCLUDED.album,
			duration = EXCLUDED.duration,
			artwork = EXCLUDED.artwork`, 
			userID, songID, song.Title, song.Artist, song.Album, song.Duration, song.Artwork)
	return err
}

func (r *PostgresMusicRepository) UnlikeSong(ctx context.Context, userID, songID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM liked_songs WHERE user_id = $1 AND song_id = $2`, userID, songID)
	return err
}



func (r *PostgresMusicRepository) GetPlayerState(ctx context.Context, userID string) (*PlayerState, error) {
	// For now, return a default state. In a real implementation, this would be stored in the database
	return &PlayerState{
		CurrentSong: nil,
		IsPlaying:   false,
		Position:    0,
		Queue:       []Song{},
	}, nil
}


func (r *PostgresMusicRepository) ListAdminSongs(ctx context.Context, limit, offset int) ([]Song, error) {
	rows, err := r.pool.Query(ctx, `
		WITH all_songs AS (
			-- Provider songs
			SELECT id::text AS song_id, title, artist, COALESCE(album, '') as album, COALESCE(genre, '') as genre, 
				COALESCE(release_year::text, '') as release_year, provider_id::text, uploaded_at, COALESCE(duration, '0') as duration
			FROM provider_songs
			
			UNION ALL
			
			-- Liked songs (might be catalog)
			SELECT song_id::text, COALESCE(title, '') as title, COALESCE(artist, '') as artist, COALESCE(album, ''), '' as genre, '' as release_year, CAST(NULL AS text) as provider_id, created_at, COALESCE(duration, '0')
			FROM liked_songs
			
			UNION ALL
			
			-- Played songs (might be catalog)
			SELECT song_id::text, COALESCE(title, ''), COALESCE(artist, ''), COALESCE(album, ''), '' as genre, '' as release_year, CAST(NULL AS text) as provider_id, updated_at, COALESCE(duration, '0')
			FROM song_play_counts
			WHERE title IS NOT NULL
		),
		unique_songs AS (
			SELECT DISTINCT ON (song_id) *
			FROM all_songs
			ORDER BY song_id, uploaded_at DESC
		)
		SELECT 
			u.song_id, u.title, u.artist, u.album, u.genre, u.release_year, COALESCE(u.provider_id, ''), u.uploaded_at,
			COALESCE(spc.play_count, 0)::int as plays,
			COALESCE(l.like_count, 0)::int as likes,
			COALESCE(u.duration, '0')
		FROM unique_songs u
		LEFT JOIN song_play_counts spc ON spc.song_id::text = u.song_id
		LEFT JOIN (
			SELECT song_id::text as song_id, COUNT(*) as like_count FROM liked_songs GROUP BY song_id
		) l ON l.song_id = u.song_id
		ORDER BY plays DESC, likes DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.ReleaseYear, &song.ProviderID, &song.UploadedAt, &song.PlayCount, &song.LikeCount, &song.Duration); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}
