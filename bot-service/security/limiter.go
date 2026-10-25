
package security

import (
    "sync"
    "time"
)

type userWindow struct {
    count int
    start time.Time
}

type Limiter struct {
    mu sync.Mutex

    users map[int64]userWindow
    busy  map[int64]bool

    downloads chan struct{}
    searches  chan struct{}

    lastCleanup time.Time
}

func NewLimiter() *Limiter {
    return &Limiter{
        users: make(map[int64]userWindow),
        busy:  make(map[int64]bool),

        downloads: make(chan struct{}, 2),
        searches:  make(chan struct{}, 8),

        lastCleanup: time.Now(),
    }
}

// AllowMessage ограничивает частоту сообщений
// до 10 в минуту на пользователя.
func (l *Limiter) AllowMessage(userID int64) bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now()

    if now.Sub(l.lastCleanup) >= 10*time.Minute {
        for id, window := range l.users {
            if now.Sub(window.start) >= time.Minute {
                delete(l.users, id)
            }
        }
        l.lastCleanup = now
    }

    window := l.users[userID]

    if window.start.IsZero() ||
        now.Sub(window.start) >= time.Minute {
        window = userWindow{start: now}
    }

    if window.count >= 10 {
        return false
    }

    window.count++
    l.users[userID] = window
    return true
}

// TryDownload запрещает несколько загрузок
// от одного пользователя одновременно.
func (l *Limiter) TryDownload(userID int64) bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.busy[userID] {
        return false
    }

    select {
    case l.downloads <- struct{}{}:
        l.busy[userID] = true
        return true
    default:
        return false
    }
}

func (l *Limiter) FinishDownload(userID int64) {
    l.mu.Lock()
    defer l.mu.Unlock()

    if !l.busy[userID] {
        return
    }

    delete(l.busy, userID)
    <-l.downloads
}

func (l *Limiter) TrySearch() bool {
    select {
    case l.searches <- struct{}{}:
        return true
    default:
        return false
    }
}

func (l *Limiter) FinishSearch() {
    <-l.searches
}