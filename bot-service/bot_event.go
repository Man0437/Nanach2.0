
package main

import (
    "bytes"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
    "time"
)

var eventHTTPClient = &http.Client{
    Timeout: 3 * time.Second,
}


func emitEvent(
    level string,
    event string,
    userID int64,
    username string,
    detail string,
) {
    endpoint := os.Getenv("ADMIN_EVENTS_URL")
    token := os.Getenv("EVENT_TOKEN")

    if endpoint == "" || token == "" {
        log.Println("Admin events not configured")
        return
    }

    payload := map[string]any{
        "level":       level,
        "service":     "bot-service",
        "event":       event,
        "telegram_id": userID,
        "username":    username,
        "detail":      detail,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        log.Println("JSON error:", err)
        return
    }

    req, err := http.NewRequest(
        http.MethodPost,
        endpoint,
        bytes.NewReader(data),
    )
    if err != nil {
        log.Println("Request error:", err)
        return
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Event-Token", token)

    resp, err := eventHTTPClient.Do(req)
    if err != nil {
        log.Println("Event send error:", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        var body bytes.Buffer
        _, _ = body.ReadFrom(
            io.LimitReader(resp.Body, 4096),
        )

        log.Printf(
            "Admin API error: HTTP %d, response: %s",
            resp.StatusCode,
            body.String(),
        )
    }
}

