package onebot

import (
	"testing"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
)

func TestOneBotFeatures(t *testing.T) {
	s := Create()
	ctx := s.getContext()

	if !ctx.SupportsFeature(botc.FeatureText) {
		t.Errorf("expected onebot to support FeatureText")
	}
	if !ctx.SupportsFeature(botc.FeatureImage) {
		t.Errorf("expected onebot to support FeatureImage")
	}
	if ctx.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("did not expect onebot to support FeatureInlineKeyboard natively")
	}

	mc := &MessageContext{
		service: s,
	}
	if !mc.SupportsFeature(botc.FeatureText) {
		t.Errorf("expected MessageContext to support FeatureText")
	}
	if mc.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("did not expect MessageContext to support FeatureInlineKeyboard")
	}
}
