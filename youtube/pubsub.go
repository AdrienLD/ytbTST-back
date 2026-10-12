package youtube

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	hubURL      = "https://pubsubhubbub.appspot.com/subscribe"
	callbackURL = "https://ytst-back.flgr.fr/youtube/callback"

	LeaseDuration = 10 * 24 * time.Hour
	// Renouvellement un jour avant l'expiration du bail
	RenewInterval = LeaseDuration - 24*time.Hour
)

func SubscribeToChannel(channelId string) error {
	topicURL := "https://www.youtube.com/xml/feeds/videos.xml?channel_id=" + channelId

	form := url.Values{}
	form.Add("hub.mode", "subscribe")
	form.Add("hub.topic", topicURL)
	form.Add("hub.callback", callbackURL)
	form.Add("hub.lease_seconds", fmt.Sprintf("%d", int(LeaseDuration.Seconds())))
	form.Add("hub.verify", "async")
	form.Add("hub.verify_token", os.Getenv("YTBToken"))

	resp, err := httpClient.PostForm(hubURL, form)
	if err != nil {
		return fmt.Errorf("erreur subscribe: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status: %d, body: %s", resp.StatusCode, body)
	}
	return nil
}
