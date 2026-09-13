package wasm_plugin

// CommandDefinition 描述 Wasm 插件向宿主注册的指令结构
type CommandDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Arguments   []CommandArgument `json:"arguments,omitempty"`
	Options     []CommandOption   `json:"options,omitempty"`
	Handler     string            `json:"handler,omitempty"` // Wasm 导出函数名，默认 "on_command"
}

// CommandArgument 描述指令参数
type CommandArgument struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "string", "number", "bool"
	Required bool   `json:"required"`
	Help     string `json:"help"`
}

// CommandOption 描述指令选项（如 -t / --times）
type CommandOption struct {
	Name      string `json:"name"`
	ShortName string `json:"short,omitempty"`
	Type      string `json:"type"` // "string", "number", "bool"
	Required  bool   `json:"required"`
	Default   string `json:"default,omitempty"`
	Help      string `json:"help"`
}

// CommandEvent 是指令被触发时传给 Wasm 插件的上下文 JSON 结构
type CommandEvent struct {
	Command      string            `json:"command"`
	Commands     []string          `json:"commands"`
	SenderID     string            `json:"sender_id"`
	Protocol     string            `json:"protocol"`
	BotContextID string            `json:"context_id"`
	Arguments    []string          `json:"arguments"`
	KvArgs       map[string]string `json:"kv_args"`
	Options      map[string]string `json:"options"`
	Raw          string            `json:"raw"`
	ContextToken string            `json:"context_token"`
}

// MessageEventPayload 是接收到聊天消息时传给 Wasm 插件的结构
type MessageEventPayload struct {
	Protocol     string `json:"protocol"`
	BotContextID string `json:"context_id"`
	SenderID     string `json:"sender_id"`
	Text         string `json:"text"`
	ContextToken string `json:"context_token"`
}

// SendMessageRequest 是 Wasm 插件主动发送消息的请求参数
type SendMessageRequest struct {
	BotContextID string `json:"context_id,omitempty"` // 可选，指定机器人上下文ID
	TargetID     string `json:"target_id"`            // 接收者 ID (User/Group 或统一 Entity ID)
	Text         string `json:"text"`                 // 发送的文本内容
}

// SendMessageResponse 是主动发送消息后的返回结果
type SendMessageResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// HttpRequestPayload 是插件调用网络请求时的入参
type HttpRequestPayload struct {
	URL       string            `json:"url"`
	Method    string            `json:"method,omitempty"` // 默认 "GET"
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	TimeoutMs int64             `json:"timeout_ms,omitempty"`
}

// HttpResponsePayload 是网络请求的响应数据
type HttpResponsePayload struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// HttpListenRequest 是插件向宿主申请监听 HTTP 网络服务的请求结构
type HttpListenRequest struct {
	Addr    string `json:"addr"`              // 监听端口或地址，如 ":8080"
	Path    string `json:"path,omitempty"`    // 路由路径，默认为 "/"
	Handler string `json:"handler,omitempty"` // 收到 HTTP 请求时回调的 Wasm 导出函数名，默认 "on_http_request"
}

// HttpListenResponse 是监听 HTTP 网络服务后的返回结构
type HttpListenResponse struct {
	Success bool   `json:"success"`
	Addr    string `json:"addr"`
	Path    string `json:"path"`
	Error   string `json:"error,omitempty"`
}

// HttpIncomingRequest 是宿主收到外部 HTTP 请求后传给 Wasm 插件的数据
type HttpIncomingRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   string            `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// HttpOutgoingResponse 是 Wasm 插件处理完 HTTP 请求后返回给外部客户端的响应
type HttpOutgoingResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
}

// SubscribeEventRequest 是事件订阅请求
type SubscribeEventRequest struct {
	Event   string `json:"event"`             // 事件名，例如 "message"
	Handler string `json:"handler,omitempty"` // 触发时回调的 Wasm 导出函数名
}
