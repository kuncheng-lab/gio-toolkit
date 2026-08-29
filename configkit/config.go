// Package configkit 提供桌面应用 YAML 配置的通用加载/保存：
// 以代码内 defaults 为基线，配置文件只覆盖出现的字段；
// 首次运行（文件不存在）自动用 defaults 落盘，方便现场直接改文件。
package configkit

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load 用 path 处的 YAML 覆盖 cfg 中已填好的默认值。
// cfg 必须传入已填充默认值的指针；文件不存在时用当前 cfg 落盘并返回 nil。
//
//	cfg := defaultConfig()
//	if err := configkit.Load(path, &cfg); err != nil {
//	    log.Fatal(err)
//	}
func Load(path string, cfg any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Save(path, cfg)
		}
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

// Save 将 cfg 序列化为 YAML 写入 path（必要时创建父目录）。
func Save(path string, cfg any) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}
