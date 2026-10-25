package youtube

import "testing"

func TestIsYouTubeURL(t *testing.T) {
    tests := []struct { url string; want bool }{
        {"https://youtube.com/watch?v=123", true},
        {"https://www.youtube.com/watch?v=123", true},
        {"https://youtu.be/123", true},
        {"http://youtube.com/watch?v=123", false},
        {"https://youtube.com.evil.test/watch?v=123", false},
        {"https://127.0.0.1/private", false},
        {"https://localhost/", false},
        {"https://user:pass@youtube.com/watch?v=123", false},
        {"https://youtube.com:443/watch?v=123", false},
        {"file:///etc/passwd", false},
        {"https://youtube.com@evil.test/watch?v=123", false},
    }
    for _, tc := range tests {
        t.Run(tc.url, func(t *testing.T) {
            if got := IsYouTubeURL(tc.url); got != tc.want {
                t.Errorf("IsYouTubeURL(%q)=%v; want %v", tc.url, got, tc.want)
            }
        })
    }
}
