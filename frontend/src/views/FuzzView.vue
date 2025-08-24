<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElInput, ElButton, ElTable, ElTableColumn, ElCard, ElForm, ElFormItem, ElInputNumber } from 'element-plus'
import {QuestionFilled} from "@element-plus/icons-vue";

interface Wordlist { path: string; key: string }

const ffufPath = ref('ffuf')
const url = ref('')
const threads = ref(40)
const proxy = ref('')
const extra = ref('')
const wordlists = ref<Wordlist[]>([{ path:'', key:'FUZZ' }])
const logs = ref<string[]>([])
const jsonResults = ref<any[]>([])
const connected = ref(false)
let es: EventSource | null = null

function addWordlist(){ wordlists.value.push({ path:'', key:'FUZZ' }) }
function removeWordlist(idx: number){ wordlists.value.splice(idx,1) }

function start() {
  let finalURL = url.value
  wordlists.value.forEach(wl => {
    if(!finalURL.includes(wl.key)) {
      if(!finalURL.endsWith('/')) finalURL += '/'
      finalURL += wl.key
    }
  })

  fetch('http://127.0.0.1:52870/fuzz/run', {
    method:'POST',
    headers:{ 'Content-Type':'application/json' },
    body: JSON.stringify({
      cmd: ffufPath.value,
      url: finalURL,
      threads: threads.value,
      proxy: proxy.value,
      extra: extra.value,
      wordlists: wordlists.value
    })
  }).then(r=>{
    if(!r.ok) r.text().then(t=>addLog('[ERR] '+t))
    else { addLog('[JOB] started'); openStream() }
  }).catch(e=>addLog('[ERR] '+e.message))
}

function stop() {
  fetch('http://127.0.0.1:52870/fuzz/stop', { method:'POST' })
      .then(r=>r.text().then(t=>addLog('[STOP] '+t)))
      .catch(e=>addLog('[ERR] '+e.message))
}

function openStream() {
  if(es){ es.close(); es=null }
  es = new EventSource('http://127.0.0.1:52870/fuzz/stream')
  es.onopen = ()=>{ connected.value = true }
  es.onmessage = e=>{
    try {
      const obj = JSON.parse(e.data)
      jsonResults.value.push(obj)
      if(jsonResults.value.length>5000) jsonResults.value.splice(0,jsonResults.value.length-5000)
    } catch {
      addLog(e.data)
    }
  }
  es.onerror = ()=>{ connected.value=false }
}

function addLog(line: string){
  logs.value.push(line)
  if(logs.value.length>5000) logs.value.splice(0,logs.value.length-5000)
  requestAnimationFrame(()=>{ const el = document.querySelector('pre'); if(el) el.scrollTop = el.scrollHeight })
}

function clearLogs() { logs.value=[]; jsonResults.value=[] }

onMounted(()=>openStream())
onUnmounted(()=>{ if(es) es.close() })

const formattedLogs = computed(()=>logs.value.join("\n"))
const formattedJSON = computed(()=>{
  return jsonResults.value.map(j=>{
    let s = JSON.stringify(j,null,2)
    s = s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;")
    s = s.replace(/"(.*?)":/g,'"<span class="json-key">$1</span>":')
    s = s.replace(/"status": (\d{3})/g,(m,p)=>`"status": <span class="json-status200">${p}</span>`)
    return s
  }).join("\n\n")
})
</script>

<template>
  <div class="app-container">
    <div class="left-panel">
      <el-card class="panel-card">
        <h2 class="panel-title">FFUF 配置
          <el-tooltip
              effect="dark"
              content="ffuf建议配置成环境变量，这样使用方便些"
              placement="top"
          >
            <el-icon><QuestionFilled /></el-icon>
          </el-tooltip>
        </h2>
        <el-form
            label-width="top"
            class="left-align-form"
            style="width: 100%;"
        >
          <el-form-item label="ffuf 可执行路径" style="width: 100%;">
            <el-input v-model="ffufPath" placeholder="ffuf 或 /path/to/ffuf" style="width: 100%;" />
          </el-form-item>

          <el-form-item label="Target URL" style="width: 100%;">
            <el-input v-model="url" placeholder="https://127.0.0.1/FUZZ" style="width: 100%;" />
          </el-form-item>

          <el-form-item label="Threads" style="width: 100%;">
            <el-input-number v-model="threads" :min="1" style="width: 100%;" />
          </el-form-item>

          <el-form-item label="代理" style="width: 100%;">
            <el-input v-model="proxy" placeholder="socks5://127.0.0.1:1080" style="width: 100%;" />
          </el-form-item>

          <el-form-item label="Extra 参数" style="width: 100%;">
            <el-input v-model="extra" placeholder="-mc 200 -fs 0 -c -rate 10" style="width: 100%;" />
          </el-form-item>
        </el-form>

        <h3 class="sub-title">Wordlists</h3>
        <el-table :data="wordlists" stripe highlight-current-row class="highlight-table" style="margin-bottom:16px">
          <el-table-column prop="path" label="Path" min-width="150">
            <template #default="{ row }">
              <el-input v-model="row.path" placeholder="wordlist path"/>
            </template>
          </el-table-column>
          <el-table-column prop="key" label="Key" min-width="120">
            <template #default="{ row }">
              <el-input v-model="row.key" placeholder="关键字 FUZZ 默认"/>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ $index }">
              <el-button type="danger" size="small" @click="removeWordlist($index)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-button type="primary" @click="addWordlist" class="hover-btn mb-3">添加 Wordlist</el-button>

        <div class="action-buttons">
          <el-button type="success"  @click="start" class="hover-btn">Start</el-button>
          <el-button type="danger"  @click="stop" class="hover-btn">Stop</el-button>
          <el-button  @click="clearLogs" class="hover-btn">Clear</el-button>
          <span class="sse-status" :class="{ connected: connected, disconnected: !connected }">
            SSE: {{ connected ? 'connected':'disconnected' }}
          </span>
        </div>
        <!-- 添加的命令参数介绍卡片 -->
        <el-collapse>
          <el-collapse-item title="命令使用实例">
            <div class="params-card">
      <pre>
虚拟主机枚举
ffuf -u https://www.example.com -H "Host: FUZZ.example.com" -w /path/to/wordlist

多域名多目录同时扫描
ffuf -w subdomains.txt:SUB -w payloads/senstivejs.txt:FILE -u https://SUB/FILE -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; rv:78.0) Gecko/20100101 Firefox/78.0" -fs 0 -c -mc 200 -rate 10 -t 10

POST数据fuzzing
ffuf -w /path/to/postdata.txt -X POST -d "username=admin\&password=FUZZ" -u https://target/login.php -fc 401

目录递归扫描
ffuf -w list.txt -u http://site.com/FUZZ -e .js -recursion

</pre>
            </div>
          </el-collapse-item>
        </el-collapse>
        <el-collapse>
          <el-collapse-item title="命令参数介绍">
            <div class="params-card">
      <pre>
"compressed", true, "用于 curl 复制功能的虚拟标志（忽略）"
"i", true, "用于 curl 复制功能的虚拟标志（忽略）"
"k", false, "向后兼容的虚拟标志"
"or", "无结果时不创建输出文件"
"ac", "自动校准过滤选项"
"ach", "按主机自动校准"
"c", "彩色输出"
"json", "JSON 输出（换行分隔的 JSON 记录）"
"noninteractive", "禁用交互式控制台功能"
"s", "静默模式（不打印额外信息）"
"V", "显示版本信息"
"sf", "当 >95% 的响应返回 403 时停止"
"sa", "所有错误情况下停止（隐含 -sf 和 -se）"
"se", "发生意外错误时停止"
"v", "详细输出（显示完整 URL 和重定向位置）"
"r", "跟随重定向"
"ignore-body", "不获取响应内容"
"raw", "不对 URI 编码"
"recursion", "递归扫描（仅支持 FUZZ 关键字，URL 必须以它结尾）"
"http2", "使用 HTTP2 协议"
"D", "DirSearch 字典兼容模式（与 -e 配合使用）"
"ic", "忽略字典注释"
"maxtime", "进程最大运行时间（秒）"
"maxtime-job", "单任务最大运行时间（秒）"
"rate", "每秒请求速率"
"t", "并发线程数"
"recursion-depth", "最大递归深度"
"timeout", "HTTP 请求超时（秒）"
"input-num", "输入数量（与 --input-cmd 配合使用）"
"ack", "自动校准关键字"
"cc", "", "客户端证书（需同时定义客户端密钥）"
"ck", "", "客户端密钥（需同时定义客户端证书）"
"config", "", "从文件加载配置"
"scraperfile", "", "自定义爬虫文件路径"
"scrapers", "活动爬虫组"
"fmode", "过滤器运算符（and/or）"
"fl", "按响应行数过滤（逗号分隔的计数和范围）"
"fr", "正则表达式过滤"
"fs", "按 HTTP 响应大小过滤（逗号分隔的大小和范围）"
"fc", "按 HTTP 状态码过滤（逗号分隔的代码和范围）"
"ft", "按首字节响应时间过滤（毫秒），例如：>100 或 <100"
"fw", "按响应词数过滤（逗号分隔的计数和范围）"
"p", "请求间延迟秒数（支持随机范围），例如 \"0.1\" 或 \"0.1-2.0\""
"search", "从 ffuf 历史记录中搜索 FFUFHASH 有效负载"
"d", "POST 数据"
"data", "POST 数据（-d 的别名）"
"data-ascii", "POST 数据（-d 的别名）"
"data-binary", "POST 数据（-d 的别名）"
"X", "使用的 HTTP 方法"
"x", "代理 URL（SOCKS5/HTTP），例如：http://127.0.0.1:8080"
"replay-proxy", "使用此代理重放匹配的请求"
"recursion-strategy", "递归策略：\"default\"（基于重定向）或 \"greedy\"（贪婪匹配）"
"u", "目标 URL"
"sni", "目标 TLS SNI（不支持 FUZZ 关键字）"
"e", "扩展名列表（逗号分隔，扩展 FUZZ 关键字）"
"mode", "多字典操作模式：clusterbomb, pitchfork, sniper"
"input-shell", "用于运行命令的 shell"
"request", "包含原始 HTTP 请求的文件"
"request-proto", "原始请求使用的协议"
"mmode", "匹配器运算符（and/or）"
"ml", "匹配响应行数"
"mr", "正则表达式匹配"
"ms", "匹配 HTTP 响应大小"
"mc", "匹配 HTTP 状态码（或 \"all\" 匹配所有）"
"mt", "匹配首字节响应时间（毫秒），例如：>100 或 <100"
"mw", "匹配响应词数"
"audit-log", "写入审计日志（包含所有请求、响应和配置）"
"debug-log", "将所有内部日志写入指定文件"
"od", "存储匹配结果的目录路径"
"o", "输出到文件"
"of",  "输出文件格式：json, ejson, html, md, csv, ecsv（或 'all' 表示所有格式）"
"acc", "自定义自动校准字符串（可多次使用，隐含 -ac）"
"acs", "自定义自动校准策略（可多次使用，隐含 -ac）"
"b", "Cookie 数据 `\"NAME1=VALUE1; NAME2=VALUE2\"`"
"cookie", "Cookie 数据（-b 的别名）"
"H", "请求头 `\"Name: Value\"`（冒号分隔，可多次使用）"
"input-cmd", "生成输入的命令（需配合 --input-num 使用，覆盖 -w）"
"w", "字典文件路径和（可选）关键字，格式：'/path/to/wordlist:KEYWORD'"
"enc", "关键字编码器，例如：'FUZZ:urlencode b64encode'"
}</pre>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-card>
    </div>

    <div class="right-panel">
      <el-card class="panel-card half-panel">
        <h2 class="panel-title">实时日志</h2>
        <pre v-html="formattedLogs"></pre>
      </el-card>

      <el-card class="panel-card half-panel">
        <h2 class="panel-title">JSON 结果</h2>
        <pre v-html="formattedJSON"></pre>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
/* 主容器响应式 */
.app-container {
  display: flex;
  gap: 20px;
  height: auto; /* 改为自动高度 */
  min-height: 95vh; /* 最小高度 */
  padding: 20px;
  background: #f9fafb;
  overflow-y: auto; /* 允许整体滚动 */
}

.left-panel {
  width: 42%;
  flex-shrink: 0; /* 防止收缩 */
}

.right-panel {
  width: 58%;
  display: flex;
  flex-direction: column;
  gap: 20px;
  flex-shrink: 0; /* 防止收缩 */
}

.panel-card {
  padding: 24px;
  background: #ffffff;
  border-radius: 14px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.08);
  display: flex;
  flex-direction: column;
  height: auto; /* 改为自动高度 */
}

.half-panel {
  flex: 1; /* 均分空间 */
  overflow: visible; /* 移除溢出隐藏 */
}

pre {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.5;
  height: auto; /* 改为自动高度 */
  overflow: visible; /* 移除溢出设置 */
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 500px; /* 设置最大高度 */
  overflow-y: auto; /* 内容超出时滚动 */
}

h2, h3 {
  margin-bottom: 16px;
  color: #1f2937;
  font-weight: 600;
}

.left-align-form .el-form-item {
  width: 100%;
}
.left-align-form .el-input,
.left-align-form .el-input-number {
  width: 100%;
}

.action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 16px;
  align-items: center;
}

.params-card {
  background: #f8f8f8;
  border-radius: 4px;
  padding: 15px;
  max-height: 400px;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.4;
}

.params-card pre {
  margin: 0;
  white-space: pre-wrap;
}

.el-collapse {
  margin-top: 15px;
}

.sse-status.connected {
  color: #16a34a;
  font-weight: 600;
}
.sse-status.disconnected {
  color: #ef4444;
  font-weight: 600;
}

.el-form-item {
  margin-bottom: 18px;
}
.el-table .el-input {
  height: 36px;
}
.el-button {
  border-radius: 8px;
  transition: all 0.25s;
  font-weight: 500;
}
.hover-btn:hover {
  background-color: #e1efff !important;
  color: #1d4ed8 !important;
}

.panel-title {
  margin-bottom: 20px;
  font-size: 20px;
}
.sub-title {
  margin-bottom: 12px;
  font-size: 16px;
  color: #374151;
  font-weight: 500;
}

.highlight-table .el-table__row:hover {
  background-color: #e1efff;
}

/* 响应式布局 */
@media (max-width: 900px) {
  .app-container {
    flex-direction: column;
    height: auto; /* 自动高度 */
    overflow-y: visible; /* 移除溢出设置 */
  }

  .left-panel, .right-panel {
    width: 100%;
    flex-shrink: 0; /* 防止收缩 */
  }

  .right-panel {
    flex-direction: column;
    height: auto; /* 自动高度 */
  }

  .half-panel {
    height: auto; /* 自动高度 */
    max-height: none; /* 移除最大高度限制 */
  }

  pre {
    max-height: 300px; /* 移动端设置较小最大高度 */
    overflow-y: auto; /* 内容超出时滚动 */
  }
}

/* 自定义滚动条样式 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: #c5c5c5;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>