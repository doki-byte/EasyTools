/** 云服务器「命令执行区 / 终端」与会话相关的本地键（与 CloudServerPanel 一致） */

export const CLOUD_SERVER_TERMINAL_SESSION_KEY = 'easytools:cloud_server:terminal_v1';
export const CLOUD_SERVER_ECHO_LOCAL_KEY = 'easytools:cloud_server:echo_v1';
export const CLOUD_SERVER_ECHO_LEGACY_KEY = 'king_cloud_server_echo_v1';

/**
 * 退出应用时清空终端输出缓存。
 * - sessionStorage：清除本次会话中的终端文本
 * - localStorage 旧 echo 键：防止下次启动时再次把历史灌回终端
 */
export function clearCloudServerTerminalStorageOnExit() {
  try {
    if (typeof sessionStorage !== 'undefined') {
      sessionStorage.removeItem(CLOUD_SERVER_TERMINAL_SESSION_KEY);
    }
    localStorage.removeItem(CLOUD_SERVER_ECHO_LOCAL_KEY);
    localStorage.removeItem(CLOUD_SERVER_ECHO_LEGACY_KEY);
  } catch (e) {
    /* ignore */
  }
}
