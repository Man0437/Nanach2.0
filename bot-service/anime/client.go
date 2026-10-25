
package anime

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type Client struct {
    BaseURL string
    HTTP    *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        BaseURL: strings.TrimRight(baseURL, "/"),
        HTTP: &http.Client{
            Timeout: 15 * time.Second,
        },
    }
}

func (c *Client) Search(query string) (
    json.RawMessage,
    error,
) {
    endpoint := c.BaseURL + "/search?q=" +
        url.QueryEscape(query)

    resp, err := c.HTTP.Get(endpoint)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf(
            "anime-service returned HTTP %d",
            resp.StatusCode,
        )
    }

    var result json.RawMessage

    err = json.NewDecoder(resp.Body).Decode(&result)
    if err != nil {
        return nil, err
    }

    return result, nil
}

func (c *Client) Random() (json.RawMessage, error) {
    endpoint := c.BaseURL + "/random"

    resp, err := c.HTTP.Get(endpoint)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf(
            "anime-service returned HTTP %d",
            resp.StatusCode,
        )
    }

    var result json.RawMessage

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result, nil
}