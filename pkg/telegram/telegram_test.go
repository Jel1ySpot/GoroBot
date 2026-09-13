package telegram

import (
	"testing"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/go-telegram/bot/models"
)

func TestTelegramFeaturesAndKeyboard(t *testing.T) {
	service := Create()
	if !service.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("expected telegram to support FeatureInlineKeyboard")
	}
	if !service.SupportsFeature(botc.FeatureMarkdown) {
		t.Errorf("expected telegram to support FeatureMarkdown")
	}
	if service.SupportsFeature("unsupported_feature") {
		t.Errorf("did not expect telegram to support unsupported_feature")
	}

	msgContext := NewMessageContext(&models.Message{
		ID: 1,
		Chat: models.Chat{
			ID:   123456,
			Type: "private",
		},
	}, service)

	if !msgContext.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("expected msgContext to support FeatureInlineKeyboard")
	}

	kb := botc.NewInlineKeyboard().AddRow(
		botc.NewURLButton("Google", "https://google.com"),
		botc.NewCallbackButton("Like", "like_123"),
		botc.NewCommandButton("Echo", "/echo hi", true),
	)

	jsonStr, err := kb.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal keyboard: %v", err)
	}

	elements := []*botc.MessageElement{
		{
			Type:    botc.InlineKeyboardElement,
			Content: jsonStr,
		},
	}

	markup := extractReplyMarkup(elements)
	inlineMarkup, ok := markup.(*models.InlineKeyboardMarkup)
	if !ok || inlineMarkup == nil {
		t.Fatalf("expected *models.InlineKeyboardMarkup, got: %T", markup)
	}

	if len(inlineMarkup.InlineKeyboard) != 1 || len(inlineMarkup.InlineKeyboard[0]) != 3 {
		t.Fatalf("unexpected inline keyboard dimensions: %+v", inlineMarkup.InlineKeyboard)
	}

	btn0 := inlineMarkup.InlineKeyboard[0][0]
	btn1 := inlineMarkup.InlineKeyboard[0][1]
	btn2 := inlineMarkup.InlineKeyboard[0][2]

	if btn0.Text != "Google" || btn0.URL != "https://google.com" {
		t.Errorf("btn0 mismatch: %+v", btn0)
	}
	if btn1.Text != "Like" || btn1.CallbackData != "like_123" {
		t.Errorf("btn1 mismatch: %+v", btn1)
	}
	if btn2.Text != "Echo" || btn2.SwitchInlineQueryCurrentChat == nil || *btn2.SwitchInlineQueryCurrentChat != "/echo hi" {
		t.Errorf("btn2 mismatch: %+v", btn2)
	}
}
