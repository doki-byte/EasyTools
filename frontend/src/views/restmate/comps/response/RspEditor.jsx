import React, { useState, useRef, useCallback, useEffect } from "react";
import { toast } from "react-toastify";
import { LuBraces, LuCopy, LuWrapText } from "react-icons/lu";
import Tippy from "@tippyjs/react";

const RspEditor = ({ lang = "JSON", bodyContent, onBodyChange }) => {
    const [wrap, setWrap] = useState(true);
    const [displayContent, setDisplayContent] = useState("");
    const contentRef = useRef(null);

    // 当bodyContent变化时更新显示内容
    useEffect(() => {
        setDisplayContent(bodyContent ?? "");
    }, [bodyContent]);

    // 语法高亮函数
    const highlightContent = useCallback((text, language) => {
        if (!text) return "";

        // 转义HTML特殊字符
        const escapedText = text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");

        if (language.toLowerCase() === "json") {
            try {
                // 尝试解析JSON来确保格式正确
                JSON.parse(text);
                return escapedText
                    .replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, (match) => {
                        let cls = "json-number";
                        if (/^"/.test(match)) {
                            if (/:$/.test(match)) {
                                cls = "json-key";
                            } else {
                                cls = "json-string";
                            }
                        } else if (/true|false/.test(match)) {
                            cls = "json-boolean";
                        } else if (/null/.test(match)) {
                            cls = "json-null";
                        }
                        return `<span class="${cls}">${match}</span>`;
                    });
            } catch {
                // 如果不是有效的JSON，返回普通文本
                return escapedText;
            }
        } else if (language.toLowerCase() === "html") {
            return escapedText
                .replace(/(&lt;\/?)([a-zA-Z][a-zA-Z0-9]*)([^&]*?&gt;)/g, '$1<span class="html-tag">$2</span>$3')
                .replace(/([a-zA-Z-]+)=&quot;([^&quot;]*)&quot;/g, '<span class="html-attr">$1</span>=&quot;<span class="html-value">$2</span>&quot;')
                .replace(/&lt;!--([^&]*)--&gt;/g, '<span class="html-comment">&lt;!--$1--&gt;</span>');
        }

        return escapedText;
    }, []);

    // 格式化内容
    const formatBody = useCallback(() => {
        if (!displayContent.trim()) return;

        try {
            let formatted = displayContent;

            if (lang.toLowerCase() === "json") {
                const parsed = JSON.parse(displayContent);
                formatted = JSON.stringify(parsed, null, 2);
            } else if (lang.toLowerCase() === "html") {
                // 简单的 HTML 格式化
                formatted = displayContent
                    .replace(/(>)(<)(\/*)/g, '$1\n$2$3')
                    .replace(/^(.*)$/gm, (match) => {
                        // 简单的缩进处理
                        const trimmed = match.trim();
                        if (trimmed.startsWith('</')) {
                            return match;
                        }
                        return match;
                    });
            }

            setDisplayContent(formatted);
            // 如果有回调函数，通知父组件内容已更新
            if (onBodyChange) {
                onBodyChange(formatted);
            }
            toast.success(`已格式化 ${lang}`);
        } catch (error) {
            toast.error(`格式化失败: ${error.message}`);
        }
    }, [displayContent, lang, onBodyChange]);

    const onCopy = useCallback(() => {
        if (!displayContent) return;
        navigator.clipboard.writeText(displayContent)
            .then(() => toast.success("复制到剪贴板！"))
            .catch(() => toast.error("复制失败，请手动复制"));
    }, [displayContent]);

    const highlightedContent = highlightContent(displayContent, lang);

    return (
        <div className="h-full w-full flex flex-col">
            <div className="flex justify-end items-center gap-x-2 text-txtsec mb-2">
                <Tippy content="格式化">
                    <button
                        className="hover:text-lit cursor-pointer relative group p-1 rounded hover:bg-gray-100"
                        onClick={formatBody}
                    >
                        <LuBraces size="16" />
                    </button>
                </Tippy>
                <Tippy content={wrap ? "关闭换行" : "开启换行"}>
                    <button
                        className="hover:text-lit cursor-pointer p-1 rounded hover:bg-gray-100"
                        onClick={() => setWrap(!wrap)}
                    >
                        <LuWrapText size="16" />
                    </button>
                </Tippy>
                <Tippy content="复制响应">
                    <button
                        className="hover:text-lit cursor-pointer p-1 rounded hover:bg-gray-100"
                        onClick={onCopy}
                    >
                        <LuCopy size="16" />
                    </button>
                </Tippy>
            </div>

            <div className="flex-1 border border-gray-300 relative bg-white overflow-auto rounded">
        <pre
            ref={contentRef}
            className={`p-3 font-mono text-sm h-full overflow-auto bg-gray-50 ${
                wrap ? "whitespace-pre-wrap" : "whitespace-pre overflow-x-auto"
            }`}
            style={{
                margin: 0,
                minHeight: '100px'
            }}
            dangerouslySetInnerHTML={{
                __html: highlightedContent || '<span class="text-gray-400">无内容</span>'
            }}
        />
            </div>
        </div>
    );
};

export default React.memo(RspEditor);