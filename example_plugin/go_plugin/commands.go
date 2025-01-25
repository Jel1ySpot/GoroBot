package go_plugin

import (
	"bytes"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"strings"
	"text/template"
)

func (s *Service) initCmd() {
	grb := s.grb

	cmd := grb.Command("plugin")

	_, _ = cmd.SubCommand("lookup").
		Action(func(ctx *command.Context) {
			if id, ok := grb.GetOwner(ctx.BotContext().ID()); ok && id != ctx.SenderID() {
				_, _ = ctx.ReplyText("Permission denied.")
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
				_, _ = ctx.ReplyText("Permission denied.")
				return
			}
			name := ctx.Args["name"]
			if strings.ToLower(name) == "all" {
				for name, stat := range s.pluginStat {
					if stat {
						if err := s.ReleasePlugin(name); err != nil {
							_, _ = ctx.ReplyText(err)
							continue
						}
					}
					if err := s.InitPlugin(name); err != nil {
						_, _ = ctx.ReplyText(err)
					}
				}

				_, _ = ctx.ReplyText("Done.")
				return
			}

			if _, ok := s.pluginStat[name]; !ok {
				_, _ = ctx.ReplyText("Plugin not found.")
				return
			}

			if s.pluginStat[name] {
				if err := s.ReleasePlugin(name); err != nil {
					_, _ = ctx.ReplyText(err)
					return
				}
			}

			if err := s.InitPlugin(name); err != nil {
				_, _ = ctx.ReplyText(err)
				return
			}

			_, _ = ctx.ReplyText("Done.")
		}).Build()

	_, _ = cmd.SubCommand("enable <name>").
		Action(func(ctx *command.Context) {
			if id, ok := grb.GetOwner(ctx.BotContext().ID()); ok && id != ctx.SenderID() {
				_, _ = ctx.ReplyText("Permission denied.")
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
				_, _ = ctx.ReplyText("Permission denied.")
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

	_, _ = cmd.SubCommand("list").
		Action(func(ctx *command.Context) {
			const temp = `插件列表：
{{- range $Name, $Stat := .Plugins }}
{{ $Name }}：
{{- if $Stat -}}
✅
{{- else -}}
❎
{{- end -}}
{{ end }}`
			var buf bytes.Buffer

			if err := template.Must(template.New("temp").Parse(temp)).Execute(&buf, map[string]any{
				"Plugins": s.pluginStat,
			}); err != nil {
				_, _ = ctx.ReplyText(err)
				return
			}
			_, _ = ctx.ReplyText(buf.String())
		}).Build()

	if _, err := cmd.Build(); err != nil {
		s.logger.Failed("Failed to build command: %v", err)
	}
}
