package browser

import (
	"testing"
	"time"
)

func TestParseTokenFromURL(t *testing.T) {
	url := "https://oauth.vk.com/blank.html#access_token=vk1.a.test_token_123&expires_in=86400&user_id=999888"
	tok, err := ParseTokenFromURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "vk1.a.test_token_123" {
		t.Errorf("got token %s, want vk1.a.test_token_123", tok.AccessToken)
	}
	if tok.UserID != 999888 {
		t.Errorf("got user_id %d, want 999888", tok.UserID)
	}
	if tok.Expires <= time.Now().Unix() {
		t.Errorf("expected expires to be in future, got %d", tok.Expires)
	}
}

func TestDetectAllBrowsers(t *testing.T) {
	browsers := DetectAllBrowsers()
	t.Logf("Detected %d browsers on this machine", len(browsers))
	for _, b := range browsers {
		t.Logf("Browser: %s (%s) at %s", b.Name, b.Type, b.Path)
	}
}
