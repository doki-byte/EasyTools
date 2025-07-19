<template>
    <el-container class="container">
        <!-- 标签栏 -->
        <el-tabs v-model="activeTab" class="tabs">
            <el-tab-pane label="Google语法" name="google-syntax" />
            <el-tab-pane label="默认密码查询" name="password-query" />
            <el-tab-pane label="反弹Shell" name="shell-syntax" />
            <el-tab-pane label="杀软进程查询" name="process-query" />
            <el-tab-pane label="地图测试" name="map-query"></el-tab-pane>
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
                    <h5>识别结果：</h5>
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
        </el-main>
    </el-container>
</template>


<script>
import { QueryAntivirusProcesses, QueryGoogleQueries, QueryPasswordsAPI } from "../../wailsjs/go/controller/InfoSearch"
import axios from 'axios'
export default {
    name: "InfoSearchView",
    data() {
        return {
            activeTab: "google-syntax",
            queryPasswordInput: "", // 查询输入框的绑定数据
            tabTitles: {
                "password-query": "默认密码查询",
                "process-query": "杀软进程查询",
                "google-syntax": "Google语法",
                "shell-syntax": "反弹Shell",
            },
            passwordData: [],
            total: 0,
            currentPage: 1,
            pageSize: 10,

            tasklistInput: "", // 用户输入的 tasklist 内容
            avResults: [], // 查询结果
            isAvQueried: false, // 是否已进行查询

            googleDomainInput: '',
            googleQueries: [],
            activeCategories: [], // 默认展开的分类

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
        };
    },

    mounted() {
        // 页面加载时自动加载第一页数据
        this.fetchPasswords(false);
        // 禁用右键菜单
        document.addEventListener('contextmenu', (event) => {
            event.preventDefault();
        });
    },
    beforeUnmount() {
        // 在组件卸载时移除事件监听器，避免潜在的内存泄漏
        document.removeEventListener('contextmenu', (event) => {
            event.preventDefault();
        });
    },
    methods: {
      // 添加搜索处理函数
      handleSearch() {
        this.currentPage = 1; // 重置到第一页
        this.fetchPasswords(true); // 执行带条件的查询
      },
      // 修改加载数据方法
      async fetchPasswords(isSearch = false) {
        const query = isSearch ? this.queryPasswordInput.trim() : "";

        // 添加加载状态
        this.loading = true;

        try {
          let response = await QueryPasswordsAPI(this.currentPage, this.pageSize, query);

          // 更健壮的数据检查
          if (!response || !response.data || typeof response.total !== 'number') {
            this.$message.error("查询结果格式不正确！");
            return;
          }

          this.passwordData = response.data || [];
          this.total = response.total;

          // 如果查询结果为空
          if (this.passwordData.length === 0 && this.currentPage > 1) {
            this.currentPage = 1; // 自动回到第一页
            await this.$nextTick(() => {
              this.fetchPasswords(isSearch); // 重新获取数据
            });
          }
        } catch (err) {
          console.error("查询错误：", err);
          this.$message.error(`查询失败: ${err.message || "请重试！"}`);
        } finally {
          this.loading = false;
        }
      },
      // 修改分页切换处理函数
      handlePageChange(newPage) {
        this.currentPage = newPage; // 确保更新当前页码
        this.fetchPasswords(!!this.queryPasswordInput.trim()); // 保留当前查询状态
      },

      async handleCopeClick(row, column, cell, event) {
        const textToCopy = cell.innerText; // 获取单元格内容
        try {
          await navigator.clipboard.writeText(textToCopy); // 复制到剪贴板
          this.$message.success(`已复制: ${textToCopy}`);
        } catch (error) {
          this.$message.error("复制失败，请重试");
        }
      },

      async handleAVQuery() {
        // 如果用户输入为空，返回错误提示
        if (!this.tasklistInput.trim()) {
          this.$message.error("请输入有效的 tasklist /svc 输出内容！");
          this.avResults = [];
          this.isAvQueried = false;
          return;
        }

        try {
          // 调用后端接口查询杀软进程
          const response = await QueryAntivirusProcesses(this.tasklistInput);

          this.avResults = response.map(item => ({
            program: item.program,    // 数据库中匹配的进程名
            match: item.match,        // 可选，显示完整的进程名
            description: item.description         // 杀软描述
          }));

          this.isAvQueried = true; // 标记查询完成
        } catch (err) {
          this.isAvQueried = true; // 标记查询完成
        }
      },

      async generateGoogleQueries() {
        // 如果用户输入为空，返回错误提示
        if (!this.googleDomainInput.trim()) {
          this.$message.warning('请输入域名或IP！');
          return;
        }

        const googleDomain = this.googleDomainInput.trim();
        try {
          // 调用后端接口查询生成的 Google 查询语法
          let response = await QueryGoogleQueries(googleDomain);

          // 如果返回的是单一对象而非数组，包装成数组
          if (!Array.isArray(response)) {
            response = [response];
          }

          // 更新查询结果
          this.googleQueries = response.map(item => ({
            category: item.category,           // 查询分类
            commands: [{
              id: "google1",                 // 查询命令的ID（根据实际数据格式修改）
              description: item.description, // 查询命令的描述
              command: item.command          // 生成的查询语法
            }]
          }));

          // 设置所有分类为展开状态
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

      // 地图key泄露
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
              // JSONP 调用百度 API，避免 CORS
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
              return; // JSONP 内部已结束 loading

            case 'qq-web':
            {
              // 1. 生成 JSONP 回调名
              const cbName = `jsonp_qq_${Date.now()}`;
              // 2. 构建带 callback 和 output=jsonp 的请求 URL
              const url = `https://apis.map.qq.com/ws/place/v1/search?keyword=${encodeURIComponent(
                  this.mapForm.keyword
              )}&boundary=nearby(${this.mapForm.center},${this.mapForm.radius})&key=${k}&output=jsonp&callback=${cbName}`;
              // 3. JSONP 调用
              await this.jsonp(url, cbName, (data) => {
                this.mapResult = data;
                this.mapLoading = false;
              });
            }
              return; // JSONP 回调内部已结束 loading
          }
        } catch (err) {
          this.mapResult = err.toString();
        } finally {
          this.mapLoading = false;
        }
      },

      /**
       * JSONP 工具函数：动态插入 <script> 并执行回调
       * @param {string} url 带 callback 参数的请求 URL
       * @param {string} cbName 全局回调函数名
       * @param {Function} onSuccess 回调执行后调用
       */
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
    }
};
</script>

<style scoped>
/* 页面容器 */
.container {
    height: 100vh;
    display: flex;
    margin-left: 10px;
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
</style>
