<template>
  <div class="system">
    <!-- 1. 右上角目录树 -->
    <div ref="treeWrapper" class="cate-tree">
      <el-tree
          ref="tree"
          :data="treeData"
          :props="{ children: 'children', label: 'label' }"
          node-key="id"
          @node-click="onTreeNodeClick"
          :highlight-current="true"
          :default-expand-all="false"
          :expand-on-click-node="true"
          class="small-tree"
      />
    </div>

    <!-- 搜索框：图标/展开切换，支持拖拽 -->
    <div
        ref="searchBar"
        class="search-bar"
        :class="{ expanded: searchExpanded }"
        :style="searchBarStyle"
        @mousedown.stop="startDrag"
        @click.stop="!searchExpanded && openSearch()"
    >
      <!-- 收起：仅图标 -->
      <el-icon v-if="!searchExpanded" class="search-icon-collapsed">
        <Search />
      </el-icon>

      <!-- 展开：输入框 -->
      <transition name="fade">
        <el-input
            v-if="searchExpanded"
            ref="searchInput"
            v-model="searchQuery"
            placeholder="请输入关键字"
            clearable
            @clear="collapseSearch"
            @blur="collapseSearch"
            @input="handleSearchInput"
            class="custom-search-input-expanded"
            @mousedown.stop
        >
          <template #prefix>
            <el-icon class="search-icon-expanded"><Search /></el-icon>
          </template>
        </el-input>
      </transition>
    </div>

    <!-- 工具列表 -->
    <div class="nav" v-for="(category, catIndex) in processedToolList" :key="catIndex"
         draggable="true"
         @dragstart.stop="handleDragStart($event, 'category', catIndex)"
         @dragover.prevent="handleDragOver($event, 'category', catIndex)"
         @drop="handleDrop($event, 'category', catIndex)">
      <div class="cate" :data-index="catIndex">{{ category.title }}</div>
      <div class="site-list">
        <div
            class="item"
            v-for="(cmd, cmdIndex) in category.list"
            :key="cmdIndex"
            draggable="true"
            @dragstart.stop="handleDragStart($event, 'command', catIndex, cmdIndex)"
            @dragover.prevent.stop="handleDragOver($event, 'command', catIndex, cmdIndex)"
            @drop.stop="handleDrop($event, 'command', catIndex, cmdIndex)"
            @click="runCmd(cmd.cmd, cmd.param, cmd.terminal)"
            @contextmenu.prevent="showContextMenu($event, cmd, index)"
        >
          <div class="image">
            <!-- 使用解析后的图标 -->
            <el-image
                style="width: 36px; height: 36px"
                :src="cmd.resolvedIcon"
                fit="cover"
                :alt="cmd.name"
                @error="handleImageError(cmd)"
            >
              <template #error>
                <div class="image-error">
                  <span>图标加载失败</span>
                </div>
              </template>
            </el-image>
          </div>
          <div class="desc" :title="cmd.desc">
            <span class="title">{{ cmd.name }}</span>
            <span class="remark">{{ cmd.desc }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 右键菜单 -->
    <ul class="context-menu" v-if="contextMenuVisible"
      :style="{ top: `${contextMenuPosition.y}px`, left: `${contextMenuPosition.x}px` }">
      <li v-for="(option, index) in contextMenuOptions" :key="index" @click="option.action">
        <i :class="option.icon"></i> {{ option.label }}
      </li>
    </ul>

    <!-- 弹出框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form :model="form" label-width="80px">
        <!-- 分类编辑仅显示分类名 -->
        <template v-if="dialogMode === 'cateEdit'">
          <el-form-item label="分类" prop="category">
            <el-input v-model="form.category" placeholder="请输入新分类名称"></el-input>
          </el-form-item>
        </template>

        <!-- 工具新增/编辑表单 -->
        <template v-else>
          <el-form-item label="分类" prop="category">
            <el-autocomplete
                v-model="form.category"
                :fetch-suggestions="queryCategories"
                placeholder="请输入或选择分类"
                @select="handleCategorySelect"
            ></el-autocomplete>
          </el-form-item>
          <el-form-item label="命令" prop="cmd">
            <el-input v-model="form.cmd" placeholder="请输入工具命令"></el-input>
          </el-form-item>
          <el-form-item label="参数" prop="param">
            <el-input v-model="form.param" placeholder="请输入工具参数"></el-input>
          </el-form-item>
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="请输入工具名称"></el-input>
          </el-form-item>
          <el-form-item label="路径" prop="path">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input
                  v-model="form.path"
                  placeholder="请输入工具路径"
                  style="flex: 1;"
              ></el-input>
              <el-button
                  type="primary"
                  @click="browseForFolder"
                  style="height: 30px; padding: 0 10px;"
              >浏览</el-button>
            </div>
          </el-form-item>
          <el-form-item label="描述" prop="desc">
            <el-input v-model="form.desc" placeholder="请输入工具描述"></el-input>
          </el-form-item>
          <el-form-item label="图标" prop="icon">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input
                  v-model="form.icon"
                  placeholder="请输入图标地址"
                  style="flex: 1;"
              ></el-input>
              <el-button
                  type="primary"
                  @click="browseForIconFile"
                  style="height: 30px; padding: 0 10px;"
              >浏览</el-button>
            </div>
          </el-form-item>
          <el-form-item label="终端" prop="terminal">
            <el-switch
                v-model="form.terminal"
                :active-value="1"
                :inactive-value="0"
                active-text="是"
                inactive-text="否"
            ></el-switch>
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {GetOpenDir, GetOpenFilePath, OpenPath, ShellCMD} from "../../wailsjs/go/controller/System";
import {
  AddTool,
  DeleteTool,
  DeleteToolCategory,
  GetAllTools,
  GetCategoryList,
  GetSearchTools,
  MoveCommandToCategory,
  ReadImageAsBase64,
  UpdateCategorySorts,
  UpdateCommandSorts,
  UpdateTool,
  UpdateToolCategory,
} from "../../wailsjs/go/controller/Tool";
import {Search} from "@element-plus/icons-vue";
import {ElNotification, ElTree} from "element-plus";

export default {
  name: "ToolsView",
  components: { ElTree,Search },
  data() {
    return {
      toolList: [], // 初始化命令列表
      resolvedIcons: {}, // 图标缓存 {id: resolvedPath}

      // 搜索框状态
      searchExpanded: false,
      // 拖拽相关
      dragging: false,
      dragOffset: { x: 0, y: 0 },
      // 当前位置
      searchPosition: {
        x: window.innerWidth - 80,
        y: window.innerHeight - 80,
      },
      searchQuery: "",

      defaultExpandedKeys: [],
      contextMenuVisible: false,
      contextMenuOptions: [], // 动态右键菜单选项
      contextMenuPosition: { x: 0, y: 0 },
      selectedCmd: null,
      selectedIndex: null,
      dialogVisible: false,
      dialogMode: "add", // 'add' or 'edit'
      form: {
        id: null,
        category: "",
        cmd: "",
        param: "",
        name: "",
        path: "",
        desc: "",
        icon: "",
        terminal: 1,
      },
      categories: [], // 初始化为空数组
      selectedCategory: "",
      dragData: {
        type: null,    // 'category' 或 'command'
        catIndex: -1,
        cmdIndex: -1
      },
    };
  },
  computed: {
    dialogTitle() {
      switch (this.dialogMode) {
        case "add": return "新增工具";
        case "toolEdit": return "修改工具";
        case "cateEdit": return "修改分类";
        default: return "编辑";
      }
    },
    searchBarStyle() {
      return {
        position: "fixed",
        left: this.searchPosition.x + "px",
        top: this.searchPosition.y + "px",
        zIndex: 999,
        cursor: this.dragging ? "grabbing" : "move",
        transition: this.dragging ? "none" : "all 0.2s",
      };
    },
    treeData() {
      // 为了实现折叠，引入一个根节点“目录”
      return [
        {
          label: '目录',
          id: 'root',
          children: this.toolList.map((cat, idx) => ({ label: cat.title, id: idx }))
        }
      ];
    },
    // 处理后的工具列表，包含解析后的图标
    processedToolList() {
      return this.toolList.map(category => {
        return {
          ...category,
          list: category.list.map(cmd => {
            return {
              ...cmd,
              resolvedIcon: this.resolvedIcons[cmd.id] || "/assets/tool/default.png"
            }
          })
        }
      });
    },
  },
  methods: {
    // 打开并自动调整到边缘
    openSearch() {
      this.searchExpanded = true;
      this.$nextTick(() => {
        const rect = this.$refs.searchBar.getBoundingClientRect();
        let x = this.searchPosition.x;
        let y = this.searchPosition.y;
        const overX = rect.right - window.innerWidth;
        const overY = rect.bottom - window.innerHeight;
        if (overX > 0) x = Math.max(10, x - overX - 10);
        if (overY > 0) y = Math.max(10, y - overY - 10);
        this.searchPosition = { x, y };
        this.$refs.searchInput.focus();
      });
    },
    // 收起并靠边
    collapseSearch() {
      this.searchExpanded = false;
      // 收起后让图标靠最近的屏幕边缘
      this.$nextTick(() => {
        const { x, y } = this.searchPosition;
        const midX = window.innerWidth / 2;
        const newX = x < midX ? 10 : window.innerWidth - 50;
        // y 保持不变或根据需要靠上/下，这里保持原来 y
        this.searchPosition.x = newX;
      });
    },
    // 拖拽开始
    startDrag(e) {
      this.dragging = true;
      const { left, top } = this.$refs.searchBar.getBoundingClientRect();
      this.dragOffset.x = e.clientX - left;
      this.dragOffset.y = e.clientY - top;
      document.addEventListener("mousemove", this.onDrag);
      document.addEventListener("mouseup", this.endDrag);
    },
    onDrag(e) {
      if (!this.dragging) return;
      let x = e.clientX - this.dragOffset.x;
      let y = e.clientY - this.dragOffset.y;
      // 限制边界
      x = Math.min(Math.max(0, x), window.innerWidth - this.$refs.searchBar.offsetWidth);
      y = Math.min(Math.max(0, y), window.innerHeight - this.$refs.searchBar.offsetHeight);
      this.searchPosition = { x, y };
    },
    endDrag() {
      this.dragging = false;
      document.removeEventListener("mousemove", this.onDrag);
      document.removeEventListener("mouseup", this.endDrag);
    },
    handleSearchInput(val) {
      this.searchQuery = val;
      this.handleSearch(); // 调用原来的搜索方法
    },

    onTreeNodeClick(node) {
      if (node.id === 'root') return;
      const target = document.querySelector(`.cate[data-index=\"${node.id}\"]`);
      if (target) {
        target.closest('.nav').scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    },


    // 加载命令列表
    async loadToolList() {
      try {
        this.toolList = await GetAllTools();
        await this.loadAllIcons();
      } catch (error) {
        console.error("加载命令列表失败:", error);
      }
    },

    // 搜索框
    async handleSearch() {
      try {
        if (!this.searchQuery.trim()) {
          this.toolList = await GetAllTools();
        } else {
          this.toolList = await GetSearchTools(this.searchQuery.trim());
        }
      } catch (error) {
        this.toolList = []
        ElNotification({
          title: '提示',
          message: '没有符合条件的查询数据',
          type: 'info',
          position: 'top-right',  // 默认就是右上角
          duration: 1000          // 可选：自动关闭时间，单位 ms
        });
        console.error("搜索失败:", error);
      }
    },

    // 运行命令
    runCmd(cmd, param, terminal) {
      ShellCMD(cmd, param, terminal);
    },

    // 异步加载所有图标
    async loadAllIcons() {
      const iconPromises = [];

      // 创建一个新的图标缓存对象
      const newResolvedIcons = {...this.resolvedIcons};

      // 收集所有需要解析的图标
      this.toolList.forEach(category => {
        category.list.forEach(cmd => {
          if (cmd.icon && !this.resolvedIcons[cmd.id]) {
            iconPromises.push(
                this.resolveIconPath(cmd.icon).then(icon => {
                  // 直接更新缓存对象
                  newResolvedIcons[cmd.id] = icon;
                })
            );
          }
        });
      });

      await Promise.all(iconPromises);

      // 整体更新图标缓存
      this.resolvedIcons = newResolvedIcons;
    },

    // 解析图标路径（保持异步）
    async resolveIconPath(icon) {
      // console.log("开始解析图标:", icon);

      if (!icon) {
        // console.log("图标为空，返回默认图标");
        return "/assets/tool/default.png";
      }

      // 本地资源或完整 URL
      if (icon.startsWith("/assets") || /^https?:\/\//.test(icon)) {
        // console.log("直接返回本地/网络图标:", icon);
        return icon;
      }

      // 绝对路径处理
      const isWinAbsPath = /^[A-Za-z]:[\\/]/.test(icon);
      const isUnixAbsPath = icon.startsWith("/");

      if (isWinAbsPath || isUnixAbsPath) {
        const normalizedIcon = icon.replace(/\\/g, '/');
        // console.log("处理绝对路径:", normalizedIcon);

        try {
          const base64 = await ReadImageAsBase64(normalizedIcon);
          // console.log("获取到base64数据:", base64 ? `长度: ${base64.length}` : "空");

          if (base64) {
            const extension = normalizedIcon.split('.').pop()?.toLowerCase() || 'png';
            const mimeType = this.getMimeType(extension);
            // console.log("生成Data URL:", dataUrl.substring(0, 50) + "...");
            return `data:${mimeType};base64,${base64}`;
          }
        } catch (err) {
          console.error("图标加载失败:", err);
        }
        return "/assets/tool/default.png";
      }

      // 其他情况
      // console.log("返回远程路径:", remoteUrl);
      return `http://127.0.0.1:52867/icon/${icon}`;
    },

    getMimeType(ext) {
      const types = {
        png: 'image/png',
        jpg: 'image/jpeg',
        jpeg: 'image/jpeg',
        gif: 'image/gif',
        svg: 'image/svg+xml',
        webp: 'image/webp'
      };
      return types[ext] || 'image/png';
    },

    // 图像加载失败时的处理
    handleImageError(cmd) {
      console.error(`图标加载失败: ${cmd.icon}`);
      const cmdId = cmd.id || cmd.name;
      // 更新为默认图标
      this.$set(this.resolvedIcons, cmd.id, "/assets/tool/default.png");
      this.resolvedIcons = {
        ...this.resolvedIcons,
        [cmdId]: "/assets/tool/default.png"
      };
    },



    // 显示右键菜单
    showContextMenu(event, cmd = null, index = null) {
      event.preventDefault();

      const itemElement = event.target.closest(".item");
      const cateElement = event.target.closest(".cate");

      if (itemElement) {
        // 点击的是工具
        this.selectedCmd = cmd;
        this.selectedIndex = index;
        this.contextMenuOptions = [
          { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
          { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("toolEdit") },
          { label: "删除", icon: "el-icon-delete", action: this.deleteCmd },
          { label: "打开文件位置", icon: "el-icon-folder-opened", action: () => this.openPath(cmd.path) },
        ];
      } else if (cateElement) {
        // 点击的是分类
        const indexAttr = cateElement.getAttribute("data-index");
        const catIndex = Number(indexAttr);
        const category = this.toolList[catIndex];

        this.selectedIndex = catIndex;
        this.selectedCmd = { category: category.title }; // 把分类名塞进 selectedCmd

        this.contextMenuOptions = [
          { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("cateEdit") },
          {
            label: "删除",
            icon: "el-icon-delete",
            action: () => {
              this.$confirm(
                  `是否确认删除分类「${category.title}」？该分类下的所有工具也会被删除！`,
                  '警告',
                  {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                  }
              )
                  .then(() => this.deleteToolCategory(category.title))
                  .catch(() => {});
            },
          },
        ];
      } else {
        this.selectedCmd = null;
        this.selectedIndex = null;
        this.contextMenuOptions = [
          { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
        ];
      }

      this.contextMenuPosition = { x: event.clientX, y: event.clientY };
      this.contextMenuVisible = true;
    },

    // 打开文件位置
    openPath(path) {
      if (!path) {
        this.$message.warning('该工具未设置路径');
        return;
      }

      // 使用 Wails 的 Shell 方法打开文件夹
      OpenPath(path).then(() => {
        console.log('文件夹已打开:', path);
      }).catch(err => {
        this.$message.error('打开文件夹失败: ' + err.message);
        console.error('打开文件夹失败:', err);
      });
    },

    // 浏览文件夹
    async browseForFolder() {
      try {
        // 使用 Wails 的对话框选择文件夹
        const path = await GetOpenDir();
        if (path) {
          this.form.path = path;
        }
      } catch (err) {
        this.$message.error('选择文件夹失败: ' + err.message);
        console.error('浏览文件夹错误:', err);
      }
    },

    // 浏览文件夹
    async browseForIconFile() {
      try {
        // 使用 Wails 的对话框选择文件夹
        const icon = await GetOpenFilePath();
        if (icon) {
          this.form.icon = icon;
        }
      } catch (err) {
        this.$message.error('选择图标失败: ' + err.message);
        console.error('浏览图标错误:', err);
      }
    },

    // 隐藏右键菜单
    hideContextMenu() {
      this.contextMenuVisible = false;
    },

    // 全局右键点击处理
    handleGlobalContextMenu(event) {
      const itemElement = event.target.closest(".item");
      if (!itemElement) {
        // 点击的不是 .item 时显示 "新增" 菜单
        this.showContextMenu(event);
      }
      event.preventDefault(); // 阻止默认右键菜单
    },

    // 打开弹出框
    openDialog(mode) {
      this.dialogMode = mode;

      if (mode === "toolEdit" && this.selectedCmd) {
        // 编辑工具
        this.form = { ...this.selectedCmd };
      } else if (mode === "cateEdit" && this.selectedIndex !== null) {
        // 编辑分类，只需要设置 oldCategory 和 category 字段
        this.selectedCategory = this.toolList[this.selectedIndex].title;
        this.form = {
          oldCategory: this.selectedCategory,
          category: this.selectedCategory,
        };
      } else {
        this.resetForm();
      }

      this.dialogVisible = true;
      this.hideContextMenu();
    },

    // 重置表单
    resetForm() {
      this.form = {
        id: null,
        category: "",
        cmd: "",
        param: "",
        name: "",
        path: "",
        desc: "",
        icon: "",
        terminal: 1,
      };
    },
    // 查询分类列表，支持动态筛选
    queryCategories(query, callback) {
      if (!query) {
        // 如果没有输入内容，返回所有分类
        callback(this.categories.map((cat) => ({ value: cat.title })));
      } else {
        // 根据输入内容筛选分类
        const results = this.categories
          .filter((cat) => cat.title.includes(query))
          .map((cat) => ({ value: cat.title }));
        callback(results);
      }
    },
    // 选择已有分类
    handleCategorySelect(selected) {
      this.form.category = selected.value;
    },
    // 提交表单
    async handleSubmit() {
      try {
        if (this.dialogMode === "cateEdit") {
          const { oldCategory, category: newCategory } = this.form;
          if (!newCategory) {
            this.$message.warning("请输入新的分类名称");
            return;
          }

          // 分类重命名
          await UpdateToolCategory(oldCategory, newCategory);
          this.$message.success("分类修改成功");
          this.dialogVisible = false;
          await this.loadToolList(); // ✅ 修改后立即刷新
          await this.loadAllIcons();

        } else {
          // 工具新增/修改
          const categoryExists = this.categories.some(
              (cat) => cat.title === this.form.category
          );
          if (!categoryExists) {
            // 新增分类
            this.categories.push({ title: this.form.category });
            console.log("新增分类:", this.form.category);
          }

          if (this.dialogMode === "add") {
            await AddTool(this.form);
            // 新增工具后添加到缓存
            const newResolvedIcons = {...this.resolvedIcons};
            newResolvedIcons[this.form.id] = await this.resolveIconPath(this.form.icon);
            this.resolvedIcons = newResolvedIcons;
            this.$message.success(`新增命令成功: ${this.form.name}`);
            console.log("新增命令成功:", this.form);
          } else if (this.dialogMode === "toolEdit") {
            await UpdateTool(this.form.id, this.form);
            // 修改工具后更新缓存
            const newResolvedIcons = {...this.resolvedIcons};
            newResolvedIcons[this.form.id] = await this.resolveIconPath(this.form.icon);
            this.resolvedIcons = newResolvedIcons;
            this.$message.success(`修改命令成功: ${this.form.name}`);
            console.log("修改命令成功:", this.form);
          }
        }

        this.dialogVisible = false;
        await this.loadToolList(); // 刷新数据
        await this.loadAllIcons();
      } catch (error) {
        const action = this.dialogMode === "add" ? "新增命令" :
            this.dialogMode === "toolEdit" ? "修改命令" :
                "修改分类";
        console.error(`${action}失败:`, error);
        this.$message.error(`${action}失败`);
      }
    },

    // 删除命令
    async deleteCmd() {
      this.hideContextMenu();
      if (this.selectedCmd && this.selectedCmd.id) {
        try {
          await this.$confirm("确认要删除此命令吗？", "删除确认", {
            confirmButtonText: "删除",
            cancelButtonText: "取消",
            type: "warning",
          });
          await DeleteTool(this.selectedCmd.id);
          this.$message.success("删除命令成功");
          console.log("删除命令成功:", this.selectedCmd);

          // 更新前端数据：从 toolList 中移除已删除的项
          for (let i = 0; i < this.toolList.length; i++) {
            const group = this.toolList[i];
            const index = group.list.findIndex((site) => site.id === this.selectedCmd.id);
            if (index !== -1) {
              group.list.splice(index, 1);
              // 如果该分类下没有站点，移除分类
              if (group.list.length === 0) {
                this.toolList.splice(i, 1);
              }
              break;
            }
          }

          this.selectedCmd = null; // 重置选中项
        } catch (error) {
          console.error("删除命令失败:", error);
        }
      }
    },
    // 删除分类
    async deleteToolCategory(category) {
      try {
        await DeleteToolCategory(category);
        this.$message.success(`分类「${category}」已删除`);
        await this.loadToolList(); // 重新加载工具列表
        await this.loadAllIcons();
      } catch (error) {
        console.error("删除分类失败:", error);
        this.$message.error("删除分类失败，请重试");
      }
    },

    async loadCategories() {
      try {
        this.categories = await GetCategoryList();
      } catch (error) {
        this.$message.error("加载分类失败");
        console.error("加载分类失败:", error);
      }
    },

    handleDragStart(e, type, catIndex, cmdIndex = -1) {
      this.dragData = {
        type,
        catIndex,
        cmdIndex
      }
      e.dataTransfer.effectAllowed = 'move'
    },

    handleDragOver(e, type, targetCatIndex, targetCmdIndex = -1) {
      // 添加可视化反馈
      e.target.classList.add('drag-over')
      e.dataTransfer.dropEffect = 'move'
    },
    async handleDrop(e, type, targetCatIndex, targetCmdIndex = -1) {
      e.stopPropagation();                  // 阻止冒泡
      e.currentTarget.classList.remove('drag-over');

      const { type: dragType, catIndex: srcCat, cmdIndex: srcCmd } = this.dragData;

      if (dragType === 'category' && type === 'category') {
        // —— 分类 ↔ 分类 互换 ——
        const moved = this.toolList.splice(srcCat, 1)[0];
        this.toolList.splice(targetCatIndex, 0, moved);
        await this.updateCategorySort();

      } else if (dragType === 'command' && type === 'command') {
        // —— 命令 ↔ 命令 同分类内部移动 ——
        const cmds = this.toolList[srcCat].list;
        const moved = cmds.splice(srcCmd, 1)[0];
        cmds.splice(targetCmdIndex, 0, moved);
        await this.updateCommandSort(srcCat);

      } else if (dragType === 'command' && type === 'category') {
        // —— 跨分类移动命令 ——
        const srcCmds    = this.toolList[srcCat].list;
        const targetCmds = this.toolList[targetCatIndex].list;
        const moved      = srcCmds.splice(srcCmd, 1)[0];
        targetCmds.push(moved);
        await this.moveCommandToCategory(moved.id, targetCatIndex);
      }
    },

    // 更新分类排序到后端
    // 确保发送正确的数据类型
    async updateCategorySort() {
      const sortData = this.toolList.map((cat, index) => ({
        category: cat.title,
        cateSort: Number(this.toolList.length - index) // 明确转换为Number
      }))

      console.log('排序数据:', JSON.stringify(sortData))

      try {
        await UpdateCategorySorts(sortData)
      } catch (err) {
        console.error('保存失败:', err)
      }
    },

    // 更新命令排序
    async updateCommandSort(catIndex) {
      const commands = this.toolList[catIndex].list
      const sortData = commands.map((cmd, index) => ({
        id: cmd.id,
        cmdSort: commands.length - index,
        category: this.toolList[catIndex].title
      }))

      try {
        await UpdateCommandSorts(sortData)
      } catch (err) {
        console.error('命令排序保存失败:', err)
      }
    },
    async moveCommandToCategory(commandId, targetCatIndex) {
      const targetCategory = this.toolList[targetCatIndex].title

      try {
        // 调用后端接口
        await MoveCommandToCategory({
          id: commandId,
          newCategory: targetCategory,
          // 添加到新分类的末尾，可根据需要调整排序值
          newCmdSort: this.toolList[targetCatIndex].list.length + 1
        })

        // 刷新数据
        await this.loadToolList()
        await this.loadAllIcons()
        this.$message.success('命令移动成功')
      } catch (err) {
        console.error('移动命令失败:', err)
        this.$message.error('命令移动失败：' + err.message)
        // 回退本地数据
        await this.loadToolList()
        await this.loadAllIcons()
      }
    },
  },

  mounted() {
    this.loadToolList();   // 加载工具列表
    this.loadAllIcons();
    this.loadCategories(); // 加载分类列表
    // 监听全局 contextmenu 事件
    document.addEventListener("contextmenu", this.handleGlobalContextMenu);
    document.addEventListener("click", this.hideContextMenu);
  },
  beforeUnmount() {
    // 移除全局 contextmenu 和 click 事件监听
    document.removeEventListener("contextmenu", this.handleGlobalContextMenu);
    document.removeEventListener("click", this.hideContextMenu);
  },
};
</script>


<style lang="scss" scoped>
/* 右上角小目录树 */
.cate-tree {
  position: fixed;
  top: 20px;
  right: 15px;
  width: 140px;
  max-height: 30px; /* 默认收起状态的高度 */
  overflow: hidden; /* 隐藏超出内容 */
  background: rgba(255, 255, 255, 0.9);
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  z-index: 1000;
  transition: max-height 0.3s ease; /* 添加高度过渡动画 */
}

/* 鼠标悬停时展开 */
.cate-tree:hover {
  max-height: 300px; /* 展开后的最大高度 */
  overflow: auto; /* 内容超出时显示滚动条 */
}

/* Tree 本身字体调小 */
.small-tree .el-tree-node__label {
  font-size: 12px;
  padding: 2px 0;
}


.system {
  margin: 10px 0 10px 10px;

  .nav {
    .cate {
      margin: 10px 0;
      font-size: 16px;
      color: #000000;
      font-weight: 600;
    }

    .site-list {
      display: flex;
      flex-wrap: wrap;

      .item {
        padding: 10px;
        background: #f5fafd;
        margin: 10px 10px 0 0;
        width: calc(33.33% - 30px);
        min-height: 40px;
        display: flex;
        align-items: center;
        cursor: pointer;
        border-radius: 8px;
        transition: all 0.2s;

        &:hover {
          box-shadow: 0 5px 12px rgba(0, 0, 0, 0.1);
          transform: translateY(-3px);
        }

        .desc {
          margin-left: 10px;
          display: flex;
          flex-direction: column;
          font-size: 14px;
          overflow: hidden;
          text-overflow: ellipsis;
          width: 140px;

          .title {
            white-space: nowrap;
            height: 22px;
            color: #242f40;
            font-weight: 500;
          }

          .remark {
            height: 34px;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            color: #b2b0c0;
            font-size: 12px;
          }
        }
      }
    }
  }

  .search-bar {
    display: flex;
    align-items: center;
    background: rgba(0, 0, 0, 0.63);   // 收起时半透明黑
    border-radius: 15px;
    border: 1px solid transparent; /* 占位，方便后面 override */
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    padding: 5px;
    user-select: none;

    &:focus-within {
      outline: none;
      box-shadow: none;
    }

    &.expanded {
      /* 外壳改灰色 & 去阴影 */
      background: rgb(44, 43, 43) !important;
      box-shadow: none !important;
      border: none !important;

      /* 覆盖 el-input 的 wrapper */
      :deep(.el-input__wrapper) {
        background: transparent !important;
        border: none !important;
        box-shadow: none !important;
      }
      /* 覆盖真正的 input */
      :deep(.el-input__inner) {
        border: none !important;
        box-shadow: none !important;
        background: transparent !important;
      }
    }

    .search-icon-collapsed {
      font-size: 24px;
      color: #ffffff;
      width: 40px;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .custom-search-input-expanded {
      width: 200px;
      transition: width 0.2s ease, opacity 0.2s ease;

      /* 整个 wrapper 用灰色底 */
      :deep(.el-input__wrapper) {
        background: transparent !important;
      }

      /* 覆盖 el-input__wrapper 聚焦样式 */
      :deep(.el-input__wrapper:focus-within) {
        outline: none !important;
        box-shadow: none !important;
        background: transparent !important;
      }

      /* 输入框本身也设置灰底 */
      :deep(.el-input__inner) {
        height: 36px;
        background: transparent !important;
        border: none;
        padding-left: 30px;
        color: #ffffff !important;
      }

      :deep(.el-input__inner:focus) {
        outline: none !important;
        box-shadow: none !important;
      }

      :deep(.el-input__clear) {
        font-size: 18px;
      }

      :deep(.search-icon-expanded) {
        font-size: 18px;
        color: #ffffff;
      }
    }
  }


  .context-menu {
    position: absolute;
    z-index: 1000;
    background-color: #ffffff;
    border: 1px solid #e6e6e6;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    border-radius: 8px;
    list-style: none;
    padding: 8px 0;
    margin: 0;

    li {
      padding: 10px 16px;
      font-size: 14px;
      color: #333333;
      display: flex;
      align-items: center;
      cursor: pointer;
      transition: background-color 0.2s, color 0.2s;

      i {
        margin-right: 8px;
        font-size: 16px;
        color: #666666;
      }

      &:hover {
        background-color: #f5f5f5;
        color: #1e80ff;

        i {
          color: #1e80ff;
        }
      }
    }
  }

}

</style>
