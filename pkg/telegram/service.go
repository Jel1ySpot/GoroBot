package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

type Service struct {
	config     Config
	configPath string
	conic      *conic.Conic

	grb    *GoroBot.Instant
	logger logger.Inst
	status botc.LoginStatus

	bot    *bot.Bot
	ctx    context.Context
	cancel context.CancelFunc

	botID       int64
	botName     string
	botUsername string
}

func Create() *Service {
	return &Service{
		configPath: DefaultConfigPath,
		conic:      conic.New(),
		status:     botc.Offline,
	}
}

func (s *Service) Name() string {
	return "Telegram-adapter"
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.grb = grb
	s.logger = grb.GetLogger()

	if err := s.initConfig(); err != nil {
		return err
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	opts := []bot.Option{
		bot.WithDefaultHandler(s.handleUpdate),
	}
	if s.config.ServerURL != "" {
		opts = append(opts, bot.WithServerURL(s.config.ServerURL))
	}

	b, err := bot.New(s.config.Token, opts...)
	if err != nil {
		return fmt.Errorf("创建 Telegram bot 失败: %w", err)
	}
	s.bot = b

	me, err := s.bot.GetMe(s.ctx)
	if err != nil {
		return fmt.Errorf("获取 bot 信息失败: %w", err)
	}
	s.botID = me.ID
	s.botName = me.FirstName
	s.botUsername = me.Username

	grb.AddContext(s)

	go s.bot.Start(s.ctx)

	// 延迟同步命令到 Telegram，等待其他插件完成命令注册
	go func() {
		select {
		case <-time.After(3 * time.Second):
			s.SyncCommands()
		case <-s.ctx.Done():
		}
	}()

	s.status = botc.Online
	s.logger.Success("Telegram adapter 初始化完成，Bot: %s (@%s)", s.botName, s.botUsername)
	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	if s.cancel != nil {
		s.cancel()
	}
	s.status = botc.Offline
	return nil
}

// handleUpdate 处理收到的 Telegram 更新
func (s *Service) handleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	msgCtx := NewMessageContext(update.Message, s)
	text := msgCtx.String()

	if strings.HasPrefix(text, "/") {
		cmd := strings.TrimSpace(strings.TrimPrefix(text, "/"))
		// 移除 @botname 后缀
		if at := strings.Index(cmd, "@"); at != -1 {
			cmd = cmd[:at]
		}
		if cmd != "" {
			s.grb.CommandEmit(command.NewCommandContext(msgCtx, cmd))
			return
		}
	}

	if err := s.grb.MessageEmit(msgCtx); err != nil {
		s.logger.Error("触发 message 事件失败: %v", err)
	}
}

// SyncCommands 将已注册的命令同步到 Telegram 服务端
func (s *Service) SyncCommands() {
	schemas := s.grb.GetCommandSchemas()

	var cmds []models.BotCommand
	for _, schema := range schemas {
		desc := schema.Description
		if desc == "" {
			desc = schema.Name
		}
		cmds = append(cmds, models.BotCommand{
			Command:     strings.ToLower(schema.Name),
			Description: desc,
		})
	}

	if len(cmds) == 0 {
		return
	}

	_, err := s.bot.SetMyCommands(s.ctx, &bot.SetMyCommandsParams{
		Commands: cmds,
	})
	if err != nil {
		s.logger.Warning("同步命令到 Telegram 失败: %v", err)
		return
	}
	s.logger.Info("已同步 %d 个命令到 Telegram", len(cmds))
}

// --- BotContext 接口实现 ---

func (s *Service) ID() string {
	if s.botID != 0 {
		return genUserID(s.botID)
	}
	return "telegram:"
}

func (s *Service) Protocol() string {
	return "telegram"
}

func (s *Service) Status() botc.LoginStatus {
	return s.status
}

func (s *Service) NewMessageBuilder() botc.MessageBuilder {
	return &MessageBuilder{service: s}
}

func (s *Service) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	chatID, err := parseChatID(target.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	return s.sendToChat(chatID, elements)
}

func (s *Service) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	chatID, err := parseChatID(target.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}
	return s.sendToChat(chatID, elements)
}

func (s *Service) Contacts() []entity.User {
	return nil
}

func (s *Service) Groups() []entity.Group {
	return nil
}

func (s *Service) DownloadResourceFromRefLink(refLink string) (string, error) {
	values, err := urlpkg.ParseQuery(refLink)
	if err != nil {
		return "", fmt.Errorf("invalid ref link: %w", err)
	}

	target := values.Get("target")
	ext := values.Get("ext")
	rawURL := values.Get("url")
	fileID := values.Get("file_id")

	if rawURL == "" && fileID == "" {
		return "", fmt.Errorf("ref link missing url or file_id")
	}

	if fileID != "" {
		file, err := s.bot.GetFile(s.ctx, &bot.GetFileParams{FileID: fileID})
		if err != nil {
			return "", fmt.Errorf("getFile failed: %w", err)
		}
		rawURL = s.bot.FileDownloadLink(file)
		if ext == "" {
			ext = strings.TrimPrefix(path.Ext(file.FilePath), ".")
		}
	}

	if target == "" {
		if ext == "" && rawURL != "" {
			if u, err := urlpkg.Parse(rawURL); err == nil {
				ext = strings.TrimPrefix(path.Ext(u.Path), ".")
			}
		}
		if ext == "" {
			ext = "dat"
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		target = filepath.Join("resources", uuid.NewString()+ext)
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("download resource failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download resource status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read resource failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("create dir failed: %w", err)
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", fmt.Errorf("write file failed: %w", err)
	}

	return target, nil
}
