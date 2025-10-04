package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	youtube "google.golang.org/api/youtube/v3"
)

var (
	oauthConfig *oauth2.Config
	state       = "state-token"
	tokenChan   = make(chan *oauth2.Token)
	tokenFile   string
	once        sync.Once
)

func tokenFilePath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}
	return filepath.Join(usr.HomeDir, ".youtube_token.json")
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving token to %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to create token file: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("state") != state {
		http.Error(w, "State parameter doesn't match", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found in URL", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Authorization successful! You can close this window.")
	// Send token back to main goroutine
	tokenChan <- token
}

func getClient(ctx context.Context) *http.Client {
	tokenFile = tokenFilePath()
	token, err := tokenFromFile(tokenFile)
	if err == nil {
		return oauthConfig.Client(ctx, token)
	}

	// Start local HTTP server to receive OAuth2 callback
	http.HandleFunc("/oauth2callback", handleOAuth2Callback)
	go func() {
		log.Println("Starting local server at http://localhost:8080/")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Generate auth URL and instruct user to visit
	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Println("Visit the URL for the auth dialog:")
	fmt.Println(authURL)

	// Wait for token from callback handler
	token = <-tokenChan

	// Save token for future use
	saveToken(tokenFile, token)

	return oauthConfig.Client(ctx, token)
}

func InitYoutubeClient() (*youtube.Service, error) {
	ctx := context.Background()

	// Load credentials.json (OAuth2 client secrets)
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	oauthConfig, err = google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Make sure redirect URL is correct in OAuth client config
	// It should be: http://localhost:8080/oauth2callback
	// If not, override here:
	oauthConfig.RedirectURL = "http://localhost:8080/oauth2callback"

	client := getClient(ctx)

	// Create YouTube service with authenticated client
	srv, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create YouTube service: %v", err)
	}

	return srv, nil
}

func getPlaylistItems(service *youtube.Service, playlistId string) ([]*youtube.PlaylistItem, error) {
	var allItems []*youtube.PlaylistItem
	nextPageToken := ""
	for {
		call := service.PlaylistItems.List([]string{"snippet", "contentDetails"}).
			PlaylistId(playlistId).
			MaxResults(50)
		if nextPageToken != "" {
			call = call.PageToken(nextPageToken)
		}
		response, err := call.Do()
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, response.Items...)
		if response.NextPageToken == "" {
			break
		}
		nextPageToken = response.NextPageToken
	}
	return allItems, nil
}

func ListMyPlaylists(service *youtube.Service) ([]*youtube.Playlist, error) {
	var allPlaylists []*youtube.Playlist
	nextPageToken := ""
	for {
		call := service.Playlists.List([]string{"snippet", "contentDetails"}).
			Mine(true).
			MaxResults(50)
		if nextPageToken != "" {
			call = call.PageToken(nextPageToken)
		}
		response, err := call.Do()
		if err != nil {
			return nil, err
		}
		allPlaylists = append(allPlaylists, response.Items...)
		if response.NextPageToken == "" {
			break
		}
		nextPageToken = response.NextPageToken
	}
	return allPlaylists, nil
}
