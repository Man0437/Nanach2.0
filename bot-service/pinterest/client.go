package pinterest

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

type Image struct {
    ImageURL  string `json:"image_url"`
    SourceURL string `json:"source_url"`
    Title     string `json:"title"`
}

type RandomResponse struct {
    Result Image `json:"result"`
}

type Client struct {
    BaseURL string
    HTTP    *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        BaseURL: strings.TrimRight(baseURL, "/"),
        HTTP: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *Client) Random() (Image, error) {
    var result RandomResponse

    resp, err := c.HTTP.Get(c.BaseURL + "/random")
    if err != nil {
        return result.Result, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return result.Result, fmt.Errorf(
            "pinterest-service: HTTP %d",
            resp.StatusCode,
        )
    }

    err = json.NewDecoder(resp.Body).Decode(&result)
    return result.Result, err
}