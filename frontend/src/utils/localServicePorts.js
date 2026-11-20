/**
 * 本地服务端口配置获取（兼容「网页预览 / 浏览器调试」等无 Wails 绑定的场景）：
 *
 * 优先级：
 *  1) Wails 绑定 GetPortConfig —— 桌面端窗口内（一次成功即缓存到 localStorage）；
 *  2) 本地存储缓存 —— 上次绑定成功后写入的快照；
 *  3) 直接探测本机 EasyTools 主服务 /api/port-config —— 网页预览时可自动连上
 *     正在运行的桌面主程序（端口为动态分配，不能依赖旧的固定 52867/52870 硬编码）。
 *
 * 探测全部失败时返回 null（此时说明本机没有可用的 EasyTools 主程序后端）。
 */
import { GetPortConfig } from '../../wailsjs/go/controller/Server';

const CACHE_KEY = 'easytools:local:port-config:v1';
const LEGACY_MAIN_PORT = 52867;
const MAIN_PORT_RANGE_START = 52000;
const MAIN_PORT_RANGE_END = 52999;

let cached = null;
let inflight = null;

function validPortConfig(cfg) {
  return !!(
    cfg &&
    Number(cfg.mainPort) > 0 &&
    Number(cfg.fuzzPort) > 0
  );
}

function readCache() {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const j = JSON.parse(raw);
    return validPortConfig(j) ? j : null;
  } catch (e) {
    return null;
  }
}

function writeCache(cfg) {
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(cfg));
  } catch (e) {
    /* ignore */
  }
}

/** 探测单个端口的 /api/port-config，成功返回配置对象，否则 null */
async function probePort(port, timeoutMs = 500) {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);
  try {
    const res = await fetch(`http://127.0.0.1:${port}/api/port-config`, {
      cache: 'no-store',
      signal: ctrl.signal,
    });
    if (!res.ok) return null;
    const data = await res.json();
    return validPortConfig(data) ? data : null;
  } catch (e) {
    return null;
  } finally {
    clearTimeout(timer);
  }
}

async function probeHttpConfig() {
  // 先探测历史固定端口（老版本主程序）
  const legacy = await probePort(LEGACY_MAIN_PORT, 400);
  if (legacy) return legacy;
  // 再分片并发探测动态主端口区间
  const ports = [];
  for (let p = MAIN_PORT_RANGE_START; p <= MAIN_PORT_RANGE_END; p++) ports.push(p);
  const CHUNK = 60;
  for (let i = 0; i < ports.length; i += CHUNK) {
    const batch = ports.slice(i, i + CHUNK);
    const results = await Promise.all(batch.map((p) => probePort(p, 350)));
    const found = results.find(Boolean);
    if (found) return found;
  }
  return null;
}

/**
 * 获取本地 EasyTools 服务端口配置；无后端可用时返回 null。
 * @param {{force?: boolean}} options
 */
export async function discoverLocalPortConfig({ force = false } = {}) {
  if (cached && !force) return cached;
  if (inflight) return inflight;
  inflight = (async () => {
    // 1) Wails 绑定（桌面端窗口内正常返回）
    try {
      const cfg = await GetPortConfig();
      if (validPortConfig(cfg)) {
        writeCache(cfg);
        cached = cfg;
        return cfg;
      }
    } catch (e) {
      // 无绑定（浏览器/网页预览），继续降级探测
    }
    // 2) 本地缓存快照
    const fromCache = readCache();
    if (fromCache) {
      cached = fromCache;
      return fromCache;
    }
    // 3) 探测本机正在运行的主程序（动态端口）
    const httpCfg = await probeHttpConfig();
    if (httpCfg) {
      writeCache(httpCfg);
      cached = httpCfg;
      return httpCfg;
    }
    return null;
  })();
  try {
    return await inflight;
  } finally {
    inflight = null;
  }
}

/** 探测失败后清除内存缓存，便于下次重新完整探测 */
export function resetLocalPortConfigCache() {
  cached = null;
}
