
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "telegram-bot/keyboard"
)

var menuHTTPClient = &http.Client{
    Timeout: 10 * time.Second,
}

func sendKeyboardMessage(
    token string,
    chatID int64,
    text string,
    markup keyboard.ReplyKeyboard,
) error {
    payload := map[string]any{
        "chat_id":      chatID,
        "text":         text,
        "reply_markup": markup,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    url := fmt.Sprintf(
        "https://api.telegram.org/bot%s/sendMessage",
        token,
    )

    resp, err := menuHTTPClient.Post(
        url,
        "application/json",
        bytes.NewReader(data),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(
            io.LimitReader(resp.Body, 2048),
        )
        return fmt.Errorf(
            "Telegram HTTP %d: %s",
            resp.StatusCode,
            string(body),
        )
    }

    return nil
}

func sendMainMenu(token string, chatID int64) error {
    return sendKeyboardMessage(
        token,
        chatID,
        "Выберите сервис:",
        keyboard.MainMenu(),
    )
}

func sendAnimeMenu(token string, chatID int64) error {
    return sendKeyboardMessage(
        token,
        chatID,
        "🔎 Режим поиска аниме включён.\n"+
            "Отправляйте названия аниме.\n\n"+
            "Для выхода нажмите кнопку или отправьте /cancel.",
        keyboard.ServiceMenu(),
    )
}