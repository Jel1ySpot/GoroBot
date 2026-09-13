package bot_context

import "encoding/json"

// ButtonActionType 定义按钮点击交互动作类型
type ButtonActionType int

const (
	// ActionURL 打开链接
	ActionURL ButtonActionType = 0
	// ActionCallback 回调交互型（向服务端发送回调事件，例如 Telegram callback_data / QQ INTERACTION_CREATE）
	ActionCallback ButtonActionType = 1
	// ActionCommand 指令发送型（自动发送指令或填入输入框）
	ActionCommand ButtonActionType = 2
)

// InlineKeyboardButton 跨平台统一内嵌按钮
type InlineKeyboardButton struct {
	// Text 按钮显示文本
	Text string `json:"text"`
	// Action 动作类型：ActionURL, ActionCallback, ActionCommand
	Action ButtonActionType `json:"action"`
	// Data 对应动作携带的数据：ActionURL为链接，ActionCallback为回调标识/数据，ActionCommand为待发送文本
	Data string `json:"data"`
	// DirectSend 指令型按钮是否直接发送（仅对支持的平台有效，如 QQ 机器人 enter=true）
	DirectSend bool `json:"direct_send,omitempty"`
	// ID 可选，指定按钮唯一ID（QQ等平台需要时使用，留空则自动生成）
	ID string `json:"id,omitempty"`
}

// InlineKeyboard 跨平台统一内嵌键盘（多行按钮）
type InlineKeyboard struct {
	Rows [][]InlineKeyboardButton `json:"rows"`
}

// NewInlineKeyboard 创建空内嵌键盘
func NewInlineKeyboard() *InlineKeyboard {
	return &InlineKeyboard{
		Rows: make([][]InlineKeyboardButton, 0),
	}
}

// AddRow 添加一行按钮
func (kb *InlineKeyboard) AddRow(buttons ...InlineKeyboardButton) *InlineKeyboard {
	kb.Rows = append(kb.Rows, buttons)
	return kb
}

// NewURLButton 创建打开网址按钮
func NewURLButton(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:   text,
		Action: ActionURL,
		Data:   url,
	}
}

// NewCallbackButton 创建回调型按钮
func NewCallbackButton(text, callbackData string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:   text,
		Action: ActionCallback,
		Data:   callbackData,
	}
}

// NewCommandButton 创建指令型按钮
func NewCommandButton(text, cmd string, directSend bool) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:       text,
		Action:     ActionCommand,
		Data:       cmd,
		DirectSend: directSend,
	}
}

// ToJSON 将 InlineKeyboard 序列化为 JSON 字符串
func (kb *InlineKeyboard) ToJSON() (string, error) {
	bytes, err := json.Marshal(kb)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON 从 JSON 反序列化 InlineKeyboard
func ParseInlineKeyboard(data string) (*InlineKeyboard, error) {
	var kb InlineKeyboard
	if err := json.Unmarshal([]byte(data), &kb); err != nil {
		return nil, err
	}
	return &kb, nil
}
