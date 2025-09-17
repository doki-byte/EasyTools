import React, { useEffect, useRef, useState } from "react";
import { Editor } from "@monaco-editor/react";
import { toast } from "react-toastify";
import {LuBraces, LuCopy, LuWrapText} from "react-icons/lu";
import Tippy from "@tippyjs/react";

const RspEditor = ({ lang = "JSON", bodyContent }) => {
  const [editorLang, setEditorLang] = useState(lang);
  const [wrap, setWrap] = useState(true);
  const editorRef = useRef(null);
  const monacoRef = useRef(null);

  const safeBody = bodyContent ?? "";

  const onEditorMount = (editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    setMonacoTheme(monaco);
  };

  const setMonacoTheme = (monaco) => {
    const bg = getComputedStyle(document.documentElement).getPropertyValue("--color-brand") || "#1e1e1e";
    monaco.editor.defineTheme("restTheme", {
      base: "vs-dark",
      inherit: true,
      rules: [],
      colors: {
        "editor.background": bg,
        "editorCursor.foreground": "#000",
        "editor.lineHighlightBackground": "#ffffff",
        "editor.selectionBackground": "#0280fa",
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
    }
  };

  const onCopy = () => {
    if (!safeBody) return;
    navigator.clipboard.writeText(safeBody).then(() => toast.success("复制到剪贴板！"));
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
            <div className="hover:text-lit cursor-pointer" onClick={() => onCopy(bodyContent)}>
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
              theme="restTheme"
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
