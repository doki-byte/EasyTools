import { nextTick } from 'vue';
import { ElMessageBox } from 'element-plus';

/** 通用删除确认口令（消息中展示完整对象名，口令保持简短便于手输） */
export const DANGER_DELETE_TOKEN = '确认删除';

export interface ConfirmDangerOptions {
  title?: string;
  message: string;
  /** 须与输入框内容完全一致（默认前后 trim） */
  expectedInput?: string;
  confirmButtonText?: string;
  cancelButtonText?: string;
  trimInput?: boolean;
  /** 对话框 message 区域类型 */
  type?: 'warning' | 'error';
  closeOnClickModal?: boolean;
  /** 与 ElMessageBox 一致：关闭按钮与取消区分，关闭时 reject 'close' */
  distinguishCancelAndClose?: boolean;
}

function attachClipboardGuards(tokenClass: string) {
  const stopAt = Date.now() + 1500;
  const block = (e: Event) => {
    e.preventDefault();
    e.stopPropagation();
  };
  const tryAttach = (): boolean => {
    const root = document.querySelector(`.${tokenClass}`);
    if (!root) return false;
    const input = root.querySelector(
      '.el-message-box__input input, .el-message-box__input textarea'
    ) as HTMLInputElement | HTMLTextAreaElement | null;
    if (!input) return false;
    for (const ev of ['paste', 'copy', 'cut', 'drop'] as const) {
      input.addEventListener(ev, block);
    }
    return true;
  };
  const tick = () => {
    if (tryAttach()) return;
    if (Date.now() < stopAt) requestAnimationFrame(tick);
  };
  nextTick(() => requestAnimationFrame(tick));
}

/**
 * 高危操作二次确认：须手动输入指定文本；输入框禁止复制/粘贴/剪切/拖入。
 * 取消时抛出与 ElMessageBox 一致的 reject（如 'cancel'）。
 */
export async function confirmDangerousAction(options: ConfirmDangerOptions): Promise<void> {
  const title = options.title ?? '危险操作确认';
  const expected = options.expectedInput ?? DANGER_DELETE_TOKEN;
  const trim = options.trimInput !== false;
  const hint = '请在下框中手动输入以下口令以继续（禁止复制粘贴；须与口令完全一致）：';
  const fullMessage = `${options.message}\n\n${hint}\n「${expected}」`;
  const cls = `et-dg-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 9)}`;
  attachClipboardGuards(cls);

  await ElMessageBox.prompt(fullMessage, title, {
    type: options.type ?? 'warning',
    confirmButtonText: options.confirmButtonText ?? '确定',
    cancelButtonText: options.cancelButtonText ?? '取消',
    customClass: cls,
    inputPlaceholder: expected,
    closeOnClickModal: options.closeOnClickModal ?? false,
    distinguishCancelAndClose: options.distinguishCancelAndClose,
    inputValidator: (val) => {
      const v = trim ? String(val).trim() : String(val);
      const e = trim ? String(expected).trim() : String(expected);
      if (v === e) return true;
      return '口令不一致，请逐项手动输入';
    },
    inputErrorMessage: '口令不一致',
  });
}
