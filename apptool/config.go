// Package apptool 收录桌面应用共用的小工具函数。
package apptool

import (
	"os"
	"path/filepath"
)

// DefaultConfigPath 返回应用配置文件路径：
// 工作目录下存在 config.yaml 时优先使用（现场覆盖），
// 否则回退到仓库内 apps/<appName>/config.yaml。
func DefaultConfigPath(appName string) string {
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}
	return filepath.Join("apps", appName, "config.yaml")
}
