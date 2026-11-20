package util

import (
	"testing"
)

// TestLoaderRegistryUniform 模板规范命名回归防护：
//   - 所有模板目录必须为 loaderN（禁止再混入带技术信息的目录名，防止"暴露特征"复发）
//   - 编号必须从 loader1 连续无空洞
//   - 关键角色常量必须能解析到真实模板（模板集增删后不得漂移）
//
// 历史 bug：模板集从 loader1..loader53 收缩到 loader1..loader20 后，
// LoaderWindowsDefault/LoaderUnixMmap/LoaderUnixCallback 常量未同步，
// 导致"未指定运行模式"与"Linux/macOS 目标"两条路径稳定失败。
// 因此这里不再断言固定数量/固定编号，只断言"角色可解析"，由 DefaultWindowsLoader/
// DefaultUnixLoader 负责在首选名缺失时回退。
func TestLoaderRegistryUniform(t *testing.T) {
	names := allModuleNames()
	if len(names) == 0 {
		t.Fatal("loader 模板数量为 0")
	}

	prev := 0
	for _, name := range names {
		if !HasLoaderTemplate(name) {
			t.Fatalf("模板 %s 未注册或为空", name)
		}
		n, ok := loaderNumber(name)
		if !ok {
			t.Fatalf("发现非规范模板目录名（疑似技术名泄露）: %q", name)
		}
		if n != prev+1 {
			t.Fatalf("loader 编号空洞: 期望 loader%d，实际为 %s", prev+1, name)
		}
		prev = n
	}
	t.Logf("loader 模板共 %d 个，命名规范且连续（loader1..loader%d）", len(names), prev)

	// 默认 Windows 模板必须可解析，且指向真实存在的模板
	defWin, ok := DefaultWindowsLoader()
	if !ok {
		t.Fatal("DefaultWindowsLoader 无法解析出任何模板")
	}
	if !HasLoaderTemplate(defWin) {
		t.Fatalf("默认 Windows 模板 %q 不存在", defWin)
	}
	if !IsWindowsLoader(defWin) {
		t.Fatalf("默认模板 %q 不应归类为 Unix 模板", defWin)
	}
	t.Logf("默认 Windows 模板: %s（首选常量 %s）", defWin, LoaderWindowsDefault)

	// 白加黑模板必须存在（前端白加黑模式依赖该编号）
	if !HasLoaderTemplate(LoaderBaiJiaHei) {
		t.Fatalf("白加黑模板常量 %q 未命中任何模板（编号漂移？）", LoaderBaiJiaHei)
	}

	// Unix 模板：存在则必须能解析；不存在则不解析出 Windows 模板冒充
	if unixLoader, ok := DefaultUnixLoader(); ok {
		if IsWindowsLoader(unixLoader) {
			t.Fatalf("Unix 模板解析结果 %q 被归类为 Windows 模板", unixLoader)
		}
		t.Logf("默认 Unix 模板: %s", unixLoader)
	} else {
		t.Logf("当前模板集不含 Linux/macOS 模板（DefaultUnixLoader 返回 false，符合预期）")
	}

	// 平台过滤与常量集合一致
	all := allModuleNames()
	win := FilterLoadersByTargetOS(all, "windows")
	lin := FilterLoadersByTargetOS(all, "linux")
	if len(win)+len(lin) != len(all) {
		t.Fatal("平台过滤后模板数不一致")
	}
	for _, name := range win {
		if IsWindowsLoader(name) == false {
			t.Fatalf("Windows 过滤结果混入 Unix 模板: %s", name)
		}
	}
	for _, name := range lin {
		if !unixLoaderNames[name] {
			t.Fatalf("非 Windows 过滤结果混入 Windows 模板: %s", name)
		}
	}
	// 注意：只能按"注册表里真实存在的 Unix 模板数"校验，不能直接用 unixLoaderNames 的长度
	// （该表登记的是角色名，模板集可能已不含对应模板）。
	existingUnix := 0
	for _, name := range all {
		if unixLoaderNames[name] {
			existingUnix++
		}
	}
	if len(win) != len(all)-existingUnix {
		t.Fatalf("Windows 模板数异常: windows=%d all=%d unix=%d", len(win), len(all), existingUnix)
	}
	t.Logf("平台分类正常: windows=%d unix=%d", len(win), len(lin))
}
