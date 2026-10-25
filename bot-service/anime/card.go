
package anime

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"
    "telegram-bot/keyboard"
)

type Anime struct {
    ID     int64   `json:"id"`
    Title  string  `json:"title"`
    Year   int     `json:"year"`
    Poster string  `json:"poster"`
    URL    string  `json:"url"`
    Rating float64 `json:"rating"`
    Status string  `json:"status"`
}

type SearchResponse struct {
    Count   int     `json:"count"`
    Results []Anime `json:"results"`
}

type RandomResponse struct {
    Result Anime `json:"result"`
}

type TelegramClient struct {
    Token  string
    Client *http.Client
}

func NewTelegramClient(token string) *TelegramClient {
    return &TelegramClient{
        Token: token,
        Client: &http.Client{
            Timeout: 15 * time.Second,
        },
    }
}

func (t *TelegramClient) SendAnime(
    chatID int64,
    anime Anime,
) error {

    year := "Неизвестен"
    if anime.Year > 0 {
        year = fmt.Sprint(anime.Year)
    }

    caption := fmt.Sprintf(
		"🎬 %s\n"+
			"📅 Год: %s\n"+
			"⭐ Рейтинг: %.2f/10\n"+
			"📺 Статус: %s",
		anime.Title,
		year,
		anime.Rating,
		anime.Status,
	)

    payload := map[string]any{
        "chat_id": chatID,
        "text": caption,
    }

    method := "sendMessage"

    if validHTTPS(anime.Poster) {
        method = "sendPhoto"
        delete(payload, "text")
        payload["photo"] = anime.Poster
        payload["caption"] = caption
    }

    if validHTTPS(anime.URL) {
        payload["reply_markup"] = map[string]any{
            "inline_keyboard": [][]map[string]string{
                {
                    {
                        "text": "🔗 Открыть на YummyAnime",
                        "url":  anime.URL,
                    },
                },
            },
        }
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    endpoint := fmt.Sprintf(
        "https://api.telegram.org/bot%s/%s",
        t.Token,
        method,
    )

    resp, err := t.Client.Post(
        endpoint,
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf(
            "Telegram API returned HTTP %d",
            resp.StatusCode,
        )
    }

    var result struct {
        OK          bool   `json:"ok"`
        Description string `json:"description"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    if !result.OK {
        return fmt.Errorf("Telegram API: %s", result.Description)
    }

    return nil
}

func validHTTPS(raw string) bool {
    u, err := url.Parse(raw)
    return err == nil &&
        u.Scheme == "https" &&
        u.Host != "" &&
        !strings.ContainsAny(u.Host, " \t\n")
}

func SendSearchResults(
    bot *TelegramClient,
    chatID int64,
    results SearchResponse,
) error {

    if len(results.Results) == 0 {
        return fmt.Errorf("аниме не найдены")
    }

    for i, item := range results.Results {
        if i >= 5 {
            break
        }

        if err := bot.SendAnime(chatID, item); err != nil {
            return err
        }
    }

    return nil
}
func (t *TelegramClient) SendText(
    chatID int64,
    message string,
) error {
    payload := map[string]any{
        "chat_id": chatID,
        "text": message,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    endpoint := fmt.Sprintf(
        "https://api.telegram.org/bot%s/sendMessage",
        t.Token,
    )

    resp, err := t.Client.Post(
        endpoint,
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf(
            "Telegram HTTP %d",
            resp.StatusCode,
        )
    }

    return nil
}


func (t *TelegramClient) SendMenu(chatID int64) error {
    payload := map[string]any{
        "chat_id": chatID,
        "text": "Выберите сервис:",
        "reply_markup": keyboard.MainMenu(),
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    endpoint := fmt.Sprintf(
        "https://api.telegram.org/bot%s/sendMessage",
        t.Token,
    )

    resp, err := t.Client.Post(
        endpoint,
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf(
            "Telegram API returned HTTP %d",
            resp.StatusCode,
        )
    }

    return nil
}