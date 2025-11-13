import React, { useState, useRef, useCallback } from "react";
import { LuBraces } from "react-icons/lu";
import { toast } from "react-toastify";
import { useStore } from "../../store/store";

const BodyJson = ({ tabId, bodyRaw }) => {
    const textareaRef = useRef(null);
    const updateReqBody = useStore((x) => x.updateReqBody);

    // 格式化 JSON
    const formatBody = useCallback(() => {
        if (!textareaRef.current) return;

        try {
            const text = (bodyRaw || "").trim();
            if (!text) {
                toast.info("请输入 JSON 数据");
                return;
            }

            const parsed = JSON.parse(text);
            const formatted = JSON.stringify(parsed, null, 2);
            updateReqBody(tabId, "bodyRaw", formatted);
            toast.success("已格式化 JSON");
        } catch (error) {
            toast.error(`JSON 格式错误: ${error.message}`);
        }
    }, [tabId, updateReqBody, bodyRaw]);

    // 处理内容变化
    const handleChange = useCallback((e) => {
        updateReqBody(tabId, "bodyRaw", e.target.value);
    }, [tabId, updateReqBody]);

    // 处理按键事件
    const handleKeyDown = useCallback((e) => {
        if (e.ctrlKey && e.key === 'Enter') {
            e.preventDefault();
            formatBody();
        }

        if (e.key === 'Tab') {
            e.preventDefault();
            const start = e.target.selectionStart;
            const end = e.target.selectionEnd;
            const value = e.target.value;

            // 使用4个空格作为制表符
            e.target.value = value.substring(0, start) + '    ' + value.substring(end);
            e.target.selectionStart = e.target.selectionEnd = start + 2;
            handleChange(e);
        }
    }, [formatBody, handleChange]);

    return (
        <div className="pt-2 h-full w-full relative">
            <div className="h-full w-full border border-gray-300 rounded relative bg-white overflow-hidden">
                {/* 文本编辑框 */}
                <textarea
                    ref={textareaRef}
                    value={bodyRaw || ""}
                    onChange={handleChange}
                    onKeyDown={handleKeyDown}
                    className="w-full h-full p-4 font-mono text-sm leading-relaxed text-gray-800 bg-white resize-none outline-none border-none"
                    spellCheck={false}
                    placeholder="输入 JSON 数据..."
                    style={{
                        fontSize: '14px',
                        lineHeight: '1.5',
                        fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace'
                    }}
                />
            </div>

            <div className="absolute top-3 right-3 z-20">
                <button
                    onClick={formatBody}
                    className="p-2 hover:bg-gray-100 rounded transition-colors bg-white border border-gray-300 shadow-sm"
                    title="格式化 JSON (Ctrl+Enter)"
                    type="button"
                >
                    <LuBraces size={16} className="text-gray-600" />
                </button>
            </div>
        </div>
    );
};

export default React.memo(BodyJson);