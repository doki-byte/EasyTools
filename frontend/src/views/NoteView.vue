<template>
  <div class="app-container note-views">
    <el-button class="btn btn-secondary toggle-sidebar-btn" @click="toggleSidebar" circle>
      <el-icon>
        <component :is="isSidebarVisible ? Fold : Expand" />
      </el-icon>
    </el-button>
    <div v-if="isSidebarVisible"  class="file-manager">
      <div class="toolbar">
        <button class="btn" @click="openDirectory">
          <i class="fas fa-folder-open"></i> 打开
        </button>
        <button class="btn btn-secondary" @click="createFile">
          <i class="fas fa-file"></i> 新建
        </button>
        <button class="btn btn-secondary" @click="createFolder">
          <i class="fas fa-folder-plus"></i> 文件夹
        </button>
      </div>
      <div class="tree-container">
        <ul class="tree-list">
          <TreeNode
              v-for="node in treeData"
              :key="node.path"
              :node="node"
              :depth="0"
              :selected="selectedFile"
              @toggle="toggle"
              @delete="onDeleteItem"
              @rename="onRenameItem"
              @context-menu="onContextMenu"
          />

        </ul>
      </div>
    </div>

    <div class="editor-container">
      <div class="editor-header">
        <div class="editor-title" v-if="selectedFile">
          <i class="fas fa-file-alt"></i>
          {{ selectedFile.name }}
          <span class="save-status">
            <span class="indicator" :class="{ saved: saved }"></span>
            {{ saved ? '已保存' : '未保存' }}
          </span>
        </div>
        <div v-else class="editor-title">
          <i class="fas fa-sticky-note"></i>
          备忘录
        </div>

        <div class="editor-title">
          <input
              v-model="searchKeyword"
              @input="onSearch"
              placeholder="搜索内容..."
              class="search-input"
          />
          <button v-if="searchKeyword" class="btn btn-secondary" @click="prevMatch" :disabled="searchMatches.length === 0">↑</button>
          <button v-if="searchKeyword" class="btn btn-secondary" @click="nextMatch" :disabled="searchMatches.length === 0">↓</button>
        </div>

        <div class="header-actions">
          <button class="btn" @click="saveContent" :disabled="!selectedFile || saved">
            <i class="fas fa-save"></i> 保存
          </button>
          <button class="btn btn-secondary" @click="togglePreview">
            <i class="fas fa-eye"></i> 预览
          </button>
          <button class="btn btn-secondary" @click="toggleEdit">
            <i class="fas fa-eye"></i> <span v-if="showEdit">关闭</span><span v-else>打开</span>编辑窗口
          </button>
        </div>
      </div>
      <div class="editor-main-horizontal">
        <div v-if="!selectedFile" class="empty-state">
          <i class="fas fa-sticky-note"></i>
          <h3>未选择文件</h3>
          <p>请从左侧选择一个文件进行编辑</p>
        </div>
        <div v-else class="editor-content-wrapper" style="display: flex; height: 100%; overflow: hidden;">
          <!-- 编辑区 -->
          <textarea
              v-if="showEdit"
              class="editor-textarea"
              v-model="fileContent"
              @input="onContentChange"
              @paste="onPaste"
              ref="editor"
              :style="{
      flex: showPreview ? '1 1 50%' : '1 1 100%',
      minWidth: 0
    }"
          ></textarea>

          <!-- 预览区 -->
          <div
              v-if="showPreview"
              class="preview-wrapper"
              :style="{
      flex: showEdit ? '1 1 50%' : '1 1 100%',
      minWidth: 0
    }"
          >
            <div class="preview-content" v-html="previewContent"></div>
          </div>
        </div>


      </div>
    </div>

    <!-- 右键菜单 -->
    <div
        v-if="contextMenuVisible"
        class="custom-context-menu"
        :style="{ top: contextMenuPos.y + 'px', left: contextMenuPos.x + 'px' }"
        @click.stop
    >
      <ul class="context-menu-list">
        <li @click="onContextMenuCommand('rename')">重命名</li>
        <li @click="onContextMenuCommand('delete')">删除</li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, h, ref } from 'vue';
import { ElMessageBox, ElMessage } from 'element-plus';
import { marked } from 'marked';
import {
  CreateFile,
  CreateFolder,
  DeleteItem,
  GetFiles, GetNotesDir,
  OpenDirectory,
  ReadFile,
  RenameItem,
  SaveFile
} from "../../wailsjs/go/controller/Note";
import {Expand, Fold} from "@element-plus/icons-vue";

interface Node {
  name: string;
  path: string;
  isDir: boolean;
  expanded: boolean;
  children: Node[];
}

const TreeNode = defineComponent({
  name: 'TreeNode',
  props: {
    node: { type: Object as () => Node, required: true },
    depth: { type: Number, default: 0 },
    selected: { type: Object as () => Node | null, default: null }
  },
  emits: ['toggle', 'delete', 'rename', 'context-menu'],
  setup(props, { emit }) {
    const onContextMenu = (e: MouseEvent) => {
      e.preventDefault();
      emit('context-menu', { event: e, node: props.node });
    };

    return () =>
        h('li', [
          h('div', {
            class: ['tree-node', { 'active-node': props.selected?.path === props.node.path }],
            style: { '--depth': props.depth.toString() },
            onClick: (e) => { e.stopPropagation(); emit('toggle', props.node); },
            onContextmenu: onContextMenu
          }, [
            h('i', {
              class: props.node.isDir
                  ? (props.node.expanded ? 'fas fa-folder-open' : 'fas fa-folder')
                  : 'fas fa-file-alt'
            }),
            h('span', props.node.name)
          ]),
          props.node.isDir && props.node.expanded && props.node.children.length > 0
              ? h('ul', props.node.children.map(child =>
                  h(TreeNode, {
                    node: child,
                    key: child.path,
                    depth: props.depth + 1,
                    onToggle: (payload) => emit('toggle', payload),
                    onDelete: (payload) => emit('delete', payload),
                    onRename: (payload) => emit('rename', payload),
                    onContextMenu: (payload) => emit('context-menu', payload),
                    selected: props.selected
                  })
              ))
              : null
        ]);
  }
});



export default defineComponent({
  name: 'NoteViews',
  components: { TreeNode },
  data() {
    return {
      treeData: [] as Node[],
      selectedFile: null as Node | null,
      fileContent: '',
      saved: true,
      showEdit: true,
      showPreview: false,
      showFileManager: true,
      isSidebarVisible: true,
      contextMenuVisible: false,
      contextMenuPos: { x: 0, y: 0 },
      contextMenuNode: null as Node | null,

      searchKeyword: '',
      searchMatches: [] as number[],
      currentMatchIndex: 0,
    };
  },
  computed: {
    Fold() {
      return Fold
    },
    Expand() {
      return Expand
    },
    previewContent(): string {
      if (!this.fileContent) {
        return '<p class="empty-preview">输入内容以预览...</p>'
      }
      // 断言为 string
      return (marked.parse(this.fileContent) as string)
    }
  },
  async mounted() {
    document.addEventListener('click', this.closeContextMenu);
    // 绑定 Ctrl+S 保存快捷键
    window.addEventListener('keydown', this.onKeyDown);

    try {
      const dir = await GetNotesDir();
      const list = await GetFiles(dir);

      // 完全复用 openDirectory 的逻辑——包括重置 selectedFile
      this.treeData = [{
        name: dir.split(/[\\/]/).pop() || '',
        path: dir,
        isDir: true,
        expanded: true,
        children: (list || []).map(f => ({
          name: f.name,
          path: f.path,
          isDir: f.isDir,
          expanded: false,
          children: []
        }))
      }];
      // this.selectedFile = this.treeData[0]; // 默认选中根目录
    } catch (e) {
      this.$message.error("无法加载默认笔记目录：" + e.message);
    }
  },
  beforeUnmount() {
    document.removeEventListener('click', this.closeContextMenu);
    // 移除快捷键监听
    window.removeEventListener('keydown', this.onKeyDown);
  },
  methods: {
    onKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
        e.preventDefault();
        if (this.selectedFile && !this.saved) {
          this.saveContent();
        }
      }
    },
    onContextMenu({ event, node }: { event: MouseEvent, node: Node }) {
      this.contextMenuNode = node;
      this.contextMenuPos = { x: event.clientX, y: event.clientY };
      this.contextMenuVisible = true;
    },
    closeContextMenu(e: MouseEvent) {
      if (!(e.target as HTMLElement).closest('.custom-context-menu')) {
        this.contextMenuVisible = false;
      }
    },
    onContextMenuCommand(command: string) {
      this.contextMenuVisible = false;

      const node = this.contextMenuNode;
      if (!node) {
        ElMessage.error('未选择节点');
        return;
      }

      if (command === 'rename') {
        try {
          ElMessageBox.prompt(`重命名 "${node.name}"`, '重命名', {
            inputValue: node.name,
            confirmButtonText: '确定',
            cancelButtonText: '取消',
          }).then(({ value }) => {
            RenameItem(node.path, value).then(() => {
              ElMessage.success('重命名成功');
              this.refreshNode(this.treeData[0]);
            }).catch(err => {
              ElMessage.error('重命名失败：' + err);
            });
          }).catch(() => {});
        } catch (e) {
          console.error('重命名弹窗异常:', e);
        }
      } else if (command === 'delete') {
        try {
          ElMessageBox.confirm(`确认删除 "${node.name}" 吗？`, '删除确认', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
          }).then(() => {
            DeleteItem(node.path).then(() => {
              ElMessage.success('删除成功');
              this.refreshNode(this.treeData[0]);
            }).catch(err => {
              ElMessage.error('删除失败：' + err);
            });
          }).catch(() => {});
        } catch (e) {
          console.error('删除弹窗异常:', e);
        }
      }
    },
    async openDirectory() {
      const dir = await OpenDirectory();
      if (!dir) return;
      const list = await GetFiles(dir);
      this.treeData = [
        {
          name: dir.split(/[\\/]/).pop() || '',
          path: dir,
          isDir: true,
          expanded: true,
          children: (list || []).map(f => ({
            name: f.name,
            path: f.path,
            isDir: f.isDir,
            expanded: false,
            children: []
          }))
        }
      ];
      this.selectedFile = null;
    },
    async createFile() {
      if (!this.selectedFile || !this.selectedFile.isDir) {
        ElMessage.warning('请先选择一个目录');
        return;
      }

      try {
        const { value: name } = await ElMessageBox.prompt('请输入文件名', '新建文件', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputValue: '新备忘录.md',
          inputPattern: /.+/,
          inputErrorMessage: '文件名不能为空'
        });

        await CreateFile(this.selectedFile.path, name);
        if (!this.selectedFile.expanded) this.selectedFile.expanded = true;
        await this.refreshNode(this.selectedFile);
        ElMessage.success('文件创建成功');
      } catch {
        // 用户取消
      }
    },

    async createFolder() {
      if (!this.selectedFile || !this.selectedFile.isDir) {
        ElMessage.warning('请先选择一个目录');
        return;
      }

      try {
        const { value: name } = await ElMessageBox.prompt('请输入文件夹名', '新建文件夹', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputValue: '新建文件夹',
          inputPattern: /.+/,
          inputErrorMessage: '文件夹名不能为空'
        });

        await CreateFolder(this.selectedFile.path, name);
        if (!this.selectedFile.expanded) this.selectedFile.expanded = true;
        await this.refreshNode(this.selectedFile);
        ElMessage.success('文件夹创建成功');
      } catch {
        // 用户取消
      }
    },
    async toggle(node: Node) {
      console.log('toggle:', node.name, node.isDir);

      // 确保所有点击都更新选中状态
      this.selectedFile = node;

      if (!node.isDir) {
        try {
          this.fileContent = await ReadFile(node.path);
        } catch(e) {
          console.error('ReadFile error:', e);
          this.fileContent = '读取文件失败';
        }
        this.saved = true;
        console.log('selectedFile set:', this.selectedFile.name);
        return;
      }

      // 文件夹处理逻辑
      node.expanded = !node.expanded;
      if (node.expanded && node.children.length === 0) {
        const subs = await GetFiles(node.path);
        node.children = (subs || []).map(f => ({
          name: f.name,
          path: f.path,
          isDir: f.isDir,
          expanded: false,
          children: []
        }));
      }
    },
    onContentChange() {
      this.saved = false;
    },
    async saveContent() {
      if (this.selectedFile && !this.saved) {
        await SaveFile(this.selectedFile.path, this.fileContent);
        this.saved = true;
        ElMessage.success('保存成功');
      }
    },
    async onDeleteItem(node: Node) {
      try {
        await DeleteItem(node.path);

        const lastSlashIndex = Math.max(node.path.lastIndexOf('/'), node.path.lastIndexOf('\\'));
        const parentPath = lastSlashIndex >= 0 ? node.path.substring(0, lastSlashIndex) : '';

        if (parentPath) {
          const parentNode = this.findNodeByPath(this.treeData[0], parentPath);
          if (parentNode) {
            await this.refreshNode(parentNode);
          }
        } else {
          await this.openDirectory();
        }

        if (this.selectedFile && this.selectedFile.path === node.path) {
          this.selectedFile = null;
          this.fileContent = '';
          this.saved = true;
        }
        ElMessage.success('删除成功');
      } catch (error: any) {
        ElMessage.error('删除失败：' + (error.message || error));
      }
    },
    findNodeByPath(node: Node, path: string): Node | null {
      if (node.path === path) return node;
      if (!node.children) return null;
      for (const child of node.children) {
        const res = this.findNodeByPath(child, path);
        if (res) return res;
      }
      return null;
    },
    async onRenameItem(payload: { node: Node, newName: string }) {
      try {
        await RenameItem(payload.node.path, payload.newName);
        const lastSlashIndex = Math.max(payload.node.path.lastIndexOf('/'), payload.node.path.lastIndexOf('\\'));
        const parentPath = lastSlashIndex >= 0 ? payload.node.path.substring(0, lastSlashIndex) : '';

        if (parentPath) {
          const parentNode = this.findNodeByPath(this.treeData[0], parentPath);
          if (parentNode) {
            await this.refreshNode(parentNode);
          }
        } else {
          await this.openDirectory();
        }

        if (this.selectedFile && this.selectedFile.path === payload.node.path) {
          this.selectedFile = this.findNodeByPath(this.treeData[0], parentPath + (parentPath ? '/' : '') + payload.newName);
          if (this.selectedFile) {
            this.fileContent = await ReadFile(this.selectedFile.path);
            this.saved = true;
          }
        }
        ElMessage.success('重命名成功');
      } catch (error: any) {
        ElMessage.error('重命名失败：' + (error.message || error));
      }
    },

    toggleEdit() {
      this.showEdit = !this.showEdit;
    },
    togglePreview() {
      this.showPreview = !this.showPreview;
    },
    toggleSidebar() {
      this.isSidebarVisible = !this.isSidebarVisible;
    },

    onPaste(e: ClipboardEvent) {
      const items = e.clipboardData?.items;
      if (!items) return;
      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.kind === 'file') {
          const file = item.getAsFile();
          if (file && file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = () => {
              const base64 = reader.result as string;
              const imageMarkdown = `![image](${base64})\n`;
              const textarea = this.$refs.editor as HTMLTextAreaElement;
              const start = textarea.selectionStart;
              this.fileContent = `${this.fileContent.slice(0, start)}${imageMarkdown}${this.fileContent.slice(start)}`;
            };
            reader.readAsDataURL(file);
          }
        }
      }
    },
    async refreshNode(node: Node) {
      const subs = await GetFiles(node.path);
      node.children = (subs || []).map(f => ({
        name: f.name,
        path: f.path,
        isDir: f.isDir,
        expanded: false,
        children: []
      }));
    },

    onSearch() {
      const keyword = this.searchKeyword.trim();
      this.searchMatches = [];
      this.currentMatchIndex = 0;

      if (!keyword || !this.fileContent) return;

      const indices = [];
      const content = this.fileContent.toLowerCase();
      const lowerKeyword = keyword.toLowerCase();

      let index = content.indexOf(lowerKeyword);
      while (index !== -1) {
        indices.push(index);
        index = content.indexOf(lowerKeyword, index + keyword.length);
      }

      this.searchMatches = indices;
      this.highlightMatches();
    },

    highlightMatches() {
      if (!this.$refs.editor) return;

      const textarea = this.$refs.editor as HTMLTextAreaElement;
      const startIndex = this.searchMatches[this.currentMatchIndex];
      if (startIndex != null) {
        textarea.focus();
        textarea.setSelectionRange(startIndex, startIndex + this.searchKeyword.length);

        // ✅ 滚动到光标位置
        this.scrollToCursor(textarea);
      }
    },
    scrollToCursor(textarea: HTMLTextAreaElement) {
      // 当前光标所在的行
      const cursorPosition = textarea.selectionStart;

      // 将光标前面的内容放入临时容器计算高度
      const beforeText = textarea.value.substring(0, cursorPosition);
      const lines = beforeText.split('\n');
      const lineHeight = parseFloat(getComputedStyle(textarea).lineHeight || '20');

      const targetScrollTop = Math.max(0, (lines.length - 1) * lineHeight - textarea.clientHeight / 2);
      textarea.scrollTop = targetScrollTop;
    },

    nextMatch() {
      if (this.searchMatches.length === 0) return;
      this.currentMatchIndex = (this.currentMatchIndex + 1) % this.searchMatches.length;
      this.highlightMatches();
    },

    prevMatch() {
      if (this.searchMatches.length === 0) return;
      this.currentMatchIndex = (this.currentMatchIndex - 1 + this.searchMatches.length) % this.searchMatches.length;
      this.highlightMatches();
    }

  }
});
</script>

<style lang="scss" >
$primary: #ccd1ef;
$primary-dark: #e3e6ef;
$secondary: #7209b7;
$dark: #1e1e2e;
$darker: #ddddf1;
$light: #f8f9fa;
$gray: #6c757d;
$gray-light: #e9ecef;

$border-radius: 8px;
$transition: all 0.3s ease;
$shadow: 0 10px 20px rgba(0, 0, 0, 0.2);
$shadow-sm: 0 4px 8px rgba(0, 0, 0, 0.1);
$glass: rgba(30, 30, 46, 0.7);

.toggle-sidebar-btn {
  position: absolute;
  top: 17px;
  left: 334px;
  z-index: 999;
  background: rgba(255, 255, 255, 0.5);
  border: none;
  backdrop-filter: blur(6px);
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.2);
  transition: all 0.3s ease;

  &:hover {
    background: rgba(255, 255, 255, 0.3);
  }
}

.search-input {
  padding: 6px 8px;
  font-size: 13px;
  border-radius: 6px;
  border: 1px solid #ccc;
  outline: none;
  width: 180px;
}


.note-views {
  display: flex;
  height: 100%;
  padding: 0;
  gap: 10px;
  background: linear-gradient(186deg, #DDDDDB, #7f7f88);
  color: $light;

  * {
    box-sizing: border-box;
  }

  .file-manager {
    width: 250px;
    min-width: 250px;
    background: $glass;
    backdrop-filter: blur(10px);
    border-radius: $border-radius;
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: $shadow;
    display: flex;
    flex-direction: column;
    padding: 15px;

    .toolbar {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin-bottom: 15px;

      .btn {
        background: $primary;
        color: white;
        border: none;
        padding: 8px 12px;
        border-radius: 8px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 5px;
        font-weight: 500;
        font-size: 13px;
        transition: $transition;
        box-shadow: $shadow-sm;

        &:hover {
          background: $primary-dark;
          transform: translateY(-2px);
        }

        &.btn-secondary {
          background: rgba(255, 255, 255, 0.1);

          &:hover {
            background: rgba(255, 255, 255, 0.2);
          }
        }

        i {
          font-size: 13px;
        }
      }
    }

    .tree-container {
      flex: 1;
      overflow-y: auto;

      .tree-list {
        list-style: none;
        padding-left: 0;
        margin: 0;

        .tree-node {
          display: flex;
          align-items: center;
          gap: 6px;
          cursor: pointer;
          padding-left: 10px;
          transition: background 0.2s;
          user-select: none;
          white-space: nowrap;

          &:hover {
            background-color: rgba(76, 175, 80, 0.2);
            border-radius: $border-radius;
          }

          i {
            color: #4a90e2;
          }
        }
      }
    }
  }
  .tree-node.active-node {
    background-color: rgba(72, 149, 239, 0.1);
    color: #fff;
    border-radius: 6px;
    font-weight: bold;
  }

  .editor-container {
    flex: 1; /* 让它撑满剩余空间 */
    display: flex;
    flex-direction: column;
    /* 保留你原有的样式 */
    padding: 20px;
    background: rgba(0, 0, 0, 0.5);
    color: $light;
    font-family: 'Segoe UI', monospace;
    font-size: 15px;
    overflow: hidden; /* 隐藏滚动条外 */

    .editor-header {
      padding: 15px 20px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.1);
      display: flex;
      justify-content: space-between;
      align-items: center;

      .editor-title {
        font-size: 16px;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 10px;

        .save-status {
          font-size: 13px;
          display: flex;
          align-items: center;
          gap: 6px;

          .indicator {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: #e63946;

            &.saved {
              background: #2a9d8f;
            }
          }
        }
      }

      .header-actions {
        display: flex;
        gap: 8px;

        .btn {
          background: $primary;
          color: white;
          border: none;
          padding: 8px 12px;
          border-radius: 8px;
          cursor: pointer;
          display: flex;
          align-items: center;
          gap: 5px;
          font-weight: 500;
          font-size: 13px;
          transition: $transition;
          box-shadow: $shadow-sm;

          &:hover {
            background: $primary-dark;
            transform: translateY(-2px);
          }

          &.btn-secondary {
            background: rgba(255, 255, 255, 0.1);

            &:hover {
              background: rgba(255, 255, 255, 0.2);
            }
          }
        }
      }
    }

    .editor-main-horizontal {
      display: flex;
      flex: 1;
      overflow: hidden;

      .editor-content-wrapper {
        display: flex;
        flex: 1;
        height: 100%;
        overflow: hidden;

        .editor-textarea {
          flex: 1;
          padding: 20px;
          background: rgba(0, 0, 0, 0.2);
          color: $light;
          font-family: 'Segoe UI', monospace;
          font-size: 15px;
          border: none;
          resize: none;
          line-height: 1.6;
          overflow-y: auto;

          &:focus {
            outline: none;
          }
        }

        .preview-wrapper {
          padding: 15px;
          border-left: 1px solid #ccc;
          overflow-y: auto;
          background: rgba(0, 0, 0, 1.2);
          min-width: 0;

          .preview-content {
            white-space: normal;
            line-height: 1.5;

            p, h1, h2, h3, ul, ol {
              margin: 0.6em 0;
            }

            a,
            a:visited,
            a:hover,
            a:active {
              color: inherit !important;
              text-decoration: none !important;
              background: none !important;
              box-shadow: none !important;
            }

            a:hover {
              text-decoration: underline !important;
              color: #4a90e2 !important;
            }

            code {
              background: rgba(255, 255, 255, 0.1);
              padding: 2px 6px;
              border-radius: 4px;
              font-family: monospace;
              font-size: 14px;
            }

            pre {
              background: rgba(0, 0, 0, 0.2);
              padding: 15px;
              border-radius: 8px;
              overflow-x: auto;
              margin: 15px 0;

              code {
                background: none;
                padding: 0;
              }
            }

            blockquote {
              border-left: 4px solid $primary;
              padding-left: 15px;
              margin: 15px 0;
              color: $gray-light;
            }

            a {
              color: inherit;
              text-decoration: none;
            }
          }
        }
      }

      .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: $gray;
        width: 780px;

        i {
          font-size: 48px;
          margin-bottom: 15px;
          color: $primary;
        }

        h3 {
          font-size: 20px;
          margin-bottom: 12px;
          color: $light;
        }

        p {
          font-size: 14px;
          max-width: 300px;
        }
      }
    }
  }

  .custom-context-menu {
    position: fixed;
    z-index: 9999;
    background: #fff;
    border: 1px solid #ccc;
    border-radius: 6px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    min-width: 120px;

    .context-menu-list {
      list-style: none;
      margin: 0;
      padding: 6px 0;

      li {
        padding: 8px 16px;
        font-size: 14px;
        color: #333;
        cursor: pointer;

        &:hover {
          background-color: #f5f5f5;
        }
      }
    }
  }

  @media (max-width: 900px) {
    flex-direction: column;

    .file-manager {
      width: 100%;
      max-height: 200px;
    }
  }
}
</style>
