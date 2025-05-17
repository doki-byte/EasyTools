<template>
    <el-container class="container">
        <!-- 标签栏 -->
        <el-tabs v-model="activeTab" class="tabs">
            <el-tab-pane label="Fscan结果处理" name="fscan-deal" />
          <el-tab-pane label="蓝队大批量封禁IP处置" name="ip-ban-deal" />
          <el-tab-pane label="OSS存储桶遍历" name="oss-list" />
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
              <el-button type="success" @click="openDir">
                打开EasyToolsFiles文件夹
              </el-button>
            </div>
          </div>
        </el-main>
    </el-container>
</template>

<script>
import {DealOssList, FscanResultDeal, GetExcelContent, UploadFile} from "../../wailsjs/go/controller/InfoDeal";
import {ElMessage, ElMessageBox} from "element-plus";
import * as XLSX from "xlsx";
import axios from "axios";

export default {
    data() {
        return {
            activeTab: "fscan-deal",
            tabTitles: {
                "fscan-deal": "Fscan结果处理",
                "ip-ban-deal": "蓝队大批量封禁IP处置",
                "oss-list": "OSS资源桶遍历",
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
          this.$message.error('请输入Oss资源桶链接！');
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
          this.$message.error("处理失败，请重试！");
        }
      },
      // 打开文件夹
      openDir() {
        axios
            .get("http://localhost:10086/opendir")
            .catch(() => {
              ElMessage.error({
                message: "打开文件夹失败",
                type: "error",
              });
            });
      },
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

:deep(.el-tabs__item) {
    font-weight: 700;
    /* 加粗字体 */
    font-size: 16px;
    /* 设置字体大小 */
    transition: all 0.3s;
}

.el-tabs__item:hover {
    color: #409eff !important;
}

.el-tabs__item.is-active {
    color: #ffffff !important;
    background-color: #409eff !important;
    border-radius: 5px 5px 0 0;
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


</style>
