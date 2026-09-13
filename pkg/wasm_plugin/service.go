package wasm_plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
)

const (
	DefaultPluginPath         = "plugin/"
	PluginsListTemplateString = `Wasm 插件列表：
-------------------
{{- range $Name, $Stat := .Plugins }}
{{ $Name }}: {{ if $Stat }}✅ 已启用{{ else }}❎ 未启用{{ end }}
{{- end }}`
)

type routeEntry struct {
	pluginID string
	handler  string
	inst     *PluginInstance
}

type managedHttpServer struct {
	addr     string
	server   *http.Server
	mux      *http.ServeMux
	routes   map[string]*routeEntry
	routesMu sync.RWMutex
}

type Service struct {
	grb    *GoroBot.Instant
	logger logger.Inst

	PluginPath string
	mu         sync.RWMutex
	pluginStat map[string]bool
	instances  map[string]*PluginInstance

	activeContexts   map[string]botc.MessageContext
	activeContextsMu sync.RWMutex

	httpServers   map[string]*managedHttpServer
	httpServersMu sync.Mutex
}

func (s *Service) Name() string {
	return "WasmPluginManager"
}

func Create() *Service {
	return &Service{
		PluginPath:     DefaultPluginPath,
		pluginStat:     make(map[string]bool),
		instances:      make(map[string]*PluginInstance),
		activeContexts: make(map[string]botc.MessageContext),
		httpServers:    make(map[string]*managedHttpServer),
	}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	s.grb = grb
	s.logger = grb.GetLogger()

	// 保证插件搜索目录存在
	if _, err := os.Stat(s.PluginPath); os.IsNotExist(err) {
		_ = os.MkdirAll(s.PluginPath, 0755)
	}

	if _, err := s.LookupPlugins(); err != nil {
		return err
	}

	s.InitPlugins()
	s.initCmd()

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	s.mu.RLock()
	var enabled []string
	for name, stat := range s.pluginStat {
		if stat {
			enabled = append(enabled, name)
		}
	}
	s.mu.RUnlock()

	for _, name := range enabled {
		if err := s.ReleasePlugin(name); err != nil {
			s.logger.Failed("释放 Wasm 插件 %s 失败: %v", name, err)
		}
	}

	// 彻底停止所有残留的 HTTP 监听服务
	s.httpServersMu.Lock()
	for addr, srv := range s.httpServers {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.server.Shutdown(ctx)
		cancel()
		delete(s.httpServers, addr)
	}
	s.httpServersMu.Unlock()

	return nil
}

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
	s.logger.Info("开始初始化 Wasm 插件...")

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
				s.logger.Failed("重新初始化插件 %s 释放旧实例失败: %v", name, err)
				continue
			}
		}
		if err := s.InitPlugin(name); err != nil {
			s.logger.Failed("加载 Wasm 插件 %s 失败: %v", name, err)
		}
	}
}

func (s *Service) InitPlugin(name string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("插件 %s 初始化过程中 Panic: %v", name, r)
			s.logger.Failed(err.Error())
		}
	}()

	wasmPath := filepath.Join(s.PluginPath, name+".wasm")
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		return fmt.Errorf("wasm 文件不存在: %s", wasmPath)
	}

	s.logger.Debug("正在加载 Wasm 插件: %s (%s)", name, wasmPath)
	inst, err := newPluginInstance(s, name, wasmPath)
	if err != nil {
		return fmt.Errorf("初始化插件 %s 失败: %w", name, err)
	}

	s.mu.Lock()
	s.instances[name] = inst
	s.pluginStat[name] = true
	s.mu.Unlock()

	s.logger.Success("Wasm 插件 %s 加载成功", name)
	return nil
}

func (s *Service) ReleasePlugin(name string) (err error) {
	s.mu.RLock()
	_, ok := s.pluginStat[name]
	inst := s.instances[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("插件 %s 未找到", name)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("插件 %s 释放过程中 Panic: %v", name, r)
			s.logger.Failed(err.Error())
		}
	}()

	if inst != nil {
		s.logger.Debug("正在释放 Wasm 插件 %s", name)
		inst.Release()
	}

	s.mu.Lock()
	delete(s.instances, name)
	s.pluginStat[name] = false
	s.mu.Unlock()

	s.logger.Success("Wasm 插件 %s 已成功释放", name)
	return nil
}

func (s *Service) EnablePlugin(name string) error {
	s.mu.RLock()
	stat, ok := s.pluginStat[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("插件 %s 未找到", name)
	}
	if stat {
		return fmt.Errorf("插件 %s 已经处于启用状态", name)
	}
	return s.InitPlugin(name)
}

func (s *Service) DisablePlugin(name string) error {
	s.mu.RLock()
	stat, ok := s.pluginStat[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("插件 %s 未找到", name)
	}
	if !stat {
		return fmt.Errorf("插件 %s 已经处于禁用状态", name)
	}
	return s.ReleasePlugin(name)
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

func (s *Service) getBotContext(id string) botc.BotContext {
	if id != "" {
		if ctx := s.grb.GetContext(id); ctx != nil {
			return ctx
		}
	}
	for _, name := range []string{"telegram", "onebot", "qq", "lagrange", "qbot"} {
		if ctx := s.grb.GetContext(name); ctx != nil {
			return ctx
		}
	}
	return nil
}

// registerHttpRoute 注册外部 HTTP 监听路径
func (s *Service) registerHttpRoute(inst *PluginInstance, addr string, path string, handler string) error {
	s.httpServersMu.Lock()
	defer s.httpServersMu.Unlock()

	managed, ok := s.httpServers[addr]
	if !ok {
		mux := http.NewServeMux()
		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		managed = &managedHttpServer{
			addr:   addr,
			server: server,
			mux:    mux,
			routes: make(map[string]*routeEntry),
		}
		s.httpServers[addr] = managed

		go func() {
			s.logger.Info("启动插件 HTTP 监听服务: %s", addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Failed("插件 HTTP 监听服务 %s 异常退出: %v", addr, err)
			}
		}()
	}

	managed.routesMu.Lock()
	defer managed.routesMu.Unlock()

	entry := &routeEntry{
		pluginID: inst.id,
		handler:  handler,
		inst:     inst,
	}
	managed.routes[path] = entry

	managed.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		managed.routesMu.RLock()
		targetEntry, exists := managed.routes[path]
		managed.routesMu.RUnlock()

		if !exists || targetEntry.inst == nil {
			http.NotFound(w, r)
			return
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		headers := make(map[string]string)
		for k := range r.Header {
			headers[k] = r.Header.Get(k)
		}

		incoming := HttpIncomingRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Headers: headers,
			Body:    string(bodyBytes),
		}

		inBytes, err := json.Marshal(incoming)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		out, err := targetEntry.inst.Call(targetEntry.handler, inBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var outgoing HttpOutgoingResponse
		if err := json.Unmarshal(out, &outgoing); err != nil {
			// 若返回值非 JSON，直接将原始输出作为响应正文返回
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(out)
			return
		}

		for k, v := range outgoing.Headers {
			w.Header().Set(k, v)
		}
		if outgoing.StatusCode <= 0 {
			outgoing.StatusCode = http.StatusOK
		}
		w.WriteHeader(outgoing.StatusCode)
		_, _ = w.Write([]byte(outgoing.Body))
	})

	return nil
}

// cleanupPluginHttpRoutes 清理特定插件所注册的 HTTP 路由，若某端口已无任何路由，则自动关闭服务
func (s *Service) cleanupPluginHttpRoutes(pluginID string) {
	s.httpServersMu.Lock()
	defer s.httpServersMu.Unlock()

	for addr, managed := range s.httpServers {
		managed.routesMu.Lock()
		for path, entry := range managed.routes {
			if entry.pluginID == pluginID {
				delete(managed.routes, path)
			}
		}
		remaining := len(managed.routes)
		managed.routesMu.Unlock()

		if remaining == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = managed.server.Shutdown(ctx)
			cancel()
			delete(s.httpServers, addr)
			s.logger.Info("插件 HTTP 监听服务 %s 已关闭 (无活跃路由)", addr)
		}
	}
}
