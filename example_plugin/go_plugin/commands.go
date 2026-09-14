package go_plugin

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

var (
	pluginsListTemplate *template.Template
)

func (s *Service) isOwner(ctx *command.Context) bool {
	if ctx.Authority() >= entity.Owner {
		return true
	}
	var ctxID string
	if botCtx := ctx.BotContext(); botCtx != nil {
		ctxID = botCtx.ID()
	}
	if s.grb.IsOwner(ctxID, ctx.Protocol(), ctx.SenderID()) {
		return true
	}
	id, ok := s.grb.GetOwner(ctxID)
	return ok && id == ctx.SenderID()
}

func (s *Service) initCmd() {
	grb := s.grb

	cmd := grb.Command("plugin")

	_, _ = cmd.SubCommand("lookup").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			n, err := s.LookupPlugins()
			if err != nil {
				return err
			}

			_, _ = ctx.ReplyText("Done. Found ", n, " plugins.")
			return nil
		}).Build()

	_, _ = cmd.SubCommand("load").
		Argument("name", command.String, true, "需要加载的插件").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			name := ctx.KvArgs["name"]
			if strings.ToLower(name) == "all" {
				stats := s.GetPluginStat()
				for pName, stat := range stats {
					if stat {
						if err := s.ReleasePlugin(pName); err != nil {
							_, _ = ctx.ReplyText("plugin ", pName, " release failed with error: ", err.Error())
						}
					}
					if err := s.InitPlugin(pName); err != nil {
						_, _ = ctx.ReplyText("plugin ", pName, " initialization failed with error: ", err.Error())
					}
				}

				_, _ = ctx.ReplyText("Done.")
				return nil
			}

			stat, ok := s.HasPlugin(name)
			if !ok {
				return fmt.Errorf("plugin not found: %s", name)
			}

			if stat {
				if err := s.ReleasePlugin(name); err != nil {
					return err
				}
			}

			if err := s.InitPlugin(name); err != nil {
				return err
			}

			_, _ = ctx.ReplyText("Done.")
			return nil
		}).Build()

	_, _ = cmd.SubCommand("enable").
		Argument("name", command.String, true, "需要启用的插件").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			name := ctx.KvArgs["name"]
			if strings.ToLower(name) == "all" {
				s.InitPlugins()
				_, _ = ctx.ReplyText("Done.")
				return nil
			}

			if err := s.EnablePlugin(name); err != nil {
				return fmt.Errorf("plugin enable failed: %s", err)
			}
			_, _ = ctx.ReplyText("Done.")
			return nil
		}).Build()

	_, _ = cmd.SubCommand("disable").
		Argument("name", command.String, true, "需要禁用的插件").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			name := ctx.KvArgs["name"]
			if strings.ToLower(name) == "all" {
				stats := s.GetPluginStat()
				for pName, stat := range stats {
					if stat {
						if err := s.DisablePlugin(pName); err != nil {
							_, _ = ctx.ReplyText(err)
						}
					}
				}
				_, _ = ctx.ReplyText("Done.")
				return nil
			}
			if err := s.DisablePlugin(name); err != nil {
				_, _ = ctx.ReplyText(err)
			}
			_, _ = ctx.ReplyText("Done.")
			return nil
		}).Build()

	_, _ = cmd.SubCommand("list").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			var buf bytes.Buffer

			if err := pluginsListTemplate.Execute(&buf, map[string]any{
				"Plugins": s.GetPluginStat(),
			}); err != nil {
				return err
			}
			_, _ = ctx.ReplyText(buf.String())
			return nil
		}).Build()

	if _, err := cmd.Build(); err != nil {
		s.logger.Failed("Failed to build command: %v", err)
	}
}

func init() {
	pluginsListTemplate = template.Must(template.New("pluginsListTemplate").Parse(PluginsListTemplateString))
}
