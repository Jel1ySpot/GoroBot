package GoroBot

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/event"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/GoroBot/pkg/util"
	"github.com/Jel1ySpot/conic"
)

const (
	ConfigPath = "conf/config.json"
)

type Instant struct {
	services   []Service
	servicesMu sync.RWMutex
	logger     logger.Inst
	db         *sql.DB
	contexts   map[string]botc.BotContext
	contextsMu sync.RWMutex
	config     Config

	event      *event.System
	middleware *MiddlewareSystem
	commands   *command.System

	// 没有连接数据库时使用
	resourceMap map[string]Resource
}

func Create() *Instant {
	inst := Instant{
		services: []Service{},
		logger: &logger.DefaultLogger{
			LogLevel: logger.Info,
		},
		contexts: map[string]botc.BotContext{},
		event:    event.NewEventSystem(),
		middleware: &MiddlewareSystem{
			middlewares: make(map[string]MiddlewareCallback),
		},
		commands: command.NewCommandSystem(),
		config: Config{
			Owner:    make(map[string]string),
			Admin:    make(map[string]StringList),
			LogLevel: logger.Info,
		},

		resourceMap: make(map[string]Resource),
	}

	inst.EventRegister("message")
	inst.EventRegister("command")

	return &inst
}

func (i *Instant) UseLogger(logger logger.Inst) {
	i.logger = logger
}

func (i *Instant) GetLogger() logger.Inst {
	return i.logger
}

func (i *Instant) GetOwner(id string) (owner string, ok bool) {
	owner, ok = i.config.Owner[id]
	return
}

func (i *Instant) GetAdmins(id string) []string {
	if i.config.Admin == nil {
		return nil
	}
	return i.config.Admin[id]
}

// matchContext 匹配配置中的上下文/协议标识
func matchContext(cfgKey, contextID, protocol string) bool {
	if cfgKey == "*" || cfgKey == "default" {
		return true
	}
	if contextID != "" && strings.EqualFold(cfgKey, contextID) {
		return true
	}
	if protocol != "" && strings.EqualFold(cfgKey, protocol) {
		return true
	}
	// 兼容 qq 别名映射（lagrange 和 onebot 均可配置为 qq）
	if strings.EqualFold(cfgKey, "qq") {
		p := strings.ToLower(protocol)
		if p == "lagrange" || p == "onebot" || p == "qq" {
			return true
		}
		c := strings.ToLower(contextID)
		if strings.HasPrefix(c, "lagrange") || strings.HasPrefix(c, "onebot") || strings.HasPrefix(c, "qq") {
			return true
		}
	}
	return false
}

// matchID 匹配配置的用户 ID 与实际 senderID（支持 Universal ID 与原始 ID）
func matchID(configured, senderID string) bool {
	if configured == "" || senderID == "" {
		return false
	}
	if configured == senderID {
		return true
	}
	// 解析 senderID Universal ID (protocol:type&id)
	if info, ok := entity.ParseInfo(senderID); ok {
		for _, arg := range info.Args {
			if arg == configured {
				return true
			}
		}
	}
	// 解析 configured Universal ID
	if info, ok := entity.ParseInfo(configured); ok {
		for _, arg := range info.Args {
			if arg == senderID {
				return true
			}
		}
	}
	return false
}

// IsOwner 检查指定发送者是否为 Owner (等级 5)
func (i *Instant) IsOwner(ctxID, protocol, senderID string) bool {
	for cfgKey, cfgOwnerID := range i.config.Owner {
		if matchContext(cfgKey, ctxID, protocol) && matchID(cfgOwnerID, senderID) {
			return true
		}
	}
	return false
}

// IsAdmin 检查指定发送者是否为 Admin (等级 4)
func (i *Instant) IsAdmin(ctxID, protocol, senderID string) bool {
	for cfgKey, adminList := range i.config.Admin {
		if matchContext(cfgKey, ctxID, protocol) {
			for _, adminID := range adminList {
				if matchID(adminID, senderID) {
					return true
				}
			}
		}
	}
	return false
}

// ResolveSenderAuthority 自动修复默认权限并提升 Owner/Admin 等级
func (i *Instant) ResolveSenderAuthority(msg botc.MessageContext) {
	if msg == nil {
		return
	}
	base := msg.Message()
	if base == nil {
		return
	}
	if base.Sender == nil {
		base.Sender = &entity.Sender{}
	}
	if base.Sender.User == nil {
		base.Sender.User = &entity.User{
			Base: &entity.Base{
				ID: msg.SenderID(),
			},
			Authority: entity.Member,
		}
	}

	// 1. 修复默认值：若小于 Member(1)，兜底设为 Member(1)
	if base.Sender.Authority < entity.Member {
		base.Sender.Authority = entity.Member
	}

	var ctxID string
	if botCtx := msg.BotContext(); botCtx != nil {
		ctxID = botCtx.ID()
	}
	protocol := msg.Protocol()
	senderID := msg.SenderID()

	// 2. 自动赋值等级 5 (Owner)
	if i.IsOwner(ctxID, protocol, senderID) {
		base.Sender.Authority = entity.Owner
		return
	}

	// 3. 自动赋值等级 4 (Admin)
	if i.IsAdmin(ctxID, protocol, senderID) {
		if base.Sender.Authority < entity.Admin {
			base.Sender.Authority = entity.Admin
		}
		return
	}
}

func (i *Instant) Use(service Service) {
	i.servicesMu.Lock()
	defer i.servicesMu.Unlock()
	i.services = append(i.services, service)
}

func (i *Instant) Remove(service Service) error {
	i.logger.Debug("Removing service %s", service.Name())
	if err := service.Release(i); err != nil {
		i.logger.Failed("Failed to remove service %s: %s", service.Name(), err.Error())
		return err
	}
	i.servicesMu.Lock()
	for idx, s := range i.services {
		if s == service {
			i.services = append(i.services[:idx], i.services[idx+1:]...)
			break
		}
	}
	i.servicesMu.Unlock()
	i.logger.Success("Removed service %s success", service.Name())
	return nil
}

func (i *Instant) initServices() error {
	i.servicesMu.RLock()
	services := make([]Service, len(i.services))
	copy(services, i.services)
	i.servicesMu.RUnlock()

	for _, service := range services {
		i.logger.Debug("Initializing service %s", service.Name())
		if err := service.Init(i); err != nil {
			i.logger.Failed("Failed to initialize service %s: %v", service.Name(), err)
			continue
		}
		i.logger.Success("Initialized service %s success", service.Name())
	}
	return nil
}

func (i *Instant) releaseServices() {
	i.servicesMu.RLock()
	services := make([]Service, len(i.services))
	copy(services, i.services)
	i.servicesMu.RUnlock()

	for _, service := range services {
		i.logger.Debug("Releasing service %s", service.Name())
		if err := service.Release(i); err != nil {
			i.logger.Failed("Failed to release service %s: %v", service.Name(), err)
			continue
		}
		i.logger.Success("Released service %s success", service.Name())
	}
}

func (i *Instant) AddContext(context botc.BotContext) bool {
	i.logger.Debug("Adding %s bot context %s", context.Protocol(), context.ID())
	i.contextsMu.Lock()
	defer i.contextsMu.Unlock()
	if _, ok := i.contexts[context.ID()]; ok {
		i.logger.Failed("Duplicated bot context %s", context.ID())
		return false
	}
	i.contexts[context.ID()] = context
	i.logger.Success("Added %s bot context %s", context.Protocol(), context.ID())
	return true
}

func (i *Instant) GetContext(protocol string) botc.BotContext {
	i.contextsMu.RLock()
	defer i.contextsMu.RUnlock()
	if context, ok := i.contexts[protocol]; ok {
		return context
	}
	return nil
}

func (i *Instant) RemoveContext(protocol string) bool {
	i.contextsMu.Lock()
	defer i.contextsMu.Unlock()
	if _, ok := i.contexts[protocol]; ok {
		delete(i.contexts, protocol)
		return true
	}
	return false
}

func (i *Instant) Run() error {
	conic.SetConfigFile(ConfigPath)
	conic.WatchConfig()
	conic.BindRef("", &i.config)
	conic.SetLogger(i.logger.Debug)

	if !util.FileExists(ConfigPath) {
		if err := util.MkdirIfNotExists("conf/"); err != nil {
			return err
		}
		if err := os.WriteFile(ConfigPath, DefaultConfig, 0644); err != nil {
			return fmt.Errorf("failed to create config file: %v", err)
		}
		i.logger.Debug("已写入默认核心配置文件: %s", ConfigPath)
		i.logger.Warning("Config file does not exist, using default config.")
	}

	i.logger.Debug("正在读取核心配置文件: %s", ConfigPath)
	if err := conic.ReadConfig(); err != nil {
		return err
	}
	i.logger.Debug("核心配置读取完成 (LogLevel: %d)", i.config.LogLevel)

	i.logger.SetLogLevel(i.config.LogLevel)

	if err := i.initServices(); err != nil {
		i.releaseServices()
		return err
	}
	defer i.releaseServices()

	waitForInterrupt()

	return nil
}

func waitForInterrupt() {
	mc := make(chan os.Signal, 2)
	signal.Notify(mc, os.Interrupt, syscall.SIGTERM)
	for {
		switch <-mc {
		case os.Interrupt, syscall.SIGTERM:
			return
		}
	}
}
