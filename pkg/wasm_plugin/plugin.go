package wasm_plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	extism "github.com/extism/go-sdk"
)

// PluginInstance 表示一个已加载并运行的 Wasm 插件实例
type PluginInstance struct {
	id      string
	path    string
	dataDir string
	plugin  *extism.Plugin
	mu      sync.Mutex // 保护对 Wasm 运行时的单线程调用

	commandReleases []func()
	eventReleases   []func()

	timersMu sync.Mutex
	timers   map[string]context.CancelFunc

	service *Service
}

// newPluginInstance 初始化并编译加载一个 Wasm 插件
func newPluginInstance(s *Service, id string, wasmPath string) (*PluginInstance, error) {
	// 为该插件准备独立的 data/<plugin_id>/ 目录
	dataDir := filepath.Join("data", id)
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("无法获取数据目录绝对路径: %w", err)
	}
	if err := os.MkdirAll(absDataDir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建插件数据目录 %s: %w", absDataDir, err)
	}

	absWasmPath, err := filepath.Abs(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("无法获取 Wasm 文件绝对路径: %w", err)
	}

	s.logger.Debug("Wasm 插件编译加载文件: %s (数据目录: %s)", absWasmPath, absDataDir)

	inst := &PluginInstance{
		id:      id,
		path:    absWasmPath,
		dataDir: absDataDir,
		service: s,
		timers:  make(map[string]context.CancelFunc),
	}

	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: absWasmPath},
		},
		AllowedPaths: map[string]string{
			absDataDir: "/",
			absDataDir: "/data",
		},
		AllowedHosts: []string{"*"}, // 允许插件使用内置 Extism HTTP 请求能力
	}

	config := extism.PluginConfig{
		EnableWasi: true,
	}

	// 创建专属于此插件实例的宿主函数集合
	hostFunctions := s.createHostFunctions(inst)

	ctx := context.Background()
	p, err := extism.NewPlugin(ctx, manifest, config, hostFunctions)
	if err != nil {
		return nil, fmt.Errorf("创建 Wasm 插件实例失败: %w", err)
	}
	inst.plugin = p

	// 若插件导出了初始化钩子函数，则自动调用
	inst.mu.Lock()
	if p.FunctionExists("on_init") {
		if _, _, err := p.Call("on_init", nil); err != nil {
			s.logger.Warning("插件 %s 执行 on_init 发生错误: %v", id, err)
		}
	} else if p.FunctionExists("init") {
		if _, _, err := p.Call("init", nil); err != nil {
			s.logger.Warning("插件 %s 执行 init 发生错误: %v", id, err)
		}
	}
	inst.mu.Unlock()

	return inst, nil
}

// Call 线程安全地调用 Wasm 插件导出的函数
func (inst *PluginInstance) Call(name string, input []byte) ([]byte, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.plugin == nil {
		return nil, fmt.Errorf("插件 %s 已经释放", inst.id)
	}
	if !inst.plugin.FunctionExists(name) {
		return nil, fmt.Errorf("插件 %s 未导出函数 %s", inst.id, name)
	}

	_, out, err := inst.plugin.Call(name, input)
	if err != nil {
		return nil, fmt.Errorf("调用插件 %s 导出函数 %s 失败: %w", inst.id, name, err)
	}
	return out, nil
}

// FunctionExists 检查插件是否导出了指定函数
func (inst *PluginInstance) FunctionExists(name string) bool {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	if inst.plugin == nil {
		return false
	}
	return inst.plugin.FunctionExists(name)
}

// Release 释放插件所占用的所有系统资源（指令注销、事件解绑、HTTP服务清理、Wasm关闭）
func (inst *PluginInstance) Release() {
	// 调用退出钩子
	inst.mu.Lock()
	if inst.plugin != nil {
		if inst.plugin.FunctionExists("on_release") {
			_, _, _ = inst.plugin.Call("on_release", nil)
		} else if inst.plugin.FunctionExists("release") {
			_, _, _ = inst.plugin.Call("release", nil)
		}
	}
	inst.mu.Unlock()

	// 注销所有由该插件注册的消息指令
	for _, fn := range inst.commandReleases {
		if fn != nil {
			fn()
		}
	}
	inst.commandReleases = nil

	// 注销所有由该插件订阅的事件回调
	for _, fn := range inst.eventReleases {
		if fn != nil {
			fn()
		}
	}
	inst.eventReleases = nil

	// 清理该插件注册的所有 HTTP 监听服务
	if inst.service != nil {
		inst.service.cleanupPluginHttpRoutes(inst.id)
	}

	// 取消并停止该插件创建的所有定时任务
	inst.timersMu.Lock()
	for _, cancel := range inst.timers {
		if cancel != nil {
			cancel()
		}
	}
	inst.timers = make(map[string]context.CancelFunc)
	inst.timersMu.Unlock()

	// 关闭 Wasm 模块实例
	inst.mu.Lock()
	if inst.plugin != nil {
		_ = inst.plugin.Close(context.Background())
		inst.plugin = nil
	}
	inst.mu.Unlock()
}

// AddTimer 记录并启动一个受控的定时任务
func (inst *PluginInstance) AddTimer(id string, cancel context.CancelFunc) {
	inst.timersMu.Lock()
	defer inst.timersMu.Unlock()
	if inst.timers == nil {
		inst.timers = make(map[string]context.CancelFunc)
	}
	inst.timers[id] = cancel
}

// RemoveTimer 移除并取消一个定时任务
func (inst *PluginInstance) RemoveTimer(id string) bool {
	inst.timersMu.Lock()
	defer inst.timersMu.Unlock()
	if cancel, ok := inst.timers[id]; ok {
		if cancel != nil {
			cancel()
		}
		delete(inst.timers, id)
		return true
	}
	return false
}

// isReleased 判断插件是否已释放
func (inst *PluginInstance) isReleased() bool {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.plugin == nil
}
