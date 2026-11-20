package cloudSecurity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"EasyTools/app/model"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	hwiam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	iammodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	iamregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
	"github.com/tidwall/gjson"
)

func takeoverStrPtr(s string) *string { return &s }

func takeoverRandAlnum(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
			continue
		}
		b[i] = chars[j.Int64()]
	}
	return string(b)
}

// takeoverRamPassword 满足常见云密码策略（大小写、数字、特殊字符）。
func takeoverRamPassword() string {
	const upper = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const lower = "abcdefghijkmnopqrstuvwxyz"
	const digit = "23456789"
	const special = "!@#$%&*-_"
	var out strings.Builder
	sets := []string{upper, lower, digit, special}
	for _, s := range sets {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(len(s))))
		out.WriteByte(s[j.Int64()])
	}
	all := upper + lower + digit + special
	for out.Len() < 18 {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		out.WriteByte(all[j.Int64()])
	}
	return out.String()
}

func takeoverDefaultUsername(prefix string) string {
	p := strings.TrimSpace(prefix)
	if p != "" {
		return p
	}
	return "et_take_" + takeoverRandAlnum(8)
}

func aliyunRAMClient(ak, sk string) (*ram.Client, error) {
	return ram.NewClientWithAccessKey("cn-hangzhou", ak, sk)
}

func aliyunSTSAccountID(ak, sk string) (string, error) {
	c, err := sts.NewClientWithAccessKey("cn-hangzhou", ak, sk)
	if err != nil {
		return "", err
	}
	c.GetConfig().WithTimeout(15 * time.Second)
	req := sts.CreateGetCallerIdentityRequest()
	resp, err := c.GetCallerIdentity(req)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(resp.AccountId)
	if id == "" {
		return "", fmt.Errorf("GetCallerIdentity 未返回 AccountId")
	}
	return id, nil
}

func errStr(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

func takeoverAliyunConsole(cr *model.CloudServerCredential, username, policyName string, attachPolicy bool) (map[string]interface{}, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("请填写 AccessKey 与 SecretKey")
	}
	user := takeoverDefaultUsername(username)
	password := takeoverRamPassword()

	rc, err := aliyunRAMClient(ak, sk)
	if err != nil {
		return nil, err
	}

	cu := ram.CreateCreateUserRequest()
	cu.UserName = user
	cu.DisplayName = "EasyTools-RAM"
	if _, err := rc.CreateUser(cu); err != nil {
		return nil, fmt.Errorf("RAM CreateUser: %w", err)
	}

	cl := ram.CreateCreateLoginProfileRequest()
	cl.UserName = user
	cl.Password = password
	cl.PasswordResetRequired = requests.NewBoolean(false)
	if _, err := rc.CreateLoginProfile(cl); err != nil {
		return nil, fmt.Errorf("RAM CreateLoginProfile: %w", err)
	}

	attachedName := ""
	if attachPolicy {
		pn := strings.TrimSpace(policyName)
		if pn == "" {
			pn = "AdministratorAccess"
		}
		attachedName = pn
		ap := ram.CreateAttachPolicyToUserRequest()
		ap.UserName = user
		ap.PolicyType = "System"
		ap.PolicyName = pn
		if _, err := rc.AttachPolicyToUser(ap); err != nil {
			return nil, fmt.Errorf("RAM AttachPolicyToUser(%s): %w", pn, err)
		}
	}

	alias := ""
	if gar, err := rc.GetAccountAlias(ram.CreateGetAccountAliasRequest()); err == nil {
		alias = strings.TrimSpace(gar.AccountAlias)
	}
	accountID, aerr := aliyunSTSAccountID(ak, sk)
	loginSuffix := alias
	if loginSuffix == "" {
		loginSuffix = accountID
	}
	consoleLogin := user
	if loginSuffix != "" {
		consoleLogin = user + "@" + loginSuffix
	}

	out := map[string]interface{}{
		"vendor":             "aliyun",
		"username":           user,
		"password":           password,
		"login_url":          "https://signin.aliyun.com/",
		"console_login_name": consoleLogin,
		"account_alias":      alias,
		"account_id":         accountID,
		"sts_error":          errStr(aerr),
		"attach_policy":      attachPolicy,
		"policy_attached":    attachedName,
	}
	return out, nil
}

func huaweiIAMClient(ak, sk, projectID string) (*hwiam.IamClient, error) {
	b := basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk)
	if projectID != "" {
		b = b.WithProjectId(projectID)
	}
	auth, err := b.SafeBuild()
	if err != nil {
		return nil, err
	}
	reg, err := iamregion.SafeValueOf("cn-north-4")
	if err != nil {
		return nil, err
	}
	hc, err := hwiam.IamClientBuilder().WithRegion(reg).WithCredential(auth).SafeBuild()
	if err != nil {
		return nil, err
	}
	return hwiam.NewIamClient(hc), nil
}

func takeoverHuaweiConsole(cr *model.CloudServerCredential, username string) (map[string]interface{}, error) {
	ak, sk := strings.TrimSpace(cr.AccessKey), strings.TrimSpace(cr.SecretKey)
	projectID := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "huawei_project_id").String())
	domainID := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "huawei_domain_id").String())
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("请填写 AccessKey 与 SecretKey")
	}
	user := takeoverDefaultUsername(username)
	password := takeoverRamPassword()

	client, err := huaweiIAMClient(ak, sk, projectID)
	if err != nil {
		return nil, err
	}

	domainName := ""
	if domainID == "" {
		dresp, derr := client.KeystoneListAuthDomains(&iammodel.KeystoneListAuthDomainsRequest{})
		if derr != nil {
			return nil, fmt.Errorf("获取 domain_id 失败（可在扩展 JSON 设置 huawei_domain_id）: %w", derr)
		}
		if dresp.Domains == nil || len(*dresp.Domains) == 0 {
			return nil, fmt.Errorf("KeystoneListAuthDomains 未返回账号，请在扩展 JSON 中填写 huawei_domain_id")
		}
		d0 := (*dresp.Domains)[0]
		domainID = strings.TrimSpace(d0.Id)
		domainName = strings.TrimSpace(d0.Name)
	}

	enabled := true
	pwdNoForceReset := false
	req := &iammodel.CreateUserRequest{
		Body: &iammodel.CreateUserRequestBody{
			User: &iammodel.CreateUserOption{
				Name:        user,
				DomainId:    domainID,
				Password:    takeoverStrPtr(password),
				AccessMode:  takeoverStrPtr("default"),
				Enabled:     &enabled,
				PwdStatus:   &pwdNoForceReset,
				Description: takeoverStrPtr("created-by-easytools"),
			},
		},
	}
	if _, err := client.CreateUser(req); err != nil {
		em := strings.ToLower(err.Error())
		if strings.Contains(em, "not authorized") || strings.Contains(em, "forbidden") || strings.Contains(em, `"code":403`) {
			return nil, fmt.Errorf("IAM CreateUser 被拒绝（403）：当前 AccessKey 对应身份无权执行 iam:users:create。请使用具备「IAM 用户管理」权限的主账号密钥，或为该 IAM 用户授予 iam:users:* / IAM FullAccess 同类策略后再试。详情：%w", err)
		}
		return nil, fmt.Errorf("IAM CreateUser: %w", err)
	}

	return map[string]interface{}{
		"vendor":            "huawei",
		"username":          user,
		"password":          password,
		"login_url":         "https://console.huaweicloud.com/",
		"domain_id":         domainID,
		"domain_name":       domainName,
		"iam_login_hint":    "控制台登录时「账号名」填主账号名（或 domain_name），「用户名」填上方 IAM 用户名，「密码」为上方密码。",
		"permission_notice": "新建 IAM 用户默认无策略，请在控制台「用户」中为其授权或加入用户组。",
	}, nil
}

func takeoverUcloudIAM(cr *model.CloudServerCredential, username, email, displayName string) (map[string]interface{}, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, fmt.Errorf("UCloud 开启控制台登录必须填写有效邮箱（LoginProfileStatus=Active）")
	}
	user := takeoverDefaultUsername(username)
	pub := strings.TrimSpace(cr.AccessKey)
	priv := strings.TrimSpace(cr.SecretKey)
	projectID := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "ucloud_project_id").String())
	if projectID == "" {
		projectID = strings.TrimSpace(gjson.Get(cr.ExtraJSON, "project_id").String())
	}
	if pub == "" || priv == "" {
		return nil, fmt.Errorf("UCloud 需填写 PublicKey 与 PrivateKey")
	}

	params := map[string]string{
		"Action":           "CreateUser",
		"PublicKey":        pub,
		"UserName":         user,
		"Email":            email,
		"AccessKeyStatus":  "Active",
		"LoginProfileStatus": "Active",
		"RequestUUID":      fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	if displayName != "" {
		params["DisplayName"] = displayName
	}
	if projectID != "" {
		params["ProjectId"] = projectID
	}
	sig, err := ucloudSignature(priv, params)
	if err != nil {
		return nil, err
	}
	params["Signature"] = sig

	body, err := ucloudAPIGET(params)
	if err != nil {
		return nil, err
	}
	if gjson.GetBytes(body, "RetCode").Int() != 0 {
		return nil, fmt.Errorf("UCloud CreateUser: %s", gjson.GetBytes(body, "Message").String())
	}

	return map[string]interface{}{
		"vendor":               "ucloud",
		"username":             user,
		"email":                email,
		"login_url":            "https://console.ucloud.cn/",
		"access_key_id":        gjson.GetBytes(body, "AccessKeyID").String(),
		"access_key_secret":    gjson.GetBytes(body, "AccessKeySecret").String(),
		"user_email":           gjson.GetBytes(body, "UserEmail").String(),
		// 不再在本地自动设置 UCloud 控制台密码，改为提示用户前往邮箱激活账号并设置密码
		"console_password":     "",
		"console_password_set": false,
		"console_password_err": "",
		"console_note":         "UCloud 已通过 CreateUser 创建子账号。请前往该子账号邮箱查收激活邮件，按邮件链接完成账号激活及登录密码设置；下方为子账号 API 密钥（AccessKey / SecretKey），请妥善保存。",
	}, nil
}

func takeoverConsoleDispatch(cr *model.CloudServerCredential, jsonStr string) (map[string]interface{}, error) {
	username := strings.TrimSpace(gjson.Get(jsonStr, "username").String())
	email := strings.TrimSpace(gjson.Get(jsonStr, "email").String())
	displayName := strings.TrimSpace(gjson.Get(jsonStr, "display_name").String())
	policyName := strings.TrimSpace(gjson.Get(jsonStr, "policy_name").String())
	attachPolicy := true
	if gjson.Get(jsonStr, "attach_policy").Exists() {
		attachPolicy = gjson.Get(jsonStr, "attach_policy").Bool()
	}

	switch cr.Vendor {
	case "aliyun":
		return takeoverAliyunConsole(cr, username, policyName, attachPolicy)
	case "huawei":
		return takeoverHuaweiConsole(cr, username)
	case "ucloud":
		return takeoverUcloudIAM(cr, username, email, displayName)
	default:
		return nil, fmt.Errorf("不支持的 vendor: %s", cr.Vendor)
	}
}

// CloudServerTakeoverConsoleJSON 调用云厂商 OpenAPI 创建子用户（阿里云 RAM 登录配置、华为 IAM 用户、UCloud IAM 用户）。
// JSON: credential_id, username(可选), email(UCloud 必填), display_name(可选), policy_name(阿里云系统策略名，可选), attach_policy(阿里云，默认 true)。
func (c *CloudServer) CloudServerTakeoverConsoleJSON(jsonStr string) (string, error) {
	cr, err := c.credByID(uint(gjson.Get(jsonStr, "credential_id").Uint()))
	if err != nil {
		return "", err
	}
	payload, err := takeoverConsoleDispatch(cr, jsonStr)
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(b), nil
	}
	payload["error"] = ""
	raw, _ := json.Marshal(payload)
	return string(raw), nil
}
