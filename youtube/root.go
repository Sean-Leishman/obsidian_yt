package youtube

import (
	youtube "google.golang.org/api/youtube/v3"
	"net/http"
)

func IpnitYotubeClient(apiKey string) (*youtube.Service, error) {
	client := &http.Client{}
	service, err := youtube.NewService(
		// context.Background(),
		client,
		youtube.WithAPIKey(apiKey),
	)
	if err != nil {
		return nil, err
	}

	return service, nil
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
			mine(true).
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
