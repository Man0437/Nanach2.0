package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"telegram-bot/anime"
	"telegram-bot/keyboard"
	"telegram-bot/security"
	"telegram-bot/session"
	"telegram-bot/youtube"
    "telegram-bot/pinterest"
	"time"
)

type User struct {
	TelegramID int64  `json:"telegram_id"`
	Username   string `json:"username"`
}

type Update struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
		From *struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
	} `json:"message"`
}

type UpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

var client = &http.Client{
	Timeout: 40 * time.Second,
}

func registerUser(id int64, username string) error {
	user := User{
		TelegramID: id,
		Username:   username,
	}

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	url := os.Getenv("USER_SERVICE_URL") + "/users"

	resp, err := client.Post(
		url,
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user service: HTTP %d", resp.StatusCode)
	}

	return nil
}

func sendMessage(token string, chatID int64, text string) error {
	body, err := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}

	url := "https://api.telegram.org/bot" + token + "/sendMessage"

	resp, err := client.Post(
		url,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: HTTP %d", resp.StatusCode)
	}

	return nil
}

func main() {
	token := os.Getenv("BOT_TOKEN")

	animeClient := anime.NewClient(
		os.Getenv("ANIME_SERVICE_URL"),
	)

    pinterestClient := pinterest.NewClient(
        os.Getenv("PINTEREST_SERVICE_URL"),
    )

	telegramClient := anime.NewTelegramClient(token)

	if token == "" {
		log.Fatal("BOT_TOKEN is empty")
	}

	if os.Getenv("USER_SERVICE_URL") == "" {
		log.Fatal("USER_SERVICE_URL is empty")
	}

	sessions := session.NewManager()
	limiter := security.NewLimiter()

	var offset int64
	for {
		url := fmt.Sprintf(
			"https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30",
			token, offset,
		)

		resp, err := client.Get(url)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}

		var updates UpdatesResponse
		err = json.NewDecoder(resp.Body).Decode(&updates)
		resp.Body.Close()

		if err != nil || !updates.OK {
			log.Println("failed to get updates:", err)
			time.Sleep(time.Second)
			continue
		}

		for _, update := range updates.Result {
			offset = update.UpdateID + 1

			if update.Message == nil {
				continue
			}

			msg := update.Message

			if msg.From == nil {
				continue
			}

			if msg.Chat.Type != "private" {
				continue
			}

			userID := msg.From.ID
			chatID := msg.Chat.ID
			username := msg.From.Username
			text := strings.TrimSpace(msg.Text)

			if !limiter.AllowMessage(userID) {
				// Не отвечаем на каждое отклонённое сообщение,
				// иначе сами создадим дополнительную нагрузку.
				continue
			}

			if len(text) > 2048 {
				continue
			}

			emitEvent(
				"INFO",
				"user_activity",
				userID,
				username,
				"Получено сообщение",
			)

			switch {
			// Запуск бота и регистрация.
			case text == "/start":
				sessions.Clear(userID)

				err := registerUser(userID, username)
				if err != nil {
					log.Println("Registration error:", err)
					_ = sendMessage(
						token,
						chatID,
						"Не удалось выполнить регистрацию.",
					)
					continue
				}

				emitEvent(
					"INFO",
					"user_registered",
					userID,
					username,
					"Пользователь запустил бота",
				)

				err = sendMainMenu(token, chatID)
				if err != nil {
					log.Println("Main menu error:", err)
				}

			// Документация.
			case text == "/help" || text == keyboard.HelpButton:
				state := sessions.Get(userID)

				err := sendMessage(
					token,
					chatID,
					helpText(state),
				)
				if err != nil {
					log.Println("Help error:", err)
				}

			// Выход из режима.
			case text == "/cancel" ||
				text == "/menu" ||
				text == keyboard.ExitButton:

				sessions.Clear(userID)

				err := sendMainMenu(token, chatID)
				if err != nil {
					log.Println("Exit menu error:", err)
				}

			// Включение режима поиска аниме.
			case text == keyboard.AnimeButton:
				sessions.Set(userID, session.AnimeSearch)

				emitEvent(
					"INFO",
					"anime_mode_enabled",
					userID,
					username,
					"Включён режим поиска аниме",
				)

				err := sendAnimeMenu(token, chatID)
				if err != nil {
					log.Println("Anime menu error:", err)
				}

                        // Будущий сервис Pinterest.
            case text == keyboard.PinterestButton:
                if !limiter.TrySearch() {
                    _ = sendMessage(
                        token,
                        chatID,
                        "Сервис изображений временно занят.",
                    )
                    continue
                }

                go func() {
                    defer limiter.FinishSearch()

                    image, err := pinterestClient.Random()
                    if err != nil {
                        log.Println("Pinterest error:", err)
                        _ = sendMessage(
                            token,
                            chatID,
                            "Не удалось получить изображение.",
                        )
                        return
                    }

                    if err := pinterest.SendPhoto(
                        token,
                        chatID,
                        image,
                    ); err != nil {
                        log.Println("Pinterest send error:", err)
                        _ = sendMessage(
                            token,
                            chatID,
                            "Не удалось отправить изображение.",
                        )
                    }
                }()

			// Будущий сервис YouTube.

			case text == keyboard.YouTubeButton:
				sessions.Set(userID, session.YouTubeDownload)

				_ = sendKeyboardMessage(
					token,
					chatID,
					"🎬 Режим скачивания видео.\n"+
						"Отправьте ссылку YouTube.\n"+
						"Для выхода используйте /cancel.",
					keyboard.ServiceMenu(),
				)
			case text == keyboard.RandomAnimeButton:
				if !limiter.TrySearch() {
					_ = sendMessage(
						token,
						chatID,
						"Сервис аниме временно перегружен.",
					)
					continue
				}

				go func() {
					defer limiter.FinishSearch()

					handleRandomAnimeCommand(
						chatID,
						animeClient,
						telegramClient,
					)
				}()

				continue

			// Поиск через команду /anime.
			case text == "/anime" ||
				strings.HasPrefix(text, "/anime "):

				query := strings.TrimSpace(
					strings.TrimPrefix(text, "/anime"),
				)

				if query == "" {
					_ = sendMessage(
						token,
						chatID,
						"Укажите название: /anime Наруто",
					)
					continue
				}

				// Включаем постоянный режим поиска.
				wasAnimeMode :=
					sessions.Get(userID) == session.AnimeSearch

				sessions.Set(userID, session.AnimeSearch)

				// Отображаем клавиатуру режима, если
				// пользователь ещё не находился в нём.
				if !wasAnimeMode {
					if err := sendAnimeMenu(token, chatID); err != nil {
						log.Println("Anime menu error:", err)
					}
				}

				go handleAnimeCommand(
					chatID,
					query,
					animeClient,
					telegramClient,
				)

			// Поиск в активном режиме.

			case sessions.Get(userID) == session.YouTubeDownload:
				if !youtube.IsYouTubeURL(text) {
					_ = sendMessage(
						token,
						chatID,
						"Отправьте корректную HTTPS-ссылку YouTube.",
					)
					continue
				}

				if !limiter.TryDownload(userID) {
					_ = sendMessage(
						token,
						chatID,
						"⏳ У вас уже выполняется загрузка "+
							"или сервис сейчас занят.",
					)
					continue
				}

				go func(videoURL string, uid int64, name string) {
					defer limiter.FinishDownload(uid)

					ctx, cancel := context.WithTimeout(
						context.Background(),
						10*time.Minute,
					)
					defer cancel()

					err := youtube.DownloadAndSend(
						ctx,
						os.Getenv("YOUTUBE_SERVICE_URL"),
						token,
						chatID,
						videoURL,
					)

					if err != nil {
						log.Printf("YouTube download failed: %v", err)

						emitEvent(
							"ERROR",
							"youtube_download_error",
							uid,
							name,
							"Не удалось завершить загрузку",
						)

						_ = sendMessage(
							token,
							chatID,
							"Не удалось скачать видео.",
						)
						return
					}

					emitEvent(
						"INFO",
						"youtube_download_success",
						uid,
						name,
						"Видео успешно отправлено",
					)
				}(text, userID, username)
			case sessions.Get(userID) == session.AnimeSearch:
				if text == "" {
					_ = sendMessage(
						token,
						chatID,
						"Отправьте название аниме текстом.",
					)
					continue
				}

				// ВАЖНО: sessions.Clear() здесь нет.
				// Режим сохраняется после каждого поиска.
				if !limiter.TrySearch() {
					_ = sendMessage(
						token,
						chatID,
						"Поиск временно перегружен.",
					)
					continue
				}

				go func(query string, chatID int64) {
					defer limiter.FinishSearch()

					handleAnimeCommand(
						chatID,
						query,
						animeClient,
						telegramClient,
					)
				}(text, chatID)

			// Главное меню.
			default:
				err := sendMainMenu(token, chatID)
				if err != nil {
					log.Println("Main menu error:", err)
				}
			}
		}

	}
}
