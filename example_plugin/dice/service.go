package dice

import (
	"fmt"
	"math/rand"
	"strconv"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
)

type Service struct {
	bot *GoroBot.Instant

	releaseFunc []func()
}

func (s *Service) Name() string {
	return "Dice"
}

func Create() *Service {
	return &Service{}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.bot = grb

	delFn, _ := grb.Command("dice").
		Argument("upper_bound", command.Number, false, "骰子点数上限，默认为 6 （骰子点数范围 1 ~ Upper Bound）").
		Alias(`^d(\d+)$`, func(ctx *command.Context) *command.Context {
			ctx.KvArgs["upper_bound"] = ctx.String()[1:]
			return ctx
		}).
		Action(func(ctx *command.Context) error {
			limit, err := strconv.Atoi(ctx.KvArgs["upper_bound"])
			if err != nil {
				_, _ = ctx.ReplyText("无效的骰子点数上限: ", err.Error())
				return err
			}

			if limit <= 0 {
				_, _ = ctx.ReplyText("骰子点数上限必须大于 0")
				return fmt.Errorf("invalid upper bound")
			}

			_, _ = ctx.ReplyText(strconv.Itoa(rand.Intn(limit) + 1))
			return nil
		}).
		Build()

	s.releaseFunc = append(s.releaseFunc, delFn)

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	for _, fn := range s.releaseFunc {
		fn()
	}
	return nil
}
