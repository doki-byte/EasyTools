<template>
  <div class="system" @contextmenu="handleGlobalContextMenu" @drop.prevent="handleFileDrop" @dragover.prevent="handleDragOver">
    <div
        v-show="isDragOver"
        class="drop-zone-overlay"
        @drop="handleFileDrop"
        @dragover.prevent
        @dragleave="handleDragLeave"
    >
      <div class="drop-zone-content">
        <el-icon size="50"><UploadFilled /></el-icon>
        <div class="drop-text">释放文件到此处添加工具</div>
      </div>
    </div>

    <!-- 1. 右上角目录树 -->
    <div ref="treeWrapper" class="cate-tree">
      <el-tree
          ref="tree"
          :data="treeData"
          :props="{ children: 'children', label: 'label' }"
          node-key="id"
          @node-click="onTreeNodeClick"
          @node-contextmenu="onTreeContextMenu"
          :highlight-current="true"
          :default-expand-all="false"
          :expand-on-click-node="true"
          class="small-tree"
      />
    </div>

    <!-- 统一的右键菜单（Teleport 到 body，避免定位偏移） -->
    <Teleport to="body">
      <ul class="context-menu" v-if="contextMenuVisible"
          :style="contextMenuStyle">
        <li v-for="(option, index) in contextMenuOptions" :key="index" @click="option.action">
          <i :class="option.icon"></i> {{ option.label }}
        </li>
      </ul>
    </Teleport>

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

    <!-- 软件中心提示卡片 -->
    <transition name="fade">
      <div 
        v-if="showSoftwareCenterTip" 
        class="software-center-tip"
        :style="tipStyle"
      >
        <el-icon class="tip-icon"><InfoFilled /></el-icon>
        <span class="tip-text">不知道从哪里获取工具？可以查看软件中心哦</span>
        <el-button 
          text 
          class="tip-jump-btn"
          @click.stop="goToSoftwareCenter"
        >
          前往
        </el-button>
        <el-button 
          text 
          class="tip-close-btn"
          @click.stop="closeSoftwareCenterTip"
        >
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
    </transition>

    <!-- 工具列表 -->
    <div class="nav" v-for="(category, catIndex) in processedToolList" :key="catIndex">
      <div class="cate"
           :data-index="catIndex"
           @contextmenu.stop="showCategoryContextMenu($event, category, catIndex)"
           @mouseenter="showCategoryActions = catIndex"
           @mouseleave="showCategoryActions = -1">
        {{ category.title }}
        <!-- 分类移动按钮 -->
        <div class="category-actions" v-show="showCategoryActions === catIndex">
          <el-icon class="move-btn up" @click.stop="moveCategoryUp(catIndex)">
            <ArrowUp />
          </el-icon>
          <el-icon class="move-btn down" @click.stop="moveCategoryDown(catIndex)">
            <ArrowDown />
          </el-icon>
        </div>
      </div>
      <div class="tool-list">
        <div
            class="item"
            v-for="(cmd, cmdIndex) in category.list"
            :key="cmdIndex"
            @click="runCmd(cmd.path, cmd.cmd, cmd.param, cmd.terminal)"
            @contextmenu.stop="showToolContextMenu($event, cmd, catIndex)"
            @mouseenter="showToolActions[catIndex] = cmdIndex"
            @mouseleave="hideToolActions(catIndex, cmdIndex)"
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
          <!-- 工具移动按钮 -->
          <div class="tool-actions" v-show="showToolActions[catIndex] === cmdIndex">
            <el-icon class="move-btn left" @click.stop="moveToolLeft(catIndex, cmdIndex)">
              <ArrowLeft />
            </el-icon>
            <el-icon class="move-btn right" @click.stop="moveToolRight(catIndex, cmdIndex)">
              <ArrowRight />
            </el-icon>
          </div>
        </div>
      </div>
    </div>

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
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-autocomplete
                  v-model="form.category"
                  :fetch-suggestions="queryCategories"
                  placeholder="请输入或选择分类"
                  @select="handleCategorySelect"
              ></el-autocomplete>
              <el-tooltip
                  effect="dark"
                  content="若分类不存在则会自动创建并保存"
                  placement="bottom-start"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
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
              <el-tooltip
                  effect="dark"
                  content="指定工具执行文件所在的完整路径(eg: C:\EasyTools\tools\Godzilla)"
                  placement="bottom-start"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </el-form-item>

          <el-form-item label="命令" prop="cmd">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input
                  v-model="form.cmd"
                  placeholder="请输入工具命令"
                  style="flex: 1;"
              ></el-input>
              <el-button
                  type="primary"
                  @click="browseForCmdFile"
                  style="height: 30px; padding: 0 10px;"
              >选择</el-button>
              <el-tooltip
                  effect="dark"
                  placement="bottom-start"
              >
                <template #content>
                  <div style="text-align: left; line-height: 1.5;">
                    <div>• 输入启动工具需要执行的主命令(eg: java -jar godzilla.jar)</div>
                    <div>• 如果直接是可执行文件exe、bat、vbs等可使用'选择'功能直接选取</div>
                  </div>
                </template>
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </el-form-item>

          <el-form-item label="参数" prop="param">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input v-model="form.param" placeholder="请输入工具参数"></el-input>
              <el-tooltip
                  effect="dark"
                  content="设置工具运行时的附加参数（可选）"
                  placement="bottom-start"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </el-form-item>

          <el-form-item label="名称" prop="name">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input v-model="form.name" placeholder="请输入工具名称"></el-input>
              <el-tooltip
                  effect="dark"
                  content="输入工具的显示名称（必填）"
                  placement="bottom-start"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </el-form-item>

          <el-form-item label="描述" prop="desc">
            <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
              <el-input v-model="form.desc" placeholder="请输入工具描述"></el-input>
              <el-tooltip
                  effect="dark"
                  content="简要描述工具的功能和用途（可选）"
                  placement="bottom-start"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
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
              <el-tooltip
                  effect="dark"
                  placement="bottom-start"
              >
                <template #content>
                  <div style="text-align: left; line-height: 1.5;">
                    <div>• 为空即用默认图标</div>
                    <div>• 支持使用http加载图标</div>
                    <div>• 支持绝对路径加载图标</div>
                    <div>• 当图标位置存放在(EasyToolsFiles/icon)下时，支持直接输入图片名称即可</div>
                  </div>
                </template>
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
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
            <el-tooltip
                effect="dark"
                content="开启之后会显示终端运行窗口"
                placement="bottom-start"
            >
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
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
  GetOpenChooseFilePath,
  GetOpenDir,
  GetOpenFilePath,
  OpenPath,
  OpenTerminalInDir,
  ShellCMD
} from "../../wailsjs/go/system/System";
import {
  AddTool, AnalyzeFilePath, AutoGenerateIcon,
  DeleteTool,
  DeleteToolCategory,
  GetAllTools,
  GetCategoryList,
  GetSearchTools,
  ReadImageAsBase64,
  UpdateTool,
  UpdateToolCategory,
  UpdateCategorySorts,
  UpdateCommandSorts
} from "../../wailsjs/go/controller/Tool";
import {QuestionFilled, Search, UploadFilled, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, InfoFilled, Close} from "@element-plus/icons-vue";
import {ElNotification, ElTree} from "element-plus";
import {GetPortConfig} from "../../wailsjs/go/controller/Server";
import {OnFileDrop} from "../../wailsjs/runtime";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
export default {
  name: "ToolsView",
  components: {UploadFilled, QuestionFilled, ElTree, Search, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, InfoFilled, Close },
  data() {
    return {
      portConfig: null,
      toolList: [], // 初始化命令列表
      resolvedIcons: {}, // 图标缓存 {id: resolvedPath}

      // 移动按钮显示状态
      showCategoryActions: -1,
      showToolActions: {},

      // 搜索框状态
      searchExpanded: false,
      // 拖拽相关
      dragging: false,
      dragOffset: { x: 0, y: 0 },
      isDragOver: false,
      isAutoGenerateIcon: true,
      showSoftwareCenterTip: false, // 软件中心提示卡片显示状态（随机显示）

      // 当前位置
      searchPosition: {
        x: window.innerWidth - 80,
        y: window.innerHeight - 80,
      },
      searchQuery: "",

      defaultExpandedKeys: [],

      // 统一的右键菜单
      contextMenuVisible: false,
      contextMenuOptions: [],
      contextMenuPosition: { x: 0, y: 0 },
      menuItemHeight: 40, // 每个菜单项的大致高度
      menuWidth: 200, // 菜单宽度

      // 当前选中的项
      selectedCmd: null,
      selectedIndex: null,
      selectedTreeNode: null,
      selectedCategory: null,

      dialogVisible: false,
      dialogMode: "add", // 'add' or 'edit'
      form: {
        id: null,
        category: "",
        cmd: "",
        name: "",
        param: "",
        path: "",
        desc: "",
        icon: "",
        terminal: 1,
      },
      categories: [], // 初始化为空数组
      dragData: {
        type: null,    // 'category' 或 'command'
        catIndex: -1,
        cmdIndex: -1
      },
    };
  },
  mounted() {
  },
  computed: {
    dialogTitle() {
      switch (this.dialogMode) {
        case "add": return "新增工具(支持拖拽添加工具哦)";
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
    tipStyle() {
      return {
        position: "fixed",
        left: "50%",
        top: "50px",
        transform: "translateX(-50%) translateY(0)",
        zIndex: 998,
      };
    },
    // 统一的右键菜单样式计算
    contextMenuStyle() {
      let { x, y } = this.contextMenuPosition;
      const menuHeight = this.contextMenuOptions.length * this.menuItemHeight;

      // 边界检测逻辑
      if (x + this.menuWidth > window.innerWidth) {
        x = window.innerWidth - this.menuWidth - 10;
      }

      if (y + menuHeight > window.innerHeight) {
        y = window.innerHeight - menuHeight - 10;
      }

      if (x < 10) {
        x = 10;
      }

      if (y < 10) {
        y = 10;
      }

      return {
        top: `${y}px`,
        left: `${x}px`,
        position: 'fixed'
      };
    },
    treeData() {
      // 为了实现折叠，引入一个根节点"目录"
      return [
        {
          label: '目录',
          id: 'root',
          children: this.toolList.map((cat, idx) => ({
            label: cat.title,
            id: idx,
            category: cat.title // 保存分类名称用于操作
          }))
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

    // 修改分类移动方法，添加错误处理和回滚
    async moveCategoryUp(catIndex) {
      if (catIndex <= 0) return;

      // 保存原始顺序用于回滚
      const originalOrder = [...this.toolList];

      try {
        // 交换分类位置 - 直接使用数组操作
        const temp = this.toolList[catIndex - 1];
        this.toolList[catIndex - 1] = this.toolList[catIndex];
        this.toolList[catIndex] = temp;

        // 触发响应式更新
        this.toolList = [...this.toolList];

        // 更新分类顺序到后端
        await this.updateCategoryOrder();

        this.$message.success('分类顺序已更新');
      } catch (error) {
        // 回滚到原始顺序
        this.toolList = originalOrder;
        console.error('移动分类失败:', error);
        this.$message.error('移动分类失败');
      }
    },

    async moveCategoryDown(catIndex) {
      if (catIndex >= this.toolList.length - 1) return;

      // 保存原始顺序用于回滚
      const originalOrder = [...this.toolList];

      try {
        // 交换分类位置 - 直接使用数组操作
        const temp = this.toolList[catIndex + 1];
        this.toolList[catIndex + 1] = this.toolList[catIndex];
        this.toolList[catIndex] = temp;

        // 触发响应式更新
        this.toolList = [...this.toolList];

        // 更新分类顺序到后端
        await this.updateCategoryOrder();

        this.$message.success('分类顺序已更新');
      } catch (error) {
        // 回滚到原始顺序
        this.toolList = originalOrder;
        console.error('移动分类失败:', error);
        this.$message.error('移动分类失败');
      }
    },

    // 修改工具移动方法，添加错误处理和回滚
    async moveToolLeft(catIndex, cmdIndex) {
      if (cmdIndex <= 0) return;

      const category = this.toolList[catIndex];
      // 保存原始顺序用于回滚
      const originalOrder = [...category.list];

      try {
        // 交换工具位置 - 直接使用数组操作
        const temp = category.list[cmdIndex - 1];
        category.list[cmdIndex - 1] = category.list[cmdIndex];
        category.list[cmdIndex] = temp;

        // 触发响应式更新
        category.list = [...category.list];

        // 更新工具顺序到后端
        await this.updateToolOrder(catIndex);

        this.$message.success('工具顺序已更新');
      } catch (error) {
        // 回滚到原始顺序
        category.list = originalOrder;
        console.error('移动工具失败:', error);
        this.$message.error('移动工具失败');
      }
    },

    async moveToolRight(catIndex, cmdIndex) {
      const category = this.toolList[catIndex];
      if (cmdIndex >= category.list.length - 1) return;

      // 保存原始顺序用于回滚
      const originalOrder = [...category.list];

      try {
        // 交换工具位置 - 直接使用数组操作
        const temp = category.list[cmdIndex + 1];
        category.list[cmdIndex + 1] = category.list[cmdIndex];
        category.list[cmdIndex] = temp;

        // 触发响应式更新
        category.list = [...category.list];

        // 更新工具顺序到后端
        await this.updateToolOrder(catIndex);

        this.$message.success('工具顺序已更新');
      } catch (error) {
        // 回滚到原始顺序
        category.list = originalOrder;
        console.error('移动工具失败:', error);
        this.$message.error('移动工具失败');
      }
    },

    // 隐藏工具操作按钮
    hideToolActions(catIndex, cmdIndex) {
      this.showToolActions[catIndex] = -1;
    },


    // 更新分类顺序到后端
    async updateCategoryOrder() {
      try {
        // 构造分类顺序数据，匹配后端 UpdateCategorySorts 方法期望的数据结构
        const categorySorts = this.toolList.map((category, index) => ({
          category: category.title,
          cateSort: this.toolList.length - index  // 按显示顺序设置排序值，第一个分类排序值最大
        }));

        await UpdateCategorySorts(categorySorts);
        console.log('分类顺序更新成功');
      } catch (error) {
        console.error('更新分类顺序失败:', error);
        this.$message.error('更新分类顺序失败');
      }
    },

    // 更新工具顺序到后端
    async updateToolOrder(catIndex) {
      try {
        const category = this.toolList[catIndex];
        // 构造工具顺序数据，匹配后端 UpdateCommandSorts 方法期望的数据结构
        const commandSorts = category.list.map((tool, index) => ({
          id: tool.id,
          cmdSort: category.list.length - index,  // 按显示顺序设置排序值，第一个工具排序值最大
          category: category.title
        }));

        await UpdateCommandSorts(commandSorts);
        console.log('工具顺序更新成功');
      } catch (error) {
        console.error('更新工具顺序失败:', error);
        this.$message.error('更新工具顺序失败');
      }
    },

    // 新增拖拽相关方法
    handleDragOver(event) {
      event.preventDefault();
      this.isDragOver = true;
    },

    handleDragLeave(event) {
      if (!event.currentTarget.contains(event.relatedTarget)) {
        this.isDragOver = false;
      }
    },

    async handleFileDrop(event) {
      event.preventDefault();
      this.isDragOver = false;

      let filePaths = [];

      // 处理两种拖拽情况：浏览器文件对象和 Wails 路径数组
      if (event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files.length > 0) {
        // 浏览器文件拖拽
        console.log('浏览器文件拖拽事件:', event.dataTransfer.files);
        const files = Array.from(event.dataTransfer.files);
        filePaths = files.map(file => file.path);
      } else if (event.detail && event.detail.paths) {
        // Wails 文件拖拽 (通过自定义事件)
        console.log('Wails 文件拖拽事件:', event.detail.paths);
        filePaths = event.detail.paths;
      } else {
        // 直接通过 Wails OnFileDrop 调用
        console.log('直接 Wails 拖拽事件');
        // 这里需要从事件参数中获取路径
        if (event.paths && Array.isArray(event.paths)) {
          filePaths = event.paths;
        }
      }

      console.log('最终处理的文件路径:', filePaths);

      if (filePaths.length === 0) {
        console.log('没有找到有效的文件路径');
        return;
      }

      // 只处理第一个文件
      const filePath = filePaths[0];
      console.log('处理文件路径:', filePath);

      try {
        // 分析文件路径
        const fileInfo = await this.analyzeFilePath(filePath);
        console.log('文件分析结果:', fileInfo);

        // 重置表单
        this.resetForm();

        // 自动填充表单，默认使用第一个分类
        this.form.path = fileInfo.directory;
        this.form.cmd = fileInfo.suggestedCmd;
        this.form.name = fileInfo.suggestedName || fileInfo.filenameWithoutExt;

        // 设置默认分类为第一个分类
        if (this.toolList.length > 0) {
          this.form.category = this.toolList[0].title;
        }

        // 自动生成图标
        if (this.isAutoGenerateIcon) {
          try {
            const iconFileName = await this.autoGenerateIcon(filePath);
            if (iconFileName) {
              this.form.icon = iconFileName;
              console.log('已设置图标:', iconFileName);
            }
          } catch (iconError) {
            console.warn('自动生成图标失败:', iconError);
          }
        }

        // 根据文件类型设置终端选项
        const terminalExtensions = ['jar', 'py', 'bat', 'cmd', 'ps1', 'sh'];
        if (terminalExtensions.includes(fileInfo.extension)) {
          this.form.terminal = 1; // 需要终端
        } else {
          this.form.terminal = 0; // 不需要终端
        }

        // 直接设置对话框模式和显示对话框，不调用 openDialog
        this.dialogMode = "add";
        this.dialogVisible = true;

        console.log('表单填充完成:', this.form);
        this.$message.success(`已自动填充文件信息: ${fileInfo.filename}`);

      } catch (error) {
        console.error('处理拖拽文件失败:', error);
        // this.$message.error('处理文件失败: ' + error.message);
      }
    },

    async analyzeFilePath(filePath) {
      try {
        // 调用后端的 AnalyzeFilePath 方法
        const fileInfo = await AnalyzeFilePath(filePath);
        console.log('后端文件分析结果:', fileInfo);
        return fileInfo;
      } catch (error) {
        console.error('后端分析文件路径失败:', error);
      }
    },

    async autoGenerateIcon(filePath) {
      try {
        const iconFileName = await AutoGenerateIcon(filePath);
        if (iconFileName) {
          console.log('图标生成成功:', iconFileName);
          return iconFileName;
        } else {
          console.log('未生成图标，将使用默认图标');
          return null;
        }
      } catch (error) {
        console.warn('自动生成图标失败:', error);
        // 生成失败时返回 null，前端会使用默认图标
        return null;
      }
    },

    async loadPortConfig() {
      try {
        this.portConfig = await GetPortConfig();
        console.log('端口配置:', this.portConfig);
      } catch (error) {
        console.error('获取端口配置失败:', error);
        // 使用默认端口作为后备
        this.portConfig = {
          mainPort: 52867,
        };
      }
    },
    // 树节点点击
    onTreeNodeClick(node) {
      if (node.id === 'root') return;
      const target = document.querySelector(`.cate[data-index="${node.id}"]`);
      if (target) {
        target.closest('.nav').scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    },

    // 当前是否处于可见状态（限制拖拽触发范围）
    isViewActive() {
      if (!this.$el || !document.body.contains(this.$el)) return false;
      return this.$el.offsetParent !== null;
    },

    // 工具项右键菜单
    showToolContextMenu(event, cmd = null, catIndex = null) {
      event.preventDefault();
      event.stopPropagation();

      this.selectedCmd = cmd; // 仍然设置，以防其他地方使用，但删除操作不再依赖它
      this.selectedIndex = catIndex;
      this.selectedCategory = null;

      this.contextMenuOptions = [
        { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
        { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("toolEdit", cmd) },
        { label: "删除", icon: "el-icon-delete", action: () => this.deleteCmd(cmd) },
        { label: "打开文件位置", icon: "el-icon-folder-opened", action: () => this.openPath(cmd.path) },
        { label: "打开所在目录终端", icon: "el-icon-monitor", action: () => this.openTerminalInDir(cmd.path) },
      ];

      this.showContextMenu(event.clientX, event.clientY);
    },

    // 分类右键菜单
    showCategoryContextMenu(event, category, catIndex) {
      event.preventDefault();
      event.stopPropagation();

      this.selectedIndex = catIndex;
      this.selectedCmd = { category: category.title };
      this.selectedCategory = category;

      this.contextMenuOptions = [
        { label: "修改", icon: "el-icon-edit", action: () => this.openDialog("cateEdit") },
        {
          label: "删除",
          icon: "el-icon-delete",
          action: async () => {
            this.hideContextMenu();
            try {
              await confirmDangerousAction({
                title: '警告',
                message: `是否确认删除分类「${category.title}」？该分类下的所有工具也会被删除。`,
                expectedInput: DANGER_DELETE_TOKEN
              });
              await this.deleteToolCategory(category.title);
            } catch {
              /* 取消 */
            }
          },
        },
      ];

      this.showContextMenu(event.clientX, event.clientY);
    },

    // 树节点右键菜单
    onTreeContextMenu(event, node) {
      event.preventDefault();
      event.stopPropagation();

      // 排除根节点
      if (node.id === 'root') return;

      this.selectedTreeNode = node;
      this.selectedCategory = { title: node.category || node.label };

      // 设置分类树右键菜单选项
      this.contextMenuOptions = [
        {
          label: "重命名",
          icon: "el-icon-edit",
          action: () => this.renameCategory(node)
        },
        {
          label: "删除",
          icon: "el-icon-delete",
          action: () => this.deleteCategory(node)
        }
      ];

      this.showContextMenu(event.clientX, event.clientY);
    },

    // 显示右键菜单
    showContextMenu(x, y) {
      this.contextMenuPosition = { x, y };
      this.contextMenuVisible = true;
    },

    // 隐藏右键菜单
    hideContextMenu() {
      this.contextMenuVisible = false;
      this.selectedCmd = null;
      this.selectedIndex = null;
      this.selectedTreeNode = null;
      this.selectedCategory = null;
    },

    // 重命名分类
    renameCategory(node) {
      this.hideContextMenu();

      const categoryTitle = node.category || node.label;
      this.selectedCategory = categoryTitle;

      // 打开分类编辑对话框
      this.dialogMode = "cateEdit";
      this.form = {
        oldCategory: categoryTitle,
        category: categoryTitle,
      };

      this.dialogVisible = true;
    },

    // 删除分类
    async deleteCategory(node) {
      this.hideContextMenu();

      const categoryTitle = node.category || node.label;

      try {
        await this.$confirm(
          `是否确认删除分类「${categoryTitle}」？该分类下的所有工具也会被删除。`,
          '警告',
          { type: 'warning' }
        );

        await this.deleteToolCategory(categoryTitle);
      } catch (error) {
        // 用户取消删除
        console.log('取消删除分类');
      }
    },

    // 全局右键菜单处理
    handleGlobalContextMenu(event) {
      // 检查是否点击在已有的右键菜单上
      const contextMenuElement = event.target.closest(".context-menu");
      if (contextMenuElement) {
        return;
      }

      event.preventDefault();
      event.stopPropagation();

      // 重置选中状态
      this.selectedCmd = null;
      this.selectedIndex = null;
      this.selectedTreeNode = null;
      this.selectedCategory = null;

      // 全局菜单只显示新增选项
      this.contextMenuOptions = [
        { label: "新增", icon: "el-icon-circle-plus", action: () => this.openDialog("add") },
      ];

      this.showContextMenu(event.clientX, event.clientY);
    },

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
    runCmd(path, cmd, param, terminal) {
      ShellCMD(path, cmd, param, terminal);
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
      console.log("开始解析图标:", icon);

      if (!icon) {
        console.log("图标为空，返回默认图标");
        return "/assets/tool/default.png";
      }

      // 自动生成的图标（在 icon 目录下）
      if (icon.startsWith("auto_") || !icon.includes("/") && !icon.includes("\\")) {
        const remoteUrl = `http://127.0.0.1:${this.portConfig.mainPort}/icon/${icon}`;
        console.log("返回自动生成的图标URL:", remoteUrl);
        return remoteUrl;
      }

      // 本地资源或完整 URL
      if (icon.startsWith("/assets") || /^https?:\/\//.test(icon)) {
        console.log("直接返回本地/网络图标:", icon);
        return icon;
      }

      // 绝对路径处理
      const isWinAbsPath = /^[A-Za-z]:[\\/]/.test(icon);
      const isUnixAbsPath = icon.startsWith("/");

      if (isWinAbsPath || isUnixAbsPath) {
        const normalizedIcon = icon.replace(/\\/g, '/');
        console.log("处理绝对路径:", normalizedIcon);

        try {
          const base64 = await ReadImageAsBase64(normalizedIcon);
          console.log("获取到base64数据:", base64 ? `长度: ${base64.length}` : "空");

          if (base64) {
            const extension = normalizedIcon.split('.').pop()?.toLowerCase() || 'png';
            const mimeType = this.getMimeType(extension);
            const dataUrl = `data:${mimeType};base64,${base64}`;
            console.log("生成Data URL成功");
            return dataUrl;
          }
        } catch (err) {
          console.error("图标加载失败:", err);
        }
        return "/assets/tool/default.png";
      }

      // 其他情况视为相对路径图标
      const remoteUrl = `http://127.0.0.1:${this.portConfig.mainPort}/icon/${icon}`;
      console.log("返回远程路径:", remoteUrl);
      return remoteUrl;
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
      // 更新为默认图标 - 直接赋值
      this.resolvedIcons[cmd.id] = "/assets/tool/default.png";
      // 触发响应式更新
      this.resolvedIcons = {...this.resolvedIcons};
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

    // 在工具路径所在目录打开终端（路径可为可执行文件或目录）
    openTerminalInDir(path) {
      if (!path) {
        this.$message.warning('该工具未设置路径');
        return;
      }
      OpenTerminalInDir(path).then(() => {
        console.log('终端已在目录打开:', path);
      }).catch(err => {
        this.$message.error('打开终端失败: ' + (err?.message || String(err)));
        console.error('打开终端失败:', err);
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

    async browseForCmdFile() {
      try {
        // 检查是否设置了路径
        if (!this.form.path) {
          this.$message.warning('请先设置工具路径');
          return;
        }

        // 使用 Wails 的对话框选择文件，传入路径作为初始目录
        const cmdFile = await GetOpenChooseFilePath(this.form.path);
        if (cmdFile) {
          // 只获取文件名，不包含完整路径
          this.form.cmd = this.extractFileName(cmdFile);
        }
      } catch (err) {
        this.$message.error('选择命令文件失败: ' + err.message);
        console.error('浏览命令文件错误:', err);
      }
    },

    // 从完整路径中提取文件名
    extractFileName(fullPath) {
      // 处理 Windows 路径
      if (fullPath.includes('\\')) {
        return fullPath.split('\\').pop();
      }
      // 处理 Linux/Mac 路径
      return fullPath.split('/').pop();
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

    // 打开弹出框
    openDialog(mode, cmd = null) {
      this.dialogMode = mode;

      if (mode === "toolEdit" && cmd) {
        this.form = { ...cmd };
      } else if (mode === "cateEdit" && this.selectedCategory) {
        const categoryTitle = this.selectedCategory.title;
        this.form = {
          oldCategory: categoryTitle,
          category: categoryTitle,
        };
      } else {
        this.resetForm();
        // 新增工具时默认选择第一个分类
        if (mode === "add" && this.toolList.length > 0) {
          this.form.category = this.toolList[0].title;
        }
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
        name: "",
        param: "",
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
    async deleteCmd(cmd) {
      this.hideContextMenu();

      if (!cmd || !cmd.id) {
        console.error('没有选中的命令或命令ID为空:', cmd);
        this.$message.warning('请选择要删除的命令');
        return;
      }

      try {
        await confirmDangerousAction({
          title: '删除确认',
          message: '确认要删除此命令吗？',
          expectedInput: DANGER_DELETE_TOKEN,
          confirmButtonText: '删除'
        });

        console.log('开始删除命令:', cmd);

        await DeleteTool(cmd.id);
        this.$message.success("删除命令成功");
        console.log("删除命令成功:", cmd);

        // 重新加载数据
        await this.loadToolList();
        await this.loadAllIcons();

      } catch (error) {
        if (error === 'cancel') {
          console.log('用户取消了删除操作');
          return;
        }
        console.error("删除命令失败:", error);
        this.$message.error("删除命令失败");
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

    // 关闭软件中心提示
    closeSoftwareCenterTip() {
      this.showSoftwareCenterTip = false;
    },

    // 跳转到软件中心
    goToSoftwareCenter() {
      this.$router.push({
        name: 'systemManage',
        query: { tab: 'software-center' }
      });
      // 关闭提示
      this.showSoftwareCenterTip = false;
    },
  },

  async mounted() {
    // 随机显示提示（30% 概率）
    const shouldShowTip = Math.random() < 0.3;
    this.showSoftwareCenterTip = shouldShowTip;

    await this.loadPortConfig();
    await this.loadToolList();
    await this.loadAllIcons();
    await this.loadCategories();

    this.$nextTick(() => {
      // 设置文件拖拽事件 - 直接调用处理方法
      OnFileDrop((x, y, paths) => {
        // 仅在当前页面可见时处理
        if (!this.isViewActive()) return;
        console.log('Wails OnFileDrop 收到拖拽文件:', paths);
        console.log('位置:', x, y);

        if (paths && paths.length > 0) {
          // 直接调用 handleFileDrop 方法，传递路径
          const mockEvent = {
            preventDefault: () => {},
            paths: paths // 直接传递 paths
          };
          this.handleFileDrop(mockEvent);
        }
      });
    });

    // 点击其他地方隐藏右键菜单
    document.addEventListener("click", () => {
      this.hideContextMenu();
    });

    // ESC键隐藏右键菜单
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") {
        this.hideContextMenu();
      }
    });
  },

  beforeUnmount() {
    document.removeEventListener("click", this.hideContextMenu);
    document.removeEventListener("keydown", this.hideContextMenu);
  },
};
</script>

<style lang="scss" scoped>
/* 拖拽区域样式 */
.drop-zone-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.7);
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 3px dashed #409EFF;
}

.drop-zone-content {
  text-align: center;
  color: white;
  padding: 40px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.drop-text {
  margin-top: 20px;
  font-size: 18px;
  font-weight: bold;
}

/* 系统元素支持拖拽 */
.system {
  --wails-drop-target: drop;
  min-height: 100vh;
}


/* 确保.system元素覆盖整个可用区域 */
.system {
  margin: 10px 0 10px 10px;
  min-height: calc(100vh - 20px); /* 确保有足够的高度 */
  position: relative; /* 为右键菜单定位提供参考 */
  --wails-drop-target: drop;
}

/* 右上角小目录树 */
.cate-tree {
  position: fixed;
  top: 50px;
  right: 15px;
  width: 140px;
  max-height: 30px; /* 默认收起状态的高度 */
  overflow: hidden; /* 隐藏超出内容 */
  background: rgba(209, 193, 193, 0.55);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.3);
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

.nav {
  .cate {
    margin: 10px 0;
    font-size: 16px;
    color: #ffffff;
    font-weight: 700;
    position: relative;
    padding-right: 60px; /* 为移动按钮留出空间 */

    .category-actions {
      position: absolute;
      left: 120px;
      top: 50%;
      transform: translateY(-50%);
      display: flex;
      gap: 8px;
      color: #ba0f08;

      .move-btn {
        width: 24px;
        height: 24px;
        border-radius: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: #f5f5f5;
        border: 1px solid #ddd;
        cursor: pointer;
        transition: all 0.2s;
        font-size: 12px;

        &:hover {
          background: #409EFF;
          color: white;
          border-color: #409EFF;
        }
      }
    }
  }

  .tool-list {
    display: flex;
    flex-wrap: wrap;

    .item {
      padding: 10px;
      /* background 由 themeBackground.js 动态控制 */
      margin: 10px 10px 0 0;
      width: calc(33.33% - 30px);
      min-height: 40px;
      display: flex;
      align-items: center;
      cursor: pointer;
      border-radius: 8px;
      transition: all 0.2s;
      position: relative;

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
          font-weight: 700;
        }

        .remark {
          height: 38px;          /* 增加高度以容纳更大的行间距 */
          display: -webkit-box;
          -webkit-line-clamp: 2;
          line-clamp: 2;         /* 标准属性，提高兼容性 */
          -webkit-box-orient: vertical;
          color: #42404f;
          font-size: 12px;
          line-height: 18px;     /* 新增：设置行间距为20px */
          overflow: hidden;      /* 确保超出部分被隐藏 */
        }
      }

      .tool-actions {
        position: absolute;
        right: 10px;
        top: 50%;
        transform: translateY(-50%);
        display: flex;
        gap: 8px;
        color: #ba0f08;

        .move-btn {
          width: 24px;
          height: 24px;
          border-radius: 4px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: #f5f5f5;
          border: 1px solid #ddd;
          cursor: pointer;
          transition: all 0.2s;
          font-size: 12px;

          &:hover {
            background: #409EFF;
            color: white;
            border-color: #409EFF;
          }
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
  position: fixed;
  z-index: 9999;
  background-color: #ffffff;
  border: 1px solid #e6e6e6;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border-radius: 8px;
  list-style: none;
  padding: 8px 0;
  margin: 0;
  width: 150px;
  overflow: visible;

  li {
    padding: 10px 16px;
    font-size: 14px;
    color: #333333;
    display: flex;
    align-items: center;
    cursor: pointer;
    transition: background-color 0.2s, color 0.2s;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;

    i {
      margin-right: 8px;
      font-size: 16px;
      color: #666666;
      flex-shrink: 0;
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

/* 软件中心提示卡片 - 参考搜索框样式 */
.software-center-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(0, 0, 0, 0.63);
  border-radius: 15px;
  border: 1px solid transparent;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  padding: 8px 16px;
  user-select: none;
  max-width: 450px;
  transition: all 0.3s ease;

  .tip-icon {
    font-size: 20px;
    color: #ffffff;
    flex-shrink: 0;
    animation: tipPulse 2s ease-in-out infinite;
  }

  .tip-text {
    font-size: 13px;
    color: #ffffff;
    line-height: 1.5;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }

  .tip-jump-btn {
    flex-shrink: 0;
    padding: 4px 12px;
    height: 28px;
    font-size: 12px;
    color: #409EFF;
    background: rgba(64, 158, 255, 0.1);
    border: 1px solid rgba(64, 158, 255, 0.3);
    border-radius: 6px;
    transition: all 0.2s;

    &:hover {
      background: rgba(64, 158, 255, 0.2);
      border-color: rgba(64, 158, 255, 0.5);
      color: #66B1FF;
    }
  }

  .tip-close-btn {
    flex-shrink: 0;
    padding: 4px;
    color: rgba(255, 255, 255, 0.7);
    transition: all 0.2s;

    &:hover {
      color: #ffffff;
      transform: scale(1.1);
    }

    :deep(.el-icon) {
      font-size: 16px;
    }
  }

  &:hover {
    background: rgba(0, 0, 0, 0.75);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
  }
}

/* 提示卡片淡入动画 */
.fade-enter-active {
  animation: fadeIn 0.3s ease-out;
}

.fade-leave-active {
  animation: fadeIn 0.3s ease-out reverse;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes tipPulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}
</style>