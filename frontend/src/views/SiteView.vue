<template>
  <div class="site">
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

    <!-- 站点列表 -->
    <div class="nav" v-for="(category, catIndex) in processedSiteList" :key="catIndex"
         draggable="true"
         @dragstart.stop="handleDragStart($event, 'category', catIndex)"
         @dragover.prevent="handleDragOver($event, 'category', catIndex)"
         @drop="handleDrop($event, 'category', catIndex)">
      <div class="cate" :data-index="catIndex">{{ category.title }}</div>
      <div class="site-list">
        <div
            class="item"
            v-for="(site, siteIndex) in category.list"
            :key="siteIndex"
            draggable="true"
            @dragstart.stop="handleDragStart($event, 'command', catIndex, siteIndex)"
            @dragover.prevent.stop="handleDragOver($event, 'command', catIndex, siteIndex)"
            @drop.stop="handleDrop($event, 'command', catIndex, siteIndex)"
            @click="openUrl(site.url, false)"
            @contextmenu.prevent="showContextMenu($event, site, index)"
        >
          <div class="image">
            <!-- 使用解析后的图标 -->
            <el-image
                style="width: 36px; height: 36px"
                :src="site.resolvedIcon"
                fit="cover"
                :alt="site.name"
                @error="handleImageError(site)"
            >
              <template #error>
                <div class="image-error">
                  <span>图标加载失败</span>
                </div>
              </template>
            </el-image>
          </div>
          <div class="desc" :title="site.remark">
            <span class="title">{{ site.title}}</span>
            <span class="remark">{{ site.remark}}</span>
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

        <template v-else>
          <el-form-item label="分类" prop="category">
            <el-autocomplete v-model="form.category" :fetch-suggestions="queryCategories" placeholder="请输入或选择分类"
              @select="handleCategorySelect"></el-autocomplete>
          </el-form-item>
          <el-form-item label="URL" prop="url">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input v-model="form.url" placeholder="请输入站点URL" style="width: 100%;"></el-input>
              <el-button type="warning" @click="autoFetchSiteInfo" title="自动获取站点信息" style="height: 30px; padding: 0 10px;">自动获取</el-button>
            </div>
          </el-form-item>
          <el-form-item label="标题" prop="title">
            <el-input v-model="form.title" placeholder="请输入站点标题"></el-input>
          </el-form-item>
          <el-form-item label="名称" prop="remark">
            <el-input v-model="form.remark" placeholder="请输入站点描述"></el-input>
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
import {
  AddSite,
  DeleteSite,
  DeleteSiteCategory,
  GetAllSites,
  GetCategoryList,
  GetSearchSites,
  MoveCommandToCategory,
  UpdateCategorySorts,
  UpdateCommandSorts,
  UpdateSite,
  UpdateSiteCategory,
  FetchSiteInfo // <- 新增导入
} from "../../wailsjs/go/controller/Site";
import {ReadImageAsBase64} from "../../wailsjs/go/controller/Tool";
import {BrowserOpenURL} from "../../wailsjs/runtime"; // 打开链接的方法
import {Search} from "@element-plus/icons-vue";
import {ElNotification, ElTree} from "element-plus";
import {GetConfigDir, GetOpenFilePath, OpenPath} from "../../wailsjs/go/controller/System";

export default {
  name: "SiteView",
  components: { ElTree,Search },
  data() {
    return {
      siteList: [],
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
      defaultIcon: "/assets/site/default.png", // 默认图标路径
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
        title: "",
        url: "",
        remark: "",
        icon: "",
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
        case "add": return "新增站点";
        case "siteEdit": return "修改站点";
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
          children: this.siteList.map((cat, idx) => ({ label: cat.title, id: idx }))
        }
      ];
    },
    // 处理后的工具列表，包含解析后的图标
    processedSiteList() {
      return this.siteList.map(category => {
        return {
          ...category,
          list: category.list.map(cmd => {
            return {
              ...cmd,
              resolvedIcon: this.resolvedIcons[cmd.id] || this.defaultIcon
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
    async loadSiteList() {
      try {
        this.siteList = await GetAllSites();
        await this.loadAllIcons();
      } catch (error) {
        console.error("加载命令列表失败:", error);
      }
    },
    async handleSearch() {
      try {
        if (!this.searchQuery.trim()) {
          this.siteList = await GetAllSites();
        } else {
          this.siteList = await GetSearchSites(this.searchQuery.trim());
        }
      } catch (error) {
        this.siteList = []
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
    openUrl(url, self = false) {
      if (!url) {
        console.error("URL 不能为空");
        return;
      }

      try {
        if (self) {
          // 在当前窗口打开链接
          location.href = url;
        } else {
          // 调用 Wails 方法在默认浏览器中打开链接
          BrowserOpenURL(url);
        }
      } catch (error) {
        console.error("打开链接失败:", error);
      }
    },
    // 异步加载所有图标
    async loadAllIcons() {
      const iconPromises = [];

      // 创建一个新的图标缓存对象
      const newResolvedIcons = {...this.resolvedIcons};

      // 收集所有需要解析的图标
      this.siteList.forEach(category => {
        category.list.forEach(site => {
          if (site.icon && !this.resolvedIcons[site.id]) {
            iconPromises.push(
                this.resolveIconPath(site.icon).then(icon => {
                  // 直接更新缓存对象
                  newResolvedIcons[site.id] = icon;
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
        return this.defaultIcon;
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
        return this.defaultIcon;
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
    handleImageError(site) {
      const siteId = site.id || site.name;
      this.resolvedIcons = {
        ...this.resolvedIcons,
        [siteId]: this.defaultIcon
      };
    },

    // 显示右键菜单
    showContextMenu(event, cmd = null, index = null) {
      event.preventDefault(); // 阻止浏览器默认右键菜单

      const itemElement = event.target.closest(".item");
      const cateElement = event.target.closest(".cate");
      if (itemElement) {
        // console.log("点击了站点菜单");
        // 点击的是 .item 元素
        this.selectedCmd = cmd;
        this.selectedIndex = index;
        this.contextMenuOptions = [
          { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
          { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("siteEdit") },
          { label: "删除", icon: "el-icon-delete", action: this.deleteCmd },
          { label: "打开图标文件夹", icon: "el-icon-delete", action: this.openIconDir },
        ];
      } else if (cateElement) {
        // 点击的是分类
        const indexAttr = cateElement.getAttribute("data-index");
        const catIndex = Number(indexAttr);
        const category = this.siteList[catIndex];

        this.selectedIndex = catIndex;
        this.selectedCmd = { category: category.title }; // 把分类名塞进 selectedCmd

        this.contextMenuOptions = [
          { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("cateEdit") },
          {
            label: "删除",
            icon: "el-icon-delete",
            action: () => {
              this.$confirm(
                  `是否确认删除分类「${category.title}」？该分类下的所有网址也会被删除！`,
                  '警告',
                  {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                  }
              )
                  .then(() => this.deleteSiteCategory(category.title))
                  .catch(() => {});
            },
          },
        ];
      } else {
        // 点击的不是 .item 元素
        this.selectedCmd = null;
        this.selectedIndex = null;
        this.contextMenuOptions = [
          { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
        ];
      }

      this.contextMenuPosition = { x: event.clientX, y: event.clientY };
      this.contextMenuVisible = true;
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

    // 打开文件夹
    async openIconDir() {
      const baseDir = await GetConfigDir()
      const iconDir = baseDir + "/icon"; // 拼接icon子目录
      await OpenPath(iconDir)
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
      if (mode === "siteEdit" && this.selectedCmd) {
        this.form = { ...this.selectedCmd };
      } else if (mode === "cateEdit" && this.selectedIndex) {
        // 编辑分类，只需要设置 oldCategory 和 category 字段
        this.selectedCategory = this.siteList[this.selectedIndex].title;
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
          await UpdateSiteCategory(oldCategory, newCategory);
          this.$message.success("分类修改成功");
          this.dialogVisible = false;
          await this.loadSiteList(); // ✅ 修改后立即刷新
          await this.loadAllIcons();

        } else {
          const categoryExists = this.categories.some(
              (cat) => cat.title === this.form.category
          );

          if (!categoryExists) {
            // 如果分类不存在，新增分类
            this.categories.push({ title: this.form.category });
            console.log("新增分类:", this.form.category);
          }

          if (this.dialogMode === "add") {
            // 新增命令
            const id = await AddSite(this.form);
            const newResolvedIcons = {...this.resolvedIcons};
            newResolvedIcons[this.form.id] = await this.resolveIconPath(this.form.icon);
            this.resolvedIcons = newResolvedIcons;
            this.$message.success(`新增成功`);
          } else if (this.dialogMode === "siteEdit") {
            // 修改命令
            await UpdateSite(this.form.id, this.form);
            // 修改工具后更新缓存
            const newResolvedIcons = {...this.resolvedIcons};
            newResolvedIcons[this.form.id] = await this.resolveIconPath(this.form.icon);
            this.resolvedIcons = newResolvedIcons;
            this.$message.success("修改成功");
          }

          this.dialogVisible = false;
          await this.loadSiteList(); // 刷新站点列表
          await this.loadAllIcons();
        }
      } catch (error) {
        const action = this.dialogMode === "add" ? "新增命令" :
            this.dialogMode === "siteEdit" ? "修改命令" :
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
          await DeleteSite(this.selectedCmd.id);
          console.log("删除命令成功:", this.selectedCmd);

          // 更新前端数据：从 siteList 中移除已删除的项
          for (let i = 0; i < this.siteList.length; i++) {
            const group = this.siteList[i];
            const index = group.list.findIndex((site) => site.id === this.selectedCmd.id);
            if (index !== -1) {
              group.list.splice(index, 1);
              // 如果该分类下没有站点，移除分类
              if (group.list.length === 0) {
                this.siteList.splice(i, 1);
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
    async deleteSiteCategory(category) {
      try {
        await DeleteSiteCategory(category);
        this.$message.success(`分类「${category}」已删除`);
        await this.loadSiteList(); // 重新加载工具列表
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
        const moved = this.siteList.splice(srcCat, 1)[0];
        this.siteList.splice(targetCatIndex, 0, moved);
        await this.updateCategorySort();

      } else if (dragType === 'command' && type === 'command') {
        // —— 命令 ↔ 命令 同分类内部移动 ——
        const cmds = this.siteList[srcCat].list;
        const moved = cmds.splice(srcCmd, 1)[0];
        cmds.splice(targetCmdIndex, 0, moved);
        await this.updateCommandSort(srcCat);

      } else if (dragType === 'command' && type === 'category') {
        // —— 跨分类移动命令 ——
        const srcCmds    = this.siteList[srcCat].list;
        const targetCmds = this.siteList[targetCatIndex].list;
        const moved      = srcCmds.splice(srcCmd, 1)[0];
        targetCmds.push(moved);
        await this.moveCommandToCategory(moved.id, targetCatIndex);
      }
    },

    // 更新分类排序到后端
    // 确保发送正确的数据类型
    async updateCategorySort() {
      const sortData = this.siteList.map((cat, index) => ({
        category: cat.title,
        cateSort: Number(this.siteList.length - index) // 明确转换为Number
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
      const commands = this.siteList[catIndex].list
      const sortData = commands.map((cmd, index) => ({
        id: cmd.id,
        cmdSort: commands.length - index,
        category: this.siteList[catIndex].title
      }))

      try {
        await UpdateCommandSorts(sortData)
      } catch (err) {
        console.error('命令排序保存失败:', err)
      }
    },
    async moveCommandToCategory(commandId, targetCatIndex) {
      const targetCategory = this.siteList[targetCatIndex].title

      try {
        // 调用后端接口
        await MoveCommandToCategory({
          id: commandId,
          newCategory: targetCategory,
          // 添加到新分类的末尾，可根据需要调整排序值
          newCmdSort: this.siteList[targetCatIndex].list.length + 1
        })

        // 刷新数据
        await this.loadSiteList()
        await this.loadAllIcons();
        this.$message.success('命令移动成功')
      } catch (err) {
        console.error('移动命令失败:', err)
        this.$message.error('命令移动失败：' + err.message)
        // 回退本地数据
        await this.loadSiteList()
        await this.loadAllIcons();
      }
    },
    async autoFetchSiteInfo() {
      if (!this.form.url || !this.form.url.trim()) {
        this.$message.warning("请输入 URL 后再自动获取");
        return;
      }
      try {
        this.$message.info("正在获取信息，请稍等");
        // 调用后端
        const info = await FetchSiteInfo(this.form.url.trim());
        // info 里应该包含 title, remark, icon
        if (info) {
          if (info.title) this.form.title = info.title;
          if (info.remark) this.form.remark = info.remark;
          if (info.icon) {
            // 后端返回的是保存的文件名（例如 a1b2c3.png）
            this.form.icon = info.icon;
            // 更新预览（resolveIconPath 会把非 http 路径按照 /icon/... 处理）
            const resolved = await this.resolveIconPath(this.form.icon);
            // 使用临时 id（如果是新建还没有 id，就用一个特殊 key）
            const previewId = this.form.id || `__preview_${Date.now()}`;
            this.resolvedIcons = {
              ...this.resolvedIcons,
              [previewId]: resolved
            };
            // 如果是编辑已有 id，直接更新那个 id 的 resolved 图标
            if (this.form.id) {
              this.resolvedIcons = {
                ...this.resolvedIcons,
                [this.form.id]: resolved
              };
            }
          }
          this.$message.success("自动获取成功");
        } else {
          this.$message.info("未获取到任何信息");
        }
      } catch (err) {
        console.error("自动获取失败:", err);
        this.$message.error("自动获取失败: " + (err.message || err));
      }
    },
  },

  mounted() {
    this.loadSiteList();
    this.loadAllIcons();
    this.loadCategories();
    // 全局右键菜单与点击隐藏
    document.addEventListener('contextmenu', this.handleGlobalContextMenu);
    document.addEventListener('click', this.hideContextMenu);
  },
  beforeUnmount() {
    document.removeEventListener('contextmenu', this.handleGlobalContextMenu);
    document.removeEventListener('click', this.hideContextMenu);
  },
}

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

.site {
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

          .title {
            white-space: nowrap;
            height: 22px;
            color: #242f40;
            font-weight: 500;
          }

          .remark {
            height: 38px;          /* 增加高度以容纳更大的行间距 */
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            color: #b2b0c0;
            font-size: 12px;
            line-height: 18px;     /* 新增：设置行间距为20px */
            overflow: hidden;      /* 确保超出部分被隐藏 */
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

  /* 淡入淡出 */
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.2s;
  }
  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
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

  .custom-dialog {
    .el-dialog__header {
      background-color: #f5f5f5;
      border-bottom: 1px solid #e6e6e6;
      border-radius: 8px 8px 0 0;
      font-weight: 600;
    }

    .el-dialog__footer {
      border-top: none;
      display: flex;
      justify-content: flex-end;
    }

    .el-form-item {
      .el-input__inner {
        border-radius: 4px;
      }

      .el-switch {
        width: 60px;
      }
    }

    .el-button {
      border-radius: 4px;

      &:not(:last-child) {
        margin-right: 10px;
      }
    }
  }
}
</style>
