<script setup lang="ts">
import { computed, onMounted, ref, reactive } from "vue";
import { Config, useConfigStore } from "./store/types";
import { ElMessage } from "element-plus";
import axios from 'axios';
import { GetPortConfig } from '../../../wailsjs/go/controller/Server';
import { BrowserOpenURL } from '../../../wailsjs/runtime';

const configState = useConfigStore();
const activeTab = ref('basic'); // 选项卡状态

const formData = ref({
  Code: 0,
  Error: "",
  FilePath: computed({
    get: () => configState.getFilePath(),
    set: (value) => configState.setFilePath(value),
  }),
  CoroutineCount: computed({
    get: () => configState.getCoroutineCount(),
    set: (value) => configState.setCoroutineCount(value),
  }),
  Timeout: computed({
    get: () => configState.getTimeout(),
    set: (value) => configState.setTimeout(value),
  }),
  SocksAddress: computed({
    get: () => configState.getSocksAddress(),
    set: (value) => configState.setSocksAddress(value),
  }),

  Status: 0,

  // 便携代理不再单独维护空间测绘类平台的账号与秘钥，
  // 只保留与代理本身运行相关的必需参数（超时、并发、Socks地址等）。
  CheckTimeOut: computed({
    get: () => configState.getCheckTimeout(),
    set: (value) => configState.setCheckTimeout(value),
  }),
  MaxPage: computed({
    get: () => configState.getMaxpage(),
    set: (value) => configState.setMaxpage(value),
  }),
  Xui: computed({
    get: () => configState.getXui(),
    set: (value) => configState.setXui(value),
  }),

  // 搜索语句配置
  FofaSocks5Query: computed({
    get: () => configState.getFofaSocks5Query(),
    set: (value) => configState.setFofaSocks5Query(value),
  }),
  FofaXuiQuery: computed({
    get: () => configState.getFofaXuiQuery(),
    set: (value) => configState.setFofaXuiQuery(value),
  }),
  HunterSocks5Query: computed({
    get: () => configState.getHunterSocks5Query(),
    set: (value) => configState.setHunterSocks5Query(value),
  }),
  HunterXuiQuery: computed({
    get: () => configState.getHunterXuiQuery(),
    set: (value) => configState.setHunterXuiQuery(value),
  }),
  QuakeSocks5Query: computed({
    get: () => configState.getQuakeSocks5Query(),
    set: (value) => configState.setQuakeSocks5Query(value),
  }),
  QuakeXuiQuery: computed({
    get: () => configState.getQuakeXuiQuery(),
    set: (value) => configState.setQuakeXuiQuery(value),
  })
});

const rules = {
  CoroutineCount: [
    { required: true, message: "请输入协程数量", trigger: "blur" },
  ],
  Timeout: [
    { required: true, message: "请指定超时时长", trigger: "blur" },
  ],
  MaxPage: [
    { required: true, message: "请输入采集最大页数", trigger: "blur" },
  ],
};

const disabled = ref(false);

// -------- 空间测绘 API 配置（样式与 SpaceMapping 统一） --------
const showApiConfigDialog = ref(false);
const apiClient = ref<any | null>(null);
const loadingApiConfig = ref(false);

const apiConfigs = reactive<Record<string, { enabled: boolean; email?: string; key: string; showKey: boolean; status?: string }>>({
  '0zone': { enabled: false, key: '', showKey: false, status: 'unknown' },
  fofa: { enabled: false, email: '', key: '', showKey: false, status: 'unknown' },
  otherfofa: { enabled: false, key: '', showKey: false, status: 'unknown' },
  hunter: { enabled: false, key: '', showKey: false, status: 'unknown' },
  quake: { enabled: false, key: '', showKey: false, status: 'unknown' },
});

const apiSettings = reactive({
  fofaThirdBaseURL: 'https://fofa.red',
});

// 供便携代理使用的平台选择（仅在基础配置页勾选使用）
const selectedPlatforms = ref<string[]>([]);
const availablePlatforms = computed(() =>
  Object.entries(apiConfigs)
    .filter(([_, cfg]) => cfg.enabled)
    .map(([key]) => ({
      value: key,
      label: getPlatformLabel(key),
    }))
);

const initApiClient = async () => {
  if (apiClient.value) return;
  const portConfig = await GetPortConfig();
  const port = portConfig?.mainPort || 52867;
  const baseURL = `http://127.0.0.1:${port}`;
  apiClient.value = axios.create({
    baseURL,
    timeout: 30000,
  });
};

const loadApiConfig = async () => {
  try {
    loadingApiConfig.value = true;
    await initApiClient();
    if (!apiClient.value) return;
    const response = await apiClient.value.get('/api/info-collection/config');
    if (response.data && response.data.success) {
      const config = response.data.data;
      if (config.apiKeys) {
        Object.keys(config.apiKeys).forEach((key: string) => {
          const keyMap: Record<string, string> = {
            'fofa.red': 'otherfofa',
            '0.zone': '0zone',
            zone0: '0zone',
          };
          const localKey = keyMap[key] || key;
          if (!apiConfigs[localKey]) return;
          const apiKeyValue = config.apiKeys[key] || '';
          if (localKey === 'fofa' && apiKeyValue.includes(':')) {
            const parts = apiKeyValue.split(':', 2);
            apiConfigs[localKey].email = parts[0] || '';
            apiConfigs[localKey].key = parts[1] || '';
          } else {
            apiConfigs[localKey].key = apiKeyValue;
          }
          apiConfigs[localKey].enabled = config.enabled?.[key] || false;
        });
      }
      if (config.settings) {
        if (config.settings.fofaThirdBaseURL) {
          apiSettings.fofaThirdBaseURL = config.settings.fofaThirdBaseURL;
        }
      }
      // 如果当前未选择平台，则默认选中所有已启用的平台，避免为空
      if (selectedPlatforms.value.length === 0) {
        selectedPlatforms.value = Object.entries(apiConfigs)
          .filter(([_, cfg]) => cfg.enabled)
          .map(([key]) => key);
      }
    }
  } catch (e: any) {
    console.error('加载空间测绘 API 配置失败:', e);
    ElMessage.error('加载空间测绘 API 配置失败');
  } finally {
    loadingApiConfig.value = false;
  }
};

const handleSaveApiConfig = async () => {
  try {
    await initApiClient();
    if (!apiClient.value) return;

    const apiKeys: Record<string, string> = {};
    const enabled: Record<string, boolean> = {};
    Object.keys(apiConfigs).forEach((key: string) => {
      if (key === 'fofa') {
        const email = apiConfigs[key].email || '';
        const keyValue = apiConfigs[key].key || '';
        if (email && keyValue) {
          apiKeys[key] = `${email}:${keyValue}`;
        } else if (keyValue) {
          apiKeys[key] = keyValue;
        } else {
          apiKeys[key] = '';
        }
      } else {
        apiKeys[key] = apiConfigs[key].key || '';
      }
      enabled[key] = apiConfigs[key].enabled;
    });

    const resp = await apiClient.value.post('/api/info-collection/config', {
      apiKeys,
      enabled,
      settings: {
        fofaThirdBaseURL: apiSettings.fofaThirdBaseURL || 'https://fofa.red',
      },
    });

    if (resp.data && resp.data.success) {
      ElMessage.success('API 配置保存成功');
      showApiConfigDialog.value = false;
    } else {
      ElMessage.error(resp.data?.message || '保存失败');
    }
  } catch (e: any) {
    console.error('保存空间测绘 API 配置失败:', e);
    ElMessage.error('保存空间测绘 API 配置失败: ' + (e.response?.data?.message || e.message));
  }
};

const getPlatformLabel = (platform: string) => {
  const labels: Record<string, string> = {
    '0zone': '0.zone',
    fofa: 'FOFA官方',
    otherfofa: 'FOFA第三方',
    hunter: 'Hunter',
    quake: 'Quake',
  };
  return labels[platform] || platform;
};

const openApiApplyPage = (platform: string) => {
  let url = '';
  if (platform === '0zone') {
    url = 'https://0.zone/apply';
  } else if (platform === 'fofa') {
    url = 'https://fofa.info/';
  } else if (platform === 'otherfofa') {
    url = (apiSettings.fofaThirdBaseURL || 'https://fofa.red/').trim();
  } else if (platform === 'hunter') {
    url = 'https://hunter.qianxin.com/';
  } else if (platform === 'quake') {
    url = 'https://quake.360.net/';
  }
  if (url) {
    BrowserOpenURL(url);
  }
};

const verifySingleKey = async (platform: string, apiKey: string) => {
  try {
    await initApiClient();
    if (!apiClient.value) return false;

    const response = await apiClient.value.post('/api/info-collection/space-mapping/verify-key', {
      platform,
      apiKey,
      settings: apiSettings,
    });

    if (response.data && response.data.success) {
      return response.data.valid === true;
    }
    return false;
  } catch (error: any) {
    console.error(`验证 ${platform} Key失败:`, error);
    return false;
  }
};

const handleVerifyApiKeys = async () => {
  if (!apiClient.value) {
    await initApiClient();
  }
  if (!apiClient.value) {
    ElMessage.error('API客户端未初始化');
    return;
  }

  const verifyPromises: Promise<void>[] = [];
  const verifyResults: Record<string, { status: string; valid?: boolean; error?: string }> = {};

  Object.entries(apiConfigs).forEach(([platform, config]) => {
    const hasKey =
      platform === 'fofa'
        ? !!(config.email && config.email.trim() !== '' && config.key && config.key.trim() !== '')
        : !!(config.key && config.key.trim() !== '');

    if (hasKey) {
      verifyResults[platform] = { status: 'verifying' };
      const apiKey = platform === 'fofa' ? `${config.email}:${config.key}` : config.key;
      verifyPromises.push(
        verifySingleKey(platform, apiKey)
          .then((valid) => {
            verifyResults[platform] = { status: valid ? 'valid' : 'invalid', valid };
            apiConfigs[platform].status = valid ? 'valid' : 'invalid';
          })
          .catch((err: any) => {
            verifyResults[platform] = { status: 'invalid', error: err.message };
            apiConfigs[platform].status = 'invalid';
          })
      );
    }
  });

  if (verifyPromises.length === 0) {
    ElMessage.warning('没有配置任何API Key');
    return;
  }

  ElMessage.info(`正在验证 ${verifyPromises.length} 个API Key...`);

  await Promise.all(verifyPromises);

  const validCount = Object.values(verifyResults).filter((r) => r.status === 'valid').length;
  const invalidCount = Object.values(verifyResults).filter((r) => r.status === 'invalid').length;

  ElMessage.success(`验证完成：有效 ${validCount} 个，无效 ${invalidCount} 个`);
};

const openApiConfig = async () => {
  await loadApiConfig();
  showApiConfigDialog.value = true;
};

function saveConfig() {
  disabled.value = true;

  const configData: Config = {
    Code: formData.value.Code,
    Error: formData.value.Error,
    FilePath: formData.value.FilePath,
    CoroutineCount: formData.value.CoroutineCount,
    Timeout: formData.value.Timeout.toString(),
    SocksAddress: formData.value.SocksAddress,
    Status: formData.value.Status,
    // 便携代理不再在此处保存空间测绘平台账号与秘钥，
    // 统一通过空间测绘模块的配置进行管理。
    Maxpage: formData.value.MaxPage.toString(),
    Xui: formData.value.Xui,
    LiveProxies: configState.getLiveProxies(),
    AllProxies: configState.getAllProxies(),
    LiveProxyLists: [] as string[],

    // 搜索语句配置
    FofaSocks5Query: formData.value.FofaSocks5Query,
    FofaXuiQuery: formData.value.FofaXuiQuery,
    HunterSocks5Query: formData.value.HunterSocks5Query,
    HunterXuiQuery: formData.value.HunterXuiQuery,
    QuakeSocks5Query: formData.value.QuakeSocks5Query,
    QuakeXuiQuery: formData.value.QuakeXuiQuery
  };

  // 调用 store 的 saveConfig 方法
  configState.saveConfig(configData)
    .then(() => {
      ElMessage.success("配置已保存！");
    })
    .catch((err) => {
      ElMessage.error(err.message);
    })
    .finally(() => {
      disabled.value = false;
    });
}

onMounted(() => {
  configState.getProfile();
  // 初始化时同步加载空间测绘 API 配置，保证“搜索使用的平台”有数据可选
  loadApiConfig();
});
</script>

<template>
  <el-row :gutter="12">
    <el-col :span="14">
      <el-input readonly placeholder="配置文件在EasyToolsFiles/config/proxy_config.json文件中" />
    </el-col>
    <el-col :span="10">
      <el-button style="width: 100%" @click="saveConfig">保存配置</el-button>
    </el-col>
  </el-row>

  <br />

  <el-tabs v-model="activeTab">
    <el-tab-pane label="基础配置" name="basic">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px;">
        <p class="config-viewer-title" style="margin: 0;">
          说明：「空间测绘」模块统一配置FOFA / Hunter / Quake 等空间测绘平台的账号与秘钥。
        </p>
        <el-button @click="openApiConfig">API配置</el-button>
      </div>
      <el-form ref="formRef" :model="formData" :rules="rules" label-position="top">

        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item>
              <template #label>
                <span style="color: #e6a23c;">搜索使用的平台（仅显示已启用的 API 平台）</span>
              </template>
              <el-select
                v-model="selectedPlatforms"
                multiple
                clearable
                collapse-tags
                collapse-tags-tooltip
                placeholder="请选择用于搜索的空间测绘平台"
                style="width: 100%"
              >
                <el-option
                  v-for="opt in availablePlatforms"
                  :key="opt.value"
                  :label="opt.label"
                  :value="opt.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="超时时间" prop="Timeout">
              <el-input-number v-model="formData.Timeout" :min="1" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="采集最大页数" prop="MaxPage">
              <el-input-number v-model="formData.MaxPage" :min="1" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="协程数量" prop="CoroutineCount">
              <el-input-number v-model="formData.CoroutineCount" :min="1" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="白嫖Xui(开启后只会抓取xui数据,点击使用只会保存excel,请手动打开excel进行调用)" prop="Xui">
              <el-select v-model="formData.Xui" placeholder="请选择是否开启Xui抓取" clearable style="width: 100%">
                <el-option label="否" value="0" />
                <el-option label="是（不添加http代理）" value="1" />
                <el-option label="是（添加http代理）" value="2" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Socks地址" prop="SocksAddress">
              <el-input v-model="formData.SocksAddress" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-tab-pane>

    <el-tab-pane label="搜索语句配置" name="search">
      <p class="config-viewer-title">搜索语句配置：</p>
      <el-form :model="formData" label-position="top">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="Fofa Socks5查询语句">
              <el-input v-model="formData.FofaSocks5Query" type="textarea" placeholder="请输入Fofa Socks5查询语句" :rows="3" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Fofa Xui查询语句">
              <el-input v-model="formData.FofaXuiQuery" type="textarea" placeholder="请输入Fofa Xui查询语句" :rows="3" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="Hunter Socks5查询语句">
              <el-input v-model="formData.HunterSocks5Query" type="textarea" placeholder="请输入Hunter Socks5查询语句" :rows="3" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Hunter Xui查询语句">
              <el-input v-model="formData.HunterXuiQuery" type="textarea" placeholder="请输入Hunter Xui查询语句" :rows="3" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="Quake Socks5查询语句">
              <el-input v-model="formData.QuakeSocks5Query" type="textarea" placeholder="请输入Quake Socks5查询语句" :rows="3" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Quake Xui查询语句">
              <el-input v-model="formData.QuakeXuiQuery" type="textarea" placeholder="请输入Quake Xui查询语句" :rows="3" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-tab-pane>
  </el-tabs>

  <!-- 空间测绘 API 配置对话框（样式与 SpaceMapping 统一） -->
  <el-dialog v-model="showApiConfigDialog" title="API Key配置 (与空间测绘接口共用所有key)" width="800px"
    :close-on-click-modal="false">
    <div v-if="loadingApiConfig" style="text-align: center; padding: 24px 0;">
      <el-icon class="is-loading">
        <Loading />
      </el-icon>
      <div style="margin-top: 8px;">正在加载配置...</div>
    </div>
    <div v-else class="api-config-content">
      <div
        v-for="(config, key) in apiConfigs"
        :key="key"
        class="api-config-item"
      >
        <div class="api-config-header">
          <span class="api-name">{{ getPlatformLabel(key) }}</span>
          <el-checkbox v-model="config.enabled" label="启用" />
        </div>
        <div class="api-config-body">
          <!-- FOFA官方需要邮箱和key，放在同一行 -->
          <template v-if="key === 'fofa'">
            <div class="fofa-input-row">
              <el-input
                v-model="config.email"
                type="text"
                placeholder="请输入FOFA官方邮箱"
                class="api-key-input fofa-email-input"
              />
              <el-input
                v-model="config.key"
                :type="config.showKey ? 'text' : 'password'"
                placeholder="请输入FOFA官方 API Key"
                class="api-key-input fofa-key-input"
              >
                <template #suffix>
                  <el-icon
                    class="eye-icon"
                    @click="config.showKey = !config.showKey"
                  >
                    <View v-if="!config.showKey" />
                    <Hide v-else />
                  </el-icon>
                </template>
              </el-input>
            </div>
          </template>
          <!-- 其他平台只需要key，FOFA第三方额外支持 BaseURL -->
          <template v-else>
            <!-- FOFA第三方 BaseURL + Key 一行显示 -->
            <template v-if="key === 'otherfofa'">
              <div class="fofa-input-row">
                <el-input
                  v-model="apiSettings.fofaThirdBaseURL"
                  type="text"
                  placeholder="FOFA第三方 BaseURL，例如：https://fofa.red"
                  class="api-key-input fofa-baseurl-input"
                />
                <el-input
                  v-model="config.key"
                  :type="config.showKey ? 'text' : 'password'"
                  :placeholder="`请输入${getPlatformLabel(key)} API Key`"
                  class="api-key-input"
                >
                  <template #suffix>
                    <el-icon
                      class="eye-icon"
                      @click="config.showKey = !config.showKey"
                    >
                      <View v-if="!config.showKey" />
                      <Hide v-else />
                    </el-icon>
                  </template>
                </el-input>
              </div>
            </template>
            <template v-else>
              <el-input
                v-model="config.key"
                :type="config.showKey ? 'text' : 'password'"
                :placeholder="`请输入${getPlatformLabel(key)} API Key`"
                class="api-key-input"
              >
                <template #suffix>
                  <el-icon
                    class="eye-icon"
                    @click="config.showKey = !config.showKey"
                  >
                    <View v-if="!config.showKey" />
                    <Hide v-else />
                  </el-icon>
                </template>
              </el-input>
            </template>
          </template>
          <el-tag
            :type="config.status === 'valid' ? 'success' : config.status === 'invalid' ? 'danger' : 'info'"
            size="small"
            class="api-status"
          >
            {{ config.status === 'valid' ? '有效' : config.status === 'invalid' ? '无效' : '未知' }}
          </el-tag>
          <el-button
            type="primary"
            size="small"
            link
            @click="openApiApplyPage(key)"
          >
            API申请
          </el-button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="showApiConfigDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveApiConfig">保存</el-button>
        <el-button type="warning" @click="handleVerifyApiKeys">Key可用验证</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.config-viewer-title {
  text-align: left;
  color: #16c7d3;
  font-weight: bold;
  font-size: 16px;
  margin-bottom: 16px;
}

/* API配置对话框样式，参考 SpaceMappingView */
.api-config-content {
  /* 限制对话框内容高度，避免撑开页面导致外层出现滚动条 */
  max-height: min(500px, calc(100vh - 320px));
  overflow-y: auto;
}

.api-config-item {
  margin-bottom: 16px;
  padding: 18px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #ffffff;
  transition: all 0.2s ease;
}

.api-config-item:hover {
  border-color: #67C23A;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.1);
}

.api-config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.api-name {
  font-weight: bold;
}

.api-config-body {
  display: flex;
  align-items: center;
  gap: 10px;
}

.fofa-input-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  white-space: nowrap;
}

.fofa-email-input,
.fofa-key-input,
.fofa-baseurl-input {
  min-width: 0;
}

.api-key-input {
  flex: 1;
}

.eye-icon {
  cursor: pointer;
}

.api-status {
  min-width: 60px;
  text-align: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>