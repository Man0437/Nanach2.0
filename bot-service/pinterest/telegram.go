package pinterest

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
	"time"
)

func SendPhoto(
    botToken string,
    chatID int64,
    image Image,
) error {
    parsed, err := url.Parse(image.ImageURL)
    if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
        return fmt.Errorf("invalid image URL")
    }

    caption := "📌 " + image.Title

    payload := map[string]any{
        "chat_id": chatID,
        "photo":   image.ImageURL,
        "caption": caption,
    }

    if image.SourceURL != "" {
        source, err := url.Parse(image.SourceURL)
        if err == nil && source.Scheme == "https" &&
            source.Hostname() == "pinterest.com" ||
            err == nil && source.Scheme == "https" &&
                source.Hostname() == "www.pinterest.com" {

            payload["reply_markup"] = map[string]any{
                "inline_keyboard": [][]map[string]string{
                    {
                        {
                            "text": "📌 Открыть Pinterest",
                            "url":  image.SourceURL,
                        },
                    },
                },
            }
        }
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    endpoint := fmt.Sprintf(
        "https://api.telegram.org/bot%s/sendPhoto",
        botToken,
    )

    resp, err := (&http.Client{
        Timeout: 20 * time.Second,
    }).Post(
        endpoint,
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Telegram HTTP %d", resp.StatusCode)
    }

    var answer struct {
        OK          bool   `json:"ok"`
        Description string `json:"description"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
        return err
    }
    if !answer.OK {
        return fmt.Errorf("Telegram: %s", answer.Description)
    }
    return nil
}