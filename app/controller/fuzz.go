package controller

import (
	"EasyTools/app/controller/system"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	clients = make(map[chan string]bool)
	mu      sync.Mutex
	cmd     *exec.Cmd
)

// SSE 广播消息
func broadcast(message string) {
	mu.Lock()
	defer mu.Unlock()
	for ch := range clients {
		select {
		case ch <- message:
		default:
		}
	}
}

// SSE 连接
func sseHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan string, 10)
	mu.Lock()
	clients[ch] = true
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(clients, ch)
		close(ch)
		mu.Unlock()
	}()

	ch <- "[SSE] connected"

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// stdout/stderr 日志
func streamReader(r io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					broadcast(line)
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				broadcast(fmt.Sprintf("[ERR] %v", err))
			}
			break
		}
	}
}

// result.json 推送
func streamResultJSON(path string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastSize := int64(0)

	for range ticker.C {
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		if fi.Size() == lastSize {
			continue
		}
		lastSize = fi.Size()

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			continue
		}
		if results, ok := parsed["results"].([]interface{}); ok {
			for _, res := range results {
				b, _ := json.Marshal(res)
				broadcast(string(b))
			}
		}
	}
}

// 请求结构
type Wordlist struct {
	Path string `json:"path"`
	Key  string `json:"key"`
}

type RunReq struct {
	Cmd       string            `json:"cmd"`
	URL       string            `json:"url"`
	Wordlists []Wordlist        `json:"wordlists"`
	Proxy     string            `json:"proxy"`
	Extra     string            `json:"extra"`
	Headers   map[string]string `json:"headers"`
	Threads   int               `json:"threads"`
}

// 启动 ffuf
func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORS(w)
		return
	}
	setCORS(w)

	var req RunReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Cmd == "" {
		req.Cmd = "ffuf"
	}
	if req.Threads <= 0 {
		req.Threads = 40
	}

	args := []string{}

	// 多 wordlist
	for _, wl := range req.Wordlists {
		key := wl.Key
		if key == "" {
			key = "FUZZ"
		}
		fmt.Println("Adding wordlist:", wl.Path, "key:", key)
		args = append(args, "-w", fmt.Sprintf("%s:%s", wl.Path, key))
	}

	args = append(args, "-u", req.URL)
	if req.Proxy != "" {
		args = append(args, "-x", req.Proxy)
	}
	args = append(args, "-t", fmt.Sprintf("%d", req.Threads))

	// Extra 参数
	if req.Extra != "" {
		parts := strings.Fields(req.Extra)
		for _, p := range parts {
			if p != "" {
				args = append(args, p)
			}
		}
	}

	// Headers
	for k, v := range req.Headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
	}

	baseDir := system.GetAppBaseDir()
	resultFile := filepath.Join(baseDir, "results.json")

	args = append(args, "-of", "json", "-o", resultFile)

	fmt.Println("Final ffuf args:", args)

	mu.Lock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		time.Sleep(100 * time.Millisecond)
	}

	ctx := context.Background()
	cmd = exec.CommandContext(ctx, req.Cmd, args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		mu.Unlock()
		http.Error(w, "start: "+err.Error(), http.StatusInternalServerError)
		return
	}
	mu.Unlock()

	go streamReader(stdout)
	go streamReader(stderr)
	go streamResultJSON(resultFile)

	go func(c *exec.Cmd) {
		_ = c.Wait()
		broadcast("[PROCESS EXITED] done")
		mu.Lock()
		if cmd == c {
			cmd = nil
		}
		mu.Unlock()
	}(cmd)

	w.Write([]byte("ok"))
}

// 停止
func stopHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	mu.Lock()
	defer mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		cmd = nil
		broadcast("[PROCESS KILLED]")
	}
	w.Write([]byte("stopped"))
}

// CORS
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func NewFuzzer() {
	http.HandleFunc("/fuzz/run", runHandler)
	http.HandleFunc("/fuzz/stop", stopHandler)
	http.HandleFunc("/fuzz/stream", sseHandler)

	log.Fatal(http.ListenAndServe("127.0.0.1:52870", nil))
}
