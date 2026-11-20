package cloudSecurity

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"EasyTools/app/model"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/tidwall/gjson"
)

func aliyunNewRDSClient(regionID, ak, sk string) (*rds.Client, error) {
	client, err := rds.NewClientWithAccessKey(strings.TrimSpace(regionID), strings.TrimSpace(ak), strings.TrimSpace(sk))
	if err != nil {
		return nil, err
	}
	client.GetConfig().WithTimeout(45 * time.Second)
	return client, nil
}

func aliyunRdsFetchNetInfos(client *rds.Client, dbInstanceID string) ([]rds.DBInstanceNetInfo, error) {
	req := rds.CreateDescribeDBInstanceNetInfoRequest()
	req.DBInstanceId = dbInstanceID
	resp, err := client.DescribeDBInstanceNetInfo(req)
	if err != nil {
		return nil, err
	}
	return resp.DBInstanceNetInfos.DBInstanceNetInfo, nil
}

// 取内网/非 Public 端点端口，用于开通外网时与实例实际端口一致（避免误用 3306）。
func aliyunRdsPickIntranetPort(infos []rds.DBInstanceNetInfo) string {
	for _, n := range infos {
		if strings.EqualFold(strings.TrimSpace(n.IPType), "Public") {
			continue
		}
		if p := strings.TrimSpace(n.Port); p != "" {
			return p
		}
	}
	for _, n := range infos {
		if p := strings.TrimSpace(n.Port); p != "" {
			return p
		}
	}
	return "3306"
}

// 阿里云部分实例的默认白名单分组名并非字面 "Default"；ModifySecurityIps 须使用 DescribeDBInstanceIPArrayList 返回的真实名称。
func aliyunRdsResolveIPArrayName(client *rds.Client, dbInstanceID, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	wantDefault := requested == "" || strings.EqualFold(requested, "default")

	listReq := rds.CreateDescribeDBInstanceIPArrayListRequest()
	listReq.DBInstanceId = dbInstanceID
	resp, err := client.DescribeDBInstanceIPArrayList(listReq)
	if err != nil {
		if wantDefault {
			return "Default", nil
		}
		return "", err
	}
	arr := resp.Items.DBInstanceIPArray
	if len(arr) == 0 {
		if wantDefault {
			return "Default", nil
		}
		return "", fmt.Errorf("实例上暂无白名单分组")
	}
	if !wantDefault {
		for _, g := range arr {
			if strings.TrimSpace(g.DBInstanceIPArrayName) == requested {
				return requested, nil
			}
		}
		names := make([]string, 0, len(arr))
		for _, g := range arr {
			if n := strings.TrimSpace(g.DBInstanceIPArrayName); n != "" {
				names = append(names, n)
			}
		}
		return "", fmt.Errorf("分组「%s」不存在，当前分组：%s", requested, strings.Join(names, "，"))
	}
	for _, g := range arr {
		if n := strings.TrimSpace(g.DBInstanceIPArrayName); n != "" && strings.EqualFold(n, "default") {
			return n, nil
		}
	}
	for _, g := range arr {
		a := strings.ToLower(strings.TrimSpace(g.DBInstanceIPArrayAttribute))
		if a == "default" || strings.Contains(a, "default") {
			if n := strings.TrimSpace(g.DBInstanceIPArrayName); n != "" {
				return n, nil
			}
		}
	}
	for _, g := range arr {
		if n := strings.TrimSpace(g.DBInstanceIPArrayName); n != "" {
			return n, nil
		}
	}
	return "Default", nil
}

func aliyunRdsPickPublicConnectionString(infos []rds.DBInstanceNetInfo) (string, error) {
	for _, n := range infos {
		if strings.EqualFold(strings.TrimSpace(n.IPType), "Public") {
			if s := strings.TrimSpace(n.ConnectionString); s != "" {
				return s, nil
			}
		}
	}
	for _, n := range infos {
		ct := strings.ToLower(strings.TrimSpace(n.ConnectionStringType))
		if strings.Contains(ct, "internet") || ct == "public" {
			if s := strings.TrimSpace(n.ConnectionString); s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("未找到外网连接串（可能未开通外网，可先查看网络信息）")
}

func aliyunRDSRunAction(cr *model.CloudServerCredential, regionID, dbInstanceID, action, jsonStr string) (interface{}, error) {
	if cr.Vendor != "aliyun" {
		return nil, fmt.Errorf("仅支持阿里云凭据")
	}
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	client, err := aliyunNewRDSClient(regionID, ak, sk)
	if err != nil {
		return nil, err
	}
	dbInstanceID = strings.TrimSpace(dbInstanceID)
	if dbInstanceID == "" {
		return nil, fmt.Errorf("db_instance_id 不能为空")
	}
	j := gjson.Parse(jsonStr)

	switch strings.TrimSpace(strings.ToLower(action)) {
	case "describe_ip_whitelist":
		req := rds.CreateDescribeDBInstanceIPArrayListRequest()
		req.DBInstanceId = dbInstanceID
		if wn := strings.TrimSpace(j.Get("whitelist_network_type").String()); wn != "" {
			req.WhitelistNetworkType = wn
		}
		resp, err := client.DescribeDBInstanceIPArrayList(req)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]string, 0, len(resp.Items.DBInstanceIPArray))
		for _, it := range resp.Items.DBInstanceIPArray {
			out = append(out, map[string]string{
				"name":       it.DBInstanceIPArrayName,
				"ips":        it.SecurityIPList,
				"attribute":  it.DBInstanceIPArrayAttribute,
				"network":    it.WhitelistNetworkType,
				"ipType":     it.SecurityIPType,
			})
		}
		return map[string]interface{}{"requestId": resp.RequestId, "groups": out}, nil

	case "append_security_ips", "delete_security_ips":
		ips := strings.TrimSpace(j.Get("security_ips").String())
		if ips == "" {
			return nil, fmt.Errorf("缺少 security_ips（逗号分隔 IP 或 CIDR）")
		}
		rawGroup := strings.TrimSpace(j.Get("ip_array_name").String())
		group, gerr := aliyunRdsResolveIPArrayName(client, dbInstanceID, rawGroup)
		if gerr != nil {
			return nil, gerr
		}
		mode := "Append"
		if action == "delete_security_ips" {
			mode = "Delete"
		}
		req := rds.CreateModifySecurityIpsRequest()
		req.DBInstanceId = dbInstanceID
		req.SecurityIps = ips
		req.DBInstanceIPArrayName = group
		req.ModifyMode = mode
		resp, err := client.ModifySecurityIps(req)
		if err != nil {
			return nil, err
		}
		out := map[string]interface{}{"requestId": resp.RequestId, "taskId": resp.TaskId, "modifyMode": mode, "group": group}
		if rawGroup != "" && !strings.EqualFold(rawGroup, group) {
			out["groupResolvedFrom"] = rawGroup
		}
		return out, nil

	case "describe_accounts":
		var all []rds.DBInstanceAccount
		page := 1
		for page <= 50 {
			req := rds.CreateDescribeAccountsRequest()
			req.DBInstanceId = dbInstanceID
			req.PageNumber = requests.NewInteger(page)
			req.PageSize = requests.NewInteger(100)
			resp, err := client.DescribeAccounts(req)
			if err != nil {
				return nil, err
			}
			all = append(all, resp.Accounts.DBInstanceAccount...)
			if len(resp.Accounts.DBInstanceAccount) < 100 {
				break
			}
			page++
		}
		rows := make([]map[string]interface{}, 0, len(all))
		for _, a := range all {
			rows = append(rows, map[string]interface{}{
				"accountName":        a.AccountName,
				"accountType":        a.AccountType,
				"accountStatus":      a.AccountStatus,
				"accountDescription": a.AccountDescription,
				"privExceeded":       a.PrivExceeded,
			})
		}
		return map[string]interface{}{"accounts": rows, "total": len(rows)}, nil

	case "create_account":
		name := strings.TrimSpace(j.Get("account_name").String())
		pass := j.Get("account_password").String()
		if name == "" || pass == "" {
			return nil, fmt.Errorf("缺少 account_name 或 account_password")
		}
		accType := strings.TrimSpace(j.Get("account_type").String())
		switch strings.ToLower(accType) {
		case "", "normal", "普通":
			accType = "Normal"
		case "super", "高权限":
			accType = "Super"
		default:
			// 阿里云 OpenAPI 仅接受 Normal / Super；其它输入按 Normal 处理以免调用失败
			accType = "Normal"
		}
		req := rds.CreateCreateAccountRequest()
		req.DBInstanceId = dbInstanceID
		req.AccountName = name
		req.AccountPassword = pass
		req.AccountType = accType
		if d := strings.TrimSpace(j.Get("account_description").String()); d != "" {
			req.AccountDescription = d
		}
		resp, err := client.CreateAccount(req)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"requestId": resp.RequestId, "accountName": name, "accountType": accType}, nil

	case "delete_account":
		name := strings.TrimSpace(j.Get("account_name").String())
		if name == "" {
			return nil, fmt.Errorf("缺少 account_name")
		}
		req := rds.CreateDeleteAccountRequest()
		req.DBInstanceId = dbInstanceID
		req.AccountName = name
		resp, err := client.DeleteAccount(req)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"requestId": resp.RequestId, "accountName": name}, nil

	case "describe_net_info":
		req := rds.CreateDescribeDBInstanceNetInfoRequest()
		req.DBInstanceId = dbInstanceID
		resp, err := client.DescribeDBInstanceNetInfo(req)
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]string, 0, len(resp.DBInstanceNetInfos.DBInstanceNetInfo))
		for _, n := range resp.DBInstanceNetInfos.DBInstanceNetInfo {
			rows = append(rows, map[string]string{
				"connectionString":     n.ConnectionString,
				"connectionStringType": n.ConnectionStringType,
				"ipType":               n.IPType,
				"ipAddress":            n.IPAddress,
				"port":                 n.Port,
				"availability":         n.Availability,
			})
		}
		return map[string]interface{}{
			"requestId":           resp.RequestId,
			"instanceNetworkType": resp.InstanceNetworkType,
			"securityIPMode":      resp.SecurityIPMode,
			"netInfos":            rows,
		}, nil

	case "allocate_public_connection":
		// ConnectionStringPrefix、Port 为 API 必填；控制台不展示输入时由服务端按实例网络信息推断端口并生成前缀。
		infos, nerr := aliyunRdsFetchNetInfos(client, dbInstanceID)
		if nerr != nil {
			return nil, nerr
		}
		prefix := strings.TrimSpace(j.Get("connection_string_prefix").String())
		if prefix == "" {
			prefix = "et" + takeoverRandAlnum(6)
		}
		if len(prefix) < 5 || len(prefix) > 40 {
			return nil, fmt.Errorf("connection_string_prefix 长度须为 5~40（当前 %d）", len(prefix))
		}
		for _, r := range prefix {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return nil, fmt.Errorf("connection_string_prefix 仅允许字母、数字、短横线")
		}
		port := strings.TrimSpace(j.Get("port").String())
		if port == "" {
			port = aliyunRdsPickIntranetPort(infos)
		}
		pnum, err := strconv.Atoi(port)
		if err != nil || pnum < 1000 || pnum > 5999 {
			return nil, fmt.Errorf("port 须为 1000~5999 的整数（当前为 %q，请检查实例网络端口或显式传入 port）", port)
		}
		port = strconv.Itoa(pnum)
		req := rds.CreateAllocateInstancePublicConnectionRequest()
		req.DBInstanceId = dbInstanceID
		req.ConnectionStringPrefix = prefix
		req.Port = port
		resp, err := client.AllocateInstancePublicConnection(req)
		if err != nil {
			return nil, err
		}
		out := map[string]interface{}{
			"requestId":                resp.RequestId,
			"connectionString":       resp.ConnectionString,
			"dbInstanceName":         resp.DbInstanceName,
			"connectionStringPrefix": prefix,
			"port":                   port,
		}
		// 与控制台一致：开通外网后向默认白名单分组追加 0.0.0.0/0（分组名按实例实际列表解析）
		wGroup, wGerr := aliyunRdsResolveIPArrayName(client, dbInstanceID, "Default")
		if wGerr != nil {
			out["whitelistAppendError"] = wGerr.Error()
			return out, nil
		}
		wreq := rds.CreateModifySecurityIpsRequest()
		wreq.DBInstanceId = dbInstanceID
		wreq.SecurityIps = "0.0.0.0/0"
		wreq.DBInstanceIPArrayName = wGroup
		wreq.ModifyMode = "Append"
		if wresp, werr := client.ModifySecurityIps(wreq); werr != nil {
			out["whitelistAppendError"] = werr.Error()
		} else {
			out["whitelistAppended"] = "0.0.0.0/0"
			out["whitelistTaskId"] = wresp.TaskId
		}
		return out, nil

	case "release_public_connection":
		cur := strings.TrimSpace(j.Get("current_connection_string").String())
		if cur == "" {
			infos, rerr := aliyunRdsFetchNetInfos(client, dbInstanceID)
			if rerr != nil {
				return nil, rerr
			}
			var perr error
			cur, perr = aliyunRdsPickPublicConnectionString(infos)
			if perr != nil {
				return nil, perr
			}
		}
		req := rds.CreateReleaseInstancePublicConnectionRequest()
		req.DBInstanceId = dbInstanceID
		req.CurrentConnectionString = cur
		resp, err := client.ReleaseInstancePublicConnection(req)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"requestId": resp.RequestId, "released": cur}, nil

	case "describe_db_instance_attribute":
		req := rds.CreateDescribeDBInstanceAttributeRequest()
		req.DBInstanceId = dbInstanceID
		resp, err := client.DescribeDBInstanceAttribute(req)
		if err != nil {
			return nil, err
		}
		items := resp.Items.DBInstanceAttribute
		if len(items) == 0 {
			return map[string]interface{}{"requestId": resp.RequestId}, nil
		}
		it := items[0]
		return map[string]interface{}{
			"requestId":    resp.RequestId,
			"dbInstanceId": it.DBInstanceId,
			"lockMode":     it.LockMode,
			"lockReason":   it.LockReason,
			"status":       it.DBInstanceStatus,
			"engine":       it.Engine,
			"engineVersion": it.EngineVersion,
		}, nil

	default:
		return nil, fmt.Errorf("未知 action: %s", action)
	}
}

func aliyunRDSActionJSONString(cr *model.CloudServerCredential, jsonStr string) (string, error) {
	regionID := strings.TrimSpace(gjson.Get(jsonStr, "region_id").String())
	dbInst := strings.TrimSpace(gjson.Get(jsonStr, "db_instance_id").String())
	action := strings.TrimSpace(gjson.Get(jsonStr, "action").String())
	if regionID == "" {
		return "", fmt.Errorf("缺少 region_id")
	}
	if dbInst == "" {
		return "", fmt.Errorf("缺少 db_instance_id")
	}
	if action == "" {
		return "", fmt.Errorf("缺少 action")
	}
	data, err := aliyunRDSRunAction(cr, regionID, dbInst, action, jsonStr)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
