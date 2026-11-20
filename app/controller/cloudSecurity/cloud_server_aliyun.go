package cloudSecurity

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"EasyTools/app/model"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

type csInstanceRow struct {
	InstanceID string `json:"instanceId"`
	HostName   string `json:"hostName"`
	PublicIP   string `json:"publicIp"`
	PrivateIP  string `json:"privateIp"`
	Config     string `json:"config"`
	OS         string `json:"os"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
	ExpiredAt  string `json:"expiredAt"`
}

type csRegionBlock struct {
	RegionID   string           `json:"regionId"`
	RegionName string           `json:"regionName"`
	Instances  []csInstanceRow  `json:"instances"`
}

func aliyunListInstances(cr *model.CloudServerCredential, regions []string) (map[string]interface{}, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	nameMap, err := aliyunDescribeRegionNames(ak, sk)
	if err != nil {
		nameMap = map[string]string{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var (
		mu     sync.Mutex
		blocks []csRegionBlock
		wg     sync.WaitGroup
		sem    = make(chan struct{}, 6)
	)
	for _, regionID := range regions {
		rid := strings.TrimSpace(regionID)
		if rid == "" {
			continue
		}
		rname := nameMap[rid]
		if rname == "" {
			rname = rid
		}
		wg.Add(1)
		go func(rid, rname string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			client, err := ecs.NewClientWithAccessKey(rid, ak, sk)
			if err != nil {
				return
			}
			client.GetConfig().WithTimeout(25 * time.Second)
			instances, err := aliyunDescribeAllInstances(ctx, client, rid)
			if err != nil || len(instances) == 0 {
				return
			}
			mu.Lock()
			blocks = append(blocks, csRegionBlock{
				RegionID:   rid,
				RegionName: rname,
				Instances:  instances,
			})
			mu.Unlock()
		}(rid, rname)
	}
	wg.Wait()
	return map[string]interface{}{"regions": blocks}, nil
}

func aliyunDescribeRegionNames(ak, sk string) (map[string]string, error) {
	client, err := ecs.NewClientWithAccessKey("cn-hangzhou", ak, sk)
	if err != nil {
		return nil, err
	}
	client.GetConfig().WithTimeout(20 * time.Second)
	req := ecs.CreateDescribeRegionsRequest()
	resp, err := client.DescribeRegions(req)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, r := range resp.Regions.Region {
		if r.RegionId != "" {
			loc := r.LocalName
			if loc == "" {
				loc = r.RegionId
			}
			m[r.RegionId] = loc
		}
	}
	return m, nil
}

func aliyunDescribeAllInstances(ctx context.Context, client *ecs.Client, regionID string) ([]csInstanceRow, error) {
	var out []csInstanceRow
	page := 1
	for {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		req := ecs.CreateDescribeInstancesRequest()
		req.RegionId = regionID
		req.PageNumber = requests.NewInteger(page)
		req.PageSize = requests.NewInteger(50)
		resp, err := client.DescribeInstances(req)
		if err != nil {
			return out, err
		}
		for _, ins := range resp.Instances.Instance {
			out = append(out, aliyunInstanceToRow(ins))
		}
		if len(resp.Instances.Instance) < 50 {
			break
		}
		page++
		if page > 200 {
			break
		}
	}
	return out, nil
}

func aliyunInstanceToRow(ins ecs.Instance) csInstanceRow {
	pub := ""
	if len(ins.PublicIpAddress.IpAddress) > 0 {
		pub = ins.PublicIpAddress.IpAddress[0]
	}
	if pub == "" && ins.InternetIp != "" {
		pub = ins.InternetIp
	}
	priv := ins.IntranetIp
	if priv == "" && len(ins.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		priv = ins.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}
	cpu := ins.CPU
	if cpu == 0 {
		cpu = ins.Cpu
	}
	mem := ins.Memory
	cfg := fmt.Sprintf("%d核 %dGB", cpu, mem/1024)
	if mem == 0 {
		cfg = ins.InstanceType
	}
	host := ins.InstanceName
	if host == "" {
		host = ins.HostName
	}
	osStr := ins.OSName
	exp := ins.ExpiredTime
	if exp == "" {
		exp = ins.AutoReleaseTime
	}
	return csInstanceRow{
		InstanceID: ins.InstanceId,
		HostName:   host,
		PublicIP:   pub,
		PrivateIP:  priv,
		Config:     cfg,
		OS:         osStr,
		Status:     ins.Status,
		CreatedAt:  ins.CreationTime,
		ExpiredAt:  exp,
	}
}

func aliyunRunCommand(cr *model.CloudServerCredential, regionID, instanceID, command, sysType string, waitSec int) (map[string]string, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	client, err := ecs.NewClientWithAccessKey(regionID, ak, sk)
	if err != nil {
		return nil, err
	}
	client.GetConfig().WithTimeout(30 * time.Second)

	cmdType := "RunShellScript"
	switch sysType {
	case "win_cmd":
		cmdType = "RunBatScript"
	case "powershell":
		cmdType = "RunPowerShellScript"
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(command))
	req := ecs.CreateRunCommandRequest()
	req.Type = cmdType
	req.CommandContent = b64
	req.ContentEncoding = "Base64"
	ids := []string{instanceID}
	req.InstanceId = &ids
	if waitSec > 0 {
		req.Timeout = requests.NewInteger(waitSec)
	}

	runResp, err := client.RunCommand(req)
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": err.Error()}, nil
	}
	invokeID := runResp.InvokeId
	if invokeID == "" {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "RunCommand 未返回 InvokeId"}, nil
	}

	deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
	var stdout, stderr string
	completed := false
	for time.Now().Before(deadline) {
		q := ecs.CreateDescribeInvocationResultsRequest()
		q.InvokeId = invokeID
		q.InstanceId = instanceID
		invResp, err := client.DescribeInvocationResults(q)
		if err != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		results := invResp.Invocation.InvocationResults.InvocationResult
		if len(results) == 0 {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		r0 := aliyunPickInvocationResult(results, instanceID)
		if aliyunInvocationTerminal(r0) {
			stdout = aliyunDecodeOutput(r0.Output)
			if r0.ErrorInfo != "" {
				stderr = r0.ErrorInfo
			}
			if r0.ErrorCode != "" {
				stderr = strings.TrimSpace(stderr + " " + r0.ErrorCode)
			}
			completed = true
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	errMsg := ""
	if !completed {
		errMsg = fmt.Sprintf("在 %d 秒内未等到云助手返回「已完成」状态，输出可能仍为空；请将前端「等待时间」调大（建议≥60秒）或凭 InvokeId 在控制台查看。InvokeId=%s", waitSec, invokeID)
	}
	return map[string]string{
		"stdout":    stdout,
		"stderr":    strings.TrimSpace(stderr),
		"invoke_id": invokeID,
		"error":     errMsg,
	}, nil
}

func aliyunPickInvocationResult(results []ecs.InvocationResult, instanceID string) ecs.InvocationResult {
	for _, r := range results {
		if r.InstanceId == instanceID {
			return r
		}
	}
	if len(results) > 0 {
		return results[0]
	}
	return ecs.InvocationResult{}
}

func aliyunInvocationTerminal(r ecs.InvocationResult) bool {
	if r.FinishedTime != "" {
		return true
	}
	for _, st := range []string{r.InvokeRecordStatus, r.InvocationStatus} {
		s := strings.ToLower(strings.TrimSpace(st))
		switch s {
		case "finished", "failed", "timeout", "stopped", "cancelled", "canceled", "success", "partialfailed", "partial_failed":
			return true
		}
	}
	return false
}

func aliyunDecodeOutput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return string(b)
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return string(b)
	}
	return s
}
