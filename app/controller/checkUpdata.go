package controller

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-version"
)

type Update struct {
	Base
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type CheckResult struct {
	HasUpdate      bool           `json:"hasUpdate"`
	CurrentVersion string         `json:"currentVersion"`
	LatestRelease  *GitHubRelease `json:"latestRelease"`
}

func CheckVersion() *Update {
	return &Update{}
}

func (u *Update) GetLatestRelease() (*CheckResult, error) {
	owner := "doki-byte"
	repo := "EasyTools"
	currentVersion := "v1.9.0" // 请确保与前端保持一致

	latest, err := CheckLatestRelease(owner, repo)
	if err != nil {
		return nil, err
	}

	currentVer, err := version.NewVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("当前版本解析失败: %v", err)
	}

	latestVer, err := version.NewVersion(latest.TagName)
	if err != nil {
		return nil, fmt.Errorf("最新版本解析失败: %v", err)
	}

	hasUpdate := latestVer.GreaterThan(currentVer)

	return &CheckResult{
		HasUpdate:      hasUpdate,
		CurrentVersion: currentVersion,
		LatestRelease:  latest,
	}, nil
}

func CheckLatestRelease(owner, repo string) (*GitHubRelease, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("请求创建失败: %s", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %s", err)
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %s", err)
	}

	fmt.Println("最新版本信息:", release)
	return &release, nil
}
