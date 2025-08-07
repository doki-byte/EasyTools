package request

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/imroc/req/v3"
)

type FreeProxyHubProxy struct {
	page int
	url  string
	reg  string
}

func (fp *FreeProxyHubProxy) Name() string {
	return "ProxyHub"
}

var FreeProxyHub = FreeProxyHubProxy{
	page: 20,
	url:  "https://proxyhub.me/zh/cn-socks5-proxy-list.html",
	reg:  `<td data-title="IP">([\d.]+)</td>\s*<td data-title="PORT">(\d+)</td>`,
}

func (fp *FreeProxyHubProxy) Fetch() ([]string, error) {
	var wg sync.WaitGroup
	ch := make(chan []string, fp.page)
	errCh := make(chan error, fp.page)

	for i := 1; i <= fp.page; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			proxies, err := fp.fetchPage(page)
			if err != nil {
				errCh <- err
				return
			}
			ch <- proxies
		}(i)
	}

	go func() {
		wg.Wait()
		close(ch)
		close(errCh)
	}()

	var allProxies []string
	for {
		select {
		case proxies, ok := <-ch:
			if !ok {
				return allProxies, nil
			}
			allProxies = append(allProxies, proxies...)
		case err, ok := <-errCh:
			if !ok {
				continue
			}
			return nil, err
		}
	}
}

func (fp *FreeProxyHubProxy) fetchPage(page int) ([]string, error) {
	client := req.C()
	resp, err := client.R().Get(fmt.Sprintf("%s%d", fp.url, page))
	if err != nil {
		return nil, err
	}

	body := resp.String()

	ipPortRegex := regexp.MustCompile(fp.reg)
	matches := ipPortRegex.FindAllStringSubmatch(body, -1)

	var proxies []string
	for _, match := range matches {
		if len(match) == 3 {
			ip := match[1]
			port := match[2]
			proxies = append(proxies, fmt.Sprintf("%s:%s", ip, port))
		}
	}

	return proxies, nil
}
