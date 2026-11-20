/**
 * HTTP 语言高亮配置 - 完全重写版
 * 修复空行检测问题，确保正确高亮
 */

/**
 * 注册 HTTP 语言到 Monaco Editor
 * @param {object} monaco - Monaco Editor 实例
 */
export const registerHTTPLanguage = (monaco) => {
  // 如果已经注册过，直接返回
  if (monaco.languages.getLanguages().some(lang => lang.id === 'http')) {
    return;
  }

  // 注册 HTTP 语言
  monaco.languages.register({ id: 'http' });

  // 设置语法高亮规则 - 简化版本，专注于解决问题
  monaco.languages.setMonarchTokensProvider('http', {
    defaultToken: 'text',
    ignoreCase: true,
    keywords: [
      'GET', 'POST', 'PUT', 'DELETE', 'HEAD', 'OPTIONS', 'PATCH', 'CONNECT', 'TRACE',
      'HTTP/1.0', 'HTTP/1.1', 'HTTP/2', 'HTTP/2.0'
    ],
    // 单状态 tokenizer：不再区分 header/body，始终尝试匹配常见结构
    tokenizer: {
      root: [
        // 请求/响应起始行
        [/^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|CONNECT|TRACE)\s+/i, {
          token: 'http-method',
          next: '@requestLine'
        }],
        [/^HTTP\/[0-9.]+/, {
          token: 'http-version',
          next: '@responseLine'
        }],

        // 纯空行：保持在 root，避免影响后续头部识别
        [/^\s*$/, 'white'],

        // 头部行：Key: Value（允许前置空白）
        [/^\s*[A-Za-z0-9-]+:\s?.*/, 'http-header'],

        // JSON / XML / form 直接在 root 中识别
        [/\{/, { token: 'delimiter.bracket', next: '@jsonBody' }],
        [/\[/, { token: 'delimiter.bracket', next: '@jsonBody' }],
        [/<\?xml/, { token: 'delimiter.bracket', next: '@xmlBody' }],
        [/</, { token: 'delimiter.bracket', next: '@xmlBody' }],
        [/^[A-Za-z0-9_-]+=|&/, { token: 'delimiter.form', next: '@formBody' }],

        // 其他文本
        [/./, 'text']
      ],

      requestLine: [
        [/(\/[^\s]*)/, 'http-url'],
        [/\s+(HTTP\/[0-9.]+)/, 'http-version'],
        [/$/, { token: 'white', next: '@root' }],
        // 兜底：无论是否匹配到，行结束后回到 root，避免状态滞留
        [/.*$/, { token: 'text', next: '@root' }]
      ],

      responseLine: [
        [/\s+(\d{3})/, 'http-status-code'],
        [/\s+([^\r\n]+)/, 'http-status-text'],
        [/$/, { token: 'white', next: '@root' }],
        // 兜底：无论是否匹配到，行结束后回到 root
        [/.*$/, { token: 'text', next: '@root' }]
      ],

      jsonBody: [
        [/\{/, { token: 'delimiter.bracket', next: '@push' }],
        [/\}/, { token: 'delimiter.bracket', next: '@pop' }],
        [/\[/, { token: 'delimiter.bracket', next: '@push' }],
        [/\]/, { token: 'delimiter.bracket', next: '@pop' }],
        [/"([^"\\]|\\.)*"/, 'string.json'],
        [/\b(true|false|null)\b/, 'keyword.json'],
        [/-?\d+(\.\d+)?([eE][+-]?\d+)?/, 'number.json'],
        [/[:,]/, 'delimiter.json'],
        [/\s+/, 'white'],
        [/./, 'string.json']
      ],

      xmlBody: [
        [/<\?xml/, 'keyword.xml'],
        [/<!--/, 'comment.xml', '@comment'],
        [/<(\/?)([A-Za-z][A-Za-z0-9_-]*)/, ['delimiter.bracket', 'tag.xml']],
        [/[^<]+/, 'text.xml'],
        [/</, 'delimiter.bracket']
      ],

      comment: [
        [/[^-]+/, 'comment.xml'],
        [/-->/, 'comment.xml', '@pop'],
        [/./, 'comment.xml']
      ],

      formBody: [
        [/&/, 'delimiter.form'],
        [/=/, 'delimiter.form'],
        [/$/, { token: 'white', next: '@root' }],
        [/./, 'text']
      ]
    }
  });

  // 注册格式化器
  monaco.languages.registerDocumentFormattingEditProvider('http', {
    provideDocumentFormattingEdits: function (model, options, token) {
      try {
        const text = model.getValue();
        const formattedText = formatHTTPPacket(text);

        return [{
          range: model.getFullModelRange(),
          text: formattedText
        }];
      } catch (error) {
        console.error('Formatting error:', error);
        return [];
      }
    }
  });

  // 定义黑色主题
  monaco.editor.defineTheme('http-dark-theme', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'http-method', foreground: '4EC9B0', fontStyle: 'bold' },
      { token: 'http-version', foreground: '569CD6' },
      { token: 'http-status-code', foreground: 'B5CEA8', fontStyle: 'bold' },
      { token: 'http-status-text', foreground: 'CE9178' },
      { token: 'http-header', foreground: '9CDCFE', fontStyle: 'bold' },
      { token: 'http-url', foreground: 'D7BA7D' },
      { token: 'http-host', foreground: 'CE9178' },
      { token: 'http-content-type', foreground: 'C586C0' },
      { token: 'http-charset', foreground: 'C586C0' },
      { token: 'http-encoding', foreground: 'CE9178' },
      { token: 'http-connection', foreground: 'CE9178' },
      { token: 'http-number', foreground: 'B5CEA8' },
      { token: 'string.json', foreground: 'CE9178' },
      { token: 'keyword.json', foreground: '569CD6' },
      { token: 'number.json', foreground: 'B5CEA8' },
      { token: 'delimiter.json', foreground: 'D4D4D4' },
      { token: 'keyword.xml', foreground: '808080' },
      { token: 'tag.xml', foreground: '569CD6' },
      { token: 'text.xml', foreground: 'D4D4D4' },
      { token: 'comment.xml', foreground: '6A9955' },
      { token: 'delimiter.form', foreground: 'D4D4D4' },
      { token: 'delimiter.bracket', foreground: 'FFD700' },
      { token: 'text', foreground: 'D4D4D4' },
      { token: 'white', foreground: 'D4D4D4' }
    ],
    colors: {
      'editor.background': '#1E1E1E',
      'editor.foreground': '#D4D4D4',
      'editor.lineHighlightBackground': '#2A2D2E',
      'editorCursor.foreground': '#AEAFAD',
      'editor.selectionBackground': '#264F78',
      'editor.selectionHighlightBackground': '#ADD6FF26',
      'editorLineNumber.foreground': '#858585',
      'editorLineNumber.activeForeground': '#C6C6C6',
      'editor.findMatchBackground': '#515C6A',
      'editor.findMatchHighlightBackground': '#38485C',
      'editorBracketMatch.background': '#0064001A',
      'editorBracketMatch.border': '#888888',
      'editorIndentGuide.background': '#404040',
      'editorIndentGuide.activeBackground': '#707070',
      'editorHoverWidget.background': '#252526',
      'editorHoverWidget.border': '#454545',
      'minimap.background': '#1E1E1E',
      'scrollbarSlider.background': '#424242',
      'scrollbarSlider.hoverBackground': '#4E4E4E',
      'scrollbarSlider.activeBackground': '#565656',
      'editorSuggestWidget.background': '#252526',
      'editorSuggestWidget.border': '#454545',
      'editorSuggestWidget.selectedBackground': '#2A2D2E'
    }
  });

  console.log('HTTP language registered with simplified tokenizer');
};

/**
 * HTTP 数据包格式化函数
 * @param {string} httpText - 原始 HTTP 文本
 * @returns {string} 格式化后的 HTTP 文本
 */
function formatHTTPPacket(httpText) {
  if (!httpText || typeof httpText !== 'string') {
    return httpText || '';
  }

  const lines = httpText.split(/\r?\n/);

  // 找到首个空行作为分隔（兼容只有一个空行的情况）
  const separatorIndex = lines.findIndex(line => line.trim() === '');
  const hasBody = separatorIndex !== -1 && separatorIndex < lines.length - 1;

  const headerLines = separatorIndex === -1 ? lines : lines.slice(0, separatorIndex);
  const bodyLines = hasBody ? lines.slice(separatorIndex + 1) : [];

  const result = [];

  // 处理头部，保留结尾空格去除
  headerLines.forEach(line => result.push(line.trimEnd()));

  // 补回分隔空行（单空行，保持原始协议习惯）
  if (hasBody) {
    result.push('');
  }

  if (hasBody) {
    const bodyText = bodyLines.join('\n').trim();

    if (bodyText) {
      const trimmedBody = bodyText.trim();
      const looksLikeJSON =
        (trimmedBody.startsWith('{') && trimmedBody.endsWith('}')) ||
        (trimmedBody.startsWith('[') && trimmedBody.endsWith(']'));

      if (looksLikeJSON) {
        try {
          const parsed = JSON.parse(trimmedBody);
          const formattedJSON = JSON.stringify(parsed, null, 2);
          result.push(formattedJSON);
        } catch (e) {
          result.push(...bodyLines);
        }
      } else {
        result.push(...bodyLines);
      }
    }
  }

  return result.join('\n');
}

/**
 * 格式化 HTTP 响应包
 * @param {string} responseText - HTTP 响应文本
 * @returns {string} 格式化后的响应文本
 */
export const formatHTTPResponse = (responseText) => {
  return formatHTTPPacket(responseText);
};

/**
 * 格式化 HTTP 请求包
 * @param {string} requestText - HTTP 请求文本
 * @returns {string} 格式化后的请求文本
 */
export const formatHTTPRequest = (requestText) => {
  return formatHTTPPacket(requestText);
};

/**
 * 自动格式化显示 HTTP 内容
 * @param {string} text - HTTP 文本
 * @param {boolean} isResponse - 是否为响应包
 * @returns {string} 格式化后的文本
 */
export const autoFormatHTTP = (text, isResponse = false) => {
  if (!text || typeof text !== 'string') return text || '';

  // 首先尝试使用Monaco的格式化
  try {
    return formatHTTPPacket(text);
  } catch (error) {
    console.warn('Auto-format failed:', error);
    // 如果失败，返回原始文本
    return text;
  }
};

/**
 * 粗略检测文本类型用于 Monaco 语言选择
 * - 优先识别 HTTP 报文
 * - 其次识别 JSON / XML
 * - 其他情况回落到 plaintext
 * @param {string} text
 * @returns {'http' | 'json' | 'xml' | 'plaintext'}
 */
export const detectPacketLanguage = (text) => {
  const content = (text || '').trim();
  if (!content) return 'plaintext';

  const firstLine = content.split(/\r?\n/, 1)[0];
  const httpMethodLine = /^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|CONNECT|TRACE)\s+\S+\s+HTTP\/[0-9.]+/i;
  const httpStatusLine = /^HTTP\/[0-9.]+\s+\d{3}/i;
  const httpHeaderLine = /^(Host|Content-Type|User-Agent|Accept|Cookie|Authorization|Content-Length|Connection):/i;

  if (httpMethodLine.test(firstLine) || httpStatusLine.test(firstLine) || httpHeaderLine.test(firstLine)) {
    return 'http';
  }

  const looksJson = (content.startsWith('{') && content.endsWith('}')) || (content.startsWith('[') && content.endsWith(']'));
  if (looksJson) {
    try {
      JSON.parse(content);
      return 'json';
    } catch (e) {
      // fall through
    }
  }

  const looksXml = /^<\?xml/i.test(content) || /^<[^>]+>/.test(content);
  if (looksXml) return 'xml';

  return 'plaintext';
};

export default {
  registerHTTPLanguage,
  formatHTTPResponse,
  formatHTTPRequest,
  autoFormatHTTP,
  detectPacketLanguage
};