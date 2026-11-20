package cloudSecurity

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"EasyTools/app/model"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sas"
)

type csRdsInstanceRow struct {
	Engine       string `json:"engine"`
	Version      string `json:"version"`
	Conn         string `json:"conn"`
	Status       string `json:"status"`
	Mode         string `json:"mode"`
	Created      string `json:"created"`
	Expire       string `json:"expire"`
	DBInstanceID string `json:"dbInstanceId"`
	RegionID     string `json:"regionId"`
	ZoneID       string `json:"zoneId,omitempty"`
	LockMode     string `json:"lockMode"`
	LockReason   string `json:"lockReason,omitempty"`
}

type csRdsRegionRow struct {
	RegionID   string             `json:"regionId"`
	RegionName string             `json:"regionName"`
	Alias      string             `json:"alias"`
	Count      int                `json:"count"`
	Vendor     string             `json:"vendor"`
	ZoneID     string             `json:"zoneId"`
	ZoneName   string             `json:"zoneName"`
	Instances  []csRdsInstanceRow `json:"instances"`
}

type csAlarmRow struct {
	Machine           string `json:"machine"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Level             string `json:"level"`
	Detail            string `json:"detail"`
	Status            string `json:"status"`
	Time              string `json:"time"`
	SecurityEventIds  string `json:"securityEventIds"`
	AlarmUniqueInfo   string `json:"alarmUniqueInfo,omitempty"`
}

type csNotifyRow struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Method string `json:"method"`
	Limit  string `json:"limit"`
	Route  int    `json:"route"`
}

func aliyunListRDSGrouped(cr *model.CloudServerCredential, regions []string) ([]csRdsRegionRow, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	nameMap, _ := aliyunDescribeRegionNames(ak, sk)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	byRegion := make(map[string][]csRdsInstanceRow)
	for _, rid := range regions {
		if ctx.Err() != nil {
			break
		}
		client, err := rds.NewClientWithAccessKey(rid, ak, sk)
		if err != nil {
			continue
		}
		client.GetConfig().WithTimeout(25 * time.Second)
		page := 1
		for {
			if ctx.Err() != nil {
				break
			}
			req := rds.CreateDescribeDBInstancesRequest()
			req.PageNumber = requests.NewInteger(page)
			req.PageSize = requests.NewInteger(50)
			resp, err := client.DescribeDBInstances(req)
			if err != nil {
				break
			}
			for _, it := range resp.Items.DBInstance {
				mode := strings.TrimSpace(it.ConnectionMode)
				if mode == "" {
					mode = strings.TrimSpace(it.Category)
				}
				if mode == "" {
					mode = it.DBInstanceType
				}
				byRegion[it.RegionId] = append(byRegion[it.RegionId], csRdsInstanceRow{
					Engine:       it.Engine,
					Version:      it.EngineVersion,
					Conn:         it.ConnectionString,
					Status:       it.DBInstanceStatus,
					Mode:         mode,
					Created:      it.CreateTime,
					Expire:       it.ExpireTime,
					DBInstanceID: it.DBInstanceId,
					RegionID:     it.RegionId,
					ZoneID:       it.ZoneId,
					LockMode:     it.LockMode,
					LockReason:   it.LockReason,
				})
			}
			if len(resp.Items.DBInstance) < 50 {
				break
			}
			page++
			if page > 100 {
				break
			}
		}
	}

	alias := strings.TrimSpace(cr.Name)
	if alias == "" {
		alias = "默认"
	}
	rids := make([]string, 0, len(byRegion))
	for rid := range byRegion {
		rids = append(rids, rid)
	}
	sort.Strings(rids)
	out := make([]csRdsRegionRow, 0, len(byRegion))
	for _, rid := range rids {
		instances := byRegion[rid]
		if len(instances) == 0 {
			continue
		}
		rname := nameMap[rid]
		if rname == "" {
			rname = rid
		}
		zoneID := instances[0].ZoneID
		if zoneID == "" {
			zoneID = rid
		}
		zoneName := zoneID
		out = append(out, csRdsRegionRow{
			RegionID:   rid,
			RegionName: rname,
			Alias:      alias,
			Count:      len(instances),
			Vendor:     "阿里云",
			ZoneID:     zoneID,
			ZoneName:   zoneName,
			Instances:  instances,
		})
	}
	return out, nil
}

func aliyunListSuspEventsPanel(cr *model.CloudServerCredential) ([]csAlarmRow, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	client, err := sas.NewClientWithAccessKey("cn-hangzhou", ak, sk)
	if err != nil {
		return nil, err
	}
	client.GetConfig().WithTimeout(30 * time.Second)

	var rows []csAlarmRow
	page := 1
	for page <= 20 {
		req := sas.CreateDescribeSuspEventsRequest()
		req.PageSize = "50"
		req.CurrentPage = strconv.Itoa(page)
		resp, err := client.DescribeSuspEvents(req)
		if err != nil {
			return rows, err
		}
		for _, e := range resp.SuspEvents {
			rows = append(rows, csAlarmRow{
				Machine:          aliyunAlarmMachine(e),
				Name:             firstNonEmpty(e.AlarmEventName, e.Name, e.RiskName),
				Type:             firstNonEmpty(e.AlarmEventTypeDisplay, e.TypeAlias, e.AlarmEventType),
				Level:            aliyunAlarmLevelCN(e.Level),
				Detail:           firstNonEmpty(e.Desc, e.AlarmEventNameDisplay),
				Status:           aliyunSuspStatus(e),
				Time:             firstNonEmpty(e.OccurrenceTime, e.LastTime, e.LastFoundTime),
				SecurityEventIds: strings.TrimSpace(e.SecurityEventIds),
				AlarmUniqueInfo:  e.AlarmUniqueInfo,
			})
		}
		if len(resp.SuspEvents) < 50 {
			break
		}
		page++
	}
	return rows, nil
}

func aliyunAlarmMachine(e sas.WarningSummary) string {
	if strings.TrimSpace(e.InstanceName) != "" {
		return e.InstanceName
	}
	if strings.TrimSpace(e.IntranetIp) != "" {
		return e.IntranetIp
	}
	if strings.TrimSpace(e.InternetIp) != "" {
		return e.InternetIp
	}
	if strings.TrimSpace(e.Uuid) != "" {
		return e.Uuid
	}
	return "—"
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func aliyunAlarmLevelCN(lv string) string {
	lv = strings.TrimSpace(strings.ToLower(lv))
	switch lv {
	case "serious", "high":
		return "紧急"
	case "suspicious", "medium":
		return "可疑"
	case "remind", "low":
		return "提醒"
	default:
		if strings.TrimSpace(lv) == "" {
			return "提醒"
		}
		return lv
	}
}

func aliyunSuspStatus(e sas.WarningSummary) string {
	switch e.EventStatus {
	case 1:
		return "处理完毕"
	case 2:
		return "已忽略"
	case 3:
		return "误报"
	default:
		return "待处理"
	}
}

func aliyunListNoticePanel(cr *model.CloudServerCredential) ([]csNotifyRow, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	client, err := sas.NewClientWithAccessKey("cn-hangzhou", ak, sk)
	if err != nil {
		return nil, err
	}
	client.GetConfig().WithTimeout(20 * time.Second)
	req := sas.CreateDescribeNoticeConfigRequest()
	resp, err := client.DescribeNoticeConfig(req)
	if err != nil {
		return nil, err
	}
	out := make([]csNotifyRow, 0, len(resp.NoticeConfigList))
	for _, n := range resp.NoticeConfigList {
		out = append(out, csNotifyRow{
			Code:   n.Project,
			Title:  n.Project,
			Method: fmt.Sprintf("Route=%d（详见云安全中心通知策略）", n.Route),
			Limit:  aliyunNoticeLimitText(n.TimeLimit),
			Route:  n.Route,
		})
	}
	return out, nil
}

func aliyunNoticeLimitText(tl int) string {
	if tl == 1 {
		return "08:00-22:00 时间段内发送通知"
	}
	return "无限制"
}

func aliyunHandleSecurityEvent(cr *model.CloudServerCredential, securityEventIDs []string, operationCode string) error {
	if len(securityEventIDs) == 0 {
		return fmt.Errorf("缺少安全事件 ID")
	}
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	client, err := sas.NewClientWithAccessKey("cn-hangzhou", ak, sk)
	if err != nil {
		return err
	}
	client.GetConfig().WithTimeout(30 * time.Second)
	req := sas.CreateHandleSecurityEventsRequest()
	req.OperationCode = mapAliyunHandleAction(operationCode)
	ids := securityEventIDs
	req.SecurityEventIds = &ids
	_, err = client.HandleSecurityEvents(req)
	return err
}

func mapAliyunHandleAction(action string) string {
	switch strings.TrimSpace(action) {
	case "ignore":
		return "ignore"
	case "manual":
		return "manual_handling"
	case "whitelist":
		return "whitelist"
	case "block":
		return "block"
	case "kill":
		return "kill_process"
	case "deep":
		return "sandbox_scan"
	case "virus":
		return "virus_scanning"
	default:
		return "manual_handling"
	}
}

func aliyunModifyNotice(cr *model.CloudServerCredential, project string, route int, timeLimit int) error {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	client, err := sas.NewClientWithAccessKey("cn-hangzhou", ak, sk)
	if err != nil {
		return err
	}
	client.GetConfig().WithTimeout(20 * time.Second)
	req := sas.CreateModifyNoticeConfigRequest()
	req.Project = project
	req.Route = requests.NewInteger(route)
	req.TimeLimit = requests.NewInteger(timeLimit)
	_, err = client.ModifyNoticeConfig(req)
	return err
}
