
package keyboard

const (
    AnimeButton     = "🔎 Поиск аниме"
    PinterestButton = "📌 Pinterest — изображения"
    YouTubeButton   = "🎵 YouTube — музыка"
    ExitButton      = "❌ Выйти из режима"
    HelpButton      = "❓ Помощь"
    RandomAnimeButton = "🎲 Случайное аниме"
)

type Button struct {
    Text string `json:"text"`
}

type ReplyKeyboard struct {
    Keyboard       [][]Button `json:"keyboard"`
    ResizeKeyboard bool       `json:"resize_keyboard"`
    IsPersistent   bool       `json:"is_persistent"`
}

func newKeyboard(rows [][]Button) ReplyKeyboard {
    return ReplyKeyboard{
        Keyboard:       rows,
        ResizeKeyboard: true,
        IsPersistent:   true,
    }
}

// Главная клавиатура со всеми сервисами.
func MainMenu() ReplyKeyboard {
    return newKeyboard([][]Button{
        {
            {Text: AnimeButton},
            {Text: RandomAnimeButton},
        },
        {{Text: PinterestButton}},
        {{Text: YouTubeButton}},
        {{Text: HelpButton}},
    })
}

// Клавиатура внутри активного режима.
func ServiceMenu() ReplyKeyboard {
    return newKeyboard([][]Button{
        {{Text: ExitButton}},
        {{Text: HelpButton}},
    })
}
