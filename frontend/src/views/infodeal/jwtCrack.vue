<template>
  <div class="jwt-page-wrapper">
    <el-row :gutter="20">
    <!-- 左侧：JWT Token 输入和解析结果 -->
    <el-col :span="12">
      <!-- JWT Token 输入卡片 -->
      <el-card class="jwt-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>
              JWT Token
              <el-button type="success" size="small" @click="copyToClipboard(jwtInput, 'Token')">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="jwtInput = ''">
                <el-icon><Delete /></el-icon>
              </el-button>
            </span>
            <el-tag
                v-if="jwtInput"
                :type="jwtverify ? 'success' : 'danger'"
                size="small"
            >
              {{ getVerifyStatusText() }}
            </el-tag>
          </div>
        </template>
        <div class="input-with-actions">
          <el-input
              type="textarea"
              :rows="6"
              placeholder="请输入JWT Token..."
              v-model="jwtInput"
              class="jwt-input large-font"
              resize="none"
              @input="onJWTInputChange"
          >
          </el-input>
        </div>
      </el-card>

      <!-- JWT 结构解析卡片 -->
      <el-card class="jwt-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>JWT 结构解析</span>
          </div>
        </template>

        <!-- Header -->
        <div class="jwt-section">
          <div class="section-header">
            <h5 class="section-title">
              Header
              <el-button type="success" size="small" @click="copyToClipboard(jwtheader, 'Header')">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="jwtheader = ''">
                <el-icon><Delete /></el-icon>
              </el-button>
            </h5>
          </div>
          <el-input
              type="textarea"
              :rows="3"
              placeholder="JWT Header 内容"
              v-model="jwtheader"
              class="section-input large-font"
              resize="none"
          >
          </el-input>
        </div>

        <!-- Payload -->
        <div class="jwt-section">
          <div class="section-header">
            <h5 class="section-title">
              Payload
              <el-button type="success" size="small" @click="copyToClipboard(jwtpayload, 'Payload')">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="jwtpayload = ''">
                <el-icon><Delete /></el-icon>
              </el-button>
            </h5>
          </div>
          <el-input
              type="textarea"
              :rows="5"
              placeholder="JWT Payload 内容（可编辑）"
              v-model="jwtpayload"
              class="section-input large-font editable"
              resize="none"
          >
          </el-input>
        </div>

        <!-- Signature -->
        <div class="jwt-section">
          <div class="section-header">
            <h5 class="section-title">
              Signature
              <el-button type="success" size="small" @click="copyToClipboard(jwtsignature, 'Signature')">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="jwtsignature = ''">
                <el-icon><Delete /></el-icon>
              </el-button>
            </h5>
          </div>
          <el-input
              type="textarea"
              :rows="2"
              placeholder="JWT Signature 内容"
              v-model="jwtsignature"
              readonly
              class="section-input large-font"
              resize="none"
          >
          </el-input>
        </div>
      </el-card>

      <!-- 快速操作卡片 -->
      <el-card class="jwt-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>快速操作</span>
          </div>
        </template>
        <div class="quick-actions">
          <el-button type="info" @click="clearAll" class="quick-btn">
            <el-icon><Close /></el-icon>
            清空所有
          </el-button>
          <el-button type="primary" @click="loadExample" class="quick-btn">
            <el-icon><Collection /></el-icon>
            加载示例
          </el-button>
        </div>
      </el-card>
    </el-col>

    <!-- 右侧：操作和配置区域 -->
    <el-col :span="12">
      <!-- JWT 操作卡片 -->
      <el-card class="jwt-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>JWT 操作</span>
          </div>
        </template>

        <!-- 算法选择 -->
        <div class="config-item">
          <h5 class="config-label">算法选择</h5>
          <el-select v-model="value" placeholder="请选择JWT算法" class="full-width large-font">
            <el-option
                v-for="item in options"
                :key="item.value"
                :label="item.label"
                :value="item.value">
            </el-option>
          </el-select>
        </div>

        <!-- 密钥输入 -->
        <div class="config-item">
          <div class="config-header">
            <h5 class="config-label">
              JWT 密钥
              <el-button type="success" size="small" @click="copyToClipboard(jwtsecret, 'JWT密钥')">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="jwtsecret = ''">
                <el-icon><Delete /></el-icon>
              </el-button>
            </h5>
          </div>
          <el-input
              type="textarea"
              :rows="2"
              placeholder="请输入JWT密钥"
              v-model="jwtsecret"
              class="full-width large-font"
              resize="none"
          >
          </el-input>
        </div>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <el-button type="primary" @click="startdecode" class="action-btn">
            <el-icon><Search /></el-icon>
            解析 JWT
          </el-button>
          <el-button type="success" @click="startencode" class="action-btn">
            <el-icon><Edit /></el-icon>
            编码 JWT
          </el-button>
          <el-button type="warning" @click="formatJson" class="action-btn">
            <el-icon><MagicStick /></el-icon>
            格式化 JSON
          </el-button>
        </div>
      </el-card>

      <!-- JWT 破解卡片 -->
      <el-card class="jwt-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>JWT 密钥破解</span>
          </div>
        </template>

        <!-- 字典选择 -->
        <div class="config-item">
          <h5 class="config-label">JWT 字典</h5>
          <div class="file-selector">
            <el-input v-model="dictPath" placeholder="请选择字典文件路径" readonly class="file-input large-font" />
            <el-button type="primary" @click="chooseFile" class="select-btn">选择</el-button>
            <el-button type="success" @click="useDefaultDict" class="select-btn">使用默认字典</el-button>
          </div>
        </div>

        <!-- 破解按钮和进度 -->
        <div class="crack-section">
          <el-button
              type="warning"
              @click="startCrack"
              class="crack-btn"
              :loading="cracking"
          >
            <el-icon><Unlock /></el-icon>
            {{ cracking ? '破解中...' : '开始破解' }}
          </el-button>

          <!-- 进度条 -->
          <div class="progress-section" v-if="progress > 0">
            <el-progress
                :percentage="progress"
                :text-inside="true"
                :stroke-width="20"
                :status="progress === 100 ? 'success' : ''"
            />
            <p class="progress-text">破解进度: {{ progress }}%</p>
          </div>
        </div>
      </el-card>

      <!-- JWT 漏洞生成卡片 -->
      <el-card class="jwt-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>JWT 漏洞生成</span>
            <el-button
                type="danger"
                size="small"
                @click="generateFromCurrent"
                :disabled="!parsedPayload"
            >
              <el-icon><Warning /></el-icon>
              基于当前解析生成
            </el-button>
          </div>
        </template>

        <div class="vulnerability-section">
          <div class="config-item">
            <h5 class="config-label">选择漏洞类型</h5>
            <el-select
                v-model="selectedVulnType"
                placeholder="选择漏洞类型"
                class="full-width large-font"
                @change="onVulnTypeChange"
            >
              <el-option
                  v-for="item in vulnerabilityTypes"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
              >
              </el-option>
            </el-select>
          </div>

          <!-- 简单Payload修改选项 -->
          <div class="config-item" v-if="selectedVulnType === 5">
            <div class="config-header">
              <h5 class="config-label">Payload修改（可选）</h5>
            </div>
            <div class="simple-modifications">
              <div class="mod-row">
                <span class="mod-label">修改字段：</span>
                <el-input
                    v-model="modifyField"
                    placeholder="字段名，如：role"
                    class="field-input"
                    size="small"
                />
                <span class="mod-label">值为：</span>
                <el-input
                    v-model="modifyValue"
                    placeholder="新值，如：admin"
                    class="value-input"
                    size="small"
                />
              </div>
              <div class="mod-tips">
                <el-text type="info" size="small">
                  提示：仅修改指定的字段，保持其他字段不变
                </el-text>
              </div>
            </div>
          </div>

          <div class="vuln-description" v-if="currentVulnDescription">
            <el-alert
                :title="currentVulnDescription.title"
                :type="currentVulnDescription.type"
                :description="currentVulnDescription.desc"
                show-icon
                :closable="false"
            />
          </div>

          <div class="action-buttons">
            <el-button
                type="danger"
                @click="generateVulnerability"
                class="action-btn"
                :loading="generatingVuln"
                :disabled="!parsedPayload"
            >
              <el-icon><Warning /></el-icon>
              生成漏洞 JWT
            </el-button>

            <el-button
                type="warning"
                @click="generateAllVulnerabilities"
                class="action-btn"
                :disabled="!parsedPayload"
            >
              <el-icon><Collection /></el-icon>
              批量生成所有
            </el-button>
          </div>

          <div class="vuln-result" v-if="generatedVulnJWT">
            <div class="result-header">
              <h5>生成的漏洞 JWT:</h5>
              <el-tag
                  :type="generatedVulnStatus === 'dangerous' ? 'danger' : 'warning'"
                  size="small"
              >
                {{ generatedVulnStatus === 'dangerous' ? '高危漏洞' : '测试漏洞' }}
              </el-tag>
              <el-tag v-if="generatedVulnDetails?.dataSource === 'custom'" type="success" size="small">
                使用用户修改数据
              </el-tag>
            </div>
            <el-input
                type="textarea"
                :rows="3"
                v-model="generatedVulnJWT"
                readonly
                class="vuln-jwt-input"
            >
            </el-input>

            <div class="vuln-actions">
              <el-button type="success" @click="copyToClipboard(generatedVulnJWT, '漏洞JWT')">
                复制 JWT
              </el-button>
              <el-button type="primary" @click="applyVulnJWT">
                应用到输入框
              </el-button>
              <el-button type="info" @click="testVulnerability" :disabled="!jwtsecret">
                测试漏洞
              </el-button>
            </div>

            <div class="vuln-details" v-if="generatedVulnDetails">
              <el-collapse>
                <el-collapse-item title="查看详情">
                  <div class="detail-content">
                    <h6>Header:</h6>
                    <pre>{{ generatedVulnDetails.header }}</pre>
                    <h6>Payload:</h6>
                    <pre>{{ generatedVulnDetails.payload }}</pre>
                    <div v-if="generatedVulnDetails.dataSource" class="modified-status">
                      <h6>数据来源:</h6>
                      <p>{{ generatedVulnDetails.dataSource === 'custom' ? '使用用户修改的数据' : '使用原始JWT解析的数据' }}</p>
                    </div>
                    <h6>测试说明:</h6>
                    <p>{{ generatedVulnDetails.description }}</p>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
        </div>
      </el-card>
    </el-col>
    </el-row>
  </div>
</template>

<script>
import {
  Close,
  Collection,
  CopyDocument,
  Delete,
  Edit,
  InfoFilled,
  MagicStick,
  Search,
  Unlock,
  Warning
} from "@element-plus/icons-vue";
import {
  BruteForceJWT,
  ChooseJwtFile,
  DecodeJWTWithAlg,
  EncodeJWTWithAlg,
  GetDefaultDictPath,
  GenerateVulnerableJWT,
  GenerateAllVulnerabilities,
  TestVulnerability
} from "../../../wailsjs/go/jwt/JwtCrackController";
import { ElMessage, ElMessageBox } from "element-plus";
export default {
  name: "JwtCrack",
  components: {
    Collection,
    Close,
    Unlock,
    MagicStick,
    Edit,
    Search,
    Delete,
    CopyDocument,
    Warning,
    InfoFilled
  },
  data() {
    return {
      // jwt相关
      jwtInput: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiRWFzeVRvb2xzIiwidXJsIjoiaHR0cHM6Ly9naXRodWIuY29tL2Rva2ktYnl0ZS9FYXN5VG9vbHMifQ.s0yuzN2oyhbUQl4Zrgg7vk9tvJB5hsDt0EBEfqoOeog",
      jwtheader: "",
      jwtpayload: "",
      jwtsecret: "",
      jwtsignature: "",
      jwtverify: true,
      progress: 0,
      cracking: false,
      dictPath: '选择jwt字典路径',

      // 算法选项
      options: [
        { value: 'HS256', label: 'HS256' },
        { value: 'HS384', label: 'HS384' },
        { value: 'HS512', label: 'HS512' },
        { value: 'RS256', label: 'RS256' },
        { value: 'RS384', label: 'RS384' },
        { value: 'RS512', label: 'RS512' },
        { value: 'ES256', label: 'ES256' },
        { value: 'ES384', label: 'ES384' },
        { value: 'ES512', label: 'ES512' },
        { value: 'PS256', label: 'PS256' },
        { value: 'PS384', label: 'PS384' },
        { value: 'PS512', label: 'PS512' },
        { value: 'EdDSA', label: 'EdDSA' },
        { value: 'None', label: 'None' }
      ],
      value: 'HS256',

      // 新增漏洞生成相关数据
      selectedVulnType: 1,
      generatingVuln: false,
      generatedVulnJWT: '',
      generatedVulnStatus: '',
      generatedVulnDetails: null,
      currentVulnDescription: null,
      modifyField: '',
      modifyValue: '',
      parsedPayload: null,
      parsedHeader: null,

      vulnerabilityTypes: [
        { value: 1, label: 'CVE-2015-2951 (alg=none)' },
        { value: 2, label: 'CVE-2015-2951 (alg=None)' },
        { value: 3, label: 'CVE-2015-2951 (alg=NoNe)' },
        { value: 4, label: 'CVE-2015-2951 (alg=NONE)' },
        { value: 5, label: '未验证签名攻击' },
        { value: 6, label: 'CVE-2018-0114 JWKS注入' },
        { value: 7, label: 'CVE-2020-28042 空签名攻击' }
      ],
      vulnDescriptions: {
        1: {
          title: 'alg=none 攻击',
          type: 'error',
          desc: '将JWT的算法设置为"none"，跳过签名验证。Payload保持原样。'
        },
        2: {
          title: 'alg=None 攻击',
          type: 'error',
          desc: '大小写变种，算法设置为"None"。Payload保持原样。'
        },
        3: {
          title: 'alg=NoNe 攻击',
          type: 'error',
          desc: '大小写变种，算法设置为"NoNe"。Payload保持原样。'
        },
        4: {
          title: 'alg=NONE 攻击',
          type: 'error',
          desc: '大小写变种，算法设置为"NONE"。Payload保持原样。'
        },
        5: {
          title: '未验证签名攻击',
          type: 'warning',
          desc: '修改Payload中的指定字段，但不重新签名。其他字段保持不变。'
        },
        6: {
          title: 'JWKS注入攻击',
          type: 'error',
          desc: '通过kid指向恶意JWKS端点，伪造JWT签名。Payload保持原样。'
        },
        7: {
          title: '空签名攻击',
          type: 'warning',
          desc: 'JWT签名部分为空，某些实现可能错误接受。Payload保持原样。'
        }
      },
      mitigationTips: {
        1: '1. 验证JWT时强制检查alg字段\n2. 拒绝alg为none的JWT\n3. 使用安全的JWT库',
        2: '1. 验证JWT时强制检查alg字段\n2. 拒绝alg为none的JWT\n3. 使用安全的JWT库',
        3: '1. 验证JWT时强制检查alg字段\n2. 拒绝alg为none的JWT\n3. 使用安全的JWT库',
        4: '1. 验证JWT时强制检查alg字段\n2. 拒绝alg为none的JWT\n3. 使用安全的JWT库',
        5: '1. 始终验证JWT签名\n2. 使用强密钥\n3. 定期轮换密钥\n4. 验证Payload完整性',
        6: '1. 严格验证JWKS来源\n2. 使用固定的可信密钥\n3. 禁用动态JWKS获取',
        7: '1. 验证签名长度\n2. 拒绝空签名\n3. 更新JWT库到最新版本'
      }
    };
  },
  mounted() {
    // 初始化漏洞描述
    this.onVulnTypeChange();
    // 自动解析示例JWT
    this.$nextTick(() => {
      this.startdecode();
    });
  },
  computed: {
    canGenerateVuln() {
      // 检查是否可以生成漏洞，需要已解析的payload
      return this.parsedPayload !== null;
    },

    // 检查漏洞5是否有有效修改
    hasValidModification() {
      return this.selectedVulnType === 5 && this.modifyField && this.modifyValue;
    }
  },
  watch: {
    // 监听Header输入框变化
    jwtheader: {
      handler(newVal) {
        if (newVal && newVal.trim()) {
          try {
            const parsed = JSON.parse(newVal);
            this.parsedHeader = parsed;
          } catch (e) {
            console.warn('Header JSON格式错误:', e);
          }
        } else {
          this.parsedHeader = null;
        }
      },
      immediate: true
    },

    // 监听Payload输入框变化
    jwtpayload: {
      handler(newVal) {
        if (newVal && newVal.trim()) {
          try {
            const parsed = JSON.parse(newVal);
            this.parsedPayload = parsed;
          } catch (e) {
            console.warn('Payload JSON格式错误:', e);
          }
        } else {
          this.parsedPayload = null;
        }
      },
      immediate: true
    }
  },
  methods: {
    // JWT输入变化时的处理
    onJWTInputChange() {
      // 当JWT输入变化时，清空生成的结果
      if (this.generatedVulnJWT && this.jwtInput !== this.generatedVulnJWT) {
        this.generatedVulnJWT = '';
        this.generatedVulnStatus = '';
        this.generatedVulnDetails = null;
      }
    },

    // jwt解析
    async useDefaultDict() {
      try {
        const defaultPath = await GetDefaultDictPath();
        if (defaultPath) {
          this.dictPath = defaultPath;
          ElMessage.success("已使用默认字典路径");
        } else {
          ElMessage.warning("未找到默认字典文件");
        }
      } catch (error) {
        console.error("获取默认字典路径失败:", error);
        ElMessage.error("获取默认字典路径失败");
      }
    },

    // 修改 chooseFile 方法，添加自动解析
    chooseFile() {
      ChooseJwtFile()
          .then((path) => {
            if (path) {
              this.dictPath = path;
              ElMessage.success("字典文件选择成功");

              // 如果已经有JWT输入，自动开始破解
              if (this.jwtInput && this.jwtInput.trim()) {
                this.$confirm('是否立即开始JWT破解？', '提示', {
                  confirmButtonText: '开始破解',
                  cancelButtonText: '稍后',
                  type: 'info'
                }).then(() => {
                  this.startCrack();
                });
              }
            } else {
              ElMessage.error("未选择任何文件");
            }
          })
          .catch((error) => {
            console.error("选择文件失败:", error);
            ElMessage.error("选择文件失败，请重试");
          });
    },

    // 修改 startCrack 方法，添加自动使用默认字典
    async startCrack() {
      // 如果没有选择字典，尝试使用默认字典
      if (!this.dictPath || this.dictPath === '选择jwt字典路径') {
        try {
          const defaultPath = await GetDefaultDictPath();
          if (defaultPath) {
            this.dictPath = defaultPath;
            ElMessage.info("已自动使用默认字典");
          } else {
            ElMessage.warning("请先选择字典文件或确保默认字典存在");
            return;
          }
        } catch (error) {
          ElMessage.error("无法找到默认字典，请手动选择字典文件");
          return;
        }
      }

      this.cracking = true;
      this.progress = 0;

      BruteForceJWT(this.jwtInput, this.value, this.dictPath)
          .then((result) => {
            this.cracking = false;
            if (!result.error) {
              this.jwtsecret = result.secret || "";
              this.jwtsignature = result.signature || "";
              this.jwtheader = JSON.stringify(result.header, null, 2);
              this.jwtpayload = JSON.stringify(result.payload, null, 2);
              this.parsedPayload = result.payload;
              this.parsedHeader = result.header;
              ElMessage.success("JWT 密钥破解成功");
              this.progress = 100;
              this.jwtverify = true;
              return;
            }
            this.progress = 100;
            this.jwtverify = false;
            ElMessage.error(result.error);
          })
          .catch((error) => {
            this.cracking = false;
            console.error("JWT 密钥破解失败:", error);
            ElMessage.error(error.message || "JWT 密钥破解失败");
          });
    },

    // 复制到剪贴板
    copyToClipboard(text, type = '内容') {
      if (!text || text.trim() === '') {
        ElMessage.warning(`没有可复制的${type}内容`);
        return;
      }

      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success(`${type}已复制到剪贴板`);
      }).catch(err => {
        console.error('复制失败:', err);
        // 备用方案
        const textArea = document.createElement('textarea');
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        try {
          document.execCommand('copy');
          ElMessage.success(`${type}已复制到剪贴板`);
        } catch (e) {
          ElMessage.error('复制失败，请手动复制');
        }
        document.body.removeChild(textArea);
      });
    },

    // 格式化JSON
    formatJson() {
      try {
        if (this.jwtheader) {
          const headerObj = JSON.parse(this.jwtheader);
          this.jwtheader = JSON.stringify(headerObj, null, 2);
          this.parsedHeader = headerObj;
        }
        if (this.jwtpayload) {
          const payloadObj = JSON.parse(this.jwtpayload);
          this.jwtpayload = JSON.stringify(payloadObj, null, 2);
          this.parsedPayload = payloadObj;
        }
        ElMessage.success('JSON格式化完成');
      } catch (error) {
        ElMessage.error('JSON格式错误，无法格式化');
      }
    },

    async clearAll() {
      try {
        await this.$confirm('确定要清空所有 JWT 相关内容吗？', '清空内容', { type: 'warning' });
      } catch {
        return;
      }
      this.jwtInput = '';
      this.jwtheader = '';
      this.jwtpayload = '';
      this.jwtsecret = '';
      this.jwtsignature = '';
      this.dictPath = '选择jwt字典路径';
      this.progress = 0;
      this.generatedVulnJWT = '';
      this.modifyField = '';
      this.modifyValue = '';
      this.parsedPayload = null;
      this.parsedHeader = null;
      this.generatedVulnDetails = null;
      ElMessage.success('已清空所有内容');
    },

    // 加载示例数据
    loadExample() {
      this.jwtInput = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiRWFzeVRvb2xzIiwidXJsIjoiaHR0cHM6Ly9naXRodWIuY29tL2Rva2ktYnl0ZS9FYXN5VG9vbHMifQ.s0yuzN2oyhbUQl4Zrgg7vk9tvJB5hsDt0EBEfqoOeog';
      this.jwtheader = JSON.stringify({
        alg: "HS256",
        typ: "JWT"
      }, null, 2);
      this.jwtpayload = JSON.stringify({
        name: "EasyTools",
        url: "https://github.com/doki-byte/EasyTools"
      }, null, 2);
      this.jwtsecret = "";
      this.parsedPayload = { name: "EasyTools", url: "https://github.com/doki-byte/EasyTools" };
      this.parsedHeader = { alg: "HS256", typ: "JWT" };
      ElMessage.info('已加载示例数据');
    },

    // 编码JWT（必须先验证秘钥）
    async startencode() {
      // 检查必要条件
      if (!this.jwtInput || !this.jwtInput.trim()) {
        ElMessage.warning('请先输入要验证的JWT Token');
        return;
      }

      if (!this.jwtsecret || this.jwtsecret.trim() === '') {
        ElMessage.warning('请先设置JWT秘钥');
        return;
      }

      let headerObj = {};
      let payloadObj = {};

      try {
        // 使用当前解析的值
        if (this.parsedHeader) {
          headerObj = this.parsedHeader;
        } else if (this.jwtheader && this.jwtheader.trim()) {
          headerObj = JSON.parse(this.jwtheader);
        } else {
          ElMessage.warning('请提供有效的Header');
          return;
        }

        // 使用当前解析的值
        if (this.parsedPayload) {
          payloadObj = this.parsedPayload;
        } else if (this.jwtpayload && this.jwtpayload.trim()) {
          payloadObj = JSON.parse(this.jwtpayload);
        } else {
          ElMessage.warning('请提供有效的Payload');
          return;
        }

      } catch (error) {
        ElMessage.error('Header或Payload不是有效的JSON格式');
        return;
      }

      try {
        const result = await EncodeJWTWithAlg(this.value, this.jwtsecret, payloadObj, headerObj);
        if (!result.error) {
          this.jwtInput = result.jwt_token || "";
          this.jwtsignature = result.signature || "";
          ElMessage.success("JWT编码成功");
        } else {
          ElMessage.error(result.error);
        }
      } catch (error) {
        ElMessage.error("JWT编码失败: " + error.message);
      }
    },

    // 解析JWT
    async startdecode() {
      if (!this.jwtInput || !this.jwtInput.trim()) {
        ElMessage.warning('请先输入JWT Token');
        return;
      }

      try {
        const testSecret = this.jwtsecret;

        const result = await DecodeJWTWithAlg(this.jwtInput, this.value, testSecret);

        if (!result.error) {
          this.jwtheader = JSON.stringify(result.header, null, 2);
          this.jwtpayload = JSON.stringify(result.payload, null, 2);
          this.jwtsignature = result.signature || "";
          this.parsedPayload = result.payload;
          this.parsedHeader = result.header;

          // 根据验证结果设置状态
          this.jwtverify = result.valid;

          if (result.valid) {
            ElMessage.success("JWT解析成功 - 签名验证通过");
          } else {
            ElMessage.warning("JWT解析完成，但签名验证失败 - 需要正确秘钥");
          }
        } else {
          // 解析出错，可能是格式错误
          ElMessage.error("JWT解析失败: " + result.error);
          this.jwtverify = false;
          this.parsedPayload = null;
          this.parsedHeader = null;
        }
      } catch (error) {
        ElMessage.error(error.message || "JWT 解析失败");
        this.jwtverify = false;
        this.parsedPayload = null;
        this.parsedHeader = null;
      }
    },

    // 添加获取校验状态文本的方法
    getVerifyStatusText() {
      if (!this.jwtInput) return '';

      if (this.jwtverify) {
        return this.jwtsecret && this.jwtsecret.trim() ? '校验通过' : '无签名验证';
      } else {
        return this.jwtsecret && this.jwtsecret.trim() ? '校验失败' : '需要秘钥';
      }
    },

    // 漏洞类型改变
    onVulnTypeChange() {
      this.currentVulnDescription = this.vulnDescriptions[this.selectedVulnType];

      // 清空生成的JWT
      this.generatedVulnJWT = '';
      this.generatedVulnStatus = '';
      this.generatedVulnDetails = null;
    },

    // 基于当前解析生成
    generateFromCurrent() {
      if (!this.parsedPayload) {
        ElMessage.warning('请先解析JWT');
        return;
      }

      // 自动生成漏洞JWT
      this.generateVulnerability();
    },

    // 生成漏洞Payload - 修复版：使用用户修改后的最新值
    async generateVulnerability() {
      // 使用当前解析的值
      if (!this.parsedPayload) {
        ElMessage.warning('请先解析JWT或提供Payload数据');
        return;
      }

      if (!this.selectedVulnType) {
        ElMessage.warning('请选择漏洞类型');
        return;
      }

      this.generatingVuln = true;

      try {
        // 使用当前最新的Payload和Header
        let customPayload = { ...this.parsedPayload };
        let customHeader = this.parsedHeader ? { ...this.parsedHeader } : {};

        // 漏洞1-4: alg=none变种攻击
        if (this.selectedVulnType >= 1 && this.selectedVulnType <= 4) {
          const algVariants = ['none', 'None', 'NoNe', 'NONE'];
          customHeader.alg = algVariants[this.selectedVulnType - 1];
          // 保持Payload原样
        }

        // 漏洞5: 未验证签名攻击 - 只修改用户指定的字段
        else if (this.selectedVulnType === 5) {
          if (this.modifyField && this.modifyValue) {
            // 只修改用户指定的字段，其他字段保持原样
            customPayload[this.modifyField] = this.modifyValue;
          } else {
            // 如果没有指定修改，保持原样
            ElMessage.info('未指定修改字段，将生成与原Payload相同的JWT');
          }
          // 保持Header原样，不修改算法
        }

        // 漏洞6: JWKS注入攻击
        else if (this.selectedVulnType === 6) {
          customHeader.kid = "http://evil.com/jwks.json";
          customHeader.alg = "RS256";
          // 保持Payload原样
        }

        // 漏洞7: 空签名攻击
        else if (this.selectedVulnType === 7) {
          customHeader.alg = "HS256";
          // 保持Payload原样
        }

        // 调用后端生成漏洞JWT
        const result = await GenerateVulnerableJWT(
            this.selectedVulnType,
            this.jwtInput, // 原始JWT
            customPayload,
            customHeader
        );

        if (result.error) {
          ElMessage.error('生成失败: ' + result.error);
        } else {
          this.generatedVulnJWT = result.jwt || "";
          this.generatedVulnStatus = result.severity || 'dangerous';
          this.generatedVulnDetails = {
            header: JSON.stringify(result.header || customHeader, null, 2),
            payload: JSON.stringify(result.payload || customPayload, null, 2),
            description: result.test_vector || this.vulnDescriptions[this.selectedVulnType].desc,
            dataSource: result.data_source || 'custom'
          };

          this.currentVulnDescription = {
            title: result.description || this.vulnDescriptions[this.selectedVulnType].title,
            type: 'success',
            desc: result.test_vector || this.vulnDescriptions[this.selectedVulnType].desc
          };

          ElMessage.success('漏洞JWT生成成功');
        }
      } catch (error) {
        console.error('生成漏洞失败:', error);
        ElMessage.error('生成失败: ' + error.message);
      } finally {
        this.generatingVuln = false;
      }
    },

    // 批量生成所有漏洞
    async generateAllVulnerabilities() {
      if (!this.parsedPayload) {
        ElMessage.warning('请先解析JWT');
        return;
      }

      try {
        const result = await GenerateAllVulnerabilities(
            this.jwtInput,
            this.parsedPayload,
            this.parsedHeader
        );

        if (result.error) {
          ElMessage.error('批量生成失败: ' + result.error);
          return;
        }

        // 显示所有生成的漏洞JWT
        let message = '批量生成结果:\n\n';
        let hasResults = false;

        for (let i = 1; i <= 7; i++) {
          const key = `vuln_${i}`;
          if (result[key] && result[key].jwt) {
            hasResults = true;
            message += `${this.vulnerabilityTypes[i-1].label}:\n`;
            message += `${result[key].jwt}\n\n`;
          }
        }

        if (hasResults) {
          ElMessageBox.alert(message, '所有漏洞JWT', {
            confirmButtonText: '确定',
            customClass: 'vulnerability-result-dialog',
            dangerouslyUseHTMLString: false,
            width: '80%'
          });
        } else {
          ElMessage.warning('未生成任何漏洞JWT');
        }
      } catch (error) {
        ElMessage.error('批量生成失败: ' + error.message);
      }
    },

    // 应用到输入框
    applyVulnJWT() {
      if (this.generatedVulnJWT) {
        this.jwtInput = this.generatedVulnJWT;

        // 自动解析新的JWT
        this.$nextTick(() => {
          this.startdecode();
        });

        ElMessage.success('漏洞JWT已应用到输入框并自动解析');
      } else {
        ElMessage.warning('请先生成漏洞JWT');
      }
    },

    // 测试漏洞
    async testVulnerability() {
      if (!this.generatedVulnJWT) {
        ElMessage.warning('请先生成漏洞JWT');
        return;
      }

      if (!this.jwtsecret) {
        ElMessage.warning('请先设置JWT密钥进行测试');
        return;
      }

      try {
        const result = await TestVulnerability(
            this.generatedVulnJWT,
            this.value,
            this.jwtsecret
        );

        ElMessageBox.alert(
            JSON.stringify(result, null, 2),
            '漏洞测试结果',
            {
              confirmButtonText: '确定',
              customClass: 'vulnerability-test-dialog',
              width: '80%'
            }
        );
      } catch (error) {
        ElMessage.error('测试失败: ' + error.message);
      }
    }
  }
};
</script>

<style scoped>
/* JWT 页面容器 - 防止横向滚动 */
.jwt-page-wrapper {
  overflow-x: hidden;
  width: 100%;
  box-sizing: border-box;
}

/* JWT 专用样式 */
.jwt-card {
  margin-bottom: 20px;
  border-radius: 8px;
  border: 1px solid #e6e8eb;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: bold;
  color: #303133;
}

.jwt-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}

.jwt-section {
  margin-bottom: 16px;
}

.jwt-section:last-child {
  margin-bottom: 0;
}

.section-title {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title .el-button {
  margin-left: 8px;
}

.section-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}

.config-item {
  margin-bottom: 16px;
}

.config-item:last-child {
  margin-bottom: 0;
}

.config-label {
  margin: 0 0 8px 0;
  font-size: 13px;
  color: #606266;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.config-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.full-width {
  width: 100%;
}

.file-selector {
  display: flex;
  gap: 10px;
}

.file-input {
  flex: 1;
}

.select-btn {
  flex-shrink: 0;
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.crack-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.crack-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.progress-section {
  text-align: center;
}

.progress-text {
  margin: 8px 0 0 0;
  font-size: 13px;
  color: #909399;
}

/* 添加大字体样式类 */
.large-font {
  font-size: 14px !important;
}

.large-font :deep(.el-textarea__inner) {
  font-size: 14px !important;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: #155adf;
}

.large-font :deep(.el-input__inner) {
  font-size: 14px !important;
}

/* 修改现有的JWT相关样式，确保字体大小生效 */
.jwt-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px !important;
}

.section-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px !important;
}

.full-width :deep(.el-textarea__inner) {
  font-size: 14px !important;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.file-input :deep(.el-input__inner) {
  font-size: 14px !important;
}

/* 修改文件选择器布局，适应新增按钮 */
.file-selector {
  display: flex;
  gap: 8px;
  align-items: center;
}

.file-input {
  flex: 1;
}

.select-btn {
  flex-shrink: 0;
  white-space: nowrap;
}

/* 快速操作按钮样式 */
.quick-actions {
  display: flex;
  gap: 10px;
}

.quick-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

/* 新增漏洞相关样式 */
.vulnerability-section {
  margin-top: 10px;
}

.simple-modifications {
  background-color: #f8f9fa;
  padding: 15px;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.mod-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.mod-label {
  font-size: 14px;
  color: #606266;
  white-space: nowrap;
}

.field-input {
  flex: 2;
}

.value-input {
  flex: 3;
}

.mod-tips {
  margin-top: 8px;
}

.vuln-description {
  margin: 15px 0;
}

.vuln-result {
  margin-top: 20px;
  padding: 15px;
  background-color: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.result-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.result-header h5 {
  margin: 0;
  color: #606266;
  font-weight: 600;
}

.vuln-jwt-input {
  margin: 10px 0;
}

.vuln-jwt-input :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  background-color: #fefefe;
  border: 1px solid #dcdfe6;
}

.vuln-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
  margin-bottom: 15px;
}

.vuln-details {
  margin-top: 15px;
}

.detail-content {
  background-color: #f9f9f9;
  padding: 10px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
}

.detail-content h6 {
  margin: 10px 0 5px 0;
  color: #333;
  font-weight: 600;
}

.detail-content pre {
  margin: 5px 0;
  padding: 8px;
  background-color: #f0f0f0;
  border-radius: 3px;
  overflow-x: auto;
}

.detail-content p {
  margin: 5px 0;
  color: #666;
}

.modified-status {
  background-color: #f0f9eb;
  border: 1px solid #e1f3d8;
  padding: 8px;
  border-radius: 4px;
  margin: 10px 0;
}

.modified-status h6 {
  color: #67C23A;
  margin: 0 0 5px 0;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .jwt-content {
    padding: 5px 0;
  }

  .action-buttons {
    flex-direction: column;
  }

  .file-selector {
    flex-direction: column;
  }

  .select-btn {
    width: 100%;
    margin-bottom: 8px;
  }

  .quick-actions {
    flex-direction: column;
  }

  .vuln-actions {
    flex-direction: column;
  }

  .vuln-actions .el-button {
    width: 100%;
  }

  .jwt-card {
    padding: 15px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .section-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .config-label {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .config-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .result-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .mod-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .field-input, .value-input {
    width: 100%;
  }
}

@media (max-width: 480px) {
  .jwt-card {
    margin-bottom: 15px;
  }
}

/* 对话框样式 */
:deep(.vulnerability-result-dialog) {
  max-width: 90%;
}

:deep(.vulnerability-test-dialog) {
  max-width: 90%;
}

:deep(.vulnerability-result-dialog .el-message-box__message) {
  white-space: pre-wrap;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 11px;
  max-height: 60vh;
  overflow-y: auto;
}

:deep(.vulnerability-test-dialog .el-message-box__message) {
  white-space: pre-wrap;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 11px;
  max-height: 60vh;
  overflow-y: auto;
}

/* 禁用状态按钮样式 */
.action-btn[disabled] {
  opacity: 0.6;
  cursor: not-allowed;
}

.quick-btn[disabled] {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 输入框禁用状态 */
:deep(.el-textarea.is-disabled .el-textarea__inner) {
  background-color: #f5f7fa;
  border-color: #e4e7ed;
  color: #c0c4cc;
  cursor: not-allowed;
}

/* 折叠面板样式 */
:deep(.el-collapse) {
  border: 1px solid #e4e7ed;
  border-radius: 4px;
}

:deep(.el-collapse-item__header) {
  background-color: #f5f7fa;
  padding: 0 15px;
  font-weight: 600;
}

:deep(.el-collapse-item__content) {
  padding: 15px;
}
</style>