package qbot

import "encoding/json"

// Credentials QQ机器人开放平台鉴权凭据
type Credentials struct {
	AppID     string `yaml:"appid" json:"app_id"`
	Secret    string `yaml:"secret" json:"secret"`
	AppSecret string `yaml:"client_secret" json:"client_secret"`
}

// ClientSecret 获取 Secret 或 ClientSecret
func (c *Credentials) ClientSecret() string {
	if c.Secret != "" {
		return c.Secret
	}
	return c.AppSecret
}

// FileType 媒体文件类型
const (
	ImageFileType = 1 // 图片
	VideoFileType = 2 // 视频
	VoiceFileType = 3 // 语音 (silk格式)
	FileFileType  = 4 // 普通文件
)

// WSPayload WebSocket / Webhook 传输 Payload
type WSPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  *int            `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
	ID string          `json:"id,omitempty"`
}

// ValidationPayload 回调地址验证数据 (op: 13)
type ValidationPayload struct {
	PlainToken string `json:"plain_token"`
	EventTs    string `json:"event_ts"`
}

// ValidationResponse 回调地址验证响应
type ValidationResponse struct {
	PlainToken string `json:"plain_token"`
	Signature  string `json:"signature"`
}

// WebhookAck 回调确认响应 (op: 12)
type WebhookAck struct {
	Op int `json:"op"`
	D  int `json:"d"`
}

// User QQ平台用户对象
type User struct {
	ID          string `json:"id,omitempty"`
	Username    string `json:"username,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	Bot         bool   `json:"bot,omitempty"`
	UserOpenID  string `json:"user_openid,omitempty"`
	MemberOpenID string `json:"member_openid,omitempty"`
}

// Member 频道/群成员
type Member struct {
	User     *User    `json:"user,omitempty"`
	Nick     string   `json:"nick,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	JoinedAt string   `json:"joined_at,omitempty"`
}

// Attachment 消息附件
type Attachment struct {
	ContentType  string `json:"content_type"`
	URL          string `json:"url"`
	Filename     string `json:"filename,omitempty"`
	Height       int    `json:"height,omitempty"`
	Width        int    `json:"width,omitempty"`
	Size         int    `json:"size,omitempty"`
	VoiceWavURL  string `json:"voice_wav_url,omitempty"`
	AsrReferText string `json:"asr_refer_text,omitempty"`
}

// MessageReference 引用回复
type MessageReference struct {
	MessageID             string `json:"message_id,omitempty"`
	IgnoreGetMessageError bool   `json:"ignore_get_message_error,omitempty"`
	EventID               string `json:"event_id,omitempty"`
}

// Markdown 消息内容
type Markdown struct {
	CustomTemplateID string            `json:"custom_template_id,omitempty"`
	Params           map[string]string `json:"params,omitempty"`
	Content          string            `json:"content,omitempty"`
}

// Keyboard 按钮组件
type Keyboard struct {
	ID      string          `json:"id,omitempty"`
	Content *CustomKeyboard `json:"content,omitempty"`
}

// CustomKeyboard 自定义内嵌键盘内容
type CustomKeyboard struct {
	Rows []KeyboardRow `json:"rows"`
}

// KeyboardRow 按钮行
type KeyboardRow struct {
	Buttons []KeyboardButton `json:"buttons"`
}

// KeyboardButton 单个按钮
type KeyboardButton struct {
	ID         string              `json:"id,omitempty"`
	RenderData *KeyboardRenderData `json:"render_data,omitempty"`
	Action     *KeyboardAction     `json:"action,omitempty"`
}

// KeyboardRenderData 按钮渲染属性
type KeyboardRenderData struct {
	Label        string `json:"label"`
	VisitedLabel string `json:"visited_label,omitempty"`
	Style        int    `json:"style"` // 0=灰色线框, 1=蓝色线框, 2=推荐回复, 3=红色字体, 4=蓝色背景
}

// KeyboardAction 按钮交互动作
type KeyboardAction struct {
	Type          int                 `json:"type"` // 0=跳转链接, 1=回调(INTERACTION_CREATE), 2=指令输入(文本发给机器人), 3=mqqapi
	Data          string              `json:"data,omitempty"`
	Enter         bool                `json:"enter,omitempty"` // 点击后是否直接发送
	Reply         bool                `json:"reply,omitempty"` // 指令是否发到输入框
	Permission    *KeyboardPermission `json:"permission,omitempty"`
	ClickLimit    int                 `json:"click_limit,omitempty"`
	UnsupportTips string              `json:"unsupport_tips,omitempty"`
}

// KeyboardPermission 按钮权限
type KeyboardPermission struct {
	Type           int      `json:"type"` // 0=所有人, 1=管理员, 2=指定用户, 3=指定身份组
	SpecifyRoleIDs []string `json:"specify_role_ids,omitempty"`
	SpecifyUserIDs []string `json:"specify_user_ids,omitempty"`
}

// MediaInfo 富媒体消息标识
type MediaInfo struct {
	FileInfo string `json:"file_info,omitempty"`
}

// MessageToCreate 发送消息体
type MessageToCreate struct {
	Content          string            `json:"content,omitempty"`
	MsgType          int               `json:"msg_type"`
	Markdown         *Markdown         `json:"markdown,omitempty"`
	Keyboard         *Keyboard         `json:"keyboard,omitempty"`
	Media            *MediaInfo        `json:"media,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	EventID          string            `json:"event_id,omitempty"`
	MsgID            string            `json:"msg_id,omitempty"`
	MsgSeq           int               `json:"msg_seq,omitempty"`
	Timestamp        int64             `json:"timestamp,omitempty"`
}

// ExtInfo 消息扩展信息
type ExtInfo struct {
	RefIdx string `json:"ref_idx,omitempty"`
}

// MessageResponse 发送消息响应
type MessageResponse struct {
	ID        string   `json:"id"`
	Timestamp any      `json:"timestamp"`
	ExtInfo   *ExtInfo `json:"ext_info,omitempty"`
}

// FileUpload 文件上传请求体
type FileUpload struct {
	FileType   int    `json:"file_type"`
	FileData   string `json:"file_data,omitempty"`
	URL        string `json:"url,omitempty"`
	SrvSendMsg bool   `json:"srv_send_msg"`
	FileName   string `json:"file_name,omitempty"`
}

// FileInfo 文件上传响应
type FileInfo struct {
	FileUUID string `json:"file_uuid,omitempty"`
	FileInfo string `json:"file_info,omitempty"`
	TTL      uint   `json:"ttl,omitempty"`
	ID       string `json:"id,omitempty"`
}

// InboundMessageAttachment 兼容旧名
type InboundMessageAttachment = Attachment

// Message 统一接收消息对象
type Message struct {
	ID               string            `json:"id"`
	Content          string            `json:"content"`
	Timestamp        string            `json:"timestamp"`
	Author           *User             `json:"author,omitempty"`
	GroupID          string            `json:"group_openid,omitempty"`
	ChannelID        string            `json:"channel_id,omitempty"`
	GuildID          string            `json:"guild_id,omitempty"`
	DirectMessage    bool              `json:"direct_message,omitempty"`
	Attachments      []*Attachment     `json:"attachments,omitempty"`
	Mentions         []*User           `json:"mentions,omitempty"`
	MentionEveryone  bool              `json:"mention_everyone,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	MsgType          int               `json:"message_type,omitempty"`
}

// C2CMessageEvent C2C消息结构
type C2CMessageEvent struct {
	ID          string        `json:"id"`
	Content     string        `json:"content"`
	Timestamp   string        `json:"timestamp"`
	Author      struct {
		UserOpenID string `json:"user_openid"`
	} `json:"author"`
	Attachments []*Attachment `json:"attachments,omitempty"`
	MessageType int           `json:"message_type,omitempty"`
}

// GroupMessageEvent 群消息结构
type GroupMessageEvent struct {
	ID          string        `json:"id"`
	Content     string        `json:"content"`
	Timestamp   string        `json:"timestamp"`
	GroupOpenID string        `json:"group_openid"`
	Author      struct {
		MemberOpenID string `json:"member_openid"`
		Username     string `json:"username,omitempty"`
		Bot          bool   `json:"bot,omitempty"`
	} `json:"author"`
	Attachments []*Attachment `json:"attachments,omitempty"`
	Mentions    []*User       `json:"mentions,omitempty"`
	MessageType int           `json:"message_type,omitempty"`
}

// ChannelMessageEvent 频道消息结构
type ChannelMessageEvent struct {
	ID               string            `json:"id"`
	Content          string            `json:"content"`
	Timestamp        string            `json:"timestamp"`
	ChannelID        string            `json:"channel_id"`
	GuildID          string            `json:"guild_id"`
	Author           *User             `json:"author,omitempty"`
	Member           *Member           `json:"member,omitempty"`
	Attachments      []*Attachment     `json:"attachments,omitempty"`
	Mentions         []*User           `json:"mentions,omitempty"`
	MentionEveryone  bool              `json:"mention_everyone,omitempty"`
	DirectMessage    bool              `json:"direct_message,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
}
