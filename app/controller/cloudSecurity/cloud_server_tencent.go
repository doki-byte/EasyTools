package cloudSecurity

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"EasyTools/app/model"

	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcprofile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tcvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	tclh "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/lighthouse/v20200324"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

func tcStrPtr(s string) *string { return &s }

func tencentDeref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func tencentListInstances(cr *model.CloudServerCredential, regions []string) (map[string]interface{}, error) {
	sid := strings.TrimSpace(cr.AccessKey)
	skey := strings.TrimSpace(cr.SecretKey)
	if sid == "" || skey == "" {
		return nil, fmt.Errorf("请填写 SecretId(AccessKey) 与 SecretKey")
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
		wg.Add(1)
		go func(rid string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			rows := tencentListInstancesInRegion(sid, skey, rid)
			if len(rows) == 0 {
				return
			}
			mu.Lock()
			blocks = append(blocks, csRegionBlock{
				RegionID:   rid,
				RegionName: rid,
				Instances:  rows,
			})
			mu.Unlock()
		}(rid)
	}
	wg.Wait()
	return map[string]interface{}{"regions": blocks}, nil
}

func tencentListInstancesInRegion(sid, skey, rid string) []csInstanceRow {
	var rows []csInstanceRow
	if cvmClient, err := tcvm.NewClientWithSecretId(sid, skey, rid); err == nil {
		var offset int64 = 0
		const limit int64 = 100
		for {
			req := tcvm.NewDescribeInstancesRequest()
			req.Offset = &offset
			req.Limit = tccommon.Int64Ptr(limit)
			resp, err := cvmClient.DescribeInstances(req)
			if err != nil || resp.Response == nil {
				break
			}
			set := resp.Response.InstanceSet
			if len(set) == 0 {
				break
			}
			for _, ins := range set {
				rows = append(rows, tencentInstanceToRow(ins, rid))
			}
			if len(set) < int(limit) {
				break
			}
			offset += limit
			if offset > 9500 {
				break
			}
		}
	}
	if lhClient, err := tclh.NewClient(
		tccommon.NewCredential(sid, skey),
		rid,
		tcprofile.NewClientProfile(),
	); err == nil {
		var offset int64 = 0
		var limit int64 = 100
		for {
			req := tclh.NewDescribeInstancesRequest()
			req.Offset = &offset
			req.Limit = &limit
			resp, err := lhClient.DescribeInstances(req)
			if err != nil || resp.Response == nil {
				break
			}
			set := resp.Response.InstanceSet
			if len(set) == 0 {
				break
			}
			for _, ins := range set {
				rows = append(rows, tencentLightInstanceToRow(ins, rid))
			}
			if len(set) < int(limit) {
				break
			}
			offset += limit
			if offset > 9500 {
				break
			}
		}
	}
	return rows
}

func tencentInstanceToRow(ins *tcvm.Instance, regionID string) csInstanceRow {
	if ins == nil {
		return csInstanceRow{}
	}
	host := tencentDeref(ins.InstanceName)
	if host == "" {
		host = tencentDeref(ins.InstanceId)
	}
	pub := tencentFirstIP(ins.PublicIpAddresses)
	priv := tencentFirstIP(ins.PrivateIpAddresses)
	cfg := ""
	if ins.CPU != nil && ins.Memory != nil {
		cfg = fmt.Sprintf("%d核 %dGiB", *ins.CPU, *ins.Memory)
	} else if ins.InstanceType != nil {
		cfg = *ins.InstanceType
	}
	osName := tencentDeref(ins.OsName)
	if osName == "" {
		osName = tencentDeref(ins.ImageId)
	}
	status := tencentDeref(ins.InstanceState)
	created := tencentDeref(ins.CreatedTime)
	exp := tencentDeref(ins.ExpiredTime)
	return csInstanceRow{
		InstanceID: tencentDeref(ins.InstanceId),
		HostName:   host,
		PublicIP:   pub,
		PrivateIP:  priv,
		Config:     cfg,
		OS:         osName,
		Status:     status,
		CreatedAt:  created,
		ExpiredAt:  exp,
	}
}

func tencentFirstIP(addrs []*string) string {
	for _, p := range addrs {
		if p == nil {
			continue
		}
		s := strings.TrimSpace(*p)
		if s != "" {
			return s
		}
	}
	return ""
}

func tencentLightFirstIP(addrs []*string) string {
	for _, p := range addrs {
		if p == nil {
			continue
		}
		s := strings.TrimSpace(*p)
		if s != "" {
			return s
		}
	}
	return ""
}

func tencentLightInstanceToRow(ins *tclh.Instance, regionID string) csInstanceRow {
	if ins == nil {
		return csInstanceRow{}
	}
	host := tencentDeref(ins.InstanceName)
	if host == "" {
		host = tencentDeref(ins.InstanceId)
	}
	pub := tencentLightFirstIP(ins.PublicAddresses)
	priv := tencentLightFirstIP(ins.PrivateAddresses)
	cfg := ""
	if ins.CPU != nil && ins.Memory != nil {
		cfg = fmt.Sprintf("%d核 %dGiB", *ins.CPU, *ins.Memory)
	}
	osName := ""
	status := tencentDeref(ins.InstanceState)
	created := tencentDeref(ins.CreatedTime)
	exp := tencentDeref(ins.ExpiredTime)
	return csInstanceRow{
		InstanceID: tencentDeref(ins.InstanceId),
		HostName:   host,
		PublicIP:   pub,
		PrivateIP:  priv,
		Config:     cfg,
		OS:         osName,
		Status:     status,
		CreatedAt:  created,
		ExpiredAt:  exp,
	}
}

func tencentRunCommand(cr *model.CloudServerCredential, regionID, instanceID, command, sysType string, waitSec int) (map[string]string, error) {
	sid := strings.TrimSpace(cr.AccessKey)
	skey := strings.TrimSpace(cr.SecretKey)
	if sid == "" || skey == "" {
		return nil, fmt.Errorf("请填写 SecretId 与 SecretKey")
	}
	if waitSec < 1 {
		waitSec = 60
	}
	if waitSec > 86400 {
		waitSec = 86400
	}
	cpf := tcprofile.NewClientProfile()
	client, err := tat.NewClient(
		tccommon.NewCredential(sid, skey),
		regionID,
		cpf,
	)
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(command))
	req := tat.NewRunCommandRequest()
	req.Content = &b64
	req.InstanceIds = []*string{&instanceID}
	cn := fmt.Sprintf("et-%d", time.Now().Unix())
	req.CommandName = &cn
	ct := "SHELL"
	switch sysType {
	case "powershell":
		ct = "POWERSHELL"
	case "win_cmd":
		ct = "BAT"
	}
	req.CommandType = &ct
	tu := uint64(waitSec)
	req.Timeout = &tu

	runResp, err := client.RunCommand(req)
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": err.Error()}, nil
	}
	if runResp.Response == nil || runResp.Response.InvocationId == nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "RunCommand 未返回 InvocationId"}, nil
	}
	invID := strings.TrimSpace(*runResp.Response.InvocationId)
	if invID == "" {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "InvocationId 为空"}, nil
	}

	deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
	var stdout, stderr string
	completed := false
	for time.Now().Before(deadline) {
		treq := tat.NewDescribeInvocationTasksRequest()
		hide := false
		treq.HideOutput = &hide
		treq.Filters = []*tat.Filter{
			{Name: tcStrPtr("invocation-id"), Values: []*string{&invID}},
			{Name: tcStrPtr("instance-id"), Values: []*string{&instanceID}},
		}
		tlim := uint64(20)
		treq.Limit = &tlim
		tresp, err := client.DescribeInvocationTasks(treq)
		if err != nil || tresp.Response == nil || len(tresp.Response.InvocationTaskSet) == 0 {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		task := tresp.Response.InvocationTaskSet[0]
		st := ""
		if task.TaskStatus != nil {
			st = strings.ToUpper(strings.TrimSpace(*task.TaskStatus))
		}
		switch st {
		case "SUCCESS", "FAILED", "TIMEOUT", "TASK_TIMEOUT", "CANCELLED", "TERMINATED", "DELIVER_FAILED", "START_FAILED":
			completed = true
			if task.TaskResult != nil && task.TaskResult.Output != nil {
				raw := strings.TrimSpace(*task.TaskResult.Output)
				if raw != "" {
					if b, e := base64.StdEncoding.DecodeString(raw); e == nil {
						stdout = string(b)
					} else {
						stdout = raw
					}
				}
			}
			if st != "SUCCESS" && stdout == "" {
				stderr = st
			}
		}
		if completed {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	errMsg := ""
	if !completed {
		errMsg = fmt.Sprintf("在 %d 秒内未拿到 TAT 终态，可调大等待时间；InvocationId=%s", waitSec, invID)
	}
	return map[string]string{
		"stdout":    stdout,
		"stderr":    strings.TrimSpace(stderr),
		"invoke_id": invID,
		"error":     errMsg,
	}, nil
}

