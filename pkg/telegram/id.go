package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

func genUserID(id int64) string {
	return fmt.Sprintf("telegram:%d", id)
}

func genGroupID(id int64) string {
	return fmt.Sprintf("telegram:%d", id)
}

func genMessageID(chatID int64, msgID int) string {
	return fmt.Sprintf("telegram:msg&%d&%d", chatID, msgID)
}

func parseChatID(id string) (int64, error) {
	id = strings.TrimPrefix(id, "telegram:")
	return strconv.ParseInt(id, 10, 64)
}
