package cloudSecurity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"EasyTools/app/model"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/tidwall/gjson"
)

func awsLoadConfig(ctx context.Context, cr *model.CloudServerCredential, region string) (aws.Config, error) {
	ak := strings.TrimSpace(cr.AccessKey)
	sk := strings.TrimSpace(cr.SecretKey)
	token := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "aws_session_token").String())
	if ak == "" || sk == "" {
		return aws.Config{}, fmt.Errorf("请填写 AccessKeyId 与 SecretAccessKey")
	}
	cred := credentials.NewStaticCredentialsProvider(ak, sk, token)
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(strings.TrimSpace(region)),
		config.WithCredentialsProvider(cred),
	)
}

func awsListInstances(cr *model.CloudServerCredential, regions []string) (map[string]interface{}, error) {
	ctx := context.Background()
	var blocks []csRegionBlock
	for _, rid := range regions {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		cfg, err := awsLoadConfig(ctx, cr, rid)
		if err != nil {
			continue
		}
		cli := ec2.NewFromConfig(cfg)
		paginator := ec2.NewDescribeInstancesPaginator(cli, &ec2.DescribeInstancesInput{})
		var rows []csInstanceRow
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				break
			}
			for _, res := range page.Reservations {
				for _, ins := range res.Instances {
					rows = append(rows, awsInstanceToRow(ins))
				}
			}
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

func awsInstanceToRow(ins types.Instance) csInstanceRow {
	id := aws.ToString(ins.InstanceId)
	host := ""
	for _, t := range ins.Tags {
		if aws.ToString(t.Key) == "Name" {
			host = aws.ToString(t.Value)
			break
		}
	}
	if host == "" {
		host = id
	}
	pub := aws.ToString(ins.PublicIpAddress)
	priv := aws.ToString(ins.PrivateIpAddress)
	cfg := string(ins.InstanceType)
	osName := aws.ToString(ins.PlatformDetails)
	if osName == "" {
		osName = aws.ToString(ins.ImageId)
	}
	status := ""
	if ins.State != nil {
		status = string(ins.State.Name)
	}
	created := ""
	if ins.LaunchTime != nil {
		created = ins.LaunchTime.In(time.Local).Format("2006-01-02 15:04:05")
	}
	return csInstanceRow{
		InstanceID: id,
		HostName:   host,
		PublicIP:   pub,
		PrivateIP:  priv,
		Config:     cfg,
		OS:         osName,
		Status:     status,
		CreatedAt:  created,
		ExpiredAt:  "",
	}
}

func awsRunCommand(cr *model.CloudServerCredential, regionID, instanceID, command, sysType string, waitSec int) (map[string]string, error) {
	ctx := context.Background()
	if waitSec < 30 {
		waitSec = 60
	}
	if waitSec > 3600 {
		waitSec = 3600
	}
	cfg, err := awsLoadConfig(ctx, cr, regionID)
	if err != nil {
		return nil, err
	}
	cli := ssm.NewFromConfig(cfg)

	doc := "AWS-RunShellScript"
	cmdLine := command
	switch sysType {
	case "powershell":
		doc = "AWS-RunPowerShellScript"
	case "win_cmd":
		doc = "AWS-RunPowerShellScript"
		cmdLine = "cmd /c " + command
	}

	to := int32(waitSec)
	if to < 30 {
		to = 60
	}
	sendOut, err := cli.SendCommand(ctx, &ssm.SendCommandInput{
		InstanceIds:    []string{instanceID},
		DocumentName:   aws.String(doc),
		TimeoutSeconds: aws.Int32(to),
		Parameters: map[string][]string{
			"commands": {cmdLine},
		},
	})
	if err != nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": err.Error()}, nil
	}
	if sendOut.Command == nil || sendOut.Command.CommandId == nil {
		return map[string]string{"stdout": "", "stderr": "", "invoke_id": "", "error": "SendCommand 未返回 CommandId"}, nil
	}
	cmdID := aws.ToString(sendOut.Command.CommandId)

	deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
	var stdout, stderr string
	completed := false
	for time.Now().Before(deadline) {
		inv, err := cli.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(cmdID),
			InstanceId: aws.String(instanceID),
		})
		if err != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		switch inv.Status {
		case ssmtypes.CommandInvocationStatusSuccess,
			ssmtypes.CommandInvocationStatusFailed,
			ssmtypes.CommandInvocationStatusCancelled,
			ssmtypes.CommandInvocationStatusTimedOut:
			completed = true
			stdout = aws.ToString(inv.StandardOutputContent)
			stderr = aws.ToString(inv.StandardErrorContent)
			if inv.Status != ssmtypes.CommandInvocationStatusSuccess && stderr == "" {
				stderr = string(inv.Status)
			}
		default:
			// InProgress, Pending, Delayed, etc.
		}
		if completed {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	errMsg := ""
	if !completed {
		errMsg = fmt.Sprintf("在 %d 秒内未完成 SSM 命令轮询；请确认实例已安装 SSM Agent 且具有 IAM 实例配置文件。CommandId=%s", waitSec, cmdID)
	}
	return map[string]string{
		"stdout":    stdout,
		"stderr":    strings.TrimSpace(stderr),
		"invoke_id": cmdID,
		"error":     errMsg,
	}, nil
}
