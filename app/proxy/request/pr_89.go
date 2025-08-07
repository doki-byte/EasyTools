package request

import (
	"regexp"

	"github.com/imroc/req/v3"
)

type Free89Proxy struct {
	url string
	reg string
}

var Free89 = Free89Proxy{
	url: "https://www.89ip.cn/tqdl.html?num=60&address=&kill_address=&port=&kill_port=&isp=",
	reg: `(\d+\.\d+\.\d+\.\d+):\d+`,
}

func (fp *Free89Proxy) Name() string {
	return "89代理"
}

func (fp *Free89Proxy) Fetch() ([]string, error) {
	client := req.C()
	resp, err := client.R().Get(fp.url)
	if err != nil {
		return nil, err
	}

	body := resp.String()

	ipPortRegex := regexp.MustCompile(fp.reg)
	matches := ipPortRegex.FindAllString(body, -1)

	return matches, nil
}
