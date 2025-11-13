<template>
  <el-container class="app-container"  direction="vertical">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="tabs" :key="tabsKey">
      <el-tab-pane
          v-for="tab in visibleTabs"
          :key="tab.name"
          :label="tab.title"
          :name="tab.name"
      />
    </el-tabs>

    <!-- 内容区域 -->
    <el-main :class="['content', { 'shell-syntax-active': activeTab === 'shell-syntax' }]">
      <!-- 默认密码查询 -->
      <div v-if="activeTab === 'password-query'" class="tab-content">
        <div class="header">
          <div class="search-bar">
            <el-input placeholder="请输入查询条件" v-model="queryPasswordInput" class="input" />
            <el-button type="primary" @click="handleSearch">查询</el-button>
          </div>
        </div>
        <!-- 修改点1：添加行高和行距设置 -->
        <el-table
            :data="passwordData"
            border
            class="custom-table uniform-rows"
            :row-style="{ height: '45px' }"
            :cell-style="{ padding: '8px 0', lineHeight: '1.5' }"
            @cell-click="handleCopeClick">

          <!-- 修改点2：添加溢出提示 -->
          <el-table-column prop="name" label="Name" :min-width="190" show-overflow-tooltip />
          <el-table-column prop="method" label="Method" :min-width="190" show-overflow-tooltip />
          <el-table-column prop="userId" label="User ID" :min-width="190" show-overflow-tooltip />
          <el-table-column prop="password" label="PassWord" :min-width="190" show-overflow-tooltip />
          <el-table-column prop="level" label="Level" :min-width="190" show-overflow-tooltip />
        </el-table>
        <br>

        <!-- 修改点3：统一分页按钮间距 -->
        <el-pagination
            ref="pagination"
            :current-page="currentPage"
            :page-size="pageSize"
            :total="total"
            @current-change="handlePageChange"
            layout="prev, pager, next, jumper, total"
            :pager-count="5"
            class="uniform-pagination"
            background>
          <template #prev>
            <el-button size="small" type="primary" class="pagination-btn">Previous</el-button>
          </template>
          <template #next>
            <el-button size="small" type="primary" class="pagination-btn">Next</el-button>
          </template>
        </el-pagination>
      </div>

      <!-- 杀软进程查询 -->
      <div v-if="activeTab === 'process-query'" class="tab-content">
        <div class="header">
          <h4>杀软识别 tasklist /svc</h4>
          <el-input type="textarea" v-model="tasklistInput" placeholder="请输入 tasklist /svc 输出内容" :rows="10"
                    class="process_textarea"></el-input>
          <el-button type="primary" @click="handleAVQuery">查询</el-button>
        </div>
        <div class="result" v-if="isAvQueried">
          <h4>识别结果：</h4>
          <!-- 如果有结果，显示表格 -->
          <el-table v-if="avResults.length > 0" :data="avResults" border style="width: 100%">
            <el-table-column prop="program" label="程序名" />
            <el-table-column prop="match" label="杀软匹配" />
            <el-table-column prop="description" label="描述" />
          </el-table>
          <!-- 如果没有结果，显示提示文字 -->
          <p v-else class="no-results">未识别到任何杀软进程。</p>
        </div>
      </div>

      <!-- Google语法查询 -->
      <div v-if="activeTab === 'google-syntax'" class="tab-content">
        <div class="google-syntax-container">
          <h4>Google Hacking 语法查询</h4>
          <div class="form-group">
            <el-input v-model="googleDomainInput" placeholder="请输入域名或IP" class="input" />
            <el-button type="primary" @click="generateGoogleQueries">生成语法</el-button>
          </div>

          <div v-if="googleQueries.length > 0" class="query-results">
            <h4>查询语法：</h4>
            <el-collapse v-model="activeCategories">
              <el-collapse-item v-for="category in googleQueries" :key="category.category"
                                :title="category.category" :name="category.category">
                <el-table :data="category.commands" border style="width: 100%">
                  <el-table-column class="query-item" prop="description" label="说明"/>
                  <el-table-column class="query-item" prop="command" label="语法">
                    <template #default="scope">
                      <div class="query-item">
                        <span>{{ scope.row.command }}</span>
                        <el-button type="text"
                                   @click="CopyToClipboard(scope.row.command)">复制</el-button>
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>
      </div>

      <!-- 反弹shell查询 -->
      <div v-if="activeTab === 'shell-syntax'" class="tab-content">
        <!-- 嵌入 public/shell/index.html -->
        <div class="iframe-container">
          <iframe src="/Shell/index.html"></iframe>
        </div>
      </div>

      <!-- 地图测试 -->
      <div v-if="activeTab === 'map-query'" class="tab-content">
        <el-card shadow="always" class="map-query-card">
          <div class="map-query-container">
            <!-- 左侧：选择与参数输入 -->
            <div class="map-form">
              <el-form :model="mapForm" label-width="100px" class="inner-form">
                <el-form-item label="地图服务">
                  <el-select v-model="mapForm.provider" placeholder="请选择" @change="resetMapResult">
                    <el-option
                        v-for="item in mapProviders"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value"
                    />
                  </el-select>
                </el-form-item>

                <!-- 公共 Key 输入 -->
                <el-form-item label="API Key">
                  <el-input v-model="mapForm.key" placeholder="输入你的 key" />
                </el-form-item>

                <!-- 动态参数区域 -->
                <template v-if="mapForm.provider === 'amap-walking'">
                  <el-form-item label="起点坐标">
                    <el-input v-model="mapForm.origin" placeholder="lng,lat(如：116.434307,39.90909)" />
                  </el-form-item>
                  <el-form-item label="终点坐标">
                    <el-input v-model="mapForm.destination" placeholder="lng,lat(如：116.434446,39.90816)" />
                  </el-form-item>
                </template>

                <template v-else-if="mapForm.provider === 'amap-geocode'">
                  <el-form-item label="坐标">
                    <el-input v-model="mapForm.location" placeholder="lng,lat(如：116.434446,39.90816)" />
                  </el-form-item>
                </template>

                <template v-else-if="mapForm.provider === 'amap-mini'">
                  <el-form-item label="坐标">
                    <el-input v-model="mapForm.location" placeholder="lng,lat(如：116.434446,39.90816)" />
                  </el-form-item>
                  <el-form-item label="AppName">
                    <el-input v-model="mapForm.amapMiniAppID" placeholder="如：c589cf63f592ac13bcab35f8cd18f495" />
                  </el-form-item>
                </template>

                <template v-else-if="mapForm.provider === 'baidu-web' || mapForm.provider === 'baidu-ios'">
                  <el-form-item label="关键字">
                    <el-input v-model="mapForm.query" placeholder="如：ATM机" />
                  </el-form-item>
                  <el-form-item label="关键字">
                    <el-input v-model="mapForm.baiduTag" placeholder="如：银行" />
                  </el-form-item>
                  <el-form-item label="地区">
                    <el-input v-model="mapForm.region" placeholder="如：北京" />
                  </el-form-item>
                </template>

                <template v-else-if="mapForm.provider === 'qq-web'">
                  <el-form-item label="关键词">
                    <el-input v-model="mapForm.keyword" placeholder="如：酒店" />
                  </el-form-item>
                  <el-form-item label="中心坐标">
                    <el-input v-model="mapForm.center" placeholder="lat,lng(如：39.908491,116.374328)" />
                  </el-form-item>
                  <el-form-item label="半径 (m)">
                    <el-input v-model="mapForm.radius" placeholder="1000" />
                  </el-form-item>
                </template>

                <el-form-item>
                  <el-button
                      type="primary"
                      @click="doMapQuery"
                      :loading="mapLoading"
                      class="submit-btn"
                  >
                    发起请求
                  </el-button>
                </el-form-item>
              </el-form>
            </div>

            <!-- 右侧：结果展示 -->
            <div class="map-result">
              <el-card shadow="hover" class="result-card">
                <div class="result-header">
                  <i class="el-icon-document"></i>
                  <span>返回结果</span>
                </div>
                <div class="result-content" v-if="mapResult">
                  <pre>{{ mapResult }}</pre>
                </div>
                <div v-else class="no-result">
                  暂无数据，请先发起请求
                </div>
              </el-card>
            </div>
          </div>
        </el-card>
      </div>

      <!-- Fscan结果处理 -->
      <div v-if="activeTab === 'fscan-deal'" class="tab-content">
        <div class="fscan-content">
          <div class="fscan-header-card">
            <h4>选择 Fscan 结果文件</h4>
            <el-upload class="upload-demo" drag action="" :before-upload="beforeUpload" :file-list="fileList"
                       :show-file-list="false" :on-change="handleFileChange">
              <i class="el-icon-upload"></i>
              <div class="el-upload__text">
                拖拽文件到此处，或<em>点击上传</em>
              </div>
              <div class="el-upload__tip">仅支持 .txt 文件</div>
            </el-upload>
          </div>

          <!-- 操作按钮 -->
          <div class="actions" v-if="fileName">
            <el-button type="primary" @click="processFscanFile">处理文件</el-button>
            <el-button type="success" @click="openExcelPath" v-if="isDealFile">
              打开文件夹
            </el-button>
            <p v-if="isDealFile">文件保存位置：{{ excelFilePath }}</p>
          </div>
          
        </div>
      </div>

      <!-- 蓝队大批量IP封禁处置 -->
      <div v-if="activeTab === 'ip-ban-deal'" class="tab-content">
        <div class="ip-ban-content">
          <div class="ip-ban-info-card">
            <p class="info">
              请填写威胁情报 & 恶意IP列表（每行一个IP），以及选填白名单IP列表。系统将自动去重，并排除白名单内容，避免误封。
            </p>
          </div>

          <div class="ip-ban-input-card">
            <!-- 上半部分：输入框 -->
            <div class="row upper-row">
              <div class="column">
                <h4>威胁情报 & 恶意IP</h4>
                <el-input type="textarea" v-model="maliciousIPInput" placeholder="请输入威胁情报 & 恶意IP (每行一个)"
                          :rows="10" class="input-box" />
              </div>
              <div class="column">
                <h4>IP白名单</h4>
                <el-input type="textarea" v-model="whiteListIPInput" placeholder="请输入IP白名单 (每行一个)" :rows="10"
                          class="input-box" />
              </div>
            </div>

            <!-- 下半部分：结果框 -->
            <div class="row lower-row">
              <div class="column">
                <h4>去重后IP (排除白名单)</h4>
                <div class="ip-input-container">
                  <el-input type="textarea" :value="uniqueIPs.join('\n')" readonly placeholder="去重后IP (排除白名单)"
                            :rows="8" class="input-box readonly" />
                  <el-button type="success" class="ip-copy-button" @click="IpCopyToClipboard('uniqueIPs')">
                    复制
                  </el-button>
                </div>
              </div>
              <div class="column">
                <h4>重复IP</h4>
                <div class="ip-input-container">
                  <el-input type="textarea" :value="duplicateIPs.join('\n')" readonly placeholder="重复的IP"
                            :rows="8" class="input-box readonly" />
                  <el-button type="success" class="ip-copy-button" @click="IpCopyToClipboard('duplicateIPs')">
                    复制
                  </el-button>
                </div>
              </div>
            </div>

            <!-- 去重按钮 -->
            <el-button type="primary" @click="processIPs" class="process-button">
              去重
            </el-button>
          </div>
        </div>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import { QueryAntivirusProcesses, QueryGoogleQueries, QueryPasswordsAPI } from "../../wailsjs/go/controller/Assistive"
import {FscanResultDeal, UploadFile} from "../../wailsjs/go/controller/Assistive";
import axios from 'axios'
import {GetConfigDir, OpenPath} from "../../wailsjs/go/system/System";
import { loadMenuOrder, moduleTabsConfig } from '@/utils/menuConfig';
import {ElMessage} from "element-plus";

export default {
  name: "AssistiveView",
  data() {
    return {
      activeTab: "", // 初始为空，等配置加载后设置
      queryPasswordInput: "",
      moduleTabs: [], // 模块标签页配置
      tabsKey: Date.now(),

      // 密码查询
      passwordData: [],
      total: 0,
      currentPage: 1,
      pageSize: 12,

      // fscan处理
      fileName: "", // 当前上传的文件名
      fileList: [], // 上传的文件列表
      excelData: [], // Excel 数据，用于渲染预览
      isDealFile: false,
      excelFilePath: "", // 后端生成的 Excel 文件路径
      sheetsData: [], // 存储所有表的数据
      activeSheet: "", // 当前显示的表
      maliciousIPInput: "",
      whiteListIPInput: "",
      uniqueIPs: [],
      duplicateIPs: [],
      paginatedData: [], // 分页后的数据

      // tasklist 查询
      tasklistInput: "",
      avResults: [],
      isAvQueried: false,

      // google 语法查询
      googleDomainInput: '',
      googleQueries: [],
      activeCategories: [],

      // 地图测试相关
      mapProviders: [
        { label: '高德 Walking', value: 'amap-walking' },
        { label: '高德 Geocode(JSAPI)', value: 'amap-geocode' },
        { label: '高德 小程序 RE Geocode', value: 'amap-mini' },
        { label: '百度 Web 搜索', value: 'baidu-web' },
        { label: '百度 iOS 搜索', value: 'baidu-ios' },
        { label: '腾讯 Web 搜索', value: 'qq-web' },
      ],
      mapForm: {
        provider: '',
        key: '',
        origin: '116.434307,39.90909',
        destination: '116.434446,39.90816',
        location: '116.434446,39.90816',
        query: 'ATM机',
        baiduTag:"银行",
        region: '北京',
        keyword: '酒店',
        center: '39.908491,116.374328',
        radius: '1000',
        amapMiniAppID:"",
      },
      mapResult: null,
      mapLoading: false,

      // 添加配置加载状态
      configLoaded: false
    };
  },

  computed: {
    // 计算可见的标签页 - 模仿 menu.vue 的实现
    visibleTabs() {
      if (!this.moduleTabs || this.moduleTabs.length === 0) {
        return [];
      }

      // 先过滤可见的，再排序
      const visible = this.moduleTabs.filter(tab => tab.visible !== false);
      return visible.sort((a, b) => (a.order || 0) - (b.order || 0));
    }
  },

  watch: {
    // 监听 visibleTabs 变化，确保 activeTab 始终有效
    visibleTabs: {
      handler(newTabs) {
        if (newTabs.length > 0 && this.configLoaded) {
          this.ensureActiveTabValid();
        }
      },
      immediate: false
    }
  },

  async mounted() {
    // 先加载标签页配置，再加载数据
    await this.loadTabsConfig();
    await this.fetchPasswords(false);
    this.configLoaded = true;

    // 监听菜单更新事件
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
  },

  beforeUnmount() {
    window.removeEventListener('menu-order-updated', this.handleMenuOrderUpdated);
  },

  methods: {
    // 处理菜单顺序更新事件
    async handleMenuOrderUpdated(event) {
      // console.log('收到菜单配置更新事件:', event.detail);

      // 重新加载标签页配置
      await this.loadTabsConfig();

      // 强制重新渲染 el-tabs
      this.forceTabsRerender();
    },

    // 强制重新渲染 el-tabs 组件
    forceTabsRerender() {
      // 方法1: 使用 key 强制重新渲染
      this.tabsKey = Date.now();

      // 方法2: 临时改变 activeTab 再改回来
      const currentActiveTab = this.activeTab;
      this.activeTab = '';

      this.$nextTick(() => {
        this.activeTab = currentActiveTab;
      });
    },

    // 加载标签页配置
    async loadTabsConfig() {
      try {
        const savedData = await loadMenuOrder();
        // console.log('加载的标签页数据:', savedData);

        const savedTabsOrder = savedData.tabs || {};
        const assistiveTabs = savedTabsOrder.assistive || [];

        // 获取默认配置
        const defaultTabs = moduleTabsConfig.assistive || [];

        // 创建标签页映射
        const tabMap = {};
        defaultTabs.forEach(tab => {
          tabMap[tab.name] = {
            ...tab,
            order: tab.defaultOrder,
            visible: tab.visible
          };
        });

        // 应用保存的配置
        assistiveTabs.forEach(savedTab => {
          if (tabMap[savedTab.name]) {
            tabMap[savedTab.name].order = savedTab.order;
            tabMap[savedTab.name].visible = savedTab.visible;
          }
        });

        // 转换为数组并排序
        this.moduleTabs = Object.values(tabMap)
            .sort((a, b) => (a.order || 0) - (b.order || 0));

        console.log('处理后的标签页:', this.moduleTabs);

        // 设置默认激活的标签页
        this.setDefaultActiveTab();

      } catch (error) {
        console.error('加载标签页配置失败:', error);
        this.setDefaultTabs();
      }
    },

    // 确保当前激活的标签页有效
    ensureActiveTabValid() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length === 0) {
        this.activeTab = '';
        return;
      }

      // 检查当前激活的标签页是否在可见标签页中
      const currentTabExists = visibleTabs.some(tab => tab.name === this.activeTab);
      if (!currentTabExists) {
        // 如果当前激活标签不存在，设置为第一个可见标签
        this.activeTab = visibleTabs[0].name;
      }
    },

    setDefaultActiveTab() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length > 0) {
        // 如果当前没有激活标签或者当前激活标签不可见，则设置为第一个可见标签
        if (!this.activeTab || !visibleTabs.some(tab => tab.name === this.activeTab)) {
          this.activeTab = visibleTabs[0].name;
        }
      } else {
        // 如果没有可见标签页，设置一个默认值
        this.activeTab = '';
      }
    },

    setDefaultTabs() {
      // 直接使用默认配置
      this.moduleTabs = moduleTabsConfig.assistive.map(tab => ({
        ...tab,
        order: tab.defaultOrder
      }));
      this.setDefaultActiveTab();
    },

    handleSearch() {
      this.currentPage = 1;
      this.fetchPasswords(true);
    },

    async fetchPasswords(isSearch = false) {
      const query = isSearch ? this.queryPasswordInput.trim() : "";
      this.loading = true;

      try {
        let response = await QueryPasswordsAPI(this.currentPage, this.pageSize, query);

        if (!response || !response.data || typeof response.total !== 'number') {
          this.$message.error("查询结果格式不正确！");
          return;
        }

        this.passwordData = response.data || [];
        this.total = response.total;

        if (this.passwordData.length === 0 && this.currentPage > 1) {
          this.currentPage = 1;
          await this.$nextTick(() => {
            this.fetchPasswords(isSearch);
          });
        }
      } catch (err) {
        console.error("查询错误：", err);
        this.$message.error(`查询失败: ${err.message || "请重试！"}`);
      } finally {
        this.loading = false;
      }
    },

    handlePageChange(newPage) {
      this.currentPage = newPage;
      this.fetchPasswords(!!this.queryPasswordInput.trim());
    },

    async handleCopeClick(row, column, cell, event) {
      const textToCopy = cell.innerText;
      try {
        await navigator.clipboard.writeText(textToCopy);
        this.$message.success(`已复制: ${textToCopy}`);
      } catch (error) {
        this.$message.error("复制失败，请重试");
      }
    },

    async handleAVQuery() {
      if (!this.tasklistInput.trim()) {
        this.$message.error("请输入有效的 tasklist /svc 输出内容！");
        this.avResults = [];
        this.isAvQueried = false;
        return;
      }

      try {
        const response = await QueryAntivirusProcesses(this.tasklistInput);

        this.avResults = response.map(item => ({
          program: item.program,
          match: item.match,
          description: item.description
        }));

        this.isAvQueried = true;
      } catch (err) {
        this.isAvQueried = true;
      }
    },

    async generateGoogleQueries() {
      if (!this.googleDomainInput.trim()) {
        this.$message.warning('请输入域名或IP！');
        return;
      }

      const googleDomain = this.googleDomainInput.trim();
      try {
        let response = await QueryGoogleQueries(googleDomain);

        if (!Array.isArray(response)) {
          response = [response];
        }

        this.googleQueries = response.map(item => ({
          category: item.category,
          commands: [{
            id: "google1",
            description: item.description,
            command: item.command
          }]
        }));

        this.activeCategories = this.googleQueries.map((query) => query.category);

      } catch (err) {
        this.$message.error("查询失败，请重试！");
      }
    },

    CopyToClipboard(query) {
      navigator.clipboard.writeText(query).then(() => {
        this.$message.success('复制成功！');
      }).catch(() => {
        this.$message.error('复制失败！');
      });
    },

    resetMapResult() {
      this.mapResult = null;
    },

    async doMapQuery() {
      if (!this.mapForm.provider || !this.mapForm.key) {
        this.$message.warning('请先选择服务并填写 API Key');
        return;
      }
      this.mapLoading = true;
      const k = encodeURIComponent(this.mapForm.key);
      try {
        switch (this.mapForm.provider) {
          case 'amap-walking':
            this.mapResult = (
                await axios.get(
                    `https://restapi.amap.com/v3/direction/walking?origin=${this.mapForm.origin}&destination=${this.mapForm.destination}&key=${k}`
                )
            ).data;
            break;

          case 'amap-geocode':
            this.mapResult = (
                await axios.get(
                    `https://restapi.amap.com/v3/geocode/regeo?key=${k}&s=rsv3&location=${this.mapForm.location}&callback=jsonp_258885_&platform=JS`
                )
            ).data;
            break;

          case 'amap-mini':
            this.mapResult = (
                await axios.get(
                    `https://restapi.amap.com/v3/geocode/regeo?key=${k}&location=${encodeURIComponent(
                        this.mapForm.location
                    )}&extensions=all&platform=WXJS&appname=${this.mapForm.amapMiniAppID}&sdkversion=1.2.0&logversion=2.0`
                )
            ).data;
            break;

          case 'baidu-web':
          case 'baidu-ios':
          {
            const cbName = `jsonp_cb_${Date.now()}`;
            let url = `https://api.map.baidu.com/place/v2/search?query=${encodeURIComponent(
                this.mapForm.query
            )}&tag=${this.mapForm.baiduTag}&region=${encodeURIComponent(
                this.mapForm.region
            )}&output=json&ak=${k}&callback=${cbName}`;
            if (this.mapForm.provider === 'baidu-ios') {
              url += '&mcode=com.didapinche.taxi&os=12.5.6';
            }
            await this.jsonp(url, cbName, (data) => {
              this.mapResult = data;
              this.mapLoading = false;
            });
          }
            return;

          case 'qq-web':
          {
            const cbName = `jsonp_qq_${Date.now()}`;
            const url = `https://apis.map.qq.com/ws/place/v1/search?keyword=${encodeURIComponent(
                this.mapForm.keyword
            )}&boundary=nearby(${this.mapForm.center},${this.mapForm.radius})&key=${k}&output=jsonp&callback=${cbName}`;
            await this.jsonp(url, cbName, (data) => {
              this.mapResult = data;
              this.mapLoading = false;
            });
          }
            return;
        }
      } catch (err) {
        this.mapResult = err.toString();
      } finally {
        this.mapLoading = false;
      }
    },

    jsonp(url, cbName, onSuccess) {
      window[cbName] = (res) => {
        onSuccess(res);
        delete window[cbName];
        document.head.removeChild(script);
      };
      const script = document.createElement('script');
      script.src = url;
      document.head.appendChild(script);
    },

    // fscan解析
    beforeUpload(file) {
      const isTxt = file.type === "text/plain";
      if (!isTxt) {
        ElMessage.error("仅支持上传 .txt 文件！");
        return false; // 阻止上传
      }
      this.fileName = file.name; // 保存文件名
      return true; // 允许上传
    },
    async handleFileChange(file) {
      if (file.raw.type !== "text/plain") return;
      const reader = new FileReader();
      reader.onload = async (event) => {
        this.fileContent = event.target.result;

        try {
          await UploadFile(this.fileName, this.fileContent);
          ElMessage.success("文件上传成功！");
        } catch (error) {
          console.error("文件上传失败:", error);
          ElMessage.error(error.message || "文件上传失败！");
        }
      };
      reader.onerror = () => ElMessage.error("文件读取失败");
      reader.readAsText(file.raw);
    },
    async processFscanFile() {
      try {
        const result = await FscanResultDeal(this.fileName);
        this.excelFilePath = result; // 保存路径
        this.isDealFile = true;
        ElMessage.success(`文件处理成功！生成的文件: ${result}`);
      } catch (error) {
        console.error("文件处理失败:", error);
        ElMessage.error(error.message || "处理文件失败！");
      }
    },
    // 打开file文件夹
    async openExcelPath() {
      const baseDir = await GetConfigDir()
      const fileDir = baseDir + "/file"; // 拼接file子目录
      await OpenPath(fileDir)
    },

    // 蓝队封禁IP
    processIPs() {
      const maliciousIPs = this.maliciousIPInput
          .split("\n")
          .map(ip => ip.trim())
          .filter(ip => ip);
      const whiteListIPs = new Set(
          this.whiteListIPInput
              .split("\n")
              .map(ip => ip.trim())
              .filter(ip => ip)
      );

      const uniqueIPs = new Set();
      const duplicateIPs = new Set();

      maliciousIPs.forEach(ip => {
        if (whiteListIPs.has(ip)) return;
        if (uniqueIPs.has(ip)) {
          duplicateIPs.add(ip);
        } else {
          uniqueIPs.add(ip);
        }
      });

      this.uniqueIPs = Array.from(uniqueIPs);
      this.duplicateIPs = Array.from(duplicateIPs);
    },
    IpCopyToClipboard(type) {
      const list = this[type];
      const text = list.join("\n");
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success("IP 列表已复制到剪贴板！");
      });
    },

  }
};
</script>

<style scoped>
/* 页面容器 */
.app-container {
  height: 95vh;
  display: flex;
  padding: 5px;
  flex-direction: column;
  background-color: #f8f9fb;
}

/* 顶部 Tabs 样式 */
.tabs {
  background-color: #ffffff;
  box-shadow: 0px 2px 2px rgba(0, 0, 0, 0.1);
  margin-bottom: 10px;
  /* border-bottom: 2px solid #ebeef5; */
  padding-left: 10px;
  /* 增加左边距 */
  border-radius: 10px 10px 10px 10px;
}

/* 内容区域 */
.content {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  background-color: #ffffff;
  box-shadow: 0px 0px 10px rgba(0, 0, 0, 0.05);
  border-radius: 8px;
  box-sizing: border-box;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}

.input {
  flex: 1;
}

/* 表格 */
.custom-table {
  flex-grow: 1;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  overflow: hidden;
  min-height: 550px;
}

.custom-table .el-table__row:hover {
  background-color: #f5f7fa !important;
}

/* 自定义分页控件按钮颜色 */
.el-pagination .el-pager li {
  color: #409EFF;
}

.el-pagination .el-pager li:hover {
  background-color: #f4f4f5;
}

.el-pagination .el-button--primary {
  background-color: #409EFF;
  border-color: #409EFF;
  color: white;
}

/* 控制分页按钮的大小 */
.el-pagination button {
  padding: 5px 15px;
}

/* 添加分隔线 */
.el-pagination .el-pager li:not(:last-child) {
  border-right: 1px solid #ddd;
}


/* 杀软查询输入框 */
.process_textarea {
  margin-bottom: 15px;
}

/* 精确定位 shell-syntax 的 tab-content */
.shell-syntax-active {
  padding: 0px !important;
}

/* iframe 容器样式 */
.iframe-container {
  /* margin: 0 auto; */
  max-width: 100%;
  /* 控制 iframe 的最大宽度 */
  overflow: hidden;
  border-radius: 8px;
  /* 圆角处理 */
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  /* 增加阴影效果 */
}

/* iframe 样式 */
.iframe-container iframe {
  width: 100%;
  /* 保持 iframe 宽度填充父容器 */
  height: 90vh;
  /* 高度占视口的 80% */
  border: none;
  /* 去掉默认边框 */
  border-radius: 8px;
  /* 与父容器一致的圆角 */
}

:deep(.el-textarea__inner) {
  padding-top: 15px;
  /* 增加顶部内边距，将文字往下移 */
  padding-bottom: 10px;
  /* 保持上下内边距平衡 */
  line-height: 1.5;
  /* 设置合适的行高 */
}

/* google语法 */
.google-syntax-container {
  padding: 0;
}

.form-group {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.input {
  margin-right: 10px;
  flex: 1;
}

.query-results {
  margin-top: 20px;
  color: #444744;
}

.query-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  word-wrap: break-word;
  /* 自动换行 */

  overflow-wrap: break-word;
  /* 防止溢出 */
  color: #5cb85c;

}

.query-item span {
  max-width: 90%;
  /* 限制命令显示的最大宽度 */
  overflow: hidden;
  text-overflow: ellipsis;
  /* 超出部分用省略号显示 */
  word-wrap: break-word;

}

.query-item el-button {
  margin-left: 10px;
}



/* 地图测试 样式 */
.el-card{

}

.map-query-card {
  width: auto;

}
.map-query-container {
  display: flex;
  gap: 20px;
}
.map-form {
  flex: 1;
}
.inner-form {
  background: #ffffff;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}
.submit-btn {
  width: 100%;
}
.map-result {
  flex: 1;
}
.result-card {
  height: 100%;
  padding: 0;
}
.result-header {
  display: flex;
  align-items: center;
  background: #f0f9ff;
  padding: 12px 20px;
  font-weight: 500;
  border-bottom: 1px solid #e4e7ed;
}
.result-header i {
  margin-right: 8px;
  font-size: 16px;
  color: #409EFF;
}
.result-content {
  padding: 20px;
  max-height: 500px;
  overflow: auto;
  background: #fafafa;
}
.result-content pre {
  white-space: pre-wrap;
}
.no-result {
  padding: 40px;
  text-align: center;
  color: #909399;
}

/* 主内容区域 */
.el-main {
  padding: 0;
  flex: 1;
  min-height: 0;
  overflow: auto;
}

:deep(.el-main) {
  padding: 0 !important;
}

/* 标签页内容通用样式 */
.tab-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
  padding: 5px 8px;
}

/* 通用卡片样式 */
.tab-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 200px;
}

.tab-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tab-card-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  min-height: 0;
}

/* 通用滚动条样式 */
.tab-card-content::-webkit-scrollbar {
  width: 6px;
}

.tab-card-content::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.tab-card-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.tab-card-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 文本样式 */
:deep(h4){
  display: block;
  margin-block-start: 1.33em;
  margin-block-end: 1.33em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(h5){
  display: block;
  font-size: 0.83em;
  margin-block-start: 1.67em;
  margin-block-end: 1.67em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(p){
  display: block;
  margin-block-start: 1em;
  margin-block-end: 1em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  unicode-bidi: isolate;
}

/* 使用 flex 布局确保按钮和文件路径显示在同一行 */
.actions {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

/* 表格样式 */
.excel-preview-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 20px;
}

.excel-preview-table th, .excel-preview-table td {
  border: 1px solid #ccc;
  padding: 8px;
  text-align: left;
  font-size: 14px;
}

.excel-preview-table th {
  background-color: #f0f0f0;
  font-weight: bold;
}

.excel-preview-table tr:nth-child(even) {
  background-color: #f9f9f9;
}

.excel-preview-table tr:hover {
  background-color: #f1f1f1;
}

.excel-preview-table td {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 蓝队封禁IP工具 */
.tab-content .info {
  color: #409eff;
}

.row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.column {
  width: 48%;
  box-sizing: border-box;
  min-width: 300px;
  margin-bottom: 10px;
}

.upper-row .column {
  padding: 0 1px;
}

.lower-row .column {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 0 10px;
}

/* 包裹 el-input 和按钮的容器 */
.ip-input-container {
  position: relative;
}

/* 输入框 */
.input-box {
  width: 100%;
}

/* 按钮定位到输入框右上角 */
.ip-copy-button {
  position: absolute;
  top: 5px;
  right: 5px;
  z-index: 10;
  padding: 5px 10px;
  font-size: 12px;
  background-color: #00aaff;
  color: #ffffff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.ip-copy-button:hover {
  background-color: #176bca;
}

/* 按钮优化 */
.process-button {
  display: block;
  padding: 8px 16px;
  margin: 25px auto 10px;
  border-radius: 5px;
  width: 100%;
  font-size: 18px;
  text-align: center;
  background-color: #5cb85c;
}

</style>