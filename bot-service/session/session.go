
package session

import "sync"

type State string

const (
    Idle              State = "idle"
    AnimeSearch       State = "anime_search"
    PinterestDownload State = "pinterest_download"
    YouTubeDownload   State = "youtube_download"
)

type Manager struct {
    mu     sync.RWMutex
    states map[int64]State
}

func NewManager() *Manager {
    return &Manager{
        states: make(map[int64]State),
    }
}

func (m *Manager) Set(userID int64, state State) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.states[userID] = state
}

func (m *Manager) Get(userID int64) State {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return m.states[userID]
}

func (m *Manager) Clear(userID int64) {
    m.mu.Lock()
    defer m.mu.Unlock()

    delete(m.states, userID)
}