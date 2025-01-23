package go_plugin

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"strings"
)

func (s *Service) initCmd() {
	grb := s.grb

	cmd := grb.Command("plugin")

	_, _ = cmd.SubCommand("lookup").
		Action(func(ctx *command.Context) {
			if id, ok := grb.GetOwner(ctx.BotContext().ID()); ok && id != ctx.SenderID() {
				return
			}
			if err := s.LookupPlugins(); err != nil {
				_, _ = ctx.ReplyText(err)
				return
			}

			_, _ = ctx.ReplyText("Done.")
		}).Build()

	_, _ = cmd.SubCommand("load <name>").
		Action(func(ctx *command.Context) {
			if id, ok := grb.GetOwner(ctx.BotContext().ID()); ok && id != ctx.SenderID() {
				return
			}
			name := ctx.Args["name"]
			if strings.ToLower(name) == "all" {
				for name, stat := range s.pluginStat {
					if stat == true {
						if err := s.InitPlugin(name); err != nil {
							_, _ = ctx.ReplyText(err)
						}
					}
				}

				_, _ = ctx.ReplyText("Done.")
				return
			}

			if err := s.EnablePlugin(name); err != nil {
				_, _ = ctx.ReplyText(err)
				return
			}

			_, _ = ctx.ReplyText("Done.")
		}).Build()

	_, _ = cmd.SubCommand("enable <name>").
		Action(func(ctx *command.Context) {
			if id, ok := grb.GetOwner(ctx.BotContext().ID()); ok && id != ctx.SenderID() {
				return
			}
			name := ctx.Args["name"]
			if strings.ToLower(name) == "all" {
				s.InitPlugins()
				_, _ = ctx.ReplyText("Done.")
				return
			}

			if err := s.EnablePlugin(name); err != nil {
				_, _ = ctx.ReplyText(err)
				return
			}
			_, _ = ctx.ReplyText("Done.")
		}).Build()

	_, _ = cmd.SubCommand("disable <name>").
		Action(func(ctx *command.Context) {
			if id, ok := grb.GetOwner(ctx.BotContext().ID()); ok && id != ctx.SenderID() {
				return
			}
			name := ctx.Args["name"]
			if strings.ToLower(name) == "all" {
				for name, stat := range s.pluginStat {
					if stat == true {
						if err := s.DisablePlugin(name); err != nil {
							_, _ = ctx.ReplyText(err)
						}
					}
				}
				_, _ = ctx.ReplyText("Done.")
				return
			}
			if err := s.DisablePlugin(name); err != nil {
				_, _ = ctx.ReplyText(err)
			}
			_, _ = ctx.ReplyText("Done.")
		}).Build()
}
