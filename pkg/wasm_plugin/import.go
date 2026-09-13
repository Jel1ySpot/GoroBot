package wasm_plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListPlugins 检查指定目录，返回所有以 .wasm 结尾的插件标识数组
func ListPlugins(pluginPath string) ([]string, error) {
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		// 目录不存在时尝试自动创建并返回空列表
		_ = os.MkdirAll(pluginPath, 0755)
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("无法访问插件目录 %s: %w", pluginPath, err)
	}

	var wasmFiles []string
	err := filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("无法遍历文件 %s: %w", path, err)
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".wasm") {
			// 计算相对 pluginPath 的相对路径，作为插件 ID
			relPath, err := filepath.Rel(pluginPath, path)
			if err != nil {
				relPath = info.Name()
			}
			// 统一转为标准斜杠并去除 .wasm 后缀
			id := filepath.ToSlash(strings.TrimSuffix(relPath, filepath.Ext(relPath)))
			wasmFiles = append(wasmFiles, id)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return wasmFiles, nil
}
