package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}
}

func GetDBUrl() string {
	return os.Getenv("DATABASE_URL")
}

func GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

func GetJWTRefreshSecret() string {
	return os.Getenv("JWT_REFRESH_SECRET")
}

func GetSpotifyClientID() string {
	return os.Getenv("SPOTIFY_CLIENT_ID")
}

func GetSpotifyClientSecret() string {
	return os.Getenv("SPOTIFY_CLIENT_SECRET")
}

func GetSpotifyTokenURL() string {
	if value := os.Getenv("SPOTIFY_TOKEN_URL"); value != "" {
		return value
	}
	return "https://accounts.spotify.com/api/token"
}

func GetSpotifyAPIURL() string {
	if value := os.Getenv("SPOTIFY_API_URL"); value != "" {
		return value
	}
	return "https://api.spotify.com/v1"
}

func GetSpotifyMarket() string {
	if value := os.Getenv("SPOTIFY_MARKET"); value != "" {
		return value
	}
	return "IN"
}

func GetAudioDBAPIKey() string {
	if value := os.Getenv("AUDIODB_API_KEY"); value != "" {
		return value
	}
	return "2"
}

func GetAudioDBV1URL() string {
	if value := os.Getenv("AUDIODB_API_URL"); value != "" {
		return value
	}
	return "https://www.theaudiodb.com/api/v1/json"
}

func GetAudioDBV2URL() string {
	if value := os.Getenv("AUDIODB_V2_API_URL"); value != "" {
		return value
	}
	return "https://www.theaudiodb.com/api/v2/json"
}

func GetMusicBrainzAPIURL() string {
	if value := os.Getenv("MUSICBRAINZ_API_URL"); value != "" {
		return value
	}
	return "https://musicbrainz.org/ws/2"
}

func GetDeezerAPIURL() string {
	if value := os.Getenv("DEEZER_API_URL"); value != "" {
		return value
	}
	return "https://api.deezer.com"
}

func GetAdminLoginPasscode() string {
	if value := os.Getenv("ADMIN_LOGIN_PASSCODE"); value != "" {
		return value
	}
	return "9302010921"
}
