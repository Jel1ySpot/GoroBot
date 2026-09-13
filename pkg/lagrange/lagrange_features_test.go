package lagrange

import (
	"testing"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
)

func TestLagrangeFeatures(t *testing.T) {
	s := Create()
	ctx := &Context{service: s}

	if !ctx.SupportsFeature(botc.FeatureText) {
		t.Errorf("expected lagrange to support FeatureText")
	}
	if !ctx.SupportsFeature(botc.FeatureImage) {
		t.Errorf("expected lagrange to support FeatureImage")
	}
	if ctx.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("did not expect lagrange to support FeatureInlineKeyboard")
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
