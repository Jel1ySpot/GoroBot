package GoroBot

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/core/bot"
	"github.com/Jel1ySpot/GoroBot/pkg/core/event"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/GoroBot/pkg/util"
	"github.com/Jel1ySpot/conic"
	"os"
	"os/signal"
	"syscall"
)

const (
	ConfigPath = "conf/config.json"
)

type Instant struct {
	services []Service
	logger   logger.Inst
	contexts map[string]bot.Context
	config   Config

	event      *event.System
	middleware *MiddlewareSystem
}

func Create() *Instant {
	inst := Instant{
		services: []Service{},
		logger: &logger.DefaultLogger{
			LogLevel: logger.Info,
		},
		contexts: map[string]bot.Context{},
		event:    event.NewEventSystem(),
		middleware: &MiddlewareSystem{
			middlewares: make(map[string]MiddlewareCallback),
		},
		config: Config{
			Owner:    make(map[string]string),
			LogLevel: logger.Info,
		},
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

func (i *Instant) GetOwner(protocol string) (id string, ok bool) {
	id, ok = i.config.Owner[protocol]
	return
}

func (i *Instant) Use(service Service) {
	i.services = append(i.services, service)
}

func (i *Instant) Remove(service Service) error {
	i.logger.Debug("Removing service %s", service.Name())
	if err := service.Release(i); err != nil {
		i.logger.Error("Failed to remove service %s: %s", service.Name(), err.Error())
		return err
	}
	i.logger.Debug("Removed service %s success", service.Name())
	return nil
}

func (i *Instant) initServices() error {
	for _, service := range i.services {
		i.logger.Debug("Initializing service %s", service.Name())
		if err := service.Init(i); err != nil {
			i.logger.Error("Failed to initialize service %s: %s", service.Name(), err.Error())
			return err
		}
		i.logger.Debug("Initialized service %s success", service.Name())
	}
	return nil
}

func (i *Instant) releaseServices() {
	for _, service := range i.services {
		i.logger.Debug("Releasing service %s", service.Name())
		if err := service.Release(i); err != nil {
			i.logger.Error("Failed to release service %s: %s", service.Name(), err.Error())
		}
		i.logger.Debug("Released service %s success", service.Name())
	}
}

func (i *Instant) AddContext(context bot.Context) bool {
	if _, ok := i.contexts[context.Name()]; ok {
		return false
	}
	i.contexts[context.ID()] = context
	return true
}

func (i *Instant) GetContext(id string) bot.Context {
	if context, ok := i.contexts[id]; ok {
		return context
	}
	return nil
}

func (i *Instant) RemoveContext(id string) bool {
	if _, ok := i.contexts[id]; ok {
		delete(i.contexts, id)
		return true
	}
	return false
}

func (i *Instant) Run() error {
	conic.SetConfigFile(ConfigPath)
	conic.WatchConfig()
	conic.BindRef("", &i.config)
	conic.SetLogger(i.logger.Info)

	if !util.FileExists(ConfigPath) {
		if err := util.MkdirIfNotExists("conf/"); err != nil {
			return err
		}
		if err := conic.WriteConfig(); err != nil {
			return err
		}
		i.logger.Warning("Config file created.")
		return fmt.Errorf("config file not exist")
	}

	if err := conic.ReadConfig(); err != nil {
		return err
	}

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
