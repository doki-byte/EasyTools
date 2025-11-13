package request

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/imroc/req/v3"
)

type FreeHappyProxy struct {
	page int
	url  string
	reg  string
}

func (fp *FreeHappyProxy) Name() string {
	return "开心代理"
}

var FreeHappy = FreeHappyProxy{
	page: 20,
	url:  "http://www.kxdaili.com/dailiip/1/",
	reg:  "<tr[\\s\\S]*?<td>(.*?)</td>\\s*?<td>(.*?)</td>",
}

func (fp *FreeHappyProxy) Fetch() ([]string, error) {
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

func (fp *FreeHappyProxy) fetchPage(page int) ([]string, error) {
	client := req.C()
	resp, err := client.R().Get(fmt.Sprintf("%s%d.html", fp.url, page))
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
