package wasm_plugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	extism "github.com/extism/go-sdk"
)

func TestListPlugins(t *testing.T) {
	tempDir := t.TempDir()

	// 创建模拟 wasm 文件
	f1 := filepath.Join(tempDir, "plugin_a.wasm")
	f2 := filepath.Join(tempDir, "plugin_b.WASM")
	f3 := filepath.Join(tempDir, "ignore.txt")
	_ = os.WriteFile(f1, []byte("wasm1"), 0644)
	_ = os.WriteFile(f2, []byte("wasm2"), 0644)
	_ = os.WriteFile(f3, []byte("text"), 0644)

	// 创建子目录下的插件
	subDir := filepath.Join(tempDir, "category")
	_ = os.MkdirAll(subDir, 0755)
	f4 := filepath.Join(subDir, "nested.wasm")
	_ = os.WriteFile(f4, []byte("wasm4"), 0644)

	plugins, err := ListPlugins(tempDir)
	if err != nil {
		t.Fatalf("ListPlugins 失败: %v", err)
	}

	expected := map[string]bool{
		"plugin_a":        true,
		"plugin_b":        true,
		"category/nested": true,
	}

	if len(plugins) != len(expected) {
		t.Fatalf("预期找到 %d 个插件，实际找到 %d 个: %v", len(expected), len(plugins), plugins)
	}

	for _, p := range plugins {
		normalized := filepath.ToSlash(p)
		if !expected[normalized] {
			t.Errorf("未预期的插件: %s", normalized)
		}
	}
}

func TestResolveDataPath(t *testing.T) {
	s := Create()
	inst := &PluginInstance{
		id:      "test_plugin",
		dataDir: filepath.Clean("data/test_plugin"),
	}

	// 正常子路径
	p1, err := s.resolveDataPath(inst, "config.json")
	if err != nil {
		t.Fatalf("解析正常路径失败: %v", err)
	}
	if !strings.HasSuffix(p1, filepath.Join("data", "test_plugin", "config.json")) {
		t.Errorf("路径解析不符合预期: %s", p1)
	}

	// 路径穿越攻击
	_, err = s.resolveDataPath(inst, "../../etc/passwd")
	if err == nil {
		t.Errorf("预期拒绝路径穿越访问，但未报错")
	}
}

func TestHttpListenRoute(t *testing.T) {
	s := Create()
	s.logger = &logger.DefaultLogger{LogLevel: logger.Info}

	inst := &PluginInstance{
		id:      "web_plugin",
		service: s,
	}

	// 查找空闲端口
	addr := "127.0.0.1:28989"
	path := "/test-webhook"

	err := s.registerHttpRoute(inst, addr, path, "on_request")
	if err != nil {
		t.Fatalf("注册路由失败: %v", err)
	}

	// 等待服务启动
	time.Sleep(300 * time.Millisecond)

	// 请求不存在的路径 -> 404
	resp404, err := http.Get("http://" + addr + "/not-found")
	if err != nil {
		t.Fatalf("HTTP 请求失败: %v", err)
	}
	if resp404.StatusCode != http.StatusNotFound {
		t.Errorf("预期 404，实际状态码: %d", resp404.StatusCode)
	}

	// 清理路由
	s.cleanupPluginHttpRoutes("web_plugin")
	time.Sleep(100 * time.Millisecond)
}

func TestExtismWasmExecutionWithFS(t *testing.T) {
	// 尝试寻找 extism go-sdk 中自带的 count_vowels.wasm 测试
	modPath := filepath.Join(os.Getenv("GOPATH"), "pkg", "mod")
	var wasmFile string

	// 在 mod 缓存中查找 count_vowels.wasm
	err := filepath.Walk(modPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, "count_vowels.wasm") {
			wasmFile = path
			return io.EOF // 终止搜索
		}
		return nil
	})
	if err != nil && err != io.EOF {
		t.Skip("未找到可用的测试 wasm 文件")
	}

	if wasmFile == "" {
		t.Skip("本地未找到 count_vowels.wasm，跳过 Wasm 调用测试")
	}

	// 测试加载该 wasm
	tempDataDir := t.TempDir()
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: wasmFile},
		},
		AllowedPaths: map[string]string{
			tempDataDir: "/data",
		},
		AllowedHosts: []string{"*"},
	}

	config := extism.PluginConfig{
		EnableWasi: true,
	}

	p, err := extism.NewPlugin(context.Background(), manifest, config, nil)
	if err != nil {
		t.Fatalf("初始化 Wasm 插件失败: %v", err)
	}
	defer p.Close(context.Background())

	_, out, err := p.Call("count_vowels", []byte("Hello World!"))
	if err != nil {
		t.Fatalf("调用 count_vowels 失败: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("解析 count_vowels 输出失败: %v", err)
	}

	if int(res["count"].(float64)) != 3 {
		t.Errorf("预期 3 个元音字母，实际返回: %v", res["count"])
	}
}

func TestServiceLifecycle(t *testing.T) {
	tempPluginDir := t.TempDir()

	grb := GoroBot.Create()
	srv := Create()
	srv.PluginPath = tempPluginDir

	if err := srv.Init(grb); err != nil {
		t.Fatalf("服务初始化失败: %v", err)
	}

	if srv.Name() != "WasmPluginManager" {
		t.Errorf("服务名称不匹配: %s", srv.Name())
	}

	if err := srv.Release(grb); err != nil {
		t.Fatalf("服务释放失败: %v", err)
	}
}
