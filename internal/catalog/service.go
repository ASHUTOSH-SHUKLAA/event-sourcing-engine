package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gin-quickstart/internal/config"
)

const (
	defaultBannerImage = "https://images.unsplash.com/photo-1511379938547-c1f69419868d?auto=format&fit=crop&w=900&q=80"
	defaultTrackQuery  = "love"
	defaultUserAgent   = "EventSound/1.0 (event-sourcing-demo)"
)

type Service interface {
	GetSongs(ctx context.Context) ([]Track, error)
	GetPlaylists(ctx context.Context) ([]Playlist, error)
	GetBanners(ctx context.Context) ([]Banner, error)
	SearchTracks(ctx context.Context, query string) ([]Track, error)
}

type service struct {
	client         *http.Client
	audioDBKey     string
	audioDBV1Base  string
	musicBrainzURL string
	deezerURL      string
}

type Banner struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Artwork string `json:"artwork"`
	Tag     string `json:"tag"`
}

type Playlist struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Curator    string `json:"curator"`
	Artwork    string `json:"artwork"`
	TrackCount int    `json:"track_count"`
}

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Artwork  string `json:"artwork"`
}

type deezerTracksResponse struct {
	Data []struct {
		ID       int    `json:"id"`
		Title    string `json:"title"`
		Duration int    `json:"duration"`
		Artist   struct {
			Name string `json:"name"`
		} `json:"artist"`
		Album struct {
			Title       string `json:"title"`
			CoverMedium string `json:"cover_medium"`
			CoverBig    string `json:"cover_big"`
		} `json:"album"`
	} `json:"data"`
}

type deezerPlaylistsResponse struct {
	Data []struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		Picture      string `json:"picture_medium"`
		PictureLarge string `json:"picture_big"`
		NBTracks     int    `json:"nb_tracks"`
		User         struct {
			Name string `json:"name"`
		} `json:"user"`
	} `json:"data"`
}

type musicBrainzResponse struct {
	Recordings []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Length int    `json:"length"`
		Artist []struct {
			Name string `json:"name"`
		} `json:"artist-credit"`
		Releases []struct {
			Title string `json:"title"`
		} `json:"releases"`
	} `json:"recordings"`
}

type audioDBTrackSearchResponse struct {
	Track []struct {
		StrTrackThumb string `json:"strTrackThumb"`
		StrAlbumThumb string `json:"strAlbumThumb"`
	} `json:"track"`
}

type audioDBAlbumSearchResponse struct {
	Album []struct {
		StrAlbumThumb string `json:"strAlbumThumb"`
	} `json:"album"`
}

func NewService() Service {
	return &service{
		client:         &http.Client{Timeout: 12 * time.Second},
		audioDBKey:     config.GetAudioDBAPIKey(),
		audioDBV1Base:  config.GetAudioDBV1URL(),
		musicBrainzURL: config.GetMusicBrainzAPIURL(),
		deezerURL:      config.GetDeezerAPIURL(),
	}
}

func (s *service) GetSongs(ctx context.Context) ([]Track, error) {
	allQueries := []string{
		"top hits", "lofi beats", "rock classics", "pop 2024", "jazz lounge",
		"electronic dance", "acoustic chill", "hip hop", "classical", "blues",
		"indie rock", "rnb soul", "metal", "reggae", "country", "funk",
		"latin pop", "kpop", "gospel", "ambient",
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(allQueries), func(i, j int) { allQueries[i], allQueries[j] = allQueries[j], allQueries[i] })

	// Pick 3 different genres to query in parallel
	selectedQueries := allQueries[:3]
	results := make(chan deezerTracksResponse, 3)

	for _, q := range selectedQueries {
		go func(query string) {
			u := strings.TrimRight(s.deezerURL, "/") + "/search?q=" + url.QueryEscape(query) + "&limit=25"
			var payload deezerTracksResponse
			_ = s.doJSONGet(ctx, u, nil, &payload)
			results <- payload
		}(q)
	}

	// Collect all tracks, deduplicating by ID
	seenIDs := map[int]bool{}
	items := make([]Track, 0, 75)
	for i := 0; i < 3; i++ {
		payload := <-results
		for _, song := range payload.Data {
			if seenIDs[song.ID] {
				continue
			}
			seenIDs[song.ID] = true
			items = append(items, Track{
				ID:       strconv.Itoa(song.ID),
				Title:    fallback(song.Title, "Unknown track"),
				Artist:   fallback(song.Artist.Name, "Unknown artist"),
				Album:    fallback(song.Album.Title, "Unknown album"),
				Duration: formatSeconds(song.Duration),
				Artwork:  firstImage(song.Album.CoverBig, song.Album.CoverMedium),
			})
		}
	}

	// Shuffle so order doesn't bias genre representation
	rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	if len(items) == 0 {
		return starterTracks(), nil
	}
	return items, nil
}

func (s *service) GetPlaylists(ctx context.Context) ([]Playlist, error) {
	u := strings.TrimRight(s.deezerURL, "/") + "/chart/0/playlists?limit=10"
	var payload deezerPlaylistsResponse
	if err := s.doJSONGet(ctx, u, nil, &payload); err != nil {
		return nil, err
	}

	items := make([]Playlist, 0, len(payload.Data))
	for _, playlist := range payload.Data {
		items = append(items, Playlist{
			ID:         strconv.Itoa(playlist.ID),
			Title:      fallback(playlist.Title, "Untitled playlist"),
			Curator:    fallback(playlist.User.Name, "Deezer"),
			Artwork:    firstImage(playlist.PictureLarge, playlist.Picture),
			TrackCount: playlist.NBTracks,
		})
	}
	return items, nil
}

func (s *service) GetBanners(ctx context.Context) ([]Banner, error) {
	songs, err := s.GetSongs(ctx)
	if err != nil {
		return nil, err
	}
	limit := 4
	if len(songs) < limit {
		limit = len(songs)
	}

	items := make([]Banner, 0, limit)
	for index := 0; index < limit; index++ {
		track := songs[index]
		items = append(items, Banner{
			ID:      track.ID,
			Title:   track.Title,
			Artist:  track.Artist,
			Artwork: track.Artwork,
			Tag:     "Trending",
		})
	}
	return items, nil
}

func (s *service) SearchTracks(ctx context.Context, query string) ([]Track, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = defaultTrackQuery
	}

	values := url.Values{}
	values.Set("query", q)
	values.Set("fmt", "json")
	values.Set("limit", "15")
	u := strings.TrimRight(s.musicBrainzURL, "/") + "/recording/?" + values.Encode()

	headers := map[string]string{"User-Agent": defaultUserAgent}
	var payload musicBrainzResponse
	if err := s.doJSONGet(ctx, u, headers, &payload); err != nil {
		return nil, err
	}

	items := make([]Track, 0, len(payload.Recordings))
	for _, recording := range payload.Recordings {
		artist := "Unknown artist"
		if len(recording.Artist) > 0 && strings.TrimSpace(recording.Artist[0].Name) != "" {
			artist = strings.TrimSpace(recording.Artist[0].Name)
		}
		album := "Unknown album"
		if len(recording.Releases) > 0 && strings.TrimSpace(recording.Releases[0].Title) != "" {
			album = strings.TrimSpace(recording.Releases[0].Title)
		}

		item := Track{
			ID:       fallback(recording.ID, fallback(recording.Title, "track")),
			Title:    fallback(recording.Title, "Unknown track"),
			Artist:   artist,
			Album:    album,
			Duration: formatMilliseconds(recording.Length),
			Artwork:  defaultBannerImage,
		}
		if artwork := s.fetchArtwork(ctx, item.Artist, item.Album, item.Title); artwork != "" {
			item.Artwork = artwork
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *service) fetchArtwork(ctx context.Context, artist, album, title string) string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return ""
	}

	trackValues := url.Values{}
	trackValues.Set("s", artist)
	trackValues.Set("t", strings.TrimSpace(title))
	trackURL := strings.TrimRight(s.audioDBV1Base, "/") + "/" + s.audioDBKey + "/searchtrack.php?" + trackValues.Encode()

	var trackResp audioDBTrackSearchResponse
	if err := s.doJSONGet(ctx, trackURL, nil, &trackResp); err == nil && len(trackResp.Track) > 0 {
		if artwork := firstImage(trackResp.Track[0].StrTrackThumb, trackResp.Track[0].StrAlbumThumb); artwork != defaultBannerImage {
			return artwork
		}
	}

	if strings.TrimSpace(album) == "" || strings.EqualFold(strings.TrimSpace(album), "unknown album") {
		return ""
	}
	albumValues := url.Values{}
	albumValues.Set("s", artist)
	albumValues.Set("a", strings.TrimSpace(album))
	albumURL := strings.TrimRight(s.audioDBV1Base, "/") + "/" + s.audioDBKey + "/searchalbum.php?" + albumValues.Encode()

	var albumResp audioDBAlbumSearchResponse
	if err := s.doJSONGet(ctx, albumURL, nil, &albumResp); err == nil && len(albumResp.Album) > 0 {
		art := strings.TrimSpace(albumResp.Album[0].StrAlbumThumb)
		if art != "" {
			return art
		}
	}
	return ""
}

func (s *service) doJSONGet(ctx context.Context, requestURL string, headers map[string]string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("external api error: %s", strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func firstImage(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return defaultBannerImage
}

func formatMilliseconds(ms int) string {
	if ms <= 0 {
		return "--:--"
	}
	seconds := ms / 1000
	return formatSeconds(seconds)
}

func formatSeconds(seconds int) string {
	if seconds <= 0 {
		return "--:--"
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func fallback(value, alt string) string {
	if strings.TrimSpace(value) == "" {
		return alt
	}
	return strings.TrimSpace(value)
}

func starterTracks() []Track {
	return []Track{
		{ID: "f1", Title: "Midnight Dreams", Artist: "Luna Echo", Album: "Neon Skyline", Duration: "3:42", Artwork: "https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=400"},
		{ID: "f2", Title: "Ocean Waves", Artist: "Blue Horizon", Album: "Coastal Sounds", Duration: "4:15", Artwork: "https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=400"},
		{ID: "f3", Title: "Summer Vibes", Artist: "Sunshine Band", Album: "Evening Glow", Duration: "3:58", Artwork: "https://images.unsplash.com/photo-1470229722913-7c0e2dbbafd3?w=400"},
		{ID: "f4", Title: "Neon Lights", Artist: "City Pulse", Album: "Night Drive", Duration: "5:03", Artwork: "https://images.unsplash.com/photo-1511379938547-c1f69419868d?w=400"},
		{ID: "f5", Title: "Electric Heart", Artist: "Volt", Album: "Circuitry", Duration: "3:20", Artwork: "https://images.unsplash.com/photo-1459749411177-042180ceea72?w=400"},
		{ID: "f6", Title: "Mountain High", Artist: "Peak", Album: "Summit", Duration: "4:45", Artwork: "https://images.unsplash.com/photo-1464375117522-1311d6a5b81f?w=400"},
		{ID: "f7", Title: "Desert Rose", Artist: "Sandstorm", Album: "Oasis", Duration: "3:55", Artwork: "https://images.unsplash.com/photo-1508700115892-45ecd05ae2ad?w=400"},
		{ID: "f8", Title: "Rainy Day", Artist: "Mist", Album: "Pitter Patter", Duration: "2:50", Artwork: "https://images.unsplash.com/photo-1534353436294-0dbd4bdac845?w=400"},
		{ID: "f9", Title: "Starry Night", Artist: "Cosmos", Album: "Galaxy", Duration: "5:30", Artwork: "https://images.unsplash.com/photo-1461749280684-dccba630e2f6?w=400"},
		{ID: "f10", Title: "Morning Dew", Artist: "Aura", Album: "Sunrise", Duration: "3:10", Artwork: "https://images.unsplash.com/photo-1446057032654-9d8822ee2d9b?w=400"},
		{ID: "f11", Title: "Urban Jungle", Artist: "Concrete", Album: "Street Life", Duration: "4:05", Artwork: "https://images.unsplash.com/photo-1441974231531-c6227db76b6e?w=400"},
		{ID: "f12", Title: "Golden Hour", Artist: "Amber", Album: "Twilight", Duration: "3:40", Artwork: "https://images.unsplash.com/photo-1470770841072-f978cf4d019e?w=400"},
		{ID: "f13", Title: "Lost in Time", Artist: "Relic", Album: "Artifact", Duration: "4:25", Artwork: "https://images.unsplash.com/photo-1516280440614-37939bbacd81?w=400"},
		{ID: "f14", Title: "Digital Soul", Artist: "Binary", Album: "Source Code", Duration: "3:50", Artwork: "https://images.unsplash.com/photo-1485846234645-a62644f84728?w=400"},
		{ID: "f15", Title: "Wild Spirit", Artist: "Rover", Album: "Untamed", Duration: "4:10", Artwork: "https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=400"},
	}
}
