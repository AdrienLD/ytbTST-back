package youtube

import "testing"

// Test d'intégration : appelle réellement youtube.com
func TestIsShort(t *testing.T) {
	if testing.Short() {
		t.Skip("appel réseau ignoré en mode -short")
	}

	cases := map[string]bool{
		"aNI2jv7lBPQ": true,  // Short
		"dQw4w9WgXcQ": false, // vidéo classique
	}

	for videoId, want := range cases {
		got, err := IsShort(videoId)
		if err != nil {
			t.Fatalf("IsShort(%q) : erreur inattendue : %v", videoId, err)
		}
		if got != want {
			t.Errorf("IsShort(%q) = %v, attendu %v", videoId, got, want)
		}
	}
}
