package main

import (
    "encoding/json"
    "log"

    "telegram-bot/anime"
)

func handleRandomAnimeCommand(
    chatID int64,
    animeClient *anime.Client,
    telegramClient *anime.TelegramClient,
) {
    emitEvent(
        "INFO",
        "anime_random",
        chatID,
        "",
        "Запрос случайного аниме",
    )

    data, err := animeClient.Random()
    if err != nil {
        log.Println("Random anime error:", err)

        emitEvent(
            "ERROR",
            "anime_random_error",
            chatID,
            "",
            err.Error(),
        )

        _ = telegramClient.SendText(
            chatID,
            "Не удалось получить случайное аниме.",
        )
        return
    }

    var result anime.RandomResponse

    if err := json.Unmarshal(data, &result); err != nil {
        log.Println("Random anime JSON error:", err)

        _ = telegramClient.SendText(
            chatID,
            "Ошибка обработки данных аниме.",
        )
        return
    }

    if err := telegramClient.SendAnime(
        chatID,
        result.Result,
    ); err != nil {
        log.Println("Random anime send error:", err)
        return
    }

    emitEvent(
        "INFO",
        "anime_random_success",
        chatID,
        "",
        "Случайное аниме отправлено",
    )
}