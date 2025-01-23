package go_plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListPlugins 检查 plugin/ 目录，并返回包含插件名的数组
func ListPlugins(pluginPath string) ([]string, error) {
	// 定位到 plugin/ 目录
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("无法访问目录 %s: %v", pluginPath, err)
	}

	// 遍历目录下的文件
	var soFiles []string
	err := filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("无法遍历文件 %s: %v", path, err)
		}
		// 检查是否是 .so 文件
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".so") {
			soFiles = append(soFiles, strings.TrimRight(path, ".so"))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return soFiles, nil
}
