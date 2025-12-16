package tests

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
)

type Service struct {
	bot         *GoroBot.Instant
	releaseFunc []func()
}

func (s *Service) Name() string { return "Tests" }

func Create() *Service {
	return &Service{}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb

	if err := s.CommandsRegistry(); err != nil {
		return err
	}

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
