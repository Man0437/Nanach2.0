package youtube

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "strings"
    "time"
    "os"
)

const maxVideoBytes = 45 * 1024 * 1024

var client = &http.Client{Timeout: 10 * time.Minute}

func IsYouTubeURL(raw string) bool {
    u, err := url.Parse(raw)
    if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" { 
        return false 
    }
    switch strings.ToLower(u.Hostname()) {
        case "youtube.com", "www.youtube.com", "m.youtube.com", "youtu.be": 
            return true
        default: 
            return false
    }
}

// DownloadAndSend streams from youtube-service to Telegram without storing a file in Go.
func DownloadAndSend(ctx context.Context, serviceURL, token string, chatID int64, videoURL string) error {
    body, _ := json.Marshal(map[string]string{"url": videoURL})
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(serviceURL, "/")+"/download", bytes.NewReader(body))
    if err != nil { 
        return err 
    }
    req.Header.Set(
        "X-Service-Token",
        os.Getenv("YOUTUBE_SERVICE_TOKEN"),
    )
    resp, err := client.Do(req)
    if err != nil { 
        return fmt.Errorf("youtube-service: %w", err) 
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        message, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
        return fmt.Errorf("youtube-service HTTP %d: %s", resp.StatusCode, message)
    }
    if resp.ContentLength > maxVideoBytes { 
        return fmt.Errorf("video too large") 
    }

    reader, writer := io.Pipe()
    form := multipart.NewWriter(writer)
    go func() {
        defer writer.Close()
        defer form.Close()
        if err := form.WriteField("chat_id", fmt.Sprint(chatID)); err != nil { 
            writer.CloseWithError(err); return 
        }
        part, err := form.CreateFormFile("video", "video.mp4")
        if err != nil { writer.CloseWithError(err); 
            return 
        }
        _, err = io.Copy(part, io.LimitReader(resp.Body, maxVideoBytes+1))
        if err != nil { writer.CloseWithError(err); 
            return 
        }
    }()

    uploadURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendVideo", token)
    uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, reader)
    if err != nil {
        reader.Close(); return err
    }
    uploadReq.Header.Set("Content-Type", form.FormDataContentType())
    uploadResp, err := client.Do(uploadReq)
    if err != nil {
        reader.CloseWithError(err);
        return fmt.Errorf("telegram upload: %w", err)
    }
    defer uploadResp.Body.Close()
    response, _ := io.ReadAll(io.LimitReader(uploadResp.Body, 1500))
    if uploadResp.StatusCode != http.StatusOK { 
        return fmt.Errorf("Telegram HTTP %d: %s", uploadResp.StatusCode, response) 
    }
    var telegram struct { 
        OK bool `json:"ok"` 
    }
    if err := json.Unmarshal(response, &telegram); err != nil || !telegram.OK { 
        return fmt.Errorf("Telegram rejected video: %s", response) 
    }
    return nil
}
