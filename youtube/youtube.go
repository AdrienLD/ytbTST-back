package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

func YouTubeAPIRequest(endpoint string, queryParams map[string]string) (map[string]interface{}, error) {
	youtubeApiKey := os.Getenv("GOOGLE_API_KEY")
	if youtubeApiKey == "" {
		return nil, fmt.Errorf("clé API Google manquante")
	}

	params := url.Values{}
	params.Set("key", youtubeApiKey)
	for key, value := range queryParams {
		params.Set(key, value)
	}
	reqURL := "https://www.googleapis.com/youtube/v3/" + endpoint + "?" + params.Encode()

	resp, err := httpClient.Get(reqURL)
	if err != nil {
		// L'erreur de net/http contient l'URL complète (et donc la clé API) : on ne garde que la cause
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("erreur lors de l'envoi de la requête : %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur API YouTube : %s", resp.Status)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage de la réponse JSON : %v", err)
	}

	return data, nil
}
