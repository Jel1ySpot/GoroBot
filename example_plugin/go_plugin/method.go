package go_plugin

import (
	"fmt"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"plugin"
)

func (s *Service) LookupPlugins() error {
	plugins, err := ListPlugins(s.PluginPath)
	if err != nil {
		return err
	}

	for _, name := range plugins {
		if _, ok := s.pluginStat[name]; !ok {
			s.pluginStat[name] = false
		}
	}

	return nil
}

func (s *Service) InitPlugins() {
	log := s.logger

	log.Info("Start initializing Go plugins")

	for name, stat := range s.pluginStat {
		if stat == true {
			if err := s.ReleasePlugin(name); err != nil {
				log.Failed("Failed to re-initialize plugin: %s", name)
				continue
			}
		}
		if err := s.InitPlugin(name); err != nil {
			log.Failed(err.Error())
		}
	}
}

func (s *Service) InitPlugin(name string) error {
	log := s.logger
	grb := s.grb

	p, err := plugin.Open(name + ".so")
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %v", name, err)
	}

	sym, err := p.Lookup("RegularCreate")
	if err != nil {
		return fmt.Errorf("failed to find Create function in plugin %s: %v", name, err)
	}
	createFunc, ok := sym.(func() GoroBot.Service)
	if !ok {
		return fmt.Errorf("failed to type assert to RegularCreation in plugin %s: %T", name, sym)
	}

	service := createFunc()

	log.Debug("Initializing plugin service %s", service.Name())
	if err := service.Init(grb); err != nil {
		return fmt.Errorf("failed to initialize plugin service %s: %v", service.Name(), err)
	}

	s.services[name] = service
	s.pluginStat[name] = true
	log.Success("Initialized plugin service %s success", service.Name())

	return nil
}

func (s *Service) ReleasePlugin(name string) error {
	if _, ok := s.pluginStat[name]; !ok {
		return fmt.Errorf("plugin %s not found", name)
	}
	s.logger.Debug("Releasing plugin service %s", name)
	if err := s.services[name].Release(s.grb); err != nil {
		return fmt.Errorf("failed to release plugin service %s: %v", name, err)
	}
	delete(s.services, name)
	s.pluginStat[name] = false
	s.logger.Debug("Released plugin service %s success", name)
	return nil
}

func (s *Service) EnablePlugin(name string) error {
	if _, ok := s.pluginStat[name]; !ok {
		_ = s.LookupPlugins()
		if _, ok := s.pluginStat[name]; !ok {
			return fmt.Errorf("plugin %s not found", name)
		}
	}

	stat := s.pluginStat[name]
	if stat {
		return fmt.Errorf("plugin %s is already enabled", name)
	}
	if err := s.InitPlugin(name); err != nil {
		return err
	}
	return nil
}

func (s *Service) DisablePlugin(name string) error {
	if stat, ok := s.pluginStat[name]; ok {
		if !stat {
			return fmt.Errorf("plugin %s is already disabled", name)
		}
		if err := s.ReleasePlugin(name); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("plugin %s not found", name)
}
