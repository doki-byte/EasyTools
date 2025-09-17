<template>
  <el-container class="container">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="tabs">
      <el-tab-pane label="Fscan结果处理" name="fscan-deal" />
      <el-tab-pane label="蓝队大批量封禁IP处置" name="ip-ban-deal" />
      <el-tab-pane label="OSS存储桶遍历" name="oss-list" />
      <el-tab-pane label="WX小程序反编译" name="unwxapp" />
      <el-tab-pane label="JWT密钥破解" name="jwt_crack" />
    </el-tabs>

    <!-- 内容区域 -->
    <el-main>
      <div v-if="activeTab === 'fscan-deal'" class="tab-content">
        <div class="header">
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
          <el-button type="primary" @click="processFile">处理文件</el-button>
          <el-button type="success" @click="openExcelPreview" v-if="isDealFile">
            预览 Excel 文件
          </el-button>
          <p v-if="isDealFile">文件保存位置：{{ excelFilePath }}</p>
        </div>

        <!-- Excel 预览 -->
        <div v-if="sheetsData.length" class="excel-preview">
          <h5>生成的 Excel 文件预览：</h5>

          <!-- Tabs for sheet navigation -->
          <el-tabs v-model="activeSheet" class="tabs">
            <el-tab-pane v-for="(sheet, index) in sheetsData" :key="index" :label="sheet.sheetName"
                         :name="sheet.sheetName">
            </el-tab-pane>
          </el-tabs>

          <!-- 当前选中的表格 -->
          <div v-if="activeSheet">
            <table class="excel-preview-table">
              <thead>
              <tr>
                <th v-for="(col, index) in getActiveSheetData()[0]" :key="index">{{ col }}</th>
              </tr>
              </thead>
              <tbody>
              <tr v-for="(row, rowIndex) in paginatedData.slice((currentPage-1)*pageSize, currentPage*pageSize)" :key="rowIndex">
                <td v-for="(cell, colIndex) in row" :key="colIndex">{{ cell }}</td>
              </tr>
              </tbody>
            </table>

            <!-- 分页组件 -->
            <el-pagination
                v-if="paginatedData.length > pageSize"
                :current-page="currentPage"
                :page-size="pageSize"
                :total="paginatedData.length"
                @current-change="handlePageChange">
            </el-pagination>
          </div>
        </div>
      </div>

      <!-- 蓝队大批量IP封禁处置 -->
      <div v-if="activeTab === 'ip-ban-deal'" class="tab-content">
        <p class="info">
          请填写威胁情报 & 恶意IP列表（每行一个IP），以及选填白名单IP列表。系统将自动去重，并排除白名单内容，避免误封。
        </p>

        <!-- 上半部分：输入框 -->
        <div class="row upper-row">
          <div class="column">
            <h5>威胁情报 & 恶意IP</h5>
            <el-input type="textarea" v-model="maliciousIPInput" placeholder="请输入威胁情报 & 恶意IP (每行一个)"
                      :rows="6" class="input-box" />
          </div>
          <div class="column">
            <h5>IP白名单</h5>
            <el-input type="textarea" v-model="whiteListIPInput" placeholder="请输入IP白名单 (每行一个)" :rows="6"
                      class="input-box" />
          </div>
        </div>

        <!-- 下半部分：结果框 -->
        <div class="row lower-row">
          <div class="column">
            <h5>去重后IP (排除白名单)</h5>
            <div class="ip-input-container">
              <el-input type="textarea" :value="uniqueIPs.join('\n')" readonly placeholder="去重后IP (排除白名单)"
                        :rows="6" class="input-box readonly" />
              <el-button type="success" class="ip-copy-button" @click="IpCopyToClipboard('uniqueIPs')">
                复制
              </el-button>
            </div>
          </div>
          <div class="column">
            <h5>重复IP</h5>
            <div class="ip-input-container">
              <el-input type="textarea" :value="duplicateIPs.join('\n')" readonly placeholder="重复的IP"
                        :rows="6" class="input-box readonly" />
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

      <!--oss存储桶遍历-->
      <div v-if="activeTab === 'oss-list'" class="tab-content">
        <div class="header">
          <h4>选择 Oss资源桶 文件</h4>
          <div class="form-group">
            <el-input v-model="OsslistInput" placeholder="请输入Oss资源桶链接" class="input" />
            <el-button type="primary" @click="generateOssListQueries">获取数据</el-button>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="actions" v-if="OssListSuccess">
          <p> 文件保存位置：
            <span style="color: #4dcd31">{{ OssListSavePath }}</span>
          </p>
          <el-button type="success" @click="openFileDir">
            打开EasyToolsFiles文件夹
          </el-button>
        </div>
      </div>

      <!--wx小程序反编译-->
      <div v-if="activeTab === 'unwxapp'" class="tab-content">
        <div class="unwxapp-wrapper">
          <el-card shadow="hover" class="box-card">
            <template #header>
              <div class="card-header">
                <span>微信小程序反编译</span>
              </div>
            </template>

            <el-form label-width="120px" label-position="left">
              <el-form-item label="选择小程序包">
                <div style="display: flex; align-items: center; gap: 10px; width: 100%;">
                  <el-input v-model="wxpackages" readonly placeholder="请选择小程序文件或文件夹，旧版微信：C:\xxxxx\WeChat Files\Applet\wx93fde323axxxxxxxx\xxx" />
                  <el-tooltip
                      effect="dark"
                      content="新版微信：C:\Users\用户名\AppData\Roaming\Tencent\xwechat\radium\Applet\packages\wx93fde323axxxxxxxx\xxx"
                      placement="bottom-start"
                  >
                    <el-icon><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <el-button type="primary" class="ml-2" @click="selectPackage">选择</el-button>
              </el-form-item>

              <el-form-item label="AppID（可选）">
                <el-input v-model="wxappid" placeholder="留空将自动获取" />
              </el-form-item>

              <el-form-item label="格式化代码">
                <el-switch v-model="wxformat" />
              </el-form-item>

              <el-form-item>
                <el-button type="primary" @click="runUnWxapp">开始反编译</el-button>
                <el-button
                    type="success"
                    v-if="wxresult && wxoutputPath"
                    @click="openOutputFolder"
                >
                  打开文件夹
                </el-button>
              </el-form-item>
            </el-form>

            <el-divider>输出结果</el-divider>
            <el-input
                type="textarea"
                v-model="wxresult"
                rows="15"
                readonly
                resize="none"
                placeholder="运行结果将在此显示"
            />
          </el-card>
        </div>
      </div>

      <!--jwt密钥爆破-->
      <div v-if="activeTab === 'jwt_crack'" class="tab-content">
        <el-row :gutter="20">
          <el-col :span="11">
            <el-card class="box-card">
              <h4>JWT TOKEN</h4>
              <el-input
                  type="textarea"
                  :rows="12"
                  placeholder="请输入JWT内容"
                  v-model="jwtInput">
              </el-input>
              <h4>JWT Verify</h4>
              <el-input
                  type="textarea"
                  :rows="5"
                  placeholder="JWT校验内容"
                  v-model="jwtsignature">
              </el-input>
            </el-card>
            <div style="margin-top: 20px;">
              <el-card class="box-card">
                <h4>选择jwt字典</h4>
                <!-- 路径选择行 -->
                <div style="display: flex; align-items: center; gap: 10px;">
                  <el-input v-model="dictPath" placeholder="请选择文件路径" readonly style="flex: 1;" />

                  <el-button type="primary" @click="chooseFile">修改</el-button>
                </div>
              </el-card>
            </div>
          </el-col>
          <el-col :span="13">
            <el-card class="box-card">
              <h4>JWT Header</h4>
              <el-input
                  type="textarea"
                  :rows="3"
                  placeholder="jwt header内容"
                  v-model="jwtheader">
              </el-input>
              <h4>JWT Payload</h4>
              <el-input
                  type="textarea"
                  :rows="4"
                  placeholder="jwt payload内容"
                  v-model="jwtpayload">
              </el-input>
              <h4>JWT 密钥</h4>
              <el-input
                  type="textarea"
                  :rows="1"
                  placeholder="jwt Secret密钥"
                  v-model="jwtsecret">
              </el-input>
            </el-card>
            <el-card class="box-card" style="margin-top: 23px;">
              <h4>选择JWT算法</h4>
              <!-- JWT算法选择下拉框 -->
              <el-select v-model="value" placeholder="请选择JWT算法">
                <el-option
                    v-for="item in options"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value">
                </el-option>
              </el-select>
              <!-- 操作按钮 -->
              <!-- 操作按钮 -->
              <div style="margin-top: 20px;">
                <div style="display: flex; gap: 12px; align-items: center; flex-wrap: wrap;">
                  <el-button type="primary" @click="startdecode">Jwt解析</el-button>
                  <el-button type="primary" @click="startencode">Jwt编码</el-button>
                  <el-button type="success" @click="startCrack">Jwt密钥破解</el-button>

                  <el-tag
                      v-if="jwtInput"
                      :type="jwtverify ? 'success' : 'danger'">
                    {{ jwtverify ? 'JWT校验通过' : 'JWT校验失败' }}
                  </el-tag>
                </div>
              </div>

              <!-- 进度条 -->
              <div style="margin-top: 20px;">
                <el-progress :percentage="progress" :text-inside="true" :stroke-width="20" />
              </div>
              <div style="margin-top: 20px;">

              </div>
            </el-card>
          </el-col>
        </el-row>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import {DealOssList, FscanResultDeal, GetExcelContent, UploadFile} from "../../wailsjs/go/controller/InfoDeal";
import {InitCheck, RunUnWxapp, SelectDirectory} from "../../wailsjs/go/controller/UnWxapp";
import {ElMessage, ElMessageBox} from "element-plus";
import * as XLSX from "xlsx";
import {OpenPath,GetConfigDir} from "../../wailsjs/go/controller/System";
import {EncodeJWTWithAlg, DecodeJWTWithAlg, ChooseJwtFile, BruteForceJWT} from "../../wailsjs/go/controller/JwtCrackController";
import {QuestionFilled} from "@element-plus/icons-vue";


export default {
  name: "InfoDealView",
  components: {QuestionFilled},
  data() {
    return {
      activeTab: "fscan-deal",
      tabTitles: {
        "fscan-deal": "Fscan结果处理",
        "ip-ban-deal": "蓝队大批量封禁IP处置",
        "oss-list": "OSS资源桶遍历",
        "unwxapp": "wx小程序反编译",
      },
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
      currentPage: 1, // 当前页
      pageSize: 10, // 每页显示的行数
      paginatedData: [], // 分页后的数据

      OsslistInput:"", //oss资源桶路径
      OssListSuccess: false, //oss处理结果
      OssListSavePath: "", //oss处理结果文件保存位置

      wxpackages: [],
      wxappid: "",
      wxformat: true,
      wxresult: "",
      wxoutputPath:"",

      jwtInput: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiRWFzeVRvb2xzIiwiYWRtaW4iOnRydWV9.2sM8zndGv1hsGD6sSotuYDjY8Zk-ApqE5o-Gbp_U278", // JWT输入内容
      jwtheader: "", // JWT header内容
      jwtpayload: "", // JWT payload内容
      jwtsecret: "a-string-secret-at-least-256-bits-long", // JWT密钥内容
      jwtsignature: "", // JWT校验内容
      jwtverify: true, // JWT校验
      progress: 0, // 进度条百分比
      _timer: null, // 定时器
      dictPath: '选择jwt字典路径', // JWT字典路径
      options: [{
        value: '选项1',
        label: 'HS256'
      }, {
        value: '选项2',
        label: 'HS384'
      }, {
        value: '选项3',
        label: 'HS512'
      }, {
        value: '选项4',
        label: 'RS256'
      }, {
        value: '选项5',
        label: 'RS384'
      }, {
        value: '选项6',
        label: 'RS512'
      }, {
        value: '选项7',
        label: 'ES256'
      }, {
        value: '选项8',
        label: 'ES384'
      }, {
        value: '选项9',
        label: 'ES512'
      }, {
        value: '选项10',
        label: 'PS256'
      }, {
        value: '选项11',
        label: 'PS384'
      }, {
        value: '选项12',
        label: 'PS512'
      }, {
        value: '选项13',
        label: 'EdDSA'
      }, {
        value: '选项14',
        label: 'None'
      }],
      value: '选项1', // 默认选择的算法
      label: 'HS256',
    };
  },
  mounted() {
    document.addEventListener('contextmenu', (event) => {
      event.preventDefault();
    });
  },
  beforeUnmount() {
    document.removeEventListener('contextmenu', (event) => {
      event.preventDefault();
    });
  },
  methods: {
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
    async processFile() {
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
    async openExcelPreview() {
      if (!this.excelFilePath) {
        ElMessage.error("请先处理文件！");
        return;
      }

      try {
        const encodedContent = await GetExcelContent(this.excelFilePath);
        if (!encodedContent) {
          throw new Error("未能获取到有效的 Base64 数据！");
        }

        const binary = atob(encodedContent);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        const workbook = XLSX.read(bytes, { type: "array" });
        this.sheetsData = workbook.SheetNames.map(sheetName => ({
          sheetName,
          data: XLSX.utils.sheet_to_json(workbook.Sheets[sheetName])
        }));
        this.activeSheet = this.sheetsData[0]?.sheetName;
        this.paginatedData = this.getActiveSheetData(); // 初始化分页数据
      } catch (error) {
        console.error("Excel 文件预览失败:", error);
        ElMessage.error(error.message || "Excel 文件预览失败！");
      }
    },
    handlePageChange(page) {
      this.currentPage = page;
    },
    getActiveSheetData() {
      const activeSheetData = this.sheetsData.find(sheet => sheet.sheetName === this.activeSheet);
      return activeSheetData ? activeSheetData.data : [];
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




    // Oss存储桶遍历
    async generateOssListQueries() {
      // 如果用户输入为空，返回错误提示
      if (!this.OsslistInput.trim()) {
        this.$message.warning('请输入Oss资源桶链接！');
        return;
      }

      const OsslistInput = this.OsslistInput.trim();
      try {
        ElMessageBox({
          title: "提示",
          message: "正在请求数据哦",
          type: "info",
          showConfirmButton: false,
          closeOnClickModal: false,
          closeOnPressEscape: false,
        });
        // 调用后端接口处理
        this.OssListSavePath = await DealOssList(OsslistInput)

        this.OssListSuccess = true
        this.$message.success('文件生成成功：' + this.OssListSavePath)
        ElMessageBox.close()

      } catch (err) {
        ElMessageBox.close()
        this.$message.warning("请检查链接，只有存在信息泄露的链接才可以爬取哦");
      }
    },
    // 打开文件夹
    async openFileDir() {
      const baseDir = await GetConfigDir()
      const fileDir = baseDir + "/file"; // 拼接file子目录
      await OpenPath(fileDir)
    },

    async selectPackage() {
      // 先检测 Node.js 环境
      if (!await InitCheck()) {
        // 弹出是否前往官网下载的确认框
        ElMessageBox.confirm(
            "检测到未安装 Node.js 环境，是否现在前往官网下载？",
            "温馨提示",
            {
              confirmButtonText: "马上前往",
              cancelButtonText: "稍后再说",
              type: "warning",
            }
        )
            .then(() => {
              // 用户点击“马上前往”
              window.open("https://nodejs.org/", "_blank");
            })
            .catch(() => {
              // 用户点击“稍后再说”或关闭弹窗，不做任何事
            });
        // 终止后续逻辑，只有通过检测才会继续
        return;
      }

      // 只有 InitCheck() 返回 true 时，才会执行下面的选择文件夹逻辑
      try {
        const path = await SelectDirectory(); // 打开系统文件夹选择对话框
        if (path) {
          this.wxpackages = [path]; // Node.js 脚本支持多个路径，这里用数组
        }
      } catch (e) {
        ElMessage.error("选择路径失败");
      }
    },

    async runUnWxapp() {
      if (this.wxpackages.length === 0) {
        ElMessage.warning("请先选择小程序包路径");
        return;
      }

      try {
        // 关闭之前的弹窗（如果有）
        ElMessageBox.close();
        // 显示正在修改的提示框
        ElMessageBox({
          title: "提示",
          message: "正在进行反编译，请稍等~~~",
          type: "info",
          showConfirmButton: false,
          closeOnClickModal: false,
          closeOnPressEscape: false,
        });


        // 添加空对象 {} 作为第一个参数
        this.wxresult = await RunUnWxapp({}, this.wxpackages, this.wxappid, this.wxformat);

        // 通过 navigator.platform 判断是否 Windows
        const platform = navigator.platform.toLowerCase()
        const isWin = platform.startsWith('win')
        const sep = isWin ? '\\' : '/'

        // 动态拼接
        this.wxoutputPath = `${this.wxpackages}${sep}__APP__`

        // 关闭之前的弹窗（如果有）
        ElMessageBox.close();
        // 显示正在修改的提示框
        ElMessageBox({
          title: "提示",
          message: `反编译完成，文件保存在 ${this.wxoutputPath} 文件夹中`,
          type: "success",
          showConfirmButton: false,
          closeOnClickModal: false,
          closeOnPressEscape: false,
        });

      } catch (err) {
        ElMessageBox.close();
        this.wxresult = err?.message || "运行失败，请检查小程序路径是否选择合适，路径下是否存在xxxxx.wxapkg文件";
        ElMessage.error("运行失败，请检查小程序路径是否选择合适，路径下是否存在xxxxx.wxapkg文件");
      }
    },
    openOutputFolder() {
      if (!this.wxoutputPath) {
        ElMessage.warning("没有输出路径");
        return;
      }
      OpenPath(this.wxoutputPath);
    },
    chooseFile() {
      ChooseJwtFile()
          .then((path) => {
            if (path) {
              this.dictPath = path; // 更新字典路径
              ElMessage.success("字典文件选择成功");
            } else {
              ElMessage.error("未选择任何文件");
            }
          })
          .catch((error) => {
            console.error("选择文件失败:", error);
            ElMessage.error("选择文件失败，请重试");
          });
    },
    startCrack() {
      this.progress = -1; // 重置进度条
      this.progress = 0; // 重置进度条
      BruteForceJWT(this.jwtInput, this.label, this.dictPath)
          .then((result) => {
            if (!result.error) {
              this.jwtsecret = result.secret || "";
              this.jwtsignature = result.signature || "";
              this.jwtheader = JSON.stringify(result.header, null, 2);
              this.jwtpayload = JSON.stringify(result.payload, null, 2);
              ElMessage.success("JWT 密钥破解成功");
              this.progress = 100; // 设置进度条为100%
              this.jwtverify = true; // 设置校验为成功
              // clearInterval(this._timer); // 停止定时器
              return;
            }
            this.progress = 100; // 设置进度条为100%
            // clearInterval(this._timer); // 停止定时器
            this.jwtverify = false; // 设置校验为失败
            ElMessage.error(result.error);
          })
          .catch((error) => {
            console.error("JWT 密钥破解失败:", error);
            ElMessage.error(error.message || "JWT 密钥破解失败");
          });
    },
    startencode() {
      EncodeJWTWithAlg(this.label, this.jwtsecret, JSON.parse(this.jwtpayload), this.jwtheader)
          .then((result) => {
            if (!result.error) {
              this.jwtInput = result.jwt_token || "";
              this.jwtsignature = result.signature || "";
              // 显示成功消息
              ElMessage.success("jwt编码成功");
            } else {
              ElMessage.error(result.error);
            }
          })
          .catch((error) => {
            this.$message.default(error.message || "JWT 编码失败");
            ElMessage.error(error.message || "JWT 编码失败");
          });
    },
    startdecode() {
      DecodeJWTWithAlg(this.jwtInput, this.label, this.jwtsecret)
          .then((result) => {
            if (!result.error) {
              this.jwtheader = JSON.stringify(result.header,null,2);
              this.jwtpayload = JSON.stringify(result.payload,null,2);
              this.jwtsecret = result.secret || "";
              this.jwtsignature = result.signature || "";
              // 显示成功消息
              ElMessage.success("jwt解码成功");
            } else {
              ElMessage.error(result.error);
            }
          })
          .catch((error) => {
            this.$message.default(error.message || "JWT 解码失败");
            ElMessage.error(error.message || "JWT 解码失败");
          });

    }
  },
  watch: {
    activeSheet() {
      this.paginatedData = this.getActiveSheetData(); // 更新分页数据
      this.currentPage = 1; // 每次切换表格时重置为第一页
    }
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
  margin-bottom: 5px;
  /* border-bottom: 2px solid #ebeef5; */
  padding-left: 10px;
  /* 增加左边距 */
  border-radius: 10px 10px 10px 10px;
}

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
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(p){
  display: block;
  margin-block-start: 1em;
  margin-block-end: 1em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  unicode-bidi: isolate;
}


/* 使用 flex 布局确保按钮和文件路径显示在同一行 */
.actions {
  display: flex;
  align-items: center;
  gap: 20px; /* 按钮和文本之间的间隔 */
  flex-wrap: nowrap; /* 确保它们不换行 */
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
/* 上半部分和下半部分使用 flex 布局 */
.tab-content .info {
  color: #409eff;
}

.row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
}

.column {
  width: 48%;
  /* 两个元素各占一半的宽度 */
  box-sizing: border-box;
}

.upper-row .column {
  margin-bottom: 20px;
  /* 上半部分的列之间添加空隙 */
}

.lower-row .column {
  display: flex;
  flex-direction: column;
  align-items: stretch;
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
  /* 距离容器顶部 5px */
  right: 5px;
  /* 距离容器右侧 5px */
  z-index: 10;
  /* 确保按钮位于输入框上方 */
  padding: 5px 10px;
  font-size: 12px;
  background-color: #00aaff;
  /* 按钮颜色 */
  color: #ffffff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

/* 按钮 hover 效果 */
.ip-copy-button:hover {
  background-color: #176bca;
}

/* 按钮优化 */
.process-button {
  display: block;
  margin: 10px auto;
  padding: 8px 16px;
  margin-top: 25px;
  border-radius: 5px;
  /* margin-left: 0; */
  width: 100%;
  font-size: 18px;
  text-align: center;
  background-color: #5cb85c;
}

/* oss存储桶遍历 */
.form-group {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

/* 小程序反编译 */
.unwxapp-wrapper {
  padding: 4px;
}
.ml-2 {
  margin-left: 5px;
}

:deep(.el-card__body) {
  margin: -15px auto;
  padding: var(--el-card-padding);
}
.el-main {
  --el-main-padding: 0px;
}


</style>
