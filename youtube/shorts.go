package youtube

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client qui ne suit pas les redirections : on a besoin de voir la réponse de /shorts/ elle-même
var shortsClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// IsShort interroge https://www.youtube.com/shorts/{id} : YouTube répond 200 pour un Short
// et redirige vers /watch?v={id} pour une vidéo classique
func IsShort(videoId string) (bool, error) {
	req, err := http.NewRequest(http.MethodHead, "https://www.youtube.com/shorts/"+videoId, nil)
	if err != nil {
		return false, err
	}
	// Évite la redirection vers la page de consentement des cookies (UE)
	req.Header.Set("Cookie", "SOCS=CAI")

	resp, err := shortsClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("erreur lors de la vérification du Short : %v", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		return true, nil
	case resp.StatusCode >= 300 && resp.StatusCode < 400 && strings.Contains(resp.Header.Get("Location"), "/watch"):
		return false, nil
	default:
		return false, fmt.Errorf("réponse inattendue de /shorts/%s : %s (Location: %s)", videoId, resp.Status, resp.Header.Get("Location"))
	}
}
