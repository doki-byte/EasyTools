<template>
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
</template>

<script>
import {ElMessage} from "element-plus";
import {FscanResultDeal, UploadFile} from "../../../wailsjs/go/assistive/Assistive";
import {GetConfigDir, OpenPath} from "../../../wailsjs/go/system/System";

export default {
  name: "fscan",
  data() {
    return {
      // fscan处理
      fileName: "", // 当前上传的文件名
      fileList: [], // 上传的文件列表
      excelData: [], // Excel 数据，用于渲染预览
      isDealFile: false,
      excelFilePath: "", // 后端生成的 Excel 文件路径
      sheetsData: [], // 存储所有表的数据
      activeSheet: "", // 当前显示的表
      paginatedData: [], // 分页后的数据
    }
  },
  methods: {
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
  }
}
</script>


<style scoped>
.fscan-content{
  padding: 0 8px 3px 3px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}
</style>