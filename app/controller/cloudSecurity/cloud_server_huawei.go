package cloudSecurity

import (
	"fmt"
	"strings"
	"time"

	appmodel "EasyTools/app/model"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	iammodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	coc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/coc/v1"
	cocmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/coc/v1/model"
	cocregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/coc/v1/region"
	hwecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	hwmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	ecsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
	"github.com/tidwall/gjson"
)

func huaweiListInstances(cr *appmodel.CloudServerCredential, regions []string) (map[string]interface{}, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	projectID := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "huawei_project_id").String())

	b := basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk)
	if projectID != "" {
		b = b.WithProjectId(projectID)
	}
	auth, err := b.SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("华为云凭证: %w", err)
	}

	var blocks []csRegionBlock
	for _, rid := range regions {
		reg, err := ecsregion.SafeValueOf(rid)
		if err != nil {
			continue
		}
		hc, err := hwecs.EcsClientBuilder().WithRegion(reg).WithCredential(auth).SafeBuild()
		if err != nil {
			continue
		}
		client := hwecs.NewEcsClient(hc)
		servers, err := huaweiListAllServers(client)
		if err != nil || len(servers) == 0 {
			continue
		}
		rows := make([]csInstanceRow, 0, len(servers))
		for _, s := range servers {
			rows = append(rows, huaweiServerToRow(s, rid))
		}
		blocks = append(blocks, csRegionBlock{
			RegionID:   rid,
			RegionName: rid,
			Instances:  rows,
		})
	}
	return map[string]interface{}{"regions": blocks}, nil
}

func huaweiListAllServers(client *hwecs.EcsClient) ([]hwmodel.NovaServer, error) {
	var all []hwmodel.NovaServer
	var marker *string
	limit := int32(100)
	for {
		req := &hwmodel.NovaListServersDetailsRequest{}
		req.Limit = &limit
		if marker != nil {
			req.Marker = marker
		}
		resp, err := client.NovaListServersDetails(req)
		if err != nil {
			return all, err
		}
		if resp.Servers == nil {
			break
		}
		for _, s := range *resp.Servers {
			all = append(all, s)
		}
		if len(*resp.Servers) < int(limit) {
			break
		}
		last := (*resp.Servers)[len(*resp.Servers)-1]
		m := last.Id
		marker = &m
		if m == "" {
			break
		}
	}
	return all, nil
}

func huaweiServerToRow(s hwmodel.NovaServer, regionID string) csInstanceRow {
	pub, priv := huaweiPickIPs(s.Addresses)
	flavor := ""
	if s.Flavor != nil && s.Flavor.Vcpus != nil && s.Flavor.Ram != nil {
		flavor = fmt.Sprintf("%d核 %dMB", *s.Flavor.Vcpus, *s.Flavor.Ram)
	}
	img := ""
	if s.Image != nil && s.Image.Id != "" {
		img = s.Image.Id
	}
	status := strings.TrimSpace(s.Status.Value())
	if st := strings.TrimSpace(s.OSEXTSTSvmState.Value()); st != "" {
		status = st
	}
	return csInstanceRow{
		InstanceID: s.Id,
		HostName:   s.Name,
		PublicIP:   pub,
		PrivateIP:  priv,
		Config:     flavor,
		OS:         img,
		Status:     status,
		CreatedAt:  s.Created,
		ExpiredAt:  "",
	}
}

func huaweiPickIPs(addresses map[string][]hwmodel.NovaNetwork) (pub, priv string) {
	if addresses == nil {
		return "", ""
	}
	for _, nets := range addresses {
		for _, n := range nets {
			addr := strings.TrimSpace(n.Addr)
			if addr == "" {
				continue
			}
			t := strings.ToLower(strings.TrimSpace(n.OSEXTIPStype))
			if t == "floating" {
				pub = addr
			}
			if t == "fixed" && priv == "" {
				priv = addr
			}
		}
	}
	if pub == "" && priv != "" {
		return "", priv
	}
	return pub, priv
}

func huaweiRunCommand(cr *appmodel.CloudServerCredential, regionID, instanceID, command, sysType string, waitSec int) (map[string]string, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	projectID, err := huaweiResolveProjectID(cr, regionID)
	if err != nil {
		return map[string]string{
			"stdout":    "",
			"stderr":    "",
			"invoke_id": "",
			"error":     err.Error(),
		}, nil
	}
	if waitSec < 6 {
		waitSec = 60
	}
	if waitSec > 1800 {
		waitSec = 1800
	}

	auth, err := basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).WithProjectId(projectID).SafeBuild()
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云鉴权失败，请检查 AK/SK"}, nil
	}

	reg, err := cocregion.SafeValueOf("cn-north-4")
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云区域初始化失败"}, nil
	}
	hc, err := coc.CocClientBuilder().WithRegion(reg).WithCredential(auth).SafeBuild()
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云 COC 客户端创建失败"}, nil
	}
	client := coc.NewCocClient(hc)

	scriptType := cocmodel.GetAddScriptModelTypeEnum().SHELL
	execUser := "root"
	switch sysType {
	case "powershell", "win_cmd":
		scriptType = cocmodel.GetAddScriptModelTypeEnum().BAT
		execUser = "Administrator"
	}

	createResp, err := client.CreateScript(&cocmodel.CreateScriptRequest{
		XProjectId: &projectID,
		Body: &cocmodel.AddScriptModel{
			Name:        fmt.Sprintf("et_%d", time.Now().Unix()),
			Description: "EasyTools 临时命令脚本",
			Type:        scriptType,
			Content:     command,
			Properties: &cocmodel.ScriptPropertiesModel{
				RiskLevel: cocmodel.GetScriptPropertiesModelRiskLevelEnum().LOW,
				Version:   "1.0.0",
			},
		},
	})
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云创建脚本失败（请确认 COC/UniAgent/权限）"}, nil
	}
	if createResp == nil || createResp.Data == nil || strings.TrimSpace(*createResp.Data) == "" {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云 CreateScript 未返回 script_uuid"}, nil
	}
	scriptUUID := strings.TrimSpace(*createResp.Data)

	// 临时脚本执行后尽量清理，不阻塞主流程
	defer func() {
		_, _ = client.DeleteScript(&cocmodel.DeleteScriptRequest{
			ScriptUuid: scriptUUID,
			XProjectId: &projectID,
		})
	}()

	provider := "ECS"
	rsType := cocmodel.GetExecuteResourceInstanceTypeEnum().CLOUDSERVER
	rotation := cocmodel.GetExecuteInstancesBatchInfoRotationStrategyEnum().CONTINUE
	execResp, err := client.ExecuteScript(&cocmodel.ExecuteScriptRequest{
		ScriptUuid: scriptUUID,
		XProjectId: &projectID,
		Body: &cocmodel.ScriptExecuteModel{
			ExecuteParam: &cocmodel.ScriptExecuteParam{
				Timeout:     int32(waitSec),
				SuccessRate: 1.0,
				ExecuteUser: execUser,
			},
			ExecuteBatches: []cocmodel.ExecuteInstancesBatchInfo{
				{
					BatchIndex: 1,
					TargetInstances: []cocmodel.ExecuteResourceInstance{
						{
							ResourceId: instanceID,
							RegionId:   regionID,
							Provider:   &provider,
							Type:       &rsType,
						},
					},
					RotationStrategy: rotation,
				},
			},
		},
	})
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云下发命令失败（请确认实例在线与执行权限）"}, nil
	}
	if execResp == nil || execResp.Data == nil || strings.TrimSpace(*execResp.Data) == "" {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "华为云 ExecuteScript 未返回 execute_uuid"}, nil
	}
	execUUID := strings.TrimSpace(*execResp.Data)

	deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
	var stdout, stderr string
	done := false
	for time.Now().Before(deadline) {
		infoResp, qerr := client.GetScriptJobInfo(&cocmodel.GetScriptJobInfoRequest{
			ExecuteUuid: execUUID,
			XProjectId:  &projectID,
		})
		if qerr != nil || infoResp == nil || infoResp.Data == nil || infoResp.Data.Status == nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		st := strings.ToUpper(strings.TrimSpace(infoResp.Data.Status.Value()))
		if st == "READY" || st == "PROCESSING" {
			time.Sleep(400 * time.Millisecond)
			continue
		}

		batchesResp, berr := client.ListScriptJobBatches(&cocmodel.ListScriptJobBatchesRequest{
			ExecuteUuid: execUUID,
			XProjectId:  &projectID,
		})
		if berr == nil && batchesResp != nil && batchesResp.Data != nil && len(*batchesResp.Data) > 0 {
			b := (*batchesResp.Data)[0]
			if b.BatchIndex != nil {
				batchResp, gerr := client.GetScriptJobBatch(&cocmodel.GetScriptJobBatchRequest{
					BatchIndex:  *b.BatchIndex,
					ExecuteUuid: execUUID,
					Limit:       20,
					XProjectId:  &projectID,
				})
				if gerr == nil && batchResp != nil && batchResp.Data != nil && batchResp.Data.ExecuteInstances != nil {
					for _, inst := range *batchResp.Data.ExecuteInstances {
						if inst.Message != nil && strings.TrimSpace(*inst.Message) != "" {
							stdout = strings.TrimSpace(*inst.Message)
							break
						}
					}
				}
			}
		}
		if st != "FINISHED" && stderr == "" {
			stderr = st
		}
		done = true
		break
	}

	errMsg := ""
	if !done {
		errMsg = fmt.Sprintf("华为云命令执行超时，请调大等待时间（execute_id=%s）", execUUID)
	}
	return map[string]string{
		"stdout":    stdout,
		"stderr":    strings.TrimSpace(stderr),
		"invoke_id": execUUID,
		"error":     errMsg,
	}, nil
}

func huaweiResolveProjectID(cr *appmodel.CloudServerCredential, regionID string) (string, error) {
	// 优先使用用户显式配置，兼容已有配置习惯。
	if projectID := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "huawei_project_id").String()); projectID != "" {
		return projectID, nil
	}
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	if ak == "" || sk == "" {
		return "", fmt.Errorf("请填写华为云 AccessKey / SecretKey")
	}
	iamClient, err := huaweiIAMClient(ak, sk, "")
	if err != nil {
		return "", fmt.Errorf("华为云项目自动识别失败，请在扩展 JSON 设置 huawei_project_id")
	}
	enabled := true
	page := int32(1)
	perPage := int32(500)
	resp, err := iamClient.KeystoneListProjects(&iammodel.KeystoneListProjectsRequest{
		Enabled: &enabled,
		Page:    &page,
		PerPage: &perPage,
	})
	if err != nil {
		return "", fmt.Errorf("华为云项目查询失败，请在扩展 JSON 设置 huawei_project_id")
	}
	if resp == nil || resp.Projects == nil || len(*resp.Projects) == 0 {
		return "", fmt.Errorf("未查询到可用华为云项目，建议在扩展 JSON 手动设置 huawei_project_id")
	}
	targetRegion := strings.TrimSpace(strings.ToLower(regionID))
	var fallback string
	for _, p := range *resp.Projects {
		name := strings.TrimSpace(strings.ToLower(p.Name))
		if name == targetRegion && strings.TrimSpace(p.Id) != "" {
			return strings.TrimSpace(p.Id), nil
		}
		if fallback == "" && strings.TrimSpace(p.Id) != "" {
			fallback = strings.TrimSpace(p.Id)
		}
	}
	if fallback != "" {
		// 兼容“仅有一个项目”或项目名不规范场景，优先保证可执行。
		return fallback, nil
	}
	return "", fmt.Errorf("未找到 region=%s 对应的华为云项目ID，请在扩展 JSON 设置 huawei_project_id", regionID)
}
