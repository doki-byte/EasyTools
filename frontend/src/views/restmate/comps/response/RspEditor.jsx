import React, { useEffect, useRef, useState } from "react";
import { Editor } from "@monaco-editor/react";
import { toast } from "react-toastify";
import {LuBraces, LuCopy, LuWrapText} from "react-icons/lu";
import Tippy from "@tippyjs/react";

// 辅助函数：将3位十六进制颜色转换为6位
const expandColor = (color) => {
  if (color && color.startsWith('#') && color.length === 4) {
    return `#${color[1]}${color[1]}${color[2]}${color[2]}${color[3]}${color[3]}`;
  }
  return color;
};

const RspEditor = ({ lang = "JSON", bodyContent }) => {
  const [editorLang, setEditorLang] = useState(lang);
  const [wrap, setWrap] = useState(true);
  const [themeLoaded, setThemeLoaded] = useState(false); // 新增状态跟踪主题是否加载
  const editorRef = useRef(null);
  const monacoRef = useRef(null);

  const safeBody = bodyContent ?? "";

  const onEditorMount = (editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    setMonacoTheme(monaco);
    setThemeLoaded(true); // 标记主题已加载
  };

  const setMonacoTheme = (monaco) => {
    // 获取背景色并确保是6位格式
    let bg = getComputedStyle(document.documentElement).getPropertyValue("--color-brand") || "#1e1e1e";
    bg = expandColor(bg); // 处理3位颜色值的情况

    // 使用6位完整格式定义所有颜色
    monaco.editor.defineTheme("restTheme", {
      base: "vs-dark",
      inherit: true,
      rules: [],
      colors: {
        "editor.background": bg,
        "editorCursor.foreground": "#000000",
        "editor.lineHighlightBackground": "#ffffff",
        "editor.selectionBackground": "#0280fa",
        // 添加其他可能需要的颜色配置
        "editorLineNumber.foreground": "#858585",
        "editorLineNumber.activeForeground": "#cccccc"
      },
    });
    monaco.editor.setTheme("restTheme");
  };

  const formatBody = () => {
    if (!editorRef.current) return;
    try {
      const action = editorRef.current.getAction("editor.action.formatDocument");
      if (action) {
        editorRef.current.updateOptions({ readOnly: false });
        action.run().finally(() => editorRef.current.updateOptions({ readOnly: true }));
      }
    } catch (e) {
      console.warn("格式化失败", e);
      toast.error("格式化失败，请检查内容格式");
    }
  };

  const onCopy = () => {
    if (!safeBody) return;
    navigator.clipboard.writeText(safeBody)
        .then(() => toast.success("复制到剪贴板！"))
        .catch(() => toast.error("复制失败，请手动复制"));
  };

  return (
      <div className="h-full w-full flex flex-col">
        <div className="flex justify-end items-center gap-x-2 text-txtsec">
          <Tippy content="格式化">
            <div className="hover:text-lit cursor-pointer relative group" onClick={formatBody}>
              <LuBraces size="16" />
            </div>
          </Tippy>
          <Tippy content="换行">
            <div className="hover:text-lit cursor-pointer" onClick={() => setWrap(!wrap)}>
              <LuWrapText size="16" />
            </div>
          </Tippy>
          <Tippy content="复制响应">
            <div className="hover:text-lit cursor-pointer" onClick={onCopy}>
              <LuCopy size="16" />
            </div>
          </Tippy>
        </div>
        <div className="flex-1 border border-gray-500">
          <Editor
              onMount={onEditorMount}
              width="100%"
              height="100%"
              value={safeBody}
              language={editorLang.toLowerCase()}
              theme={themeLoaded ? "restTheme" : "vs-dark"} // 主题加载完成前使用默认主题
              options={{
                readOnly: true,
                wordWrap: wrap ? "on" : "off",
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
              }}
          />
        </div>
      </div>
  );
};

export default React.memo(RspEditor);
