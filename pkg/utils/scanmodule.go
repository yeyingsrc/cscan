package utils

// EnabledScanModules 与 worker.executeTask 的实际执行条件严格对齐。
// 只有显式 enable==true 才视为启用，缺失 / nil / false 一律不计数。
var EnabledScanModules = []string{
	"domainscan",
	"portscan",
	"portidentify",
	"fingerprint",
	"brutescan",
	"dirscan",
	"jsfinder",
	"pocscan",
}

// CountEnabledModules 统计 taskConfig 中启用的扫描模块数量。
// portscan 默认启用（配置缺失或 nil 时计为启用），其余模块需显式 enable==true。
// 返回真实计数，可能为 0；调用方需自行处理零值（避免除零或欠计数）。
func CountEnabledModules(configMap map[string]any) int {
	if configMap == nil {
		return 0
	}
	count := 0
	for _, mod := range EnabledScanModules {
		if mod == "portscan" {
			// portscan 默认启用：配置缺失或 nil 时计为启用
			if ps, ok := configMap["portscan"].(map[string]any); ok {
				if enable, ok := ps["enable"].(bool); ok && enable {
					count++
				}
			} else {
				count++
			}
			continue
		}
		m, ok := configMap[mod].(map[string]any)
		if !ok {
			continue
		}
		if enable, ok := m["enable"].(bool); ok && enable {
			count++
		}
	}
	return count
}
