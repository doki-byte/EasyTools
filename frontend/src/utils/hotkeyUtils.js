export const LOCALSTORAGE_KEY = 'easytools.hotkey.showHide';
export const DEFAULT_HOTKEY = '';

export function normalizeAccelerator(acc) {
    if (!acc) return '';
    return acc.split('+').map(p => {
        const key = p.trim().toLowerCase();
        if (key === 'ctrl' || key === 'control') return 'Ctrl';
        if (key === 'alt' || key === 'option') return 'Alt';
        if (key === 'shift') return 'Shift';
        if (key === 'meta' || key === 'cmd' || key === 'command') return 'Cmd';
        if (key === 'space' || key === 'spacebar') return 'Space';

        // 功能键
        if (/^f[1-9][0-2]?$/.test(key)) return key.toUpperCase();
        if (['arrowup','arrowdown','arrowleft','arrowright'].includes(key))
            return key.replace('arrow','').charAt(0).toUpperCase() + key.slice(6);

        return key.length === 1 ? key.toUpperCase() : key.toUpperCase();
    }).join('+');
}

// 从键盘事件生成 accelerator 字符串
export function accelFromEvent(e) {
    let key = e.key;

    if (key === ' ') key = 'Space';
    if (key === 'Escape') return 'Escape';
    if (['Control','Shift','Alt','Meta'].includes(key)) return '';

    if (key.startsWith('Arrow')) key = key.slice(5).charAt(0).toUpperCase() + key.slice(6); // ArrowUp -> Up

    const parts = [];
    if (e.ctrlKey) parts.push('Ctrl');
    if (e.altKey) parts.push('Alt');
    if (e.shiftKey) parts.push('Shift');
    if (e.metaKey) parts.push('Cmd');

    parts.push(key.length === 1 ? key.toUpperCase() : key);
    return parts.join('+');
}
