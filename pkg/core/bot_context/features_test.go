package bot_context

import (
	"testing"
)

type dummyProvider struct{}

func (d *dummyProvider) Features() []Feature {
	return []Feature{FeatureText, FeatureImage, FeatureInlineKeyboard}
}

func (d *dummyProvider) SupportsFeature(f Feature) bool {
	for _, feature := range d.Features() {
		if feature == f {
			return true
		}
	}
	return false
}

func TestFeaturesAndInlineKeyboard(t *testing.T) {
	dp := &dummyProvider{}
	if !dp.SupportsFeature(FeatureInlineKeyboard) {
		t.Fatal("expected dummyProvider to support FeatureInlineKeyboard")
	}
	if dp.SupportsFeature(FeatureVoice) {
		t.Fatal("did not expect dummyProvider to support FeatureVoice")
	}

	kb := NewInlineKeyboard().AddRow(
		NewURLButton("Google", "https://google.com"),
		NewCallbackButton("Like", "like_click"),
		NewCommandButton("Help", "/help", true),
	)

	if len(kb.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb.Rows))
	}
	if len(kb.Rows[0]) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(kb.Rows[0]))
	}

	jsonStr, err := kb.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal keyboard: %v", err)
	}

	parsed, err := ParseInlineKeyboard(jsonStr)
	if err != nil {
		t.Fatalf("failed to parse keyboard: %v", err)
	}

	if len(parsed.Rows) != 1 || len(parsed.Rows[0]) != 3 {
		t.Fatalf("parsed keyboard mismatch: %+v", parsed)
	}
	if parsed.Rows[0][2].DirectSend != true || parsed.Rows[0][2].Data != "/help" {
		t.Fatalf("button content mismatch: %+v", parsed.Rows[0][2])
	}
}
