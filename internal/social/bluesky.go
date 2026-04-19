package social

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Bluesky post limit is 300 graphemes. We use 300 as a conservative char limit.
const blueskyMaxChars = 300

// BlueskyClient publishes short posts to Bluesky via the AT Protocol.
// Only requires a handle + app password (https://bsky.app/settings/app-passwords).
type BlueskyClient struct {
	Handle      string // e.g. "lwcn.bsky.social" or "mfahlandt.com"
	AppPassword string
	Service     string // default "https://bsky.social"
	HTTP        *http.Client
}

type bskySession struct {
	AccessJwt string `json:"accessJwt"`
	Did       string `json:"did"`
}

// Post publishes the given post to Bluesky. Creates a clickable link facet
// for the URL so it renders as a rich-text link in the Bluesky timeline.
func (c *BlueskyClient) Post(p Post) error {
	if c.Handle == "" || c.AppPassword == "" {
		return fmt.Errorf("bluesky credentials missing")
	}
	svc := c.Service
	if svc == "" {
		svc = "https://bsky.social"
	}
	httpc := c.HTTP
	if httpc == nil {
		httpc = &http.Client{Timeout: 30 * time.Second}
	}

	// 1) Create session
	loginBody, _ := json.Marshal(map[string]string{
		"identifier": c.Handle,
		"password":   c.AppPassword,
	})
	req, _ := http.NewRequest("POST", svc+"/xrpc/com.atproto.server.createSession", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return fmt.Errorf("bluesky login: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("bluesky login failed (%d): %s", resp.StatusCode, string(body))
	}
	var session bskySession
	if err := json.Unmarshal(body, &session); err != nil {
		return fmt.Errorf("bluesky login decode: %w", err)
	}

	// 2) Build post text with URL appended + link facet at URL byte range.
	text := truncateForPlatform(p.Text, p.URL, blueskyMaxChars)
	byteStart := strings.LastIndex(text, p.URL)
	byteEnd := byteStart + len(p.URL)

	record := map[string]any{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"langs":     []string{"en"},
	}
	if byteStart >= 0 {
		record["facets"] = []map[string]any{
			{
				"index": map[string]int{"byteStart": byteStart, "byteEnd": byteEnd},
				"features": []map[string]any{
					{"$type": "app.bsky.richtext.facet#link", "uri": p.URL},
				},
			},
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"repo":       session.Did,
		"collection": "app.bsky.feed.post",
		"record":     record,
	})

	// 3) Create record
	req2, _ := http.NewRequest("POST", svc+"/xrpc/com.atproto.repo.createRecord", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+session.AccessJwt)
	resp2, err := httpc.Do(req2)
	if err != nil {
		return fmt.Errorf("bluesky post: %w", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode >= 300 {
		return fmt.Errorf("bluesky post failed (%d): %s", resp2.StatusCode, string(body2))
	}
	return nil
}
