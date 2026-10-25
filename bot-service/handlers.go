
package main

import (
    "encoding/json"
    "log"
    "strings"
    "fmt"
    "telegram-bot/anime"
)

func handleAnimeCommand(
    chatID int64,
    query string,
    animeClient *anime.Client,
    telegramClient *anime.TelegramClient,
) {

    emitEvent(
        "INFO",
        "anime_search",
        chatID,
        "",
        "Поиск аниме: "+query,
    )
    query = strings.TrimSpace(query)

    if query == "" {
        err := telegramClient.SendText(
            chatID,
            "Использование: /anime название",
        )
        if err != nil {
            log.Println(err)
        }
        return
    }

    // Запрос к Python Anime Service
    data, err := animeClient.Search(query)
    if err != nil {
        log.Println("Search error:", err)
        emitEvent(
            "ERROR",
            "anime_search_error",
            chatID,
            "",
            err.Error(),
        )
        _ = telegramClient.SendText(
            chatID,
            "Не удалось выполнить поиск.",
        )
        return
    }

    // Преобразование JSON
    var result anime.SearchResponse

    if err := json.Unmarshal(data, &result); err != nil {
        log.Println("JSON decode error:", err)
        return
    }

    if len(result.Results) == 0 {
        _ = telegramClient.SendText(
            chatID,
            "Аниме не найдено.",
        )
        return
    }

    // Отправляем максимум 5 карточек
    for i, item := range result.Results {
        if i >= 5 {
            break
        }

        if err := telegramClient.SendAnime(
            chatID,
            item,
        ); err != nil {
            log.Println(
                "Telegram send error:",
                err,
            )
        }
    }
    emitEvent(
        "INFO",
        "anime_search_success",
        chatID,
        "",
        fmt.Sprintf(
            "Запрос: %s, найдено: %d",
            query,
            len(result.Results),
        ),
    )
}
