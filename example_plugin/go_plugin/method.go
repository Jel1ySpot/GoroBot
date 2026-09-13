package go_plugin

import (
	"fmt"
	"plugin"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
)

func (s *Service) LookupPlugins() (int, error) {
	plugins, err := ListPlugins(s.PluginPath)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, name := range plugins {
		if _, ok := s.pluginStat[name]; !ok {
			s.pluginStat[name] = false
		}
	}

	return len(plugins), nil
}

func (s *Service) InitPlugins() {
	log := s.logger

	log.Info("Start initializing Go plugins")

	s.mu.RLock()
	pluginsToInit := make([]string, 0, len(s.pluginStat))
	for name := range s.pluginStat {
		pluginsToInit = append(pluginsToInit, name)
	}
	s.mu.RUnlock()

	for _, name := range pluginsToInit {
		s.mu.RLock()
		stat := s.pluginStat[name]
		s.mu.RUnlock()

		if stat {
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

func (s *Service) InitPlugin(name string) (err error) {
	log := s.logger
	grb := s.grb

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin %s panicked during initialization: %v", name, r)
			log.Failed(err.Error())
		}
	}()

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

	s.mu.Lock()
	s.services[name] = service
	s.pluginStat[name] = true
	s.mu.Unlock()

	log.Success("Initialized plugin service %s success", service.Name())

	return nil
}

func (s *Service) ReleasePlugin(name string) (err error) {
	s.mu.RLock()
	_, ok := s.pluginStat[name]
	service := s.services[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin %s panicked during release: %v", name, r)
			s.logger.Failed(err.Error())
		}
	}()

	if service != nil {
		s.logger.Debug("Releasing plugin service %s", name)
		if err := service.Release(s.grb); err != nil {
			return fmt.Errorf("failed to release plugin service %s: %v", name, err)
		}
	}

	s.mu.Lock()
	delete(s.services, name)
	s.pluginStat[name] = false
	s.mu.Unlock()

	s.logger.Success("Released plugin service %s success", name)
	return nil
}

func (s *Service) EnablePlugin(name string) error {
	s.mu.RLock()
	stat, ok := s.pluginStat[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}

	if stat {
		return fmt.Errorf("plugin %s is already enabled", name)
	}
	if err := s.InitPlugin(name); err != nil {
		return err
	}
	return nil
}

func (s *Service) DisablePlugin(name string) error {
	s.mu.RLock()
	stat, ok := s.pluginStat[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}
	if !stat {
		return fmt.Errorf("plugin %s is already disabled", name)
	}
	if err := s.ReleasePlugin(name); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetPluginStat() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statCopy := make(map[string]bool, len(s.pluginStat))
	for k, v := range s.pluginStat {
		statCopy[k] = v
	}
	return statCopy
}

func (s *Service) HasPlugin(name string) (bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stat, ok := s.pluginStat[name]
	return stat, ok
}
