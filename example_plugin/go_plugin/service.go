package go_plugin

import (
	"fmt"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"runtime"
)

const DefaultPluginPath = "plugin/"

type Service struct {
	grb    *GoroBot.Instant
	logger logger.Inst

	PluginPath string
	pluginStat map[string]bool
	services   map[string]GoroBot.Service
}

func (s *Service) Name() string {
	return "GoPluginImporter"
}

func Create() *Service {
	return &Service{
		PluginPath: DefaultPluginPath,
		pluginStat: make(map[string]bool),
	}
}

// RegularCreate 示例，一个返回标准 Service 接口的函数。动态加载的插件必须存在此函数。
func RegularCreate() GoroBot.Service {
	return Create()
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.grb = grb
	s.logger = grb.GetLogger()

	switch runtime.GOOS {
	case "darwin":
	case "linux":
	case "freebsd":
	default:
		return fmt.Errorf("%s is not a supported platform", runtime.GOOS)
	}

	if err := s.LookupPlugins(); err != nil {
		return err
	}

	s.InitPlugins()

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for name, stat := range s.pluginStat {
		if stat {
			if err := s.ReleasePlugin(name); err != nil {
				s.logger.Failed(err.Error())
			}
		}
	}
	return nil
}
