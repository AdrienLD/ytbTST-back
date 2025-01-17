package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func YouTubeAPIRequest(endpoint string, queryParams map[string]string) (map[string]interface{}, error) {
	youtubeApiKey := os.Getenv("GOOGLE_API_KEY")
	if youtubeApiKey == "" {
		return nil, fmt.Errorf("clé API Google manquante")
	}

	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/%s?key=%s", endpoint, youtubeApiKey)
	for key, value := range queryParams {
		url += fmt.Sprintf("&%s=%s", key, value)
	}

	resp, err := http.Get(url)
	if err != nil {
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
