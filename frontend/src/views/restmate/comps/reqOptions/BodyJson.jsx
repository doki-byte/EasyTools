import React, { useEffect, useRef } from "react";
import { Editor } from "@monaco-editor/react";
import { LuBraces } from "react-icons/lu";
import { toast } from "react-toastify";
import { useStore } from "../../store/store";
import { ENVIRONMENT_REGEX } from "../../utils/utils";

const BodyJson = ({ tabId, bodyRaw }) => {
    const editorRef = useRef(null);
    const monacoRef = useRef(null);
    const decorationsRef = useRef([]);
    const updateReqBody = useStore((x) => x.updateReqBody);

    // 初始化白色主题
    function monacoSetup(monaco) {
        monacoRef.current = monaco;
        monaco.editor.defineTheme("restThemeLight", {
            base: "vs",
            inherit: true,
            rules: [],
            colors: {
                "editor.background": "#ffffff",
                "editorCursor.foreground": "#000000",
                "editor.lineHighlightBackground": "#f0f0f0",
                "editor.selectionBackground": "#cce5ff",
            },
        });
        monaco.editor.setTheme("restThemeLight");
    }

    // 高亮环境变量
    function updateDecorations(editor, monaco) {
        if (!editor) return;
        const model = editor.getModel();
        if (!model) return;

        const matches = [...model.getValue().matchAll(ENVIRONMENT_REGEX)];
        const decorations =
            matches &&
            matches.map((match) => {
                const start = match.index;
                const end = start + match[0].length;
                const startPos = model.getPositionAt(start);
                const endPos = model.getPositionAt(end);
                return {
                    range: new monaco.Range(
                        startPos.lineNumber,
                        startPos.column,
                        endPos.lineNumber,
                        endPos.column
                    ),
                    options: {
                        inlineClassName: "manacoEnvFound",
                    },
                };
            });
        decorationsRef.current = editor.deltaDecorations(
            decorationsRef.current,
            decorations
        );
    }

    // 编辑器挂载
    function handleEditorDidMount(editor, monaco) {
        editorRef.current = editor;

        // Ctrl+Enter 格式化
        editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
            formatBody();
        });

        updateDecorations(editor, monaco);
        editor.onDidChangeModelContent(() => {
            updateDecorations(editor, monaco);
        });
    }

    const formatBody = () => {
        if (editorRef.current) {
            editorRef.current.getAction("editor.action.formatDocument").run();
            toast.success("已格式化响应");
        }
    };

    return (
        <div className="pt-2 h-full w-full">
            <div className="h-full w-full border border-lines">
                <Editor
                    width="100%"
                    height="100%"
                    defaultLanguage="json"
                    value={bodyRaw || ""}
                    onChange={(e) => updateReqBody(tabId, "bodyRaw", e)}
                    loading={<div className="bg-none"></div>}
                    options={{
                        readOnly: false,
                        minimap: { enabled: false },
                        wordWrap: "on",
                        scrollBeyondLastLine: false,
                        formatOnPaste: true,
                        formatOnType: true,
                        lineNumbersMinChars: 2,
                        cursorBlinking: "smooth",
                        overviewRulerBorder: false,
                    }}
                    beforeMount={monacoSetup}
                    onMount={handleEditorDidMount}
                />
            </div>
        </div>
    );
};

export default React.memo(BodyJson);
