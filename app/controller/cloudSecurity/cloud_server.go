package cloudSecurity

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"EasyTools/app/controller/system"
	"EasyTools/app/model"

	"github.com/tidwall/gjson"
)

// CloudServer 多云 ECS/主机管理与命令执行（凭据存库，调用各云 OpenAPI）。
type CloudServer struct {
	system.Base
}

func NewCloudServer() *CloudServer {
	return &CloudServer{}
}

func (c *CloudServer) credByID(id uint) (*model.CloudServerCredential, error) {
	db := c.Db()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}
	var row model.CloudServerCredential
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// CloudServerCredentialsJSON 返回全部凭据 JSON（含密钥，仅本地 Wails）。
// 说明：该接口仅从本地 config.db 读取云服务器凭据列表，不调用外部云厂商 API，
// 因此不再强制依赖软件授权，便于在授权过期时仍可查看与导出本地已保存的密钥记录。
func (c *CloudServer) CloudServerCredentialsJSON() (string, error) {
	db := c.Db()
	if db == nil {
		return "", errors.New("数据库连接失败")
	}
	var rows []model.CloudServerCredential
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		return "", err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, credentialToFrontendMap(r, nil))
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CloudServerSaveCredentialJSON 保存或更新凭据；id 为 0 时新增。成功返回 {"id":n}。
func (c *CloudServer) CloudServerSaveCredentialJSON(jsonStr string) (string, error) {
	db := c.Db()
	if db == nil {
		return "", errors.New("数据库连接失败")
	}
	var row model.CloudServerCredential
	if err := json.Unmarshal([]byte(jsonStr), &row); err != nil {
		return "", fmt.Errorf("凭据 JSON 无效: %w", err)
	}
	row.Vendor = strings.TrimSpace(strings.ToLower(row.Vendor))
	if row.Vendor == "" {
		return "", fmt.Errorf("vendor 不能为空")
	}
	if strings.TrimSpace(row.AccessKey) == "" || strings.TrimSpace(row.SecretKey) == "" {
		return "", fmt.Errorf("请填写 AccessKey 与 SecretKey")
	}
	if row.ID == 0 {
		if err := db.Create(&row).Error; err != nil {
			return "", err
		}
	} else {
		if err := db.Model(&model.CloudServerCredential{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
			"vendor":     row.Vendor,
			"name":       row.Name,
			"access_key": row.AccessKey,
			"secret_key": row.SecretKey,
			"extra_json": row.ExtraJSON,
			"remark":     row.Remark,
		}).Error; err != nil {
			return "", err
		}
	}
	b, _ := json.Marshal(map[string]uint{"id": row.ID})
	return string(b), nil
}

// CloudServerDeleteCredential 删除凭据。
func (c *CloudServer) CloudServerDeleteCredential(id uint) error {
	db := c.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}
	return db.Delete(&model.CloudServerCredential{}, id).Error
}

// CloudServerListInstancesJSON 拉取指定凭据下的实例，按地域分组，返回与前端一致的 JSON。
func (c *CloudServer) CloudServerListInstancesJSON(credentialID uint) (string, error) {
	cr, err := c.credByID(credentialID)
	if err != nil {
		return "", err
	}
	regions := regionsFromExtra(cr.ExtraJSON, cr.Vendor)
	var payload interface{}
	switch cr.Vendor {
	case "aliyun":
		payload, err = aliyunListInstances(cr, regions)
	case "huawei":
		payload, err = huaweiListInstances(cr, regions)
	case "ucloud":
		payload, err = ucloudListInstances(cr)
	case "tencent":
		payload, err = tencentListInstances(cr, regions)
	case "aws":
		payload, err = awsListInstances(cr, regions)
	case "baidu":
		payload, err = baiduListInstances(cr, regions)
	default:
		return "", fmt.Errorf("不支持的 vendor: %s", cr.Vendor)
	}
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CloudServerRunCommandJSON 在实例上执行命令。阿里云云助手、腾讯云 TAT(RunCommand)、AWS Systems Manager SendCommand；华为云 / UCloud 返回说明错误。
// 请求 JSON：credential_id, region_id, instance_id, command, system_type(linux|win_cmd|powershell), wait_sec
func (c *CloudServer) CloudServerRunCommandJSON(jsonStr string) (string, error) {
	cr, err := c.credByID(uint(gjson.Get(jsonStr, "credential_id").Uint()))
	if err != nil {
		return "", err
	}
	regionID := strings.TrimSpace(gjson.Get(jsonStr, "region_id").String())
	instanceID := strings.TrimSpace(gjson.Get(jsonStr, "instance_id").String())
	command := gjson.Get(jsonStr, "command").String()
	sysType := strings.TrimSpace(gjson.Get(jsonStr, "system_type").String())
	waitSec := int(gjson.Get(jsonStr, "wait_sec").Int())
	if waitSec <= 0 {
		waitSec = 60
	}
	if waitSec > 600 {
		waitSec = 600
	}
	if regionID == "" || instanceID == "" || strings.TrimSpace(command) == "" {
		return "", errors.New("缺少 region_id、instance_id 或 command")
	}
	var out map[string]string
	switch cr.Vendor {
	case "aliyun":
		out, err = aliyunRunCommand(cr, regionID, instanceID, command, sysType, waitSec)
	case "tencent":
		out, err = tencentRunCommand(cr, regionID, instanceID, command, sysType, waitSec)
	case "aws":
		out, err = awsRunCommand(cr, regionID, instanceID, command, sysType, waitSec)
	case "huawei":
		out, err = huaweiRunCommand(cr, regionID, instanceID, command, sysType, waitSec)
	case "baidu":
		out, err = baiduRunCommand(cr, regionID, instanceID, command, sysType, waitSec)
	default:
		out = map[string]string{
			"stdout":    "",
			"stderr":    "",
			"invoke_id": "",
			"error":     fmt.Sprintf("当前不支持该厂商命令执行（vendor=%s），请使用阿里云 / 腾讯云 / AWS", cr.Vendor),
		}
		err = nil
	}
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CloudServerListRDSJSON 阿里云 RDS DescribeDBInstances，按地域聚合，返回 regions 数组（结构与前端表格一致）。
func (c *CloudServer) CloudServerListRDSJSON(credentialID uint) (string, error) {
	cr, err := c.credByID(credentialID)
	if err != nil {
		return "", err
	}
	if cr.Vendor != "aliyun" {
		return "", fmt.Errorf("仅支持阿里云凭据")
	}
	regions := regionsFromExtra(cr.ExtraJSON, cr.Vendor)
	rows, err := aliyunListRDSGrouped(cr, regions)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(map[string]interface{}{"regions": rows})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CloudServerAliyunRDSActionJSON 阿里云 RDS 细粒度 OpenAPI。JSON: credential_id, region_id, db_instance_id, action 及附加字段。
// action: describe_ip_whitelist, append_security_ips, delete_security_ips, describe_accounts, create_account, delete_account,
// describe_net_info, allocate_public_connection, release_public_connection, describe_db_instance_attribute。
func (c *CloudServer) CloudServerAliyunRDSActionJSON(jsonStr string) (string, error) {
	cr, err := c.credByID(uint(gjson.Get(jsonStr, "credential_id").Uint()))
	if err != nil {
		return "", err
	}
	return aliyunRDSActionJSONString(cr, jsonStr)
}

// CloudServerListAliyunSecurityJSON 拉取云安全中心可疑事件 + 通知配置（通知失败时仍返回告警列表）。
func (c *CloudServer) CloudServerListAliyunSecurityJSON(credentialID uint) (string, error) {
	cr, err := c.credByID(credentialID)
	if err != nil {
		return "", err
	}
	if cr.Vendor != "aliyun" {
		return "", fmt.Errorf("仅支持阿里云凭据")
	}
	alarms, err := aliyunListSuspEventsPanel(cr)
	if err != nil {
		return "", err
	}
	notify, nerr := aliyunListNoticePanel(cr)
	out := map[string]interface{}{
		"alarms":      alarms,
		"notify_rows": notify,
	}
	if nerr != nil {
		out["notify_error"] = nerr.Error()
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CloudServerAliyunHandleSecurityEventJSON 处置告警。JSON: credential_id, security_event_ids(逗号分隔), action(与前端一致)。
func (c *CloudServer) CloudServerAliyunHandleSecurityEventJSON(jsonStr string) (string, error) {
	cr, err := c.credByID(uint(gjson.Get(jsonStr, "credential_id").Uint()))
	if err != nil {
		return "", err
	}
	if cr.Vendor != "aliyun" {
		return "", fmt.Errorf("仅支持阿里云凭据")
	}
	raw := strings.TrimSpace(gjson.Get(jsonStr, "security_event_ids").String())
	var ids []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ids = append(ids, p)
		}
	}
	action := strings.TrimSpace(gjson.Get(jsonStr, "action").String())
	err = aliyunHandleSecurityEvent(cr, ids, action)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b), nil
}

// CloudServerAliyunSaveNoticeJSON 更新通知时间窗等。JSON: credential_id, project, route(整数，来自列表), time_limit(0|1)。
func (c *CloudServer) CloudServerAliyunSaveNoticeJSON(jsonStr string) (string, error) {
	cr, err := c.credByID(uint(gjson.Get(jsonStr, "credential_id").Uint()))
	if err != nil {
		return "", err
	}
	if cr.Vendor != "aliyun" {
		return "", fmt.Errorf("仅支持阿里云凭据")
	}
	project := strings.TrimSpace(gjson.Get(jsonStr, "project").String())
	route := int(gjson.Get(jsonStr, "route").Int())
	tl := int(gjson.Get(jsonStr, "time_limit").Int())
	err = aliyunModifyNotice(cr, project, route, tl)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b), nil
}

// CloudServerListUcloudOrdersJSON JSON: credential_id, begin_time,end_time (unix 秒，可选), region (可选)。
func (c *CloudServer) CloudServerListUcloudOrdersJSON(jsonStr string) (string, error) {
	cr, err := c.credByID(uint(gjson.Get(jsonStr, "credential_id").Uint()))
	if err != nil {
		return "", err
	}
	if cr.Vendor != "ucloud" {
		return "", fmt.Errorf("仅支持 UCloud 凭据")
	}
	region := strings.TrimSpace(gjson.Get(jsonStr, "region").String())
	begin := gjson.Get(jsonStr, "begin_time").Int()
	end := gjson.Get(jsonStr, "end_time").Int()
	rows, err := ucloudListOrderRows(cr, region, begin, end)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(map[string]interface{}{"orders": rows})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func credentialToFrontendMap(r model.CloudServerCredential, regions interface{}) map[string]interface{} {
	if regions == nil {
		regions = []interface{}{}
	}
	return map[string]interface{}{
		"id":          r.ID,
		"vendor":      r.Vendor,
		"alias":       r.Name,
		"name":        r.Name,
		"accessKey":   r.AccessKey,
		"access_key":  r.AccessKey,
		"secretKey":   r.SecretKey,
		"secret_key":  r.SecretKey,
		"extra_json":  r.ExtraJSON,
		"remark":      r.Remark,
		"regions":     regions,
		"takeover":    nil,
		"ucloudKeys":  nil,
		"ucloudNote":  nil,
	}
}

func regionsFromExtra(extraJSON, vendor string) []string {
	raw := strings.TrimSpace(gjson.Get(extraJSON, "regions").Raw)
	if raw != "" && raw != "null" {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err == nil && len(arr) > 0 {
			out := make([]string, 0, len(arr))
			for _, s := range arr {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return defaultRegionsForVendor(vendor)
}

func defaultRegionsForVendor(vendor string) []string {
	switch vendor {
	case "aliyun":
		return []string{
			"cn-hangzhou", "cn-shanghai", "cn-beijing", "cn-shenzhen", "cn-qingdao",
			"cn-zhangjiakou", "cn-huhehaote", "cn-chengdu", "cn-heyuan", "cn-guangzhou",
			"cn-wulanchabu", "cn-hongkong",
		}
	case "huawei":
		return []string{
			"cn-north-4", "cn-north-1", "cn-east-2", "cn-east-3", "cn-south-1",
			"cn-southwest-2", "cn-south-4", "ap-southeast-1",
		}
	case "ucloud":
		// 常用地域；缺少时可于凭据扩展 JSON 的 regions 数组中补充（如 cn-wlcb 等）
		return []string{
			"cn-bj2", "cn-sh2", "cn-gd", "cn-wlcb", "hk", "cn-hk", "tw-ka",
			"us-ca", "sg", "th-bkk", "id-jakarta", "vn-sng", "br-sa", "ae-dubai",
			"rus-mosc", "ge-fra", "uk-london", "jpn-tky", "kr-seoul", "in-mumbai",
		}
	case "tencent":
		return []string{
			"ap-guangzhou", "ap-shanghai", "ap-beijing", "ap-nanjing", "ap-chengdu",
			"ap-chongqing", "ap-hongkong", "ap-singapore", "na-toronto", "na-siliconvalley",
		}
	case "aws":
		return []string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "ap-northeast-1",
			"ap-east-1", "eu-central-1", "sa-east-1",
		}
	case "baidu":
		// 典型 BCC 地域；如需更多可在扩展 JSON 的 regions 中覆盖
		return []string{
			"bj", "gz", "su", "bd", "hk",
		}
	default:
		return []string{"cn-hangzhou"}
	}
}
