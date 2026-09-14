package wasm_plugin

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

	cmd := grb.Command("wasm").Description("WebAssembly (Extism) 插件管理指令")

	// 1. wasm lookup
	_, _ = cmd.SubCommand("lookup").
		Description("扫描插件目录并发现新的 Wasm 插件").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			n, err := s.LookupPlugins()
			if err != nil {
				_, _ = ctx.ReplyText("扫描插件目录失败: ", err.Error())
				return err
			}

			_, _ = ctx.ReplyText("扫描完成，共找到 ", n, " 个 Wasm 插件。")
			return nil
		}).Build()

	// 2. wasm load <name|all>
	_, _ = cmd.SubCommand("load").
		Description("加载或重新加载 Wasm 插件").
		Argument("name", command.String, true, "插件名称或 all").
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
							_, _ = ctx.ReplyText("插件 ", pName, " 释放失败: ", err.Error())
						}
					}
					if err := s.InitPlugin(pName); err != nil {
						_, _ = ctx.ReplyText("插件 ", pName, " 加载失败: ", err.Error())
					}
				}
				_, _ = ctx.ReplyText("全部插件重载完成。")
				return nil
			}

			stat, ok := s.HasPlugin(name)
			if !ok {
				_, _ = ctx.ReplyText("未找到插件: ", name)
				return fmt.Errorf("plugin not found: %s", name)
			}

			if stat {
				if err := s.ReleasePlugin(name); err != nil {
					_, _ = ctx.ReplyText("释放旧实例失败: ", err.Error())
					return err
				}
			}

			if err := s.InitPlugin(name); err != nil {
				_, _ = ctx.ReplyText("加载插件失败: ", err.Error())
				return err
			}

			_, _ = ctx.ReplyText("插件 ", name, " 加载成功。")
			return nil
		}).Build()

	// 3. wasm enable <name|all>
	_, _ = cmd.SubCommand("enable").
		Description("启用指定或全部 Wasm 插件").
		Argument("name", command.String, true, "插件名称或 all").
		Action(func(ctx *command.Context) error {
			if !s.isOwner(ctx) {
				_, _ = ctx.ReplyText("Permission denied.")
				return fmt.Errorf("permission denied")
			}
			name := ctx.KvArgs["name"]
			if strings.ToLower(name) == "all" {
				stats := s.GetPluginStat()
				for pName, stat := range stats {
					if !stat {
						if err := s.EnablePlugin(pName); err != nil {
							_, _ = ctx.ReplyText("插件 ", pName, " 启用失败: ", err.Error())
						}
					}
				}
				_, _ = ctx.ReplyText("全部插件启用完成。")
				return nil
			}

			if err := s.EnablePlugin(name); err != nil {
				_, _ = ctx.ReplyText("启用失败: ", err.Error())
				return err
			}
			_, _ = ctx.ReplyText("插件 ", name, " 已启用。")
			return nil
		}).Build()

	// 4. wasm disable <name|all>
	_, _ = cmd.SubCommand("disable").
		Description("禁用指定或全部 Wasm 插件").
		Argument("name", command.String, true, "插件名称或 all").
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
							_, _ = ctx.ReplyText("插件 ", pName, " 禁用失败: ", err.Error())
						}
					}
				}
				_, _ = ctx.ReplyText("全部插件已禁用。")
				return nil
			}

			if err := s.DisablePlugin(name); err != nil {
				_, _ = ctx.ReplyText("禁用失败: ", err.Error())
				return err
			}
			_, _ = ctx.ReplyText("插件 ", name, " 已禁用。")
			return nil
		}).Build()

	// 5. wasm list
	_, _ = cmd.SubCommand("list").
		Description("列出所有 Wasm 插件及运行状态").
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
		s.logger.Failed("构建 wasm 指令失败: %v", err)
	}
}

func init() {
	pluginsListTemplate = template.Must(template.New("pluginsListTemplate").Parse(PluginsListTemplateString))
}
