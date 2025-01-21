package go_plugin

import (
	"fmt"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"plugin"
	"runtime"
)

const DefaultPluginPath = "plugin/"

type Service struct {
	bot    *GoroBot.Instant
	logger logger.Inst

	PluginPath string
	services   []GoroBot.Service
}

func (s *Service) Name() string {
	return "GoPluginImporter"
}

func Create() *Service {
	return &Service{
		PluginPath: DefaultPluginPath,
	}
}

// RegularCreate 示例，一个返回标准 Service 接口的函数。动态加载的插件必须存在此函数。
func RegularCreate() GoroBot.Service {
	return Create()
}

type RegularCreation func() GoroBot.Service

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb
	s.logger = grb.GetLogger()

	log := s.logger

	switch runtime.GOOS {
	case "darwin":
	case "linux":
	case "freebsd":
	default:
		return fmt.Errorf("%s is not a supported platform", runtime.GOOS)
	}

	plugins, err := ListPlugins(s.PluginPath)
	if err != nil {
		return err
	}

	for _, file := range plugins {
		p, err := plugin.Open(file)
		if err != nil {
			log.Error("Failed to open plugin %s: %v", file, err)
			continue
		}

		sym, err := p.Lookup("RegularCreate")
		if err != nil {
			log.Error("Failed to find Create function in plugin %s: %v", file, err)
			continue
		}
		createFunc, ok := sym.(RegularCreation)
		if !ok {
			log.Error("Failed to type assert to RegularCreation in plugin %s: %t", file, sym)
		}

		service := createFunc()

		log.Debug("Initializing plugin service %s", service.Name())
		if err := service.Init(grb); err != nil {
			log.Error("Failed to initialize plugin service %s: %v", service.Name(), err)
			continue
		}

		s.services = append(s.services, service)
		log.Debug("Initialized plugin service %s success", service.Name())
	}

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, service := range s.services {
		s.logger.Debug("Releasing plugin service %s", service.Name())
		if err := service.Release(grb); err != nil {
			s.logger.Error("Failed to release plugin service %s: %v", service.Name(), err)
			continue
		}
		s.logger.Debug("Released plugin service %s success", service.Name())
	}
	return nil
}
