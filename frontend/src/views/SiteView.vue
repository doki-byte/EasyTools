<template>
  <div class="site">
    <!-- 搜索框 -->
    <div class="search-bar">
      <el-input v-model="searchQuery" placeholder="请输入关键字搜索" clearable @input="handleSearch"
        class="custom-search-input">
        <template #prefix>
          <el-icon class="search-icon">
            <Search />
          </el-icon>
        </template>
      </el-input>
    </div>

    <!-- 站点列表 -->
    <div class="nav" v-for="(category, catIndex) in siteList" :key="catIndex"
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
            <el-image style="width: 36px; height: 36px" :src="resolveIconPath(site.icon)" fit="cover" :alt="site.name" />
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
          <el-form-item label="标题" prop="title">
            <el-input v-model="form.title" placeholder="请输入站点标题"></el-input>
          </el-form-item>
          <el-form-item label="URL" prop="url">
            <el-input v-model="form.url" placeholder="请输入站点URL"></el-input>
          </el-form-item>
          <el-form-item label="名称" prop="remark">
            <el-input v-model="form.remark" placeholder="请输入站点描述"></el-input>
          </el-form-item>
          <el-form-item label="图标" prop="icon">
            <el-input v-model="form.icon" placeholder="请输入图标地址(可为url)"></el-input>
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
  GetAllSites,
  GetCategoryList,
  GetSearchSites,
  UpdateSite,
  UpdateSiteCategory,
  DeleteSiteCategory,
  UpdateCommandSorts,
  UpdateCategorySorts,
  MoveCommandToCategory,
} from "../../wailsjs/go/controller/Site";
import {BrowserOpenURL} from "../../wailsjs/runtime"; // 打开链接的方法
import {Search} from "@element-plus/icons-vue";

export default {
  name: "SiteView",
  components: { Search },
  data() {
    return {
      siteList: [],
      searchQuery: "",
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
  methods: {
    // 加载命令列表
    async loadSiteList() {
      try {
        this.siteList = await GetAllSites();
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
    resolveIconPath(icon) {
      if (!icon) {
        // 如果路径不存在，返回默认图标
        return "/assets/site/default.png";
      }
      if (icon.startsWith("/assets") || icon.startsWith("http://") || icon.startsWith("https://")) {
        // 如果是本地资源或已经是完整的 URL，直接返回
        return icon;
      }
      // 其他情况，生成远程路径
      return `http://127.0.0.1:10086/icon/${icon}`;
    },

    // 显示右键菜单
    showContextMenu(event, cmd = null, index = null) {
      event.preventDefault(); // 阻止浏览器默认右键菜单

      const itemElement = event.target.closest(".item");
      const cateElement = event.target.closest(".cate");
      if (itemElement) {
        console.log("点击了站点菜单");
        // 点击的是 .item 元素
        this.selectedCmd = cmd;
        this.selectedIndex = index;
        this.contextMenuOptions = [
          { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
          { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("siteEdit") },
          { label: "删除", icon: "el-icon-delete", action: this.deleteCmd },
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
            console.log("新增命令成功:", this.form);
          } else if (this.dialogMode === "siteEdit") {
            // 修改命令
            await UpdateSite(this.form.id, this.form);
            console.log("修改命令成功:", this.form);
          }

          this.dialogVisible = false;
          await this.loadSiteList(); // 刷新站点列表
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
        this.$message.success('命令移动成功')
      } catch (err) {
        console.error('移动命令失败:', err)
        this.$message.error('命令移动失败：' + err.message)
        // 回退本地数据
        await this.loadSiteList()
      }
    },
  },
  computed: {
    dialogTitle() {
      switch (this.dialogMode) {
        case "add": return "新增站点";
        case "siteEdit": return "修改站点";
        case "cateEdit": return "修改分类";
        default: return "编辑";
      }
    }
  },
  mounted() {
    this.loadSiteList();
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
    margin: 20px;
    display: flex;
    justify-content: center;

    .custom-search-input {
      width: 100%;
      max-width: 500px;
      height: 45px;
      border: none;
      border-radius: 25px;
      box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
      overflow: hidden;
      background-color: #f9f9f9;
      position: relative;

      /* 将 ::v-deep 替换为 :deep */
      :deep(.el-input__inner) {
        height: 100%;
        border: none;
        border-radius: 25px;
        font-size: 16px;
        background-color: transparent;
        padding-left: 30px !important;
        padding-right: 15px;

        &:focus {
          outline: none;
        }
      }

      :deep(.el-input__clear) {
        font-size: 20px;
        color: #888;
      }

      :deep(.el-input__prefix) {
        padding-left: 10px !important;
      }

      :deep(.search-icon) {
        font-size: 20px;
        color: #999;
        position: absolute;
        left: 15px;
        top: 50%;
        transform: translateY(-50%);
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
