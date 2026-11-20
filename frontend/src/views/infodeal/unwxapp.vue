<template>
  <div class="unwxapp-wrapper">
    <!-- 配置区域 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>微信小程序反编译配置</span>
        </div>
      </template>

      <!-- 第一行：小程序路径选择 -->
      <el-row :gutter="20" class="config-row">
        <el-col :span="24">
          <el-form label-width="120px" label-position="left">
            <el-form-item label="微信Applet路径">
              <div class="path-selector">
                <el-input v-model="appletPath" readonly placeholder="请选择微信小程序存储路径" class="path-input" />
                <el-button type="primary" @click="selectAppletPath" class="select-btn">
                  选择路径
                </el-button>
                <el-tooltip effect="dark" placement="top">
                  <template #content>
                    <div style="text-align: left; line-height: 1.5;">
                      <div>• Win v3: C:\Users\用户名\Documents\WeChat Files\Applet (手工定位：PC微信->设置->文件管理)</div>
                      <div>• Win v4 (推荐): C:\Users\用户名\AppData\Roaming\Tencent\xwechat\radium\users
                        (程序会自动遍历所有用户目录下的applet\packages)</div>
                      <div>• Win v4 (旧): C:\Users\用户名\AppData\Roaming\Tencent\xwechat\radium\Applet\packages</div>
                      <div>• Mac v3: /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/.wxapplet/packages</div>
                      <div>• Mac v4 (推荐):
                        /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/users
                        (程序会自动遍历所有用户目录下的applet/packages)</div>
                      <div>• Mac v4 (旧):
                        /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages
                      </div>
                    </div>
                  </template>
                  <el-icon class="help-icon">
                    <QuestionFilled />
                  </el-icon>
                </el-tooltip>
              </div>
            </el-form-item>
          </el-form>
        </el-col>
      </el-row>

      <!-- 第二行：其他所有配置 -->
      <el-row :gutter="20" class="config-row">
        <el-col :span="12">
          <el-form label-width="120px" label-position="left">
            <el-form-item label="操作按钮">
              <el-space wrap>
                <el-button :type="autoDecompile ? 'success' : 'primary'" @click="toggleAutoDecompile" size="small">
                  {{ autoDecompile ? '停用自动反编译' : '启用自动反编译' }}
                </el-button>

                <el-button size="small" @click="clearApplet">清空Applet目录</el-button>

                <el-button size="small" @click="clearDecompiled">清空反编译文件</el-button>
              </el-space>
            </el-form-item>
          </el-form>
        </el-col>

        <el-col :span="12">
          <div class="config-edit-actions">
            <el-button size="small" type="success" @click="showRulesDialog = true">
              编辑正则提取规则
            </el-button>
            <el-button size="small" type="warning" @click="showAppidKeysDialog = true">
              Appid 查询
            </el-button>
            <span class="wxformat-inline">
              <span class="wxformat-label">格式化代码</span>
              <el-switch v-model="wxformat" />
            </span>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 主内容区域 -->
    <el-row :gutter="5" class="main-content">
      <!-- 左侧小程序列表 -->
      <el-col :span="11">
        <el-card class="applet-list-panel">
          <template #header>
            <div class="panel-header">
              <span>
                小程序列表
                <el-button type="primary" @click="loadAppletList" size="small" :loading="refreshing">
                  <el-icon>
                    <Refresh />
                  </el-icon>
                  {{ refreshing ? '刷新中...' : '刷新列表' }}
                </el-button>
              </span>

              <span class="total-count">共 {{ appletList.length }} 个</span>
            </div>
          </template>

          <div class="applet-list-container" :class="{ 'applet-list-container--fullscreen': isFullscreen }">
            <div v-for="app in appletList" :key="app.AppID" class="applet-item">
              <!-- 小程序头部信息 -->
              <div class="applet-header">
                <div class="applet-basic">
                  <el-tag type="primary" size="small" class="appid-tag">
                    {{ app.AppID }}
                  </el-tag>
                  <el-tag type="warning" size="small" class="date-tag">
                    {{ app.UpdateDate }}
                  </el-tag>
                </div>

                <!-- 在小程序列表显示部分-->
                <div class="applet-info" :class="{ 'applet-info--fullscreen': isFullscreen }">
                  <el-tooltip v-if="app.Info && app.Info.nickname && app.Info.nickname !== ''"
                              :content="app.Info.nickname" placement="top" effect="dark">
                    <el-tag type="danger" size="small" class="nickname-tag">
                      {{ app.Info.nickname }}
                    </el-tag>
                  </el-tooltip>
                  <el-tag v-if="app.Info && app.Info.username && app.Info.username !== ''" type="info" size="small">
                    {{ app.Info.username }}
                  </el-tag>
                  <!-- 如果没有信息，显示AppID -->
                  <el-tag v-else type="warning" size="small">
                    {{ app.AppID }}
                  </el-tag>
                </div>
              </div>

              <!-- 版本列表 -->
              <div class="version-list">
                <div v-for="version in app.Versions" :key="`${app.AppID}-${version.Number}`" class="version-item">
                  <div class="version-info">
                    <el-tag :type="getVersionTagType(version)" class="version-tag">
                      <el-icon class="version-icon">
                        <Files />
                      </el-icon>
                      {{ version.Number }}
                    </el-tag>

                    <el-tag
                      v-for="tag in (version.WxapkgTags || [])"
                      :key="`${version.Number}-${tag}`"
                      :type="wxapkgTagElType(tag)"
                      size="small"
                      class="wxapkg-kind-tag"
                    >
                      {{ tag }}
                    </el-tag>

                    <el-space class="version-actions" :size="4">
                      <!-- 反编译按钮 -->
                      <el-tooltip content="反编译">
                        <el-button :disabled="version.DecompileStatus === 'Running'"
                                   :loading="version.DecompileStatus === 'Running'" size="small"
                                   @click="decompileApplet(app, version)">
                          <template #icon>
                            <el-icon>
                              <MagicStick />
                            </el-icon>
                          </template>
                        </el-button>
                      </el-tooltip>

                      <!-- 提取敏感信息按钮 -->
                      <el-tooltip content="提取敏感信息">
                        <el-button :disabled="version.MatchStatus === 'Running'"
                                   :loading="version.MatchStatus === 'Running'" size="small"
                                   @click="extractSensitiveInfo(app, version)">
                          <template #icon>
                            <el-icon>
                              <Search />
                            </el-icon>
                          </template>
                        </el-button>
                      </el-tooltip>

                      <el-tooltip content="打开文件夹">
                        <el-button size="small" @click="openAppletFolder(app, version)">
                          <template #icon>
                            <el-icon>
                              <FolderOpened />
                            </el-icon>
                          </template>
                        </el-button>
                      </el-tooltip>

                      <el-tooltip content="敏感信息">
                        <el-button size="small" @click="showMatchedInfo(app, version)">
                          <template #icon>
                            <el-icon>
                              <Document />
                            </el-icon>
                          </template>
                        </el-button>
                      </el-tooltip>
                    </el-space>
                  </div>

                  <!-- 版本状态信息 -->
                  <div v-if="version.Message" class="version-message">
                    <el-text type="info" size="small">{{ version.Message }}</el-text>
                  </div>
                </div>

                <!-- 如果没有版本，显示提示 -->
                <div v-if="app.Versions.length === 0" class="no-version">
                  <el-text type="info" size="small">暂无版本信息</el-text>
                </div>

              </div>
            </div>

            <!-- 空状态 -->
            <div v-if="!appletList || appletList.length === 0" class="empty-state">
              <el-empty description="暂无小程序数据" />
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧详情面板 -->
      <el-col :span="12">
        <el-card class="detail-panel">
          <template #header>
            <div class="panel-header">
              <span>
                敏感信息提取结果
                <el-button type="success" size="small" @click="MingGanInfoCopyToClipboard('matchedResult')">
                  复制
                </el-button>
              </span>
              <div class="selected-info">
                <el-tag v-if="selectedApplet" type="warning">
                  {{ selectedApplet.appid }} {{ selectedApplet.version }}
                </el-tag>
                <el-tag v-if="selectedApplet?.nickname" type="danger">
                  {{ selectedApplet.nickname }}
                </el-tag>
              </div>
            </div>
          </template>

          <el-input v-model="matchedResult" type="textarea"
                    placeholder="选中版本的反编译结果和敏感信息将显示在这里" class="result-textarea"
                    resize="none" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 规则编辑对话框 -->
    <el-dialog v-model="showRulesDialog" title="信息提取规则" width="700px" destroy-on-close>
      <div class="rules-dialog">
        <div class="rules-toolbar">
          <el-button type="primary" @click="saveRules" size="small">
            保存规则
          </el-button>
          <el-button @click="showRulesDialog = false" size="small">
            取消
          </el-button>
          <span style="color: #0a6ebd">正则测试地址: https://regex101.com/</span>
        </div>

        <el-input v-model="rulesText" type="textarea" placeholder="每行一条正则表达式规则" :rows="15" class="rules-textarea"
                  resize="none" />
      </div>
    </el-dialog>

    <el-dialog
        v-model="showAppidKeysDialog"
        title="Appid 元数据查询"
        width="700px"
        class="appid-keys-dialog-root appid-query-config-dialog"
        destroy-on-close
        align-center
    >
      <div class="appid-query-config">
        <div class="appid-query-config__actions">
          <el-space :size="8">
            <el-button type="primary" @click="saveAppidQueryAPIKeys">保存</el-button>
            <el-button @click="showAppidKeysDialog = false">取消</el-button>
          </el-space>
          <el-button text type="primary" @click="openAppidQueryDoc">
            open.kainy 说明
          </el-button>
        </div>

        <el-card shadow="never" class="appid-query-config__card appid-query-config__card--mode">
          <div class="appid-query-config__label">查询方式</div>
          <el-radio-group
              v-model="appidQueryMode"
              class="appid-query-mode"
          >
            <el-radio-button value="mp">仅微信公众平台 MP</el-radio-button>
            <el-radio-button value="open">仅 X-API-Key</el-radio-button>
            <el-radio-button value="both">MP 与 API 都启用</el-radio-button>
          </el-radio-group>
          <p class="appid-query-config__hint appid-query-config__hint--mode">
            仅 MP：从浏览器贴 Cookie 和 token，请求间隔约 1.2 秒。都启用时先走 MP，失败或无效再试 API。
            请避免短时间内多次狂点「刷新列表」。
          </p>
        </el-card>

        <el-card
            v-show="appidQueryShowMPSection"
            shadow="never"
            class="appid-query-config__card"
        >
          <div class="appid-query-config__row-title">
            <span class="appid-query-config__step">1</span>
            <div>
              <div class="appid-query-config__subheading">公众号 MP</div>
              <div class="appid-query-config__desc">登录 mp.weixin.qq.com，点击账号设置->第三方设置->半屏搜索，打开开发者工具，过滤接口wxaembedded/searchWxaembedded，复制url中的token及完整 Cookie 粘贴到下方。</div>
            </div>
          </div>
          <el-input
              v-model="wechatMPTokenText"
              placeholder="MP token（例如 123456789）"
              class="appid-query-config__textarea"
          />
          <div style="padding-top: 10px;"></div>
          <el-input
              v-model="wechatMPCookieText"
              type="textarea"
              :rows="5"
              placeholder="RK=/erKD; ptcz=51ba2698cdb476a79a9c35f4f703c; pac_uid=0_hzjwp; "
              class="appid-query-config__textarea"
              resize="none"
          />
        </el-card>

        <el-card
            v-show="appidQueryShowApiSection"
            shadow="never"
            class="appid-query-config__card"
        >
          <div class="appid-query-config__row-title">
            <span class="appid-query-config__step">{{ appidQueryMode === 'both' ? '2' : '1' }}</span>
            <div class="appid-query-config__row-title--grow">
              <div>
                <div class="appid-query-config__subheading">open.kainy X-API-Key</div>
                <div class="appid-query-config__desc">多行按顺序轮换；# 为注释。下方状态在「刷新列表」后更新。</div>
              </div>
              <el-button type="danger" plain size="small" @click="removeDepletedAppidKeys">
                删除已用尽
              </el-button>
            </div>
          </div>
          <el-table
              :data="appidKeyStatusRows"
              border
              stripe
              size="small"
              max-height="180"
              empty-text="暂无有效密钥行"
              class="appid-query-config__table"
          >
            <el-table-column prop="Line" label="#" width="44" align="center"/>
            <el-table-column prop="Masked" label="密钥预览" min-width="200"/>
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="appidKeyStatusTagType(row.Status)" size="small">
                  {{ appidKeyStatusLabel(row.Status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="Detail" label="说明" min-width="140" show-overflow-tooltip/>
          </el-table>
          <el-input
              v-model="appidQueryApiKeysText"
              type="textarea"
              placeholder="每行一个 X-API-Key"
              :rows="9"
              class="appid-query-config__textarea appid-query-config__textarea--keys"
              resize="none"
          />
        </el-card>

        <el-collapse v-model="appidKeysHelpCollapse" class="appid-query-config__collapse">
          <el-collapse-item name="help">
            <template #title>详细说明（可选展开）</template>
            <div class="appid-keys-collapse-body appid-keys-collapse-body--tight">
              <p>「都启用」且失败时，若 MP 返回页无法解析会再试 <code class="rules-code">POST open.kainy AppidQuery</code>；仅选 MP 时不会回退到 API。</p>
              <p>额度用尽时会出现全局提示，并可用「删除已用尽」清掉失效行。</p>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {
  AutoDecompile, ClearApplet, ClearDecompiled, Decompile, ExtractSensitiveInfo,
  GetAllMiniApp,
  GetAppletPath, GetAppidQueryAPIKeys, GetAppidQueryAPIKeyStatuses, GetMatchedString, GetPackagePath, GetTaskStatus,
  GetAppidQueryMode,
  GetWechatMPToken,
  GetWechatMPCookie,
  GetWechatRules,
  InitCheck, RemoveDepletedAppidQueryAPIKeys, SaveWechatRules, SelectDirectory, SetAppletPath, SetAppidQueryAPIKeys, SetAppidQueryMode,
  SetWechatMPToken,
  SetWechatMPCookie
} from "../../../wailsjs/go/unwxapp/UnWxapp";
import { ElMessage, ElMessageBox } from "element-plus";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { GetOs, OpenPath } from "../../../wailsjs/go/system/System";
import { Document, Files, FolderOpened, InfoFilled, MagicStick, QuestionFilled, Refresh, Search } from "@element-plus/icons-vue";
import { BrowserOpenURL, EventsOn } from "../../../wailsjs/runtime";

export default {
  name: "UnwxApp",
  components: { Document, Search, FolderOpened, InfoFilled, MagicStick, Files, QuestionFilled, Refresh },
  data() {
    return {
      // 小程序反编译配置相关
      appletPath: "",
      appidQueryApiKeysText: "",
      wechatMPCookieText: "",
      wechatMPTokenText: "",
      /** @type {'mp' | 'open' | 'both'} 与后台 AppidQueryMode 一致 */
      appidQueryMode: "mp",
      autoDecompile: false,
      wxformat: true,
      rulesText: "",
      showRulesDialog: false,
      showAppidKeysDialog: false,
      /** Appid 密钥弹窗内「使用说明」折叠面板，含 name 表示展开 */
      appidKeysHelpCollapse: [],
      appidKeyStatusRows: [],

      // 数据相关
      appletList: [],
      selectedApplet: null,
      matchedResult: "",

      // 状态相关
      refreshing: false,
      dataCachePath: "",
      pollInterval: null,
      pollingTasks: new Set(), // 正在轮询的任务集合

      // 用于“全屏时调整高度上限”
      isFullscreen: false,


    }
  },
  watch: {
    async showAppidKeysDialog(v) {
      if (v) {
        try {
          const mode = (await GetAppidQueryMode()) || "both";
          this.appidQueryMode = (mode === "mp" || mode === "open" || mode === "both") ? mode : "both";
        } catch (e) {
          this.appidQueryMode = "both";
          console.warn("加载 Appid 查询方式失败:", e);
        }
        try {
          this.wechatMPCookieText = (await GetWechatMPCookie()) || "";
          this.wechatMPTokenText = (await GetWechatMPToken()) || "";
        } catch (e) {
          console.warn("加载公众号 MP 配置失败:", e);
        }
        this.$nextTick(() => this.refreshAppidKeyStatusTable());
      }
    },
  },
  computed: {
    appidQueryShowMPSection() {
      return this.appidQueryMode === "mp" || this.appidQueryMode === "both";
    },
    appidQueryShowApiSection() {
      return this.appidQueryMode === "open" || this.appidQueryMode === "both";
    },
  },
  async mounted() {
    // 同步初始化：先算一次“是否全屏”，再进行耗时初始化
    this.updateFullscreenState();
    this._resizeHandler = this.updateFullscreenState.bind(this);
    window.addEventListener("resize", this._resizeHandler);

    // 加载小程序反编译配置
    await this.initUnwxAppData();

    this._offAppidQuotaEvent = EventsOn("unwxapp:appid_query_quota_exhausted", () => {
      ElMessage.warning("apikey额度已使用完毕, 查询失败的以原始ID展示");
    });
  },
  beforeUnmount() {
    if (this._resizeHandler) {
      window.removeEventListener("resize", this._resizeHandler);
    }
    if (typeof this._offAppidQuotaEvent === "function") {
      this._offAppidQuotaEvent();
      this._offAppidQuotaEvent = null;
    }
  },
  methods: {
    // 通过窗口高度接近“可用屏幕高度”来判断（兼容 Electron/Wails 的全屏体验）
    updateFullscreenState() {
      if (typeof window === "undefined" || !window.screen) return;
      const h = window.innerHeight || 0;
      const availH = window.screen.availHeight || window.screen.height || 0;
      // 留一点误差，避免边框/任务栏导致频繁抖动
      this.isFullscreen = h >= availH - 60;
    },

    // 小程序反编译
    async initUnwxAppData() {
      try {
        // 初始化时检查Node.js环境
        const hasNode = await InitCheck();
        if (!hasNode) {
          ElMessageBox.confirm(
              "检测到未安装 Node.js 环境，是否现在前往官网下载？",
              "温馨提示",
              {
                confirmButtonText: "马上前往",
                cancelButtonText: "稍后再说",
                type: "warning",
              }
          ).then(() => {
            BrowserOpenURL("https://nodejs.org/");
          });
        }

        // 加载规则
        const rules = await GetWechatRules();
        this.rulesText = rules ? rules.join("\n") : "";

        this.appletPath = await GetAppletPath();

        try {
          this.appidQueryApiKeysText = (await GetAppidQueryAPIKeys()) || "";
        } catch (e) {
          console.warn("加载 Appid 查询密钥配置失败:", e);
        }
        try {
          this.wechatMPCookieText = (await GetWechatMPCookie()) || "";
          this.wechatMPTokenText = (await GetWechatMPToken()) || "";
          const mode = (await GetAppidQueryMode()) || "both";
          this.appidQueryMode = (mode === "mp" || mode === "open" || mode === "both") ? mode : "both";
        } catch (e) {
          console.warn("加载 Appid 查询/MP 配置失败:", e);
        }

        // 加载小程序列表
        await this.loadAppletList();
      } catch (error) {
        console.error("初始化数据失败:", error);
        ElMessage.error("初始化失败");
      }
    },

    // 修复小程序列表数据处理
    async loadAppletList() {
      try {
        this.refreshing = true;
        const oldList = JSON.parse(JSON.stringify(this.appletList)); // 深拷贝旧列表

        const data = await GetAllMiniApp();

        // 处理数据格式，确保字段名一致
        const newList = (data || []).map(app => {
          return {
            AppID: app.AppID || app.app_id || app.AppId || '',
            UpdateDate: app.UpdateDate || app.update_date || app.updateDate || '',
            Info: app.Info || app.info || {},
            Versions: (app.Versions || app.versions || app.Version || []).map(version => ({
              Number: version.Number || version.number || version.Version || version.version || '',
              DecompileStatus: version.DecompileStatus || version.decompileStatus || version.status || 'Waiting',
              MatchStatus: version.MatchStatus || version.matchStatus || 'Waiting',
              Message: version.Message || version.message || '',
              WxapkgTags: version.WxapkgTags || version.wxapkgTags || []
            }))
          };
        });

        // 按小程序最近更新时间降序（与后端一致），相同时按 AppID
        newList.sort((a, b) => {
          const ta = this.parseMiniAppListDate(a.UpdateDate);
          const tb = this.parseMiniAppListDate(b.UpdateDate);
          if (tb !== ta) return tb - ta;
          return a.AppID.localeCompare(b.AppID);
        });
        this.appletList = newList;

        await this.refreshAppidKeyStatusTable();

        // 检查状态变化
        this.checkStatusChanges(oldList, this.appletList);

      } catch (error) {
        console.error("加载小程序列表失败:", error);
        this.$message.error("加载小程序列表失败: " + error);
        this.appletList = [];
      } finally {
        this.refreshing = false;
      }
    },

    // 检查状态变化
    checkStatusChanges(oldList, newList) {
      for (const newApp of newList) {
        const oldApp = oldList.find(app => app.AppID === newApp.AppID);
        if (!oldApp) continue;

        for (const newVersion of newApp.Versions) {
          const oldVersion = oldApp.Versions.find(v => v.Number === newVersion.Number);
          if (!oldVersion) continue;

          // 检查反编译状态变化
          if (oldVersion.DecompileStatus === 'Running' &&
              newVersion.DecompileStatus === 'Stopped') {
            this.$message.success(`小程序 ${newApp.AppID} 反编译完成`);
          }

          // 检查匹配状态变化
          if (oldVersion.MatchStatus === 'Running' &&
              newVersion.MatchStatus === 'Stopped') {
            this.$message.success(`小程序 ${newApp.AppID} 敏感信息提取完成`);
          }
        }
      }
    },

    async selectAppletPath() {
      try {
        const path = await SelectDirectory();
        if (path) {
          await SetAppletPath(path);
          this.appletPath = path;
          ElMessage.success("路径设置成功");

          // 立即刷新小程序列表
          await this.loadAppletList();
        }
      } catch (error) {
        ElMessage.error("选择路径失败");
      }
    },

    async toggleAutoDecompile() {
      try {
        await AutoDecompile(!this.autoDecompile);
        this.autoDecompile = !this.autoDecompile;
        ElMessage.success(
            this.autoDecompile ? "已启用自动反编译" : "已停用自动反编译"
        );

        // 如果启用自动反编译，立即开始轮询状态
        if (this.autoDecompile) {
          this.startAutoDecompilePolling();
        }
      } catch (error) {
        ElMessage.error("操作失败: " + error);
      }
    },

    // 自动反编译状态轮询
    startAutoDecompilePolling() {
      // 每30秒刷新一次列表，检查新任务
      setInterval(async () => {
        if (this.autoDecompile) {
          await this.loadAppletList();
        }
      }, 30000);
    },

    async clearApplet() {
      try {
        await confirmDangerousAction({
          title: '清空 Applet 目录',
          message: '将删除 Applet 目录下所有文件，并同时删除反编译后的所有文件。',
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        await ClearApplet();
        ElMessage.success("Applet目录已清空");
        // 立即刷新列表
        await this.loadAppletList();
      } catch (error) {
        ElMessage.error("清空失败: " + error);
      }
    },

    async clearDecompiled() {
      try {
        await confirmDangerousAction({
          title: '清空反编译文件',
          message: '将删除所有反编译后的文件，不可恢复。',
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        await ClearDecompiled();
        ElMessage.success("反编译文件已清空");
        // 立即刷新列表
        await this.loadAppletList();
      } catch (error) {
        ElMessage.error("清空失败: " + error);
      }
    },

    async saveAppidQueryAPIKeys() {
      try {
        const mode = this.appidQueryMode;
        if (mode !== "mp" && mode !== "open" && mode !== "both") {
          ElMessage.warning("查询方式无效");
          return;
        }
        await SetAppidQueryMode(mode);
        await SetWechatMPCookie((this.wechatMPCookieText || "").trim());
        await SetWechatMPToken((this.wechatMPTokenText || "").trim());
        await SetAppidQueryAPIKeys(this.appidQueryApiKeysText || "");
        ElMessage.success("已保存");
        await this.refreshAppidKeyStatusTable();
        this.showAppidKeysDialog = false;
      } catch (error) {
        ElMessage.error("保存配置失败: " + error);
      }
    },

    async removeDepletedAppidKeys() {
      const depleted = (this.appidKeyStatusRows || []).filter(
          r => String(r.Status || "").toLowerCase() === "depleted");
      if (depleted.length === 0) {
        ElMessage.info("当前没有「已用尽」密钥可删；请先刷新小程序列表以更新状态表。");
        return;
      }
      try {
        await ElMessageBox.confirm(
            `将从已保存的配置中永久移除 ${depleted.length} 条标记为「已用尽」的密钥行（与下方列表一致）。是否继续？`,
            "删除已用尽密钥",
            { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" },
        );
      } catch {
        return;
      }
      try {
        const removed = await RemoveDepletedAppidQueryAPIKeys();
        this.appidQueryApiKeysText = (await GetAppidQueryAPIKeys()) || "";
        if (removed > 0) {
          ElMessage.success(`已删除 ${removed} 条已用尽密钥`);
        } else {
          ElMessage.info("未删除任何行（状态与配置可能未同步，请先刷新列表后再试）");
        }
        await this.refreshAppidKeyStatusTable();
      } catch (error) {
        ElMessage.error("删除失败: " + error);
      }
    },

    async refreshAppidKeyStatusTable() {
      try {
        const rows = await GetAppidQueryAPIKeyStatuses();
        this.appidKeyStatusRows = Array.isArray(rows) ? rows.map(r => ({
          Line: r.Line ?? r.line,
          Masked: r.Masked ?? r.masked ?? "",
          Status: r.Status ?? r.status ?? "unknown",
          Detail: r.Detail ?? r.detail ?? "",
        })) : [];
      } catch (e) {
        console.warn("加载密钥状态表失败:", e);
        this.appidKeyStatusRows = [];
      }
    },

    appidKeyStatusLabel(status) {
      const s = String(status || "").toLowerCase();
      if (s === "ok") return "可用";
      if (s === "depleted") return "已用尽";
      if (s === "error") return "异常";
      return "未探测";
    },

    appidKeyStatusTagType(status) {
      const s = String(status || "").toLowerCase();
      if (s === "ok") return "success";
      if (s === "depleted") return "danger";
      if (s === "error") return "warning";
      return "info";
    },

    openAppidQueryDoc() {
      BrowserOpenURL("https://open.kainy.cn/");
    },

    async saveRules() {
      try {
        const rules = this.rulesText.split('\n').filter(rule => rule.trim());
        await SaveWechatRules(rules);
        this.showRulesDialog = false;
        ElMessage.success("规则保存成功");
      } catch (error) {
        ElMessage.error("保存规则失败: " + error);
      }
    },

    // 开始轮询任务状态
    startPollingTask(appID, versionNumber) {
      const taskKey = `${appID}-${versionNumber}`;

      // 如果已经在轮询，则跳过
      if (this.pollingTasks.has(taskKey)) {
        return;
      }

      this.pollingTasks.add(taskKey);

      // 设置轮询间隔
      const pollInterval = setInterval(async () => {
        try {
          // 只获取特定任务的状态，而不是刷新整个列表
          const status = await GetTaskStatus(appID, versionNumber);

          // 更新本地任务状态
          const task = this.findTask(appID, versionNumber);
          if (task && status) {
            task.DecompileStatus = status.DecompileStatus;
            task.MatchStatus = status.MatchStatus;
            task.Message = status.Message || '';
            if (status.WxapkgTags && status.WxapkgTags.length) {
              task.WxapkgTags = status.WxapkgTags;
            }
          }

          // 检查任务是否完成
          if (status &&
              status.DecompileStatus !== 'Running' &&
              status.MatchStatus !== 'Running') {
            // 任务完成，停止轮询
            clearInterval(pollInterval);
            this.pollingTasks.delete(taskKey);
            console.log(`停止轮询任务: ${taskKey}`);

            // 任务完成后，刷新一次列表以获取最新数据（如文件数量等）
            await this.loadAppletList();
          }
        } catch (error) {
          console.error('轮询任务状态失败:', error);
          // 如果获取单个状态失败，回退到刷新整个列表
          try {
            await this.loadAppletList();
          } catch (refreshError) {
            console.error('刷新列表失败:', refreshError);
          }
        }
      }, 3000); // 每3秒轮询一次

      // 10分钟后自动停止轮询（安全机制）
      setTimeout(() => {
        if (this.pollingTasks.has(taskKey)) {
          clearInterval(pollInterval);
          this.pollingTasks.delete(taskKey);
          console.log(`轮询超时，停止任务: ${taskKey}`);
        }
      }, 10 * 60 * 1000);
    },

    // 查找特定任务
    findTask(appID, versionNumber) {
      for (const app of this.appletList) {
        if (app.AppID === appID) {
          const version = app.Versions.find(v => v.Number === versionNumber);
          if (version) return version;
        }
      }
      return null;
    },

    // 修改 decompileApplet 方法中的数据结构
    async decompileApplet(app, version) {
      try {
        console.log("开始反编译:", app.AppID, version.Number);

        // 检查必要参数
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        // 检查是否设置了小程序路径
        if (!this.appletPath) {
          this.$message.error("请先设置微信小程序路径");
          return;
        }

        // 更新前端状态
        version.DecompileStatus = 'Running';
        version.Message = '反编译中...';

        const item = {
          AppID: app.AppID,
          UpdateDate: app.UpdateDate || '',
          Info: app.Info || {},
          Versions: [{
            Number: version.Number,
            DecompileStatus: 'Running',
            MatchStatus: version.MatchStatus || 'Waiting',
            Message: '反编译中...'
          }]
        };

        console.log("调用 Decompile 方法进行反编译:", JSON.stringify(item, null, 2));

        // 调用 Decompile 方法
        await Decompile(item);

        this.$message.success("开始反编译，请稍后查看结果");

        // 启动轮询
        this.startPollingTask(app.AppID, version.Number);

      } catch (error) {
        console.error("反编译失败:", error);
        if (version) {
          version.DecompileStatus = 'Error';
          version.Message = '反编译失败: ' + (error.message || "未知错误");
        }
        this.$message.error("反编译失败: " + error.message);
      }
    },

    async extractSensitiveInfo(app, version) {
      try {
        console.log("开始提取敏感信息:", app.AppID, version.Number);

        // 检查必要参数
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        // 更新前端状态
        version.MatchStatus = 'Running';
        version.Message = '正在提取敏感信息...';

        const item = {
          AppID: app.AppID,
          UpdateDate: app.UpdateDate || '',
          Info: app.Info || {},
          Versions: [{
            Number: version.Number,
            DecompileStatus: version.DecompileStatus || 'Waiting',
            MatchStatus: 'Running',
            Message: '正在提取敏感信息...'
          }]
        };

        console.log("调用 ExtractSensitiveInfo 进行敏感信息提取:", item);

        // 调用提取方法
        await ExtractSensitiveInfo(app.AppID, version.Number);

        this.$message.success("开始提取敏感信息，请稍后查看结果");

        // 启动轮询
        this.startPollingTask(app.AppID, version.Number);

      } catch (error) {
        console.error("提取敏感信息失败:", error);
        if (version) {
          version.MatchStatus = 'Error';
          version.Message = '提取失败: ' + (error.message || "未知错误");
        }
        this.$message.error("提取敏感信息失败: " + error.message);
      }
    },

    // 添加打开文件夹功能
    async openAppletFolder(app, version) {
      try {
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        if (!this.appletPath) {
          this.$message.error("请先设置微信小程序路径");
          return;
        }

        // 使用后端API获取实际的package路径（支持新版本users目录和旧版本路径）
        const packagePath = await GetPackagePath(app.AppID, version.Number);

        if (!packagePath) {
          this.$message.error("无法找到package路径");
          return;
        }

        // 调用打开路径的方法
        await OpenPath(packagePath);

      } catch (error) {
        console.error("打开文件夹失败:", error);
        this.$message.error("打开文件夹失败: " + (error.message || "未知错误"));
      }
    },

    // 在显示敏感信息的方法中，可以添加格式处理
    async showMatchedInfo(app, version) {
      try {
        // 检查必要参数
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        this.selectedApplet = {
          appid: app.AppID,
          version: version.Number,
          nickname: (app.Info && app.Info.nickname) || (app.Info && app.Info.Nickname) || ""
        };

        console.log("获取匹配信息:", app.AppID, version.Number);

        const result = await GetMatchedString(app.AppID, version.Number);

        if (result && result.length > 0) {
          // 如果有统计信息，可以特殊显示
          this.matchedResult = result.map(line => {
            if (line.includes(" -> ")) {
              const [file, content] = line.split(" -> ");
              return `📄 ${file}\n   🔍 ${content}`;
            }
            return line;
          }).join("\n\n");
        } else {
          this.matchedResult = "未找到匹配的敏感信息";
        }
      } catch (error) {
        console.error("获取敏感信息失败:", error);
        ElMessage.error("获取敏感信息失败: " + error);
        this.matchedResult = "获取敏感信息失败: " + error;
      }
    },

    MingGanInfoCopyToClipboard(type) {
      const text = this[type];
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success("敏感信息提取结果已复制到剪贴板！");
      });
    },

    /** 与后端 UpdateDate 格式 "2006/01/02 15:04" 一致，用于列表排序 */
    parseMiniAppListDate(s) {
      if (!s) return 0;
      const m = String(s).trim().match(/^(\d{4})\/(\d{2})\/(\d{2})\s+(\d{2}):(\d{2})$/);
      if (!m) return 0;
      return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]), Number(m[4]), Number(m[5])).getTime();
    },

    wxapkgTagElType(label) {
      switch (label) {
        case '小程序':
          return 'primary';
        case '插件':
          return 'warning';
        case '小游戏':
          return 'success';
        default:
          return 'info';
      }
    },

    getVersionTagType(version) {
      if (!version) return 'info';

      const status = version.DecompileStatus || 'Waiting';
      switch (status) {
        case 'Stopped':
          return 'success';
        case 'Running':
          return 'warning';
        case 'Error':
          return 'danger';
        default:
          return 'info';
      }
    },
  }
}
</script>

<style scoped>
.unwxapp-wrapper {
  height: 100%;
  min-height: 92vh;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 8px 0;
  min-height: 0;
  overflow-x: hidden;
  width: 100%;
  box-sizing: border-box;
}

.config-card {
  flex-shrink: 0;
}

.path-selector {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.path-input {
  flex: 1;
  min-width: 750px;
}

.select-btn {
  flex-shrink: 0;
}

.help-icon {
  color: #909399;
  cursor: help;
}

.rules-tip {
  margin-top: 8px;
}

/* 确保操作按钮区域对齐 */
.config-row .el-form-item {
  margin-bottom: 15px;
}

.config-edit-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding-top: 4px;
}

.wxformat-inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-left: 4px;
  padding-left: 12px;
  border-left: 1px solid var(--el-border-color-lighter);
}

.wxformat-label {
  font-size: 14px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}

.main-content {
  flex: 1;
  display: flex;
  gap: 5px;
  min-height: 0;
  align-items: stretch;
}

.applet-list-panel,
.detail-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 300px;
  min-height: 0;
  margin-left: 5px;
}

.applet-list-panel :deep(.el-card__body),
.detail-panel :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.total-count {
  color: #909399;
  font-size: 12px;
}

.selected-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* 小程序列表容器 - 添加高度限制和滚动 */
.applet-list-container {
  flex: 1;
  overflow-y: auto;
  padding: 5px;
  max-height: 440px;
  min-height: 440px;
}

.applet-list-container--fullscreen {
  max-height: 610px;
}

.applet-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: white;
  transition: all 0.3s ease;
}

.applet-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.applet-header {
  margin-bottom: 8px;
}

.applet-basic {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.applet-info {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  /* 默认非全屏：超出 480px 高度后滚动 */
  max-height: 480px;
  overflow-y: auto;
  overflow-x: hidden;
}

.applet-info--fullscreen {
  /* 全屏时放大到 650px，避免视觉上留白过多 */
  max-height: 650px;
}

.appid-tag,
.date-tag {
  font-weight: bold;
}

.nickname-tag {
  max-width: 120px;
  overflow: hidden;
  white-space: nowrap;
  display: inline-block;
}

.version-list {
  margin-top: 8px;
  max-height: none;
  overflow: visible;
  min-height: 0;
}

.version-item {
  padding: 8px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  margin-bottom: 6px;
  background: #fafafa;
}

.version-item:last-child {
  margin-bottom: 0;
}

.version-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
  flex-wrap: wrap;
  gap: 8px;
}

.version-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.wxapkg-kind-tag {
  flex-shrink: 0;
}

.version-icon {
  font-size: 12px;
}

.version-actions {
  flex-shrink: 0;
}

.version-message {
  margin-top: 4px;
}

.empty-state {
  min-height: 420px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.result-textarea {
  flex: 1;
  height: 100%;
  border: none;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  min-height: 0;
  resize: vertical;
}

.result-textarea :deep(.el-textarea__inner) {
  height: 100%;
  min-height: 0;
}

.rules-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rules-toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.rules-toolbar-hint {
  color: #0a6ebd;
  font-size: 13px;
  line-height: 1.5;
}

.rules-dialog-desc {
  margin: 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}

.rules-code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  padding: 1px 6px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  color: #303133;
}

.appid-keys-dialog-root :deep(.el-dialog__header) {
  margin-right: 0;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.appid-keys-dialog-root :deep(.el-dialog__body) {
  padding-top: 14px;
  padding-bottom: 8px;
}

.appid-query-config-dialog :deep(.el-dialog__body) {
  max-height: min(78vh, 700px);
  overflow-y: auto;
}

.appid-query-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.appid-query-config__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-bottom: 2px;
}

.appid-query-config__card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.appid-query-config__card--mode {
  --el-card-padding: 0;
}

.appid-query-config__card--mode :deep(.el-card__body) {
  padding: 12px 14px 14px;
}

.appid-query-config__card:not(.appid-query-config__card--mode) :deep(.el-card__body) {
  padding: 2px;
}

.appid-query-config__label {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
}

.appid-query-mode {
  display: flex;
  flex-wrap: wrap;
  width: 100%;
  gap: 0;
}

.appid-query-mode :deep(.el-radio-button__inner) {
  padding: 8px 10px;
  font-size: 12px;
}

.appid-query-config__hint--mode {
  margin: 10px 0 0;
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.appid-query-config__row-title {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
  align-items: flex-start;
}

.appid-query-config__row-title--grow {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px 12px;
}

.appid-query-config__step {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--el-color-primary-light-8);
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.appid-query-config__subheading {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0 0 4px;
}

.appid-query-config__desc {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  margin: 0;
}

.appid-query-config__textarea :deep(.el-textarea__inner) {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 12px;
  line-height: 1.45;
}

.appid-query-config__textarea--keys :deep(.el-textarea__inner) {
  min-height: 140px;
}

.appid-query-config__table {
  width: 100%;
  margin-bottom: 10px;
}

.appid-query-config__collapse {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
  --el-collapse-header-bg-color: var(--el-fill-color);
}

.appid-query-config__collapse :deep(.el-collapse-item__header) {
  min-height: 40px;
  font-size: 13px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.appid-query-config__collapse :deep(.el-collapse-item__content) {
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}

.appid-keys-collapse-body--tight {
  padding: 10px 12px 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}

.appid-keys-collapse-body--tight p {
  margin: 0 0 8px;
}

.appid-keys-dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.appid-keys-dialog__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.appid-keys-dialog__doc-btn {
  font-size: 13px;
  padding: 4px 8px;
}

.appid-keys-dialog__collapse {
  margin: 0;
  border: 1px solid var(--el-color-info-light-7);
  border-radius: 8px;
  overflow: hidden;
  --el-collapse-header-bg-color: var(--el-color-info-light-9);
}

.appid-keys-dialog__collapse :deep(.el-collapse-item__header) {
  height: auto;
  min-height: 44px;
  line-height: 1.45;
  padding: 10px 14px;
  font-size: 13px;
  border-bottom: none;
}

.appid-keys-dialog__collapse :deep(.el-collapse-item__arrow) {
  margin: 0 0 0 8px;
}

.appid-keys-dialog__collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.appid-keys-dialog__collapse :deep(.el-collapse-item__content) {
  padding: 0;
}

.appid-keys-collapse-title {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 10px;
  color: var(--el-text-color-primary);
}

.appid-keys-collapse-title__icon {
  font-size: 16px;
  color: var(--el-color-info);
  flex-shrink: 0;
}

.appid-keys-collapse-title__text {
  font-weight: 600;
}

.appid-keys-collapse-title__hint {
  font-size: 12px;
  font-weight: normal;
  color: var(--el-text-color-secondary);
}

.appid-keys-collapse-body {
  padding: 12px 14px 14px;
  font-size: 13px;
  line-height: 1.65;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-blank);
  border-top: 1px solid var(--el-border-color-lighter);
}

.appid-keys-collapse-body p {
  margin: 0 0 10px;
}

.appid-keys-collapse-body p:last-child {
  margin-bottom: 0;
}

.appid-keys-dialog__panel {
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px 14px 14px;
}

.appid-keys-dialog__panel--editor {
  padding-bottom: 12px;
}

.appid-keys-dialog__panel-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px 12px;
  margin-bottom: 10px;
}

.appid-keys-dialog__panel-head--status {
  align-items: flex-start;
}

.appid-keys-dialog__panel-head-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.appid-keys-dialog__panel-head--status .appid-keys-dialog__panel-meta {
  text-align: left;
}

.appid-keys-dialog__panel-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.appid-keys-dialog__panel-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  text-align: right;
  flex: 1;
  min-width: 0;
}

.appid-keys-dialog__table {
  width: 100%;
}

.appid-keys-dialog__table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 4px 4px;
}

.appid-keys-dialog__textarea :deep(.el-textarea__inner) {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 13px;
  line-height: 1.45;
  min-height: 200px;
}

.rules-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}
</style>