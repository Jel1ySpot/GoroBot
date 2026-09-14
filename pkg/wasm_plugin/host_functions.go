package wasm_plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	extism "github.com/extism/go-sdk"
	"github.com/google/uuid"
)

// extractMessageDetails 从 MessageContext 解析消息基础信息、群聊/私聊类型、发送者与元素
func extractMessageDetails(msgCtx botc.MessageContext) (msgType string, groupID string, group *GroupInfo, sender *SenderInfo, msgID string, elements []MessageElementPayload, ts int64) {
	msgType = "direct"
	if msgCtx == nil {
		return
	}
	base := msgCtx.Message()
	if base == nil {
		return
	}

	msgID = base.ID
	if !base.Time.IsZero() {
		ts = base.Time.Unix()
	}

	if base.MessageType == botc.GroupMessage {
		msgType = "group"
	}

	if base.Sender != nil {
		// 群聊详情
		if base.Sender.From != nil {
			groupID = base.Sender.From.ID
			group = &GroupInfo{
				ID:     base.Sender.From.ID,
				Name:   base.Sender.From.Name,
				Avatar: base.Sender.From.Avatar,
			}
		}

		// 发送者详情
		sender = &SenderInfo{
			Authority: int(base.Sender.Authority),
		}
		if base.Sender.User != nil {
			if base.Sender.User.Base != nil {
				sender.ID = base.Sender.User.Base.ID
				sender.Name = base.Sender.User.Base.Name
				sender.Avatar = base.Sender.User.Base.Avatar
			}
			sender.Nickname = base.Sender.User.Nickname
		}
	}

	for _, el := range base.Elements {
		if el != nil {
			elements = append(elements, MessageElementPayload{
				Type:    int(el.Type),
				Content: el.Content,
				Source:  el.Source,
			})
		}
	}

	return
}

// createDualHostFunction 创建两个相同行为的 HostFunction：一个注册在默认命名空间 "extism:host/user"，另一个在 "gorobot"
func createDualHostFunction(name string, cb extism.HostFunctionStackCallback, params []extism.ValueType, returns []extism.ValueType) []extism.HostFunction {
	f1 := extism.NewHostFunctionWithStack(name, cb, params, returns)
	f2 := extism.NewHostFunctionWithStack(name, cb, params, returns)
	f2.SetNamespace("gorobot")
	return []extism.HostFunction{f1, f2}
}

// createHostFunctions 为插件实例创建所有内置的宿主功能接口
func (s *Service) createHostFunctions(inst *PluginInstance) []extism.HostFunction {
	var functions []extism.HostFunction

	// 1. 日志输出接口: gorobot_log(level, msg)
	logCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		level, _ := p.ReadString(stack[0])
		msg, _ := p.ReadString(stack[1])

		prefix := fmt.Sprintf("[WASM:%s]", inst.id)
		switch strings.ToLower(level) {
		case "debug":
			s.logger.Debug("%s %s", prefix, msg)
		case "warn", "warning":
			s.logger.Warning("%s %s", prefix, msg)
		case "error", "failed":
			s.logger.Failed("%s %s", prefix, msg)
		default:
			s.logger.Info("%s %s", prefix, msg)
		}
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_log",
		logCb,
		[]extism.ValueType{extism.ValueTypePTR, extism.ValueTypePTR},
		[]extism.ValueType{},
	)...)

	// 2. 指令注册接口: gorobot_register_command(def_json) -> json_resp
	regCmdCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": err.Error()})
			return
		}

		var def CommandDefinition
		if err := json.Unmarshal([]byte(jsonStr), &def); err != nil {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": fmt.Sprintf("解析指令定义失败: %v", err)})
			return
		}

		if def.Name == "" {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": "指令名称不能为空"})
			return
		}

		cmd := s.grb.Command(def.Name)
		if def.Description != "" {
			cmd.Description(def.Description)
		}

		for _, arg := range def.Arguments {
			inputType := command.String
			switch strings.ToLower(arg.Type) {
			case "number", "int", "float":
				inputType = command.Number
			case "bool", "boolean":
				inputType = command.Boolean
			}
			cmd.Argument(arg.Name, inputType, arg.Required, arg.Help)
		}

		for _, opt := range def.Options {
			optType := command.String
			switch strings.ToLower(opt.Type) {
			case "number", "int", "float":
				optType = command.Number
			case "bool", "boolean":
				optType = command.Boolean
			}
			cmd.Option(opt.Name, opt.ShortName, optType, opt.Required, opt.Default, opt.Help)
		}

		handlerName := def.Handler
		if handlerName == "" {
			handlerName = "on_command"
		}

		cmd.Action(func(cmdCtx *command.Context) error {
			token := uuid.NewString()
			s.activeContextsMu.Lock()
			s.activeContexts[token] = cmdCtx
			s.activeContextsMu.Unlock()
			defer func() {
				s.activeContextsMu.Lock()
				delete(s.activeContexts, token)
				s.activeContextsMu.Unlock()
			}()

			var featureStrings []string
			if prov, ok := cmdCtx.MessageContext.(botc.FeatureProvider); ok {
				for _, f := range prov.Features() {
					featureStrings = append(featureStrings, string(f))
				}
			} else if prov, ok := cmdCtx.BotContext().(botc.FeatureProvider); ok {
				for _, f := range prov.Features() {
					featureStrings = append(featureStrings, string(f))
				}
			}

			msgType, groupID, group, sender, msgID, elements, ts := extractMessageDetails(cmdCtx)

			cmdEv := CommandEvent{
				Command:      def.Name,
				Commands:     cmdCtx.Commands,
				MessageType:  msgType,
				GroupID:      groupID,
				Group:        group,
				SenderID:     cmdCtx.SenderID(),
				Sender:       sender,
				Protocol:     cmdCtx.Protocol(),
				BotContextID: cmdCtx.BotContext().ID(),
				Features:     featureStrings,
				MessageID:    msgID,
				Elements:     elements,
				Timestamp:    ts,
				Arguments:    cmdCtx.Arguments,
				KvArgs:       cmdCtx.KvArgs,
				Options:      cmdCtx.Options,
				Raw:          cmdCtx.String(),
				ContextToken: token,
			}

			payload, err := json.Marshal(cmdEv)
			if err != nil {
				return err
			}

			out, err := inst.Call(handlerName, payload)
			if err != nil {
				s.logger.Failed("插件 %s 处理指令 %s 失败: %v", inst.id, def.Name, err)
				return err
			}

			if len(out) > 0 {
				reply := strings.TrimSpace(string(out))
				if reply != "" {
					if cmdCtx.SupportsFeature(botc.FeatureMarkdown) && isMarkdownContent(reply) {
						builder := cmdCtx.NewMessageBuilder().Markdown(reply)
						if _, err := builder.ReplyTo(cmdCtx); err != nil {
							s.logger.Debug("插件 %s 使用 Markdown 回复失败 (%v)，回退为纯文本回复", inst.id, err)
							_, _ = cmdCtx.ReplyText(reply)
						}
					} else {
						_, _ = cmdCtx.ReplyText(reply)
					}
				}
			}
			return nil
		})

		delFn, err := cmd.Build()
		if err != nil {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": err.Error()})
			return
		}

		inst.commandReleases = append(inst.commandReleases, delFn)
		s.writeJsonResponse(p, stack, map[string]any{"success": true})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_register_command",
		regCmdCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 3. 消息回复接口: gorobot_reply_text(token, text) -> status_code
	replyCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		token, _ := p.ReadString(stack[0])
		text, _ := p.ReadString(stack[1])

		s.activeContextsMu.RLock()
		msgCtx, ok := s.activeContexts[token]
		s.activeContextsMu.RUnlock()

		if !ok || msgCtx == nil {
			stack[0] = 1
			return
		}

		if msgCtx.SupportsFeature(botc.FeatureMarkdown) && isMarkdownContent(text) {
			builder := msgCtx.NewMessageBuilder().Markdown(text)
			if _, err := builder.ReplyTo(msgCtx); err == nil {
				stack[0] = 0
				return
			}
			s.logger.Debug("插件 %s 使用 Markdown 回复失败，回退为纯文本回复", inst.id)
		}

		if _, err := msgCtx.ReplyText(text); err != nil {
			s.logger.Failed("插件 %s 回复消息失败: %v", inst.id, err)
			stack[0] = 2
			return
		}
		stack[0] = 0
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_reply_text",
		replyCb,
		[]extism.ValueType{extism.ValueTypePTR, extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 4. 主动发送消息: gorobot_send_message(req_json) -> resp_json
	sendMsgCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: err.Error()})
			return
		}

		var req SendMessageRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: fmt.Sprintf("解析参数失败: %v", err)})
			return
		}

		botCtx := s.getBotContext(req.BotContextID)
		if botCtx == nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: "未找到有效的机器人上下文"})
			return
		}

		if req.TargetID == "" {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: "目标ID不能为空"})
			return
		}

		builder := botCtx.NewMessageBuilder()
		if req.Markdown != "" {
			builder.Markdown(req.Markdown)
		} else {
			builder.Text(req.Text)
		}
		if req.Keyboard != nil && len(req.Keyboard.Rows) > 0 {
			kb := botc.NewInlineKeyboard()
			for _, r := range req.Keyboard.Rows {
				var rowBtns []botc.InlineKeyboardButton
				for _, b := range r {
					rowBtns = append(rowBtns, botc.InlineKeyboardButton{
						Text:       b.Text,
						Action:     botc.ButtonActionType(b.Action),
						Data:       b.Data,
						DirectSend: b.DirectSend,
						ID:         b.ID,
					})
				}
				kb.AddRow(rowBtns...)
			}
			builder.InlineKeyboard(kb)
		}

		_, err = builder.Send(req.TargetID)
		if err != nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: err.Error()})
			return
		}

		s.writeJsonResponse(p, stack, SendMessageResponse{Success: true})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_send_message",
		sendMsgCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 4.1 上下文回复消息(带内嵌键盘支持): gorobot_reply_message(req_json) -> resp_json
	replyMsgCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: err.Error()})
			return
		}

		var req ReplyMessageRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: fmt.Sprintf("解析参数失败: %v", err)})
			return
		}

		s.activeContextsMu.RLock()
		msgCtx, ok := s.activeContexts[req.ContextToken]
		s.activeContextsMu.RUnlock()

		if !ok || msgCtx == nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: "上下文已失效或不存在"})
			return
		}

		builder := msgCtx.NewMessageBuilder()
		if req.Markdown != "" {
			builder.Markdown(req.Markdown)
		} else {
			builder.Text(req.Text)
		}
		if req.Keyboard != nil && len(req.Keyboard.Rows) > 0 {
			kb := botc.NewInlineKeyboard()
			for _, r := range req.Keyboard.Rows {
				var rowBtns []botc.InlineKeyboardButton
				for _, b := range r {
					rowBtns = append(rowBtns, botc.InlineKeyboardButton{
						Text:       b.Text,
						Action:     botc.ButtonActionType(b.Action),
						Data:       b.Data,
						DirectSend: b.DirectSend,
						ID:         b.ID,
					})
				}
				kb.AddRow(rowBtns...)
			}
			builder.InlineKeyboard(kb)
		}

		_, err = builder.ReplyTo(msgCtx)
		if err != nil {
			s.writeJsonResponse(p, stack, SendMessageResponse{Success: false, Error: err.Error()})
			return
		}

		s.writeJsonResponse(p, stack, SendMessageResponse{Success: true})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_reply_message",
		replyMsgCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 4.2 查询适配器特性接口: gorobot_get_features(req_json) -> resp_json
	getFeaturesCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, GetFeaturesResponse{Success: false, Error: err.Error()})
			return
		}

		var req GetFeaturesRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			s.writeJsonResponse(p, stack, GetFeaturesResponse{Success: false, Error: fmt.Sprintf("解析参数失败: %v", err)})
			return
		}

		var provider botc.FeatureProvider

		if req.ContextToken != "" {
			s.activeContextsMu.RLock()
			msgCtx := s.activeContexts[req.ContextToken]
			s.activeContextsMu.RUnlock()
			if msgCtx != nil {
				if p, ok := msgCtx.(botc.FeatureProvider); ok {
					provider = p
				} else if p, ok := msgCtx.BotContext().(botc.FeatureProvider); ok {
					provider = p
				}
			}
		}

		if provider == nil && req.BotContextID != "" {
			botCtx := s.getBotContext(req.BotContextID)
			if p, ok := botCtx.(botc.FeatureProvider); ok {
				provider = p
			}
		}

		if provider == nil {
			s.writeJsonResponse(p, stack, GetFeaturesResponse{Success: false, Error: "未找到有效的特性提供者"})
			return
		}

		var features []string
		for _, f := range provider.Features() {
			features = append(features, string(f))
		}

		s.writeJsonResponse(p, stack, GetFeaturesResponse{
			Success:  true,
			Features: features,
		})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_get_features",
		getFeaturesCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 5. 事件订阅接口: gorobot_subscribe_event(req_json) -> resp_json
	subEventCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": err.Error()})
			return
		}

		var req SubscribeEventRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": err.Error()})
			return
		}

		handler := req.Handler
		if handler == "" {
			handler = "on_" + req.Event
		}

		delFn, err := s.grb.On(GoroBot.EventHandler{
			Name: req.Event,
			Callback: func(args ...interface{}) {
				if len(args) == 0 {
					return
				}
				if msgCtx, ok := args[0].(botc.MessageContext); ok {
					token := uuid.NewString()
					s.activeContextsMu.Lock()
					s.activeContexts[token] = msgCtx
					s.activeContextsMu.Unlock()
					defer func() {
						s.activeContextsMu.Lock()
						delete(s.activeContexts, token)
						s.activeContextsMu.Unlock()
					}()

					var featureStrings []string
					if prov, ok := msgCtx.(botc.FeatureProvider); ok {
						for _, f := range prov.Features() {
							featureStrings = append(featureStrings, string(f))
						}
					} else if prov, ok := msgCtx.BotContext().(botc.FeatureProvider); ok {
						for _, f := range prov.Features() {
							featureStrings = append(featureStrings, string(f))
						}
					}

					msgType, groupID, group, sender, msgID, elements, ts := extractMessageDetails(msgCtx)

					payload := MessageEventPayload{
						MessageType:  msgType,
						GroupID:      groupID,
						Group:        group,
						Protocol:     msgCtx.Protocol(),
						BotContextID: msgCtx.BotContext().ID(),
						Features:     featureStrings,
						MessageID:    msgID,
						SenderID:     msgCtx.SenderID(),
						Sender:       sender,
						Text:         msgCtx.String(),
						Elements:     elements,
						Timestamp:    ts,
						ContextToken: token,
					}
					b, err := json.Marshal(payload)
					if err == nil {
						out, err := inst.Call(handler, b)
						if err == nil && len(out) > 0 {
							reply := strings.TrimSpace(string(out))
							if reply != "" {
								_, _ = msgCtx.ReplyText(reply)
							}
						}
					}
				}
			},
		})

		if err != nil {
			s.writeJsonResponse(p, stack, map[string]any{"success": false, "error": err.Error()})
			return
		}

		inst.eventReleases = append(inst.eventReleases, delFn)
		s.writeJsonResponse(p, stack, map[string]any{"success": true})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_subscribe_event",
		subEventCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 6. 网络请求访问接口 (Outgoing HTTP): gorobot_http_request(req_json) -> resp_json
	httpReqCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, HttpResponsePayload{Error: err.Error()})
			return
		}

		var req HttpRequestPayload
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			s.writeJsonResponse(p, stack, HttpResponsePayload{Error: fmt.Sprintf("解析请求失败: %v", err)})
			return
		}

		method := strings.ToUpper(req.Method)
		if method == "" {
			method = "GET"
		}

		timeout := time.Duration(req.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client := &http.Client{Timeout: timeout}

		var bodyReader io.Reader
		if req.Body != "" {
			bodyReader = strings.NewReader(req.Body)
		}

		httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, bodyReader)
		if err != nil {
			s.writeJsonResponse(p, stack, HttpResponsePayload{Error: err.Error()})
			return
		}

		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			s.writeJsonResponse(p, stack, HttpResponsePayload{Error: err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			s.writeJsonResponse(p, stack, HttpResponsePayload{Error: err.Error()})
			return
		}

		respHeaders := make(map[string]string)
		for k := range resp.Header {
			respHeaders[k] = resp.Header.Get(k)
		}

		s.writeJsonResponse(p, stack, HttpResponsePayload{
			StatusCode: resp.StatusCode,
			Headers:    respHeaders,
			Body:       string(respBody),
		})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_http_request",
		httpReqCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 7. 网络监听服务接口 (Incoming HTTP / Webhook): gorobot_listen_http(req_json) -> resp_json
	httpListenCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		jsonStr, err := p.ReadString(stack[0])
		if err != nil {
			s.writeJsonResponse(p, stack, HttpListenResponse{Success: false, Error: err.Error()})
			return
		}

		var req HttpListenRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			s.writeJsonResponse(p, stack, HttpListenResponse{Success: false, Error: fmt.Sprintf("解析参数失败: %v", err)})
			return
		}

		if req.Addr == "" {
			req.Addr = ":8080"
		}
		if req.Path == "" {
			req.Path = "/"
		}
		if req.Handler == "" {
			req.Handler = "on_http_request"
		}

		err = s.registerHttpRoute(inst, req.Addr, req.Path, req.Handler)
		if err != nil {
			s.writeJsonResponse(p, stack, HttpListenResponse{Success: false, Error: err.Error()})
			return
		}

		s.writeJsonResponse(p, stack, HttpListenResponse{
			Success: true,
			Addr:    req.Addr,
			Path:    req.Path,
		})
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_listen_http",
		httpListenCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// 8. 辅助文件系统操作 (全部严格限定在 data/<plugin_id>/ 内，防止目录穿越)
	// gorobot_fs_read(path) -> data
	fsReadCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		relPath, _ := p.ReadString(stack[0])
		safePath, err := s.resolveDataPath(inst, relPath)
		if err != nil {
			p.WriteString(fmt.Sprintf("error: %v", err))
			stack[0] = 0
			return
		}
		s.logger.Debug("Wasm 插件 %s 读取文件: %s", inst.id, safePath)
		data, err := os.ReadFile(safePath)
		if err != nil {
			stack[0] = 0
			return
		}
		offset, _ := p.WriteBytes(data)
		stack[0] = offset
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_fs_read",
		fsReadCb,
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	// gorobot_fs_write(path, data) -> status_code (0:成功)
	fsWriteCb := func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		relPath, _ := p.ReadString(stack[0])
		data, _ := p.ReadBytes(stack[1])
		safePath, err := s.resolveDataPath(inst, relPath)
		if err != nil {
			stack[0] = 1
			return
		}
		s.logger.Debug("Wasm 插件 %s 写入文件: %s (%d 字节)", inst.id, safePath, len(data))
		if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
			stack[0] = 2
			return
		}
		if err := os.WriteFile(safePath, data, 0644); err != nil {
			stack[0] = 3
			return
		}
		stack[0] = 0
	}
	functions = append(functions, createDualHostFunction(
		"gorobot_fs_write",
		fsWriteCb,
		[]extism.ValueType{extism.ValueTypePTR, extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)...)

	return functions
}

func (s *Service) resolveDataPath(inst *PluginInstance, userPath string) (string, error) {
	cleanPath := filepath.Clean(filepath.Join(inst.dataDir, userPath))
	if !strings.HasPrefix(cleanPath, inst.dataDir) {
		return "", fmt.Errorf("路径越界，禁止访问数据目录之外的文件")
	}
	return cleanPath, nil
}

func (s *Service) writeJsonResponse(p *extism.CurrentPlugin, stack []uint64, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		stack[0] = 0
		return
	}
	offset, err := p.WriteString(string(data))
	if err != nil {
		stack[0] = 0
		return
	}
	stack[0] = offset
}

func isMarkdownContent(s string) bool {
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "+ ") ||
			strings.HasPrefix(trimmed, "> ") ||
			strings.HasPrefix(trimmed, "```") {
			return true
		}
	}
	return strings.Contains(s, "**") || strings.Contains(s, "```") || strings.Contains(s, "![")
}
