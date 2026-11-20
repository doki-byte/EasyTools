package cloudSecurity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"EasyTools/app/model"

	bccbcc "github.com/baidubce/bce-sdk-go/services/bcc"
	bccapi "github.com/baidubce/bce-sdk-go/services/bcc/api"
	oos "github.com/baidubce/bce-sdk-go/services/oos"
	oosmodel "github.com/baidubce/bce-sdk-go/services/oos/model"
	"github.com/tidwall/gjson"
)

// baiduListInstances 使用百度云 BCC Go SDK 列举实例，并转换为前端统一结构。
// 说明：
//   - 仅覆盖 BCC 云服务器实例；轻量应用服务器 LS 暂无单独 API 支持，这里假定在同一账号下通过 BCC 统一管理。
//   - 若未来需额外拉取 LS 专有接口，可在此函数中追加合并逻辑。
func baiduListInstances(cr *model.CloudServerCredential, regions []string) (map[string]interface{}, error) {
	ak := strings.TrimSpace(cr.AccessKey)
	sk := strings.TrimSpace(cr.SecretKey)
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("百度云需填写 AccessKey 与 SecretKey")
	}
	cli, err := bccbcc.NewClient(ak, sk, "")
	if err != nil {
		return nil, err
	}

	var blocks []csRegionBlock
	for _, rid := range regions {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		// 按地域设置 Endpoint 与 Region；形如 bcc.bj.baidubce.com
		cli.Config.Region = rid
		cli.Config.Endpoint = fmt.Sprintf("bcc.%s.baidubce.com", rid)

		args := &bccapi.ListInstanceArgs{
			MaxKeys: 1000,
		}
		res, err := bccapi.ListInstances(cli, args)
		if err != nil || res == nil {
			continue
		}
		// Marshal 成 JSON，使用 gjson 提取字段，避免强耦合内部结构定义。
		b, err := json.Marshal(res)
		if err != nil {
			continue
		}
		instances := gjson.GetBytes(b, "instances").Array()
		if len(instances) == 0 {
			instances = gjson.GetBytes(b, "Instances").Array()
		}
		if len(instances) == 0 {
			continue
		}
		rows := make([]csInstanceRow, 0, len(instances))
		for _, it := range instances {
			id := strings.TrimSpace(it.Get("id").String())
			if id == "" {
				id = strings.TrimSpace(it.Get("instanceId").String())
			}
			if id == "" {
				continue
			}
			host := strings.TrimSpace(it.Get("instanceName").String())
			if host == "" {
				host = strings.TrimSpace(it.Get("name").String())
			}
			if host == "" {
				host = id
			}
			pub := firstNonEmptyStr(
				it.Get("publicIp").String(),
				it.Get("eip").String(),
			)
			priv := firstNonEmptyStr(
				it.Get("internalIp").String(),
				it.Get("privateIp").String(),
			)
			cpu := it.Get("cpu").Int()
			if cpu == 0 {
				cpu = it.Get("cpuCount").Int()
			}
			memMB := it.Get("memory").Int()
			config := ""
			if cpu > 0 && memMB > 0 {
				// BCC 常用单位为 MB
				config = fmt.Sprintf("%d核 %dGB", cpu, memMB/1024)
			} else {
				config = strings.TrimSpace(it.Get("instanceType").String())
			}
			osStr := firstNonEmptyStr(
				it.Get("osType").String(),
				it.Get("imageId").String(),
			)
			status := strings.TrimSpace(it.Get("status").String())
			created := strings.TrimSpace(it.Get("createTime").String())
			expire := strings.TrimSpace(it.Get("expireTime").String())

			rows = append(rows, csInstanceRow{
				InstanceID: id,
				HostName:   host,
				PublicIP:   strings.TrimSpace(pub),
				PrivateIP:  strings.TrimSpace(priv),
				Config:     config,
				OS:         osStr,
				Status:     status,
				CreatedAt:  created,
				ExpiredAt:  expire,
			})
		}
		if len(rows) == 0 {
			continue
		}
		blocks = append(blocks, csRegionBlock{
			RegionID:   rid,
			RegionName: rid,
			Instances:  rows,
		})
	}
	return map[string]interface{}{"regions": blocks}, nil
}

func baiduRunCommand(cr *model.CloudServerCredential, regionID, instanceID, command, sysType string, waitSec int) (map[string]string, error) {
	ak := strings.TrimSpace(cr.AccessKey)
	sk := strings.TrimSpace(cr.SecretKey)
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("百度云需填写 AccessKey 与 SecretKey")
	}
	if waitSec < 10 {
		waitSec = 60
	}
	if waitSec > 1800 {
		waitSec = 1800
	}

	// 支持扩展 JSON 手动覆盖，默认按地域自动拼接 endpoint，仅 AK/SK 也可执行。
	endpoint := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "baidu_oos_endpoint").String())
	if endpoint == "" {
		endpoint = strings.TrimSpace(gjson.Get(cr.ExtraJSON, "baidu_ca_endpoint").String())
	}
	if endpoint == "" {
		rid := strings.TrimSpace(regionID)
		if rid == "" {
			rid = "bj"
		}
		endpoint = fmt.Sprintf("oos.%s.baidubce.com", rid)
	}

	cli, err := oos.NewClient(ak, sk, endpoint)
	if err != nil {
		return map[string]string{
			"stdout":    "",
			"stderr":    "",
			"invoke_id": "",
			"error":     "百度云客户端初始化失败，请检查 AK/SK 或地域",
		}, nil
	}

	ops := baiduRunCommandCandidates(instanceID, command, sysType, waitSec)
	detectedOps := baiduDetectCommandOperators(cli)
	if len(detectedOps) > 0 {
		auto := make([]*oosmodel.Operator, 0, len(detectedOps))
		for i, op := range detectedOps {
			auto = append(auto, baiduBuildRunOperator(op, fmt.Sprintf("run_command_auto_%d", i+1), instanceID, command, sysType, waitSec))
		}
		ops = append(auto, ops...)
	}
	var execID string
	var runErr error
	triedOps := make([]string, 0, len(ops))
	for _, op := range ops {
		triedOps = append(triedOps, op.Operator)
		req := &oosmodel.Execution{
			Description: fmt.Sprintf("EasyTools-%d", time.Now().Unix()),
			Template: &oosmodel.Template{
				Name:      fmt.Sprintf("et-baidu-cmd-%d", time.Now().UnixNano()),
				Operators: []*oosmodel.Operator{op},
				Linear:    true,
			},
		}
		resp, err := cli.CreateExecution(req)
		if err != nil {
			runErr = err
			continue
		}
		if resp == nil || resp.Result == nil || strings.TrimSpace(resp.Result.ID) == "" {
			runErr = fmt.Errorf("CreateExecution 未返回执行 ID")
			continue
		}
		execID = strings.TrimSpace(resp.Result.ID)
		runErr = nil
		break
	}
	if runErr != nil {
		detectedPreview := ""
		if len(detectedOps) > 0 {
			detectedPreview = "；检测到可用operator=" + strings.Join(detectedOps, ",")
		}
		return map[string]string{
			"stdout":    "",
			"stderr":    "",
			"invoke_id": "",
			"error":     "百度云命令创建失败（operator不匹配）" + detectedPreview + "；已尝试=" + strings.Join(uniqueNonEmpty(triedOps), ","),
		}, nil
	}

	deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
	var stdout, stderr string
	done := false
	for time.Now().Before(deadline) {
		detail, err := cli.GetExecutionDetail(execID)
		if err != nil || detail == nil || detail.Result == nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		state := strings.ToUpper(strings.TrimSpace(detail.Result.State))
		if state == "RUNNING" || state == "PROCESSING" || state == "PENDING" || state == "READY" || state == "" {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		for _, t := range detail.Result.Tasks {
			if t == nil {
				continue
			}
			taskStdout, taskStderr := baiduExtractTaskOutput(t)
			if taskStdout != "" {
				if stdout == "" {
					stdout = taskStdout
				} else {
					stdout += "\n" + taskStdout
				}
			}
			if taskStderr != "" {
				if stderr == "" {
					stderr = taskStderr
				} else {
					stderr += "\n" + taskStderr
				}
			}
			for _, lg := range t.Log {
				if lg == nil {
					continue
				}
				msg := strings.TrimSpace(lg.Msg)
				if msg == "" {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(lg.Level), "error") {
					if stderr == "" {
						stderr = msg
					} else {
						stderr += "\n" + msg
					}
				} else {
					if stdout == "" {
						stdout = msg
					} else {
						stdout += "\n" + msg
					}
				}
			}
			if t.Reason != "" && stderr == "" {
				stderr = strings.TrimSpace(t.Reason)
			}
		}
		if state != "SUCCESS" && stderr == "" {
			stderr = state
		}
		done = true
		break
	}

	errMsg := ""
	if !done {
		errMsg = fmt.Sprintf("百度云命令执行超时，请调大等待时间（execution_id=%s）", execID)
	}
	return map[string]string{
		"stdout":    strings.TrimSpace(stdout),
		"stderr":    strings.TrimSpace(stderr),
		"invoke_id": execID,
		"error":     errMsg,
	}, nil
}

func baiduRunCommandCandidates(instanceID, command, sysType string, waitSec int) []*oosmodel.Operator {
	// 常见命名兜底：不同租户/版本可能差异较大。
	known := []struct {
		name     string
		operator string
	}{
		{"run_command", "BCE::Agent::ExecuteShell"},
		{"run_command_ca", "BCE::CA::ExecuteShell"},
		{"run_command_bcc", "BCE::BCC::RunCommand"},
		{"run_script_bcc", "BCE::BCC::ExecuteShell"},
	}
	out := make([]*oosmodel.Operator, 0, len(known))
	for _, k := range known {
		out = append(out, baiduBuildRunOperator(k.operator, k.name, instanceID, command, sysType, waitSec))
	}
	return out
}

func baiduCommandType(sysType string) string {
	switch sysType {
	case "win_cmd":
		return "BAT"
	case "powershell":
		return "POWERSHELL"
	default:
		return "SHELL"
	}
}

func baiduBuildRunOperator(operator, name, instanceID, command, sysType string, waitSec int) *oosmodel.Operator {
	user := "root"
	workDir := "/"
	switch sysType {
	case "powershell", "win_cmd":
		user = "Administrator"
		workDir = "C:\\"
	}

	return &oosmodel.Operator{
		Name:          name,
		Description:   "EasyTools Run Command",
		Operator:      operator,
		Retries:       1,
		RetryInterval: 1000,
		Timeout:       waitSec * 1000,
		Properties: map[string]interface{}{
			"__workerSelectors__": []map[string]string{
				{
					"instanceId": instanceID,
				},
			},
			"content": command,
			"user":    user,
			"workDir": workDir,
		},
	}
}

func baiduDetectCommandOperators(cli *oos.Client) []string {
	if cli == nil {
		return nil
	}
	resp, err := cli.GetOperatorList(&oosmodel.BasePageRequest{
		Ascending: false,
		Sort:      "createTime",
		PageNo:    1,
		PageSize:  100,
	})
	if err != nil || resp == nil || resp.Result == nil {
		return nil
	}
	var out []string
	for _, op := range resp.Result.Operators {
		if op == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(op.Name))
		label := strings.ToLower(strings.TrimSpace(op.Label))
		desc := strings.ToLower(strings.TrimSpace(op.Description))
		operator := strings.TrimSpace(op.Operator)
		joined := name + " " + label + " " + desc + " " + strings.ToLower(operator)
		if strings.Contains(joined, "脚本") || strings.Contains(joined, "command") || strings.Contains(joined, "shell") {
			if operator != "" {
				out = append(out, operator)
			}
			// 某些租户 name 字段里才是实际 operator 标识
			if strings.HasPrefix(strings.TrimSpace(op.Name), "BCE::") {
				out = append(out, strings.TrimSpace(op.Name))
			}
		}
	}
	return uniqueNonEmpty(out)
}

func uniqueNonEmpty(in []string) []string {
	m := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func baiduExtractTaskOutput(t *oosmodel.Task) (string, string) {
	if t == nil {
		return "", ""
	}
	var outParts []string
	var errParts []string
	collect := func(m map[string]interface{}) {
		if len(m) == 0 {
			return
		}
		baiduCollectOutputStrings(m, &outParts, &errParts)
	}
	collect(t.OutputContext)
	collect(t.Context)
	return strings.TrimSpace(strings.Join(uniqueNonEmpty(outParts), "\n")), strings.TrimSpace(strings.Join(uniqueNonEmpty(errParts), "\n"))
}

func baiduCollectOutputStrings(v interface{}, outParts *[]string, errParts *[]string) {
	switch x := v.(type) {
	case map[string]interface{}:
		for k, vv := range x {
			key := strings.ToLower(strings.TrimSpace(k))
			switch key {
			case "stdout", "output", "content", "result", "message", "msg":
				if s := baiduValueToText(vv); s != "" {
					*outParts = append(*outParts, s)
				}
			case "stderr", "error", "err", "reason":
				if s := baiduValueToText(vv); s != "" {
					*errParts = append(*errParts, s)
				}
			default:
				baiduCollectOutputStrings(vv, outParts, errParts)
			}
		}
	case []interface{}:
		for _, item := range x {
			baiduCollectOutputStrings(item, outParts, errParts)
		}
	}
}

func baiduValueToText(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []byte:
		return strings.TrimSpace(string(x))
	case fmt.Stringer:
		return strings.TrimSpace(x.String())
	case map[string]interface{}, []interface{}:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "<nil>" || s == "map[]" || s == "[]" {
			return ""
		}
		return s
	}
}
