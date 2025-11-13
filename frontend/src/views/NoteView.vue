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
          打开
        </button>
        <button class="btn btn-secondary" @click="createFile">
          文件
        </button>
        <button class="btn btn-secondary" @click="createFolder">
          文件夹
        </button>
        <!--        <button class="btn btn-secondary" @click="refreshCurrentDirectory">-->
        <!--          刷新-->
        <!--        </button>-->
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
            <div class="preview-content" v-html="previewContent" ref="preview"></div>
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
        <li @click="onContextMenuCommand('open')">打开文件目录</li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent, h} from 'vue';
import {ElMessage, ElMessageBox} from 'element-plus';
import { EventsOn } from "../../wailsjs/runtime";
import {marked} from 'marked';
import {
  CreateFile,
  CreateFolder,
  DeleteItem,
  GetFiles,
  GetNotesDir,
  OpenDirectory,
  ReadFile,
  RenameItem,
  SaveFile,
  SaveImage
} from "../../wailsjs/go/controller/Note";
import {Expand, Fold} from "@element-plus/icons-vue";
import {OpenPath} from "../../wailsjs/go/system/System";

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
                    selected: props.selected,  // ✅ 添加这一行
                    onToggle: (payload) => emit('toggle', payload),
                    onDelete: (payload) => emit('delete', payload),
                    onRename: (payload) => emit('rename', payload),
                    onContextMenu: (payload) => emit('context-menu', payload),
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
        return '<p class="empty-preview">输入内容以预览...</p>';
      }
      let content = marked.parse(this.fileContent) as string;
      // 处理图片路径，确保相对路径能正确显示
      content = this.processImagePaths(content);
      return content;
    },
  },
  async mounted() {
    document.addEventListener('click', this.closeContextMenu);
    // 绑定 Ctrl+S 保存快捷键
    window.addEventListener('keydown', this.onKeyDown);

    // 监听文件变化事件，添加更严格的防抖
    EventsOn('fileChange', (eventData: string) => {
      try {
        const event = JSON.parse(eventData);
        console.log('文件变化:', event);

        // 只处理重要的文件变化，忽略频繁的修改事件
        if (event.Type === 'modify') {
          return;
        }

        // 延迟刷新，避免频繁更新
        setTimeout(() => {
          this.refreshCurrentDirectory();
        }, 1000);
      } catch (e) {
        console.error('解析文件变化事件失败:', e);
      }
    });

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
      } else if (command === 'open'){
        try {
          let pathToOpen = node.path;

          // 如果是文件，获取其所在文件夹路径
          if (!node.isDir) {
            pathToOpen = this.getParentDirectory(node.path);
          }

          console.log(`将打开路径：${pathToOpen}`);
          OpenPath(pathToOpen);
        } catch (e){
          console.error('打开文件所在位置异常:', e);
        }
      }
    },
    // 获取父目录的辅助方法
    getParentDirectory(filePath: string): string {
      // 使用字符串处理方法获取父目录
      const pathSeparator = filePath.includes('/') ? '/' : '\\';
      const pathParts = filePath.split(pathSeparator);

      // 移除最后一个部分（文件名）
      if (pathParts.length > 1) {
        pathParts.pop();
        return pathParts.join(pathSeparator);
      }

      // 如果已经是根目录，返回原路径
      return filePath;
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
    // 添加手动刷新方法
    async refreshCurrentDirectory() {
      if (this.treeData.length === 0) return;

      const rootNode = this.treeData[0];
      await this.refreshNode(rootNode);
      // this.$message.success('目录已刷新');
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

    // 修改 onPaste 方法，确保使用统一路径格式
    async onPaste(e: ClipboardEvent) {
      const items = e.clipboardData?.items;
      if (!items) return;

      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.kind === 'file') {
          const file = item.getAsFile();
          if (file && file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = async () => {
              const base64 = reader.result as string;

              if (this.selectedFile) {
                try {
                  // 调用后端保存图片方法，返回HTTP URL用于预览
                  const imageUrl = await SaveImage(this.selectedFile.path, base64);
                  const imageMarkdown = `![image](${imageUrl})\n`;

                  const textarea = this.$refs.editor as HTMLTextAreaElement;
                  const start = textarea.selectionStart;
                  this.fileContent = `${this.fileContent.slice(0, start)}${imageMarkdown}${this.fileContent.slice(start)}`;
                  this.saved = false;
                } catch (error) {
                  console.error('保存图片失败:', error);
                  // 降级处理
                  const imageMarkdown = `![image](${base64})\n`;
                  const textarea = this.$refs.editor as HTMLTextAreaElement;
                  const start = textarea.selectionStart;
                  this.fileContent = `${this.fileContent.slice(0, start)}${imageMarkdown}${this.fileContent.slice(start)}`;
                  this.saved = false;
                }
              }
            };
            reader.readAsDataURL(file);
            break;
          }
        }
      }
    },

    // 处理图片路径，确保能正确显示
    processImagePaths(html: string): string {
      // 创建一个临时div来解析HTML
      const tempDiv = document.createElement('div');
      tempDiv.innerHTML = html;

      // 处理所有图片标签
      const images = tempDiv.querySelectorAll('img');
      images.forEach(img => {
        const src = img.getAttribute('src');
        if (src && !src.startsWith('http') && !src.startsWith('data:')) {
          // 对于相对路径图片，我们保持原样
          // 在Electron/Wails环境中，相对路径应该能正常工作
          console.log('处理图片路径:', src);
        }
      });

      return tempDiv.innerHTML;
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
      // this.highlightMatches();
    },

    highlightMatches() {
      // 编辑区高亮
      if (this.searchMatches.length > 0 && this.$refs.editor) {
        const textarea = this.$refs.editor as HTMLTextAreaElement;
        const startIndex = this.searchMatches[this.currentMatchIndex];

        textarea.focus();
        textarea.setSelectionRange(startIndex, startIndex + this.searchKeyword.length);

        // 滚动编辑区到匹配位置
        this.scrollTextareaToSelection(textarea);
      }

      // 预览区高亮
      this.$nextTick(() => {
        const previewEl = this.$refs.preview as HTMLElement;
        if (!previewEl) return;

        // 清除旧高亮
        previewEl.querySelectorAll('.mark-highlight').forEach(el => {
          const parent = el.parentNode;
          if (parent) parent.replaceChild(document.createTextNode(el.textContent || ''), el);
        });

        if (!this.searchKeyword) return;

        const walker = document.createTreeWalker(previewEl, NodeFilter.SHOW_TEXT);
        const keyword = this.searchKeyword.toLowerCase();
        let matchCount = 0;
        let currentHighlightElement: HTMLElement | null = null;

        let node: Text | null;
        while ((node = walker.nextNode() as Text | null)) {
          const text = node.nodeValue;
          if (!text) continue;

          let index = text.toLowerCase().indexOf(keyword);
          while (index !== -1) {
            if (matchCount === this.currentMatchIndex) {
              const range = document.createRange();
              range.setStart(node, index);
              range.setEnd(node, index + keyword.length);

              const span = document.createElement('span');
              span.className = 'mark-highlight current-highlight';
              range.surroundContents(span);
              currentHighlightElement = span;

              // 滚动预览区到匹配位置
              if (currentHighlightElement) {
                currentHighlightElement.scrollIntoView({
                  behavior: 'smooth',
                  block: 'center',
                });
              }

              // 节点已被修改，需要重新获取当前节点
              node = span.nextSibling as Text;
              break;
            }

            matchCount++;
            index = text.toLowerCase().indexOf(keyword, index + keyword.length);
          }

          if (currentHighlightElement) break;
        }
      });
    },

    scrollTextareaToSelection(textarea: HTMLTextAreaElement) {
      // 计算光标所在位置
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;

      // 计算行号
      const content = textarea.value;
      const lines = content.substring(0, start).split('\n');
      const lineNumber = lines.length;

      // 计算滚动位置
      const lineHeight = parseInt(getComputedStyle(textarea).lineHeight) || 20;
      const scrollTop = (lineNumber - 1) * lineHeight - textarea.clientHeight / 2;

      // 平滑滚动
      textarea.scrollTo({
        top: Math.max(0, scrollTop),
        behavior: 'smooth'
      });
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

<style lang="scss">
$primary: #4a6cf7;
$primary-light: #dcdce1;
$secondary: #6c757d;
$success: #10b981;
$warning: #f59e0b;
$danger: #ef4444;
$light: rgba(228,228,228,0.25);
$light-gray: #f8f9fa;
$medium-gray: #435568;
$dark-gray: #6c757d;
$text-dark: #212529;

$border-radius: 8px;
$transition: all 0.3s ease;
$shadow: 0 5px 15px rgba(0, 0, 0, 0.05);
$shadow-hover: 0 8px 25px rgba(0, 0, 0, 0.1);

.toggle-sidebar-btn {
  position: absolute;
  top: 15px;
  left: 205px;
  z-index: 999;
  background: $light;
  border: 1px solid $medium-gray;
  box-shadow: $shadow;
  transition: $transition;
  color: $primary;

  &:hover {
    background: $primary-light;
    box-shadow: $shadow-hover;
  }
}

.search-input {
  padding: 8px 12px;
  font-size: 14px;
  border-radius: $border-radius;
  border: 1px solid $medium-gray;
  outline: none;
  width: 180px;
  transition: $transition;

  &:focus {
    border-color: $primary;
    box-shadow: 0 0 0 3px rgba($primary, 0.1);
  }
}

.note-views {
  display: flex;
  height: 100%;
  padding: 0;
  gap: 10px;
  background: $light-gray;
  color: $text-dark;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;

  * {
    box-sizing: border-box;
  }

  .file-manager {
    width: 250px;
    min-width: 250px;
    background: $light;
    border-radius: $border-radius;
    border: 1px solid $medium-gray;
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
        background: $light;
        color: $primary;
        border: 1px solid $primary;
        padding: 8px 12px;
        border-radius: $border-radius;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 3px;
        font-weight: 500;
        font-size: 13px;
        transition: $transition;
        box-shadow: $shadow;

        &:hover {
          background: $primary;
          color: $light;
          box-shadow: $shadow-hover;
        }

        &.btn-secondary {
          background: $light;
          color: $secondary;
          border: 1px solid $medium-gray;

          &:hover {
            background: $light-gray;
            color: $text-dark;
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
          padding: 6px 10px;
          transition: background 0.2s;
          user-select: none;
          white-space: nowrap;
          border-radius: 4px;
          margin-bottom: 2px;

          &:hover {
            background-color: $primary-light;
          }

          i {
            color: $dark-gray;
          }
        }
      }
    }
  }

  .tree-node.active-node {
    background-color: $primary-light;
    color: $primary;
    font-weight: 600;

    i {
      color: $primary;
    }
  }

  .editor-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 2px;
    background: $light;
    color: $text-dark;
    border-radius: $border-radius;
    border: 1px solid $medium-gray;
    box-shadow: $shadow;
    overflow: hidden;

    .editor-header {
      padding: 15px 20px;
      border-bottom: 1px solid $medium-gray;
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: $light;
      border-radius: $border-radius $border-radius 0 0;

      .editor-title {
        font-size: 16px;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 10px;
        min-width: 0; /* 允许内容被截断 */
        flex: 1; /* 占据可用空间 */

        /* 文件名部分 - 添加文本截断 */
        &:first-child {
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          max-width: 30%; /* 限制最大宽度 */
        }

        .save-status {
          font-size: 13px;
          display: flex;
          align-items: center;
          gap: 6px;
          flex-shrink: 0; /* 防止保存状态被挤压 */

          .indicator {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: $danger;

            &.saved {
              background: $success;
            }
          }
        }
      }

      .header-actions {
        display: flex;
        gap: 8px;

        .btn {
          background: $light;
          color: $primary;
          border: 1px solid $primary;
          padding: 8px 12px;
          border-radius: $border-radius;
          cursor: pointer;
          display: flex;
          align-items: center;
          gap: 5px;
          font-weight: 500;
          font-size: 13px;
          transition: $transition;
          box-shadow: $shadow;

          &:hover {
            background: $primary;
            color: $light;
            box-shadow: $shadow-hover;
          }

          &.btn-secondary {
            background: $light;
            color: $secondary;
            border: 1px solid $medium-gray;

            &:hover {
              background: $light-gray;
              color: $text-dark;
            }
          }
        }
      }
    }

    .editor-main-horizontal {
      display: flex;
      flex: 1;
      overflow: hidden;
      background: $light;

      .editor-content-wrapper {
        display: flex;
        flex: 1;
        height: 100%;
        overflow: hidden;

        .editor-textarea {
          flex: 1;
          padding: 20px;
          background: $light;
          color: $text-dark;
          font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
          font-size: 15px;
          border: none;
          resize: none;
          line-height: 1.6;
          overflow-y: auto;
          border-right: 1px solid $medium-gray;

          &:focus {
            outline: none;
            box-shadow: inset 0 0 0 1px $primary-light;
          }
        }

        .preview-wrapper {
          padding: 20px;
          overflow-y: auto;
          background: $light;
          color: $text-dark;
          min-width: 0;

          .preview-content {
            white-space: normal;
            line-height: 1.6;
            font-size: 15px;

            p, h1, h2, h3, ul, ol {
              margin: 0.8em 0;
            }

            h1, h2, h3 {
              color: $text-dark;
              font-weight: 600;
            }

            a {
              pointer-events: none !important;
              cursor: default !important;
              color: $dark-gray !important;
              text-decoration: none !important;
            }

            img {
              max-width: 80%;
              height: auto;
              border-radius: 4px;
              box-shadow: $shadow;
              display: block;
              margin: 15px 0;
            }

            code {
              background: $light-gray;
              padding: 2px 6px;
              border-radius: 4px;
              font-family: monospace;
              font-size: 14px;
              color: $danger;
            }

            pre {
              background: $light-gray;
              padding: 15px;
              border-radius: $border-radius;
              overflow-x: auto;
              margin: 15px 0;
              border: 1px solid $medium-gray;

              code {
                background: none;
                padding: 0;
                color: inherit;
              }
            }

            blockquote {
              border-left: 4px solid $primary;
              padding-left: 15px;
              margin: 15px 0;
              color: $dark-gray;
              font-style: italic;
            }

            table {
              width: 100%;
              border-collapse: collapse;
              margin: 15px 0;

              th, td {
                padding: 8px 12px;
                border: 1px solid $medium-gray;
              }

              th {
                background: $light-gray;
              }
            }

            .mark-highlight {
              background-color: #fff59d;
              color: #000;
              font-weight: bold;
              padding: 0 2px;
              border-radius: 2px;
            }
          }
        }
      }
      .preview-wrapper {
        img {
          max-width: 100%;
          height: auto;
        }
      }

      .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: $dark-gray;
        width: 100%;

        i {
          font-size: 48px;
          margin-bottom: 15px;
          color: $primary-light;
        }

        h3 {
          font-size: 20px;
          margin-bottom: 12px;
          color: $text-dark;
        }

        p {
          font-size: 14px;
          max-width: 300px;
          text-align: center;
        }
      }
    }
  }

  .custom-context-menu {
    position: fixed;
    z-index: 9999;
    background: $light;
    border: 1px solid $medium-gray;
    border-radius: $border-radius;
    box-shadow: $shadow-hover;
    min-width: 120px;
    overflow: hidden;

    .context-menu-list {
      list-style: none;
      margin: 0;
      padding: 6px 0;
      background-color: rgba(214, 227, 241, 0.79);

      li {
        padding: 8px 16px;
        font-size: 14px;
        color: $text-dark;
        cursor: pointer;
        transition: background 0.2s;


        &:hover {
          background-color: #b8cedf;
          color: $primary;
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