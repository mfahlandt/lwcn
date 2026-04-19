package social

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Twitter/X v2 tweet limit. Free tier is write-limited but supports creating
// tweets. Credentials come from https://developer.x.com/.
const xMaxChars = 280

// XClient publishes tweets via the X API v2 using OAuth 1.0a user context
// (required for POST /2/tweets).
type XClient struct {
	APIKey       string // a.k.a. consumer key
	APISecret    string // a.k.a. consumer secret
	AccessToken  string
	AccessSecret string
	HTTP         *http.Client
}

// Post publishes the given short post as a tweet.
func (c *XClient) Post(p Post) error {
	if c.APIKey == "" || c.APISecret == "" || c.AccessToken == "" || c.AccessSecret == "" {
		return fmt.Errorf("x credentials missing")
	}
	httpc := c.HTTP
	if httpc == nil {
		httpc = &http.Client{Timeout: 30 * time.Second}
	}

	text := truncateForPlatform(p.Text, p.URL, xMaxChars)
	body, _ := json.Marshal(map[string]string{"text": text})
	endpoint := "https://api.twitter.com/2/tweets"

	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.oauth1Header("POST", endpoint, nil))

	resp, err := httpc.Do(req)
	if err != nil {
		return fmt.Errorf("x post: %w", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("x post failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// oauth1Header builds the OAuth 1.0a "Authorization" header for a request.
// `params` are any extra query/body params (URL-encoded keys and values) that
// must participate in the signature. For JSON bodies (like /2/tweets) the
// body is NOT part of the signature base string — only OAuth params are.
func (c *XClient) oauth1Header(method, fullURL string, params map[string]string) string {
	nonce := randomNonce()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	oauthParams := map[string]string{
		"oauth_consumer_key":     c.APIKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_token":            c.AccessToken,
		"oauth_version":          "1.0",
	}

	// Collect all params for signature base string
	all := make(map[string]string, len(oauthParams)+len(params))
	for k, v := range oauthParams {
		all[k] = v
	}
	for k, v := range params {
		all[k] = v
	}

	// Parse URL query params as well (not used here but spec-compliant)
	if u, err := url.Parse(fullURL); err == nil {
		for k, vs := range u.Query() {
			if len(vs) > 0 {
				all[k] = vs[0]
			}
		}
		fullURL = strings.Split(fullURL, "?")[0]
	}

	// Sort keys and build parameter string
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, percentEncode(k)+"="+percentEncode(all[k]))
	}
	paramString := strings.Join(parts, "&")

	// Signature base string
	base := strings.ToUpper(method) + "&" + percentEncode(fullURL) + "&" + percentEncode(paramString)
	signingKey := percentEncode(c.APISecret) + "&" + percentEncode(c.AccessSecret)

	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(base))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	oauthParams["oauth_signature"] = signature

	// Build header
	hKeys := make([]string, 0, len(oauthParams))
	for k := range oauthParams {
		hKeys = append(hKeys, k)
	}
	sort.Strings(hKeys)
	var hParts []string
	for _, k := range hKeys {
		hParts = append(hParts, fmt.Sprintf("%s=\"%s\"", percentEncode(k), percentEncode(oauthParams[k])))
	}
	return "OAuth " + strings.Join(hParts, ", ")
}

// percentEncode implements RFC 3986 percent-encoding as required by OAuth 1.0a.
func percentEncode(s string) string {
	var b strings.Builder
	for _, r := range []byte(s) {
		switch {
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteByte(r)
		case r == '-' || r == '.' || r == '_' || r == '~':
			b.WriteByte(r)
		default:
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

func randomNonce() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return base64.RawURLEncoding.EncodeToString(buf[:])
}
