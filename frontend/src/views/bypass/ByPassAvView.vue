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
      <div class="upload-section glass-card">
        <el-upload class="upload-demo" ref="upload" :action="BypassAvUploadUrl" drag :auto-upload="false"
          :show-file-list="true" :limit="1" :before-upload="beforeUpload" :http-request="handleUpload"
          @change="handleFileChange">
          <div class="upload-icon">
            <el-icon><upload-filled /></el-icon>
          </div>
          <div class="el-upload__text">
            将 bin（CS/MSF shellcode）文件拖拽或 <em>点击这里上传</em>，也可直接拖入 exe 自动转换
          </div>
          <template #tip>
            <div class="el-upload__tip">注：使用前需要进行环境配置，配置教程
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

      <!-- exe 载荷选项：运行参数 + 保留控制台（mimikatz 等命令行工具） -->
      <div v-if="isExeSelected" class="exe-option-row glass-card">
        <div class="exe-option-item exe-option-params">
          <span class="exe-option-label">运行参数：</span>
          <el-input
              v-model="exeParams"
              size="small"
              clearable
              placeholder='如 mimikatz："privilege::debug" "sekurlsa::logonpasswords" "exit"'
          />
        </div>
        <div class="exe-option-item">
          <el-checkbox v-model="keepConsole" size="small">保留控制台（显示命令行输出）</el-checkbox>
        </div>
        <div class="exe-option-tip">运行参数会预置进转换后的 exe；勾选“保留控制台”后运行可在窗口中看到 mimikatz 等工具的输出（未勾选时静默后台运行）。<b style="color:#e6a23c">注意：修改参数后必须重新点击“处理bin文件/预处理”再生成，否则仍使用上一次转换的无参载荷</b></div>
      </div>

      <!-- 选项区域（目标固定 Windows，无平台选择） -->
      <div class="select-section glass-card">
        <el-select class="select-container" v-model="loadWay" placeholder="选择加载模式" @change="handleLoadWayChange" :disabled="isLoadDisabled">
          <el-option v-for="(mode, index) in filteredLoadList" :key="index" :label="mode" :value="mode"></el-option>
        </el-select>

        <el-select class="select-container" v-model="mode" placeholder="选择运行模式" @change="handleModeChange" :disabled="isModeDisabled">
          <el-option v-for="(mode, index) in modeList" :key="index" :label="mode" :value="mode"></el-option>
        </el-select>

        <el-select class="select-container" v-model="encodeWay" placeholder="选择加密方式">
          <el-option v-for="(item, index) in encodeWayList" :key="index" :label="item.label || item" :value="item.value || item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="av" placeholder="选择杀软类型">
          <el-option v-for="(item, index) in avList" :key="index" :label="item" :value="item"></el-option>
        </el-select>

        <el-select class="select-container" v-model="buildWay" placeholder="选择编译方式" :disabled="isModeDisabled || isLoadDisabled">
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
      <div class="action-section glass-card">
        <!-- 反沙箱校验配置 -->
        <div v-if="passBoxWay !== '否' && passBoxWay !== ''" class="sandbox-config">
          <div class="sandbox-config-title">
            <el-icon><Lock /></el-icon>
            <span>反沙箱校验</span>
            <el-tag size="small" effect="plain" :type="passBoxWay === '双击运行' ? 'success' : 'warning'">
              {{ passBoxWay }}
            </el-tag>
          </div>

          <!-- 双击运行模式说明 -->
          <div v-if="passBoxWay === '双击运行'" class="sandbox-config-desc">
            <el-icon><Clock /></el-icon>
            <span>程序启动后延时约 6 秒，检测实际睡眠时间与 CPU 核心数，符合沙箱特征时自动退出</span>
          </div>

          <!-- 命令行运行模式：自定义运行口令 -->
          <template v-else>
            <div class="sandbox-config-row">
              <span class="sandbox-config-label">运行口令</span>
              <el-input
                v-model="passBoxToken"
                placeholder="4-64位：字母/数字/下划线/连字符"
                clearable
                class="sandbox-config-input"
              />
            </div>
            <div v-if="passBoxToken" class="sandbox-config-cmd">
              运行命令：程序.exe -s {{ passBoxToken }}
            </div>
          </template>
        </div>
        <!-- 远程 -->
        <div v-if="loadWay === '远程'">
          <p style="color: #9c7f64">
            远程加载请先点击shellcode预处理，然后将处理后的yuancheng.bin文件上传后使用(注意与url中的名称一致)
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
              <!-- 架构选择 -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="arch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">shellcode预处理</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：URL + 架构 + 按钮们 -->
            <div class="action-row">
              <el-input
                  class="action-yuancheng-button"
                  placeholder="请输入远程加载URL"
                  v-model="yuanchengUrlOrFileName"
              />
              <!-- 架构选择 -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="arch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">shellcode预处理</el-button>
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
              <!-- 架构选择 -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="arch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：文件名 + 架构 + 按钮们 -->
            <div class="action-row">
              <el-input
                  class="action-yuancheng-button"
                  placeholder="请输入分离加载文件名"
                  v-model="yuanchengUrlOrFileName"
              />
              <!-- 架构选择 -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="arch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>
        </div>

        <!-- 白加黑 -->
        <div v-else-if="loadWay === '白加黑'">
          <!-- 白加黑模式下的子模式选择：SkyShadow 白程序 / 手动 DLL 代理转发 -->
          <div class="skyshadow-selector" style="margin-bottom: 12px;">
            <el-radio-group v-model="whiteBlackSource" class="skyshadow-source-toggle">
              <el-radio-button label="skyshadow">使用 SkyShadow 白程序</el-radio-button>
              <el-radio-button label="dllproxy">手动选择 DLL 代理转发</el-radio-button>
            </el-radio-group>
          </div>

          <!-- 选项一：SkyShadow 白程序 -->
          <div v-if="whiteBlackSource === 'skyshadow'" class="skyshadow-selector" style="margin-bottom: 12px;">
            <div class="skyshadow-row">
              <el-select
                  v-model="selectedSkyShadowPayload"
                  placeholder="选择SkyShadow白程序（可选，不选择则使用默认白程序）"
                  clearable
                  filterable
                  class="skyshadow-select"
                  popper-class="skyshadow-payload-dropdown"
                  @change="handleSkyShadowPayloadChange"
              >
                <el-option
                    v-for="payload in skyShadowPayloads"
                    :key="payload.header"
                    :label="`${payload.exeName} (${payload.bit}, ${payload.signed})`"
                    :value="payload"
                >
                  <div style="display: flex; justify-content: space-between;">
                    <span>{{ payload.exeName }}</span>
                    <span style="color: #909399; font-size: 12px;">{{ payload.bit }} | {{ payload.signed }}</span>
                  </div>
                </el-option>
              </el-select>
              
              <!-- 函数选择器（多选） -->
              <el-select
                  v-if="selectedSkyShadowPayload && selectedSkyShadowPayload.dlls && selectedSkyShadowPayload.dlls.length > 0"
                  v-model="selectedFunctions"
                  placeholder="选择要插入的函数（可多选，选中的函数会追加代码，未选中的只返回0）"
                  clearable
                  filterable
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  class="skyshadow-select"
                  popper-class="skyshadow-functions-dropdown"
              >
                <el-option-group
                    v-for="dll in selectedSkyShadowPayload.dlls"
                    :key="dll.name"
                    :label="`DLL: ${dll.name}`"
                >
                  <el-option
                      v-for="func in dll.functions"
                      :key="func"
                      :label="func"
                      :value="func"
                  />
                </el-option-group>
              </el-select>
              
              <!-- 刷新按钮 -->
              <el-button
                  type="default"
                  size="small"
                  class="skyshadow-refresh-btn"
                  @click="fetchSkyShadowPayloads"
                  title="刷新白程序列表"
              >
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>

          </div>

          <!-- 选项二：手动 DLL 代理转发（仅选择 DLL，由"编译生成"统一触发） -->
          <div v-else-if="whiteBlackSource === 'dllproxy'" class="skyshadow-selector" style="margin-bottom: 12px;">
            <div class="skyshadow-row dllproxy-row">
              <span class="dllproxy-label">
                DLL 代理转发：
              </span>
              <el-upload
                  ref="dllProxyUploadRef"
                  :auto-upload="false"
                  :on-change="handleDllProxyFileChange"
                  :file-list="dllProxyFileList"
                  accept=".dll"
                  :limit="1"
                  class="dllproxy-upload"
                  :show-file-list="false"
              >
                <template #trigger>
                  <el-button type="primary" size="small">选择 DLL 文件</el-button>
                </template>
              </el-upload>
              <span
                  class="dllproxy-file-name"
                  v-if="selectedDllProxyFile"
              >
                {{ (selectedDllProxyFile.name || selectedDllProxyFile.raw?.name) }}
              </span>
            </div>
            <div class="skyshadow-row dllproxy-row" style="margin-top: 8px;">
              <span class="dllproxy-label">
                代理后 DLL 名称：
              </span>
              <el-input
                  v-model="dllProxyName"
                  placeholder="留空则自动生成（原名称.orig.dll）,不能以数字开头"
                  size="small"
                  style="width: 300px;"
                  clearable
              />
            </div>
          </div>

          <template v-if="autoDelete === 'true'">
            <!-- 一行：时间 + 架构 + 按钮们 -->
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
              <!-- 架构选择（白加黑模式） -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="whiteBlackArch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：架构 + 按钮们 -->
            <div class="action-row">
              <!-- 架构选择（白加黑模式） -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="whiteBlackArch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>
        </div>

        <!-- 一体 -->
        <div v-else>
          <template v-if="autoDelete === 'true'">
            <!-- 一行：时间 + 架构 + 按钮们 -->
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
              <!-- 架构选择 -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="arch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>

          <template v-else>
            <!-- 一行：架构 + 按钮们 -->
            <div class="action-row">
              <!-- 架构选择 -->
              <div class="arch-toggle">
                <span class="arch-label">架构：</span>
                <el-radio-group v-model="arch" size="small">
                  <el-radio-button label="amd64">amd64</el-radio-button>
                  <el-radio-button label="386">386</el-radio-button>
                </el-radio-group>
              </div>
              <el-button :disabled="!isFormValid" type="primary" @click="handleInitCheck">处理bin文件</el-button>
              <el-button :disabled="!isFormValid" type="success" @click="handleGenerate">编译生成</el-button>
            </div>
          </template>
        </div>
      </div>


      <!-- 提示信息 -->
      <el-alert v-if="uploadStatus" :title="uploadStatus" :type="uploadStatusIsError ? 'error' : 'success'"
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
    <el-dialog v-model="dialogVisible" :title="getDialogTitle()" :width="dialogMode === 'byPassQvm' ? '720px' : '500px'">
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
        <el-upload v-else class="upload-demo" ref="uploadDialog" :action="BypassAvUploadUrl" drag :auto-upload="true"
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

        <!-- 绕过Qvm：可选图标处理（QVM规避 + 图标替换集成） -->
        <div v-if="dialogMode === 'byPassQvm'" class="qvm-icon-panel">
          <div class="qvm-icon-head">
            <span class="qvm-icon-title">图标处理：</span>
            <el-radio-group v-model="qvmIconMode" size="small">
              <el-radio-button label="none">不更换图标</el-radio-button>
              <el-radio-button label="random">随机内置图标</el-radio-button>
              <el-radio-button label="spec">指定图标</el-radio-button>
            </el-radio-group>
          </div>

          <div v-if="qvmIconMode === 'spec'" class="qvm-icon-spec">
            <div class="qvm-icon-upload-row">
              <span class="icon-dlg-label">自定义图标：</span>
              <el-upload
                  :auto-upload="false"
                  :limit="1"
                  :show-file-list="false"
                  accept=".ico,.png"
                  :on-change="handleQvmIconUpload"
              >
                <el-button size="small" type="primary" plain>
                  <el-icon style="margin-right: 4px;"><UploadFilled /></el-icon>选择 .ico/.png
                </el-button>
              </el-upload>
              <img v-if="qvmCustomData" :src="qvmCustomData" class="icon-dlg-preview" alt="icon" />
              <span v-if="qvmCustomLabel" class="icon-dlg-name">{{ qvmCustomLabel }}</span>
              <span v-else-if="qvmLibSel" class="icon-dlg-name">内置图标: {{ qvmLibSel }}</span>
              <span v-else class="icon-dlg-name-placeholder">未选择（≤512KB）</span>
            </div>
            <div class="qvm-icon-lib-title">或从内置图标库选择：</div>
            <div v-if="qvmLibLoading" class="icon-dlg-loading">图标库加载中...</div>
            <div v-else-if="qvmLibIcons.length > 0" class="icon-dlg-grid">
              <div
                  v-for="ic in qvmLibIcons"
                  :key="ic.name"
                  class="icon-dlg-item"
                  :class="{ active: !qvmCustomData && qvmLibSel === ic.name }"
                  :title="ic.name"
                  @click="pickQvmLibIcon(ic.name)"
              >
                <img :src="iconLibThumbUrl(ic.name)" alt="" loading="lazy" />
                <div class="icon-dlg-item-name">{{ ic.name.replace(/\.ico$/i, '') }}</div>
              </div>
            </div>
            <div v-else class="icon-dlg-loading">内置图标库为空，请使用自定义上传</div>
          </div>

          <div class="qvm-icon-tip">处理将作用于 EasyToolsFiles/file 下的所有 exe：随机化版本信息 + 清理静态特征以绕过QVM；选择图标后一并替换图标（纯本地处理，全平台可用）</div>
        </div>

      </el-form>

      <template #footer>
        <div v-if="dialogMode === 'binder'">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleBinderSubmit">开始捆绑</el-button>
        </div>
        <div v-else-if="dialogMode === 'byPassQvm'">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleBypassQvm">开始处理</el-button>
        </div>
        <div v-else>
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleChangeExeSubmit">修改</el-button>
        </div>
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
import {Clock, DocumentCopy, QuestionFilled, UploadFilled, Lock, InfoFilled, Check, Refresh} from '@element-plus/icons-vue';
import {GetPortConfig} from "../../../wailsjs/go/controller/Server";

export default {
  name: "BypassAvView",
  data() {
    return {
      portConfig: null,
      selectedFile: null, // 选中的文件
      mode: "", // 运行模式
      av: "", // 杀软标签
      encodeWay: "",  // 加密方式标签
      buildWay: "", // 编译方式标签
      arch: "amd64", // 目标架构（amd64/386）- 用于普通模式
      targetOS: "windows", // 目标操作系统
      whiteBlackArch: "amd64", // 白加黑模式架构选择（amd64/386）
      loadWay: "", // 加载器方式标签
      passBoxWay: "", // 添加反杀箱代码标签
      autoRun: "", //添加自启动代码标签
      autoDelete: "", //添加自删除代码标签
      autoDeleteTime: "", // 添加自删除校验时间
      yuanchengUrlOrFileName:"", // 远程加载url

      modeList: [], // 运行模式列表
      filteredModeList: [], //实际显示的模式列表（会根据条件过滤）
      baiJiaHeiMode: "loader10", // 白加黑专用loader编号（后端modelist下发，无响应时回退默认值）
      isLoadDisabled: false, //是否禁用模式模式选择
      isModeDisabled: false, //是否禁用运行模式选择
      loadList: ["一体","远程","分离","白加黑"], // 加载模式列表
      avList: [], // 杀软列表
      encodeWayList: [], // 加密方式列表
      buildWayList: ["go", "garble"], // 加密方式列表
      archList: ["amd64", "386"], // 架构列表
      passBoxWayList:["否", "命令行运行","双击运行"], // 添加反杀箱代码列表
      passBoxToken: "", // 命令行运行模式的自定义运行口令（用户前端输入）
      autoRunList:["true", "false"], // 添加自启动列表
      autoDeleteList:["true", "false"], // 添加自删除列表
      // 绕过Qvm 图标处理（集成于"绕过Qvm查杀"，编译生成不涉及图标）
      qvmIconMode: "none", // none=不更换 | random=随机内置图标 | spec=指定图标
      qvmLibIcons: [], // 内置图标库 [{name:'access.ico'}]
      qvmLibLoading: false,
      qvmLibSel: "", // 指定模式下选中的内置库图标文件名
      qvmCustomData: "", // 指定模式下自定义图标 dataURL
      qvmCustomLabel: "",
      qvmProcessing: false, // Qvm 处理中标记（防重复点击）
      exeParams: "", // exe 载荷运行参数（拖入 exe 时预置进 Donut）
      keepConsole: false, // 保留控制台（命令行工具模式，如 mimikatz）
      lastUploadParams: "", // 最近一次上传(处理bin)时使用的运行参数，用于防呆校验
      uploadStatus: "", // 上传状态信息
      uploadStatusIsError: false, // 上传状态是否为错误（决定提示条样式）
      initCheckStatus: "", // 上传状态信息

      fileType: "bin",
      dialogMode: "",
      dialogVisible: false,

      changeExeSizeForm: {
        size: "",
      },

      excludeExeAccept: "*",
      // 用于在提交时直接获取文件对象
      exeFile: null,
      binderFile: null,
      
      // SkyShadow白程序相关
      skyShadowPayloads: [], // SkyShadow白程序列表
      selectedSkyShadowPayload: null, // 选中的白程序
      selectedFunctions: [], // 选中的函数名列表（支持多选）
      whiteBlackSource: "skyshadow", // 白加黑子模式：skyshadow / dllproxy

      // DLL 代理转发
      generatingDllProxy: false,
      generating: false, // 编译生成中标记（防重复点击 + 立即反馈）
      processingInit: false, // shellcode 预处理中标记（防重复点击）
      dllProxyFileList: [],
      selectedDllProxyFile: null,
      dllProxyName: "", // 用户自定义的代理后 DLL 名称（原始DLL重命名后的名称）
    };
  },
  computed: {
    BypassAvUploadUrl() {
      if (!this.portConfig) return 'about:blank';
      return `http://127.0.0.1:${this.portConfig.mainPort}/upload`;
    },

    DocumentCopy() {
      return DocumentCopy
    },
    filteredLoadList() {
      // 目标固定为 Windows，白加黑始终可用
      return this.loadList;
    },
    // 当前上传的是否为 exe（拖入 exe 自动转 shellcode，可配置运行参数/保留控制台）
    isExeSelected() {
      const f = this.selectedFile;
      const name = (f && (f.name || f.raw?.name || "")) || "";
      return name.toLowerCase().endsWith(".exe");
    },
    isFormValid() {
      // 命令行运行模式：必须填写运行口令
      if (this.passBoxWay === "命令行运行" && !(this.passBoxToken && this.passBoxToken.trim())) {
        return false;
      }
      // 一体 / 白加黑 共用一套基本校验
      if (this.loadWay === "一体" || this.loadWay === "白加黑") {
        // 白加黑 + 手动 DLL 代理模式时，必须选择 DLL 文件
        if (this.loadWay === "白加黑" && this.whiteBlackSource === "dllproxy") {
          if (!this.selectedDllProxyFile) {
            return false;
          }
        }
        if (this.autoDelete === 'true') {
          return (
              this.selectedFile &&
              this.mode &&
              this.loadWay &&
              this.av &&
              this.encodeWay &&
              this.passBoxWay &&
              this.buildWay &&
              this.whiteBlackArch &&
              this.autoRun &&
              this.autoDelete &&
              this.autoDeleteTime
          );
        } else {
          return (
              this.selectedFile &&
              this.mode &&
              this.loadWay &&
              this.av &&
              this.encodeWay &&
              this.passBoxWay &&
              this.buildWay &&
              this.whiteBlackArch &&
              this.autoRun &&
              this.autoDelete
          );
        }
      } else {
        // 远程 / 分离
        if (this.autoDelete === 'true') {
          return (
              this.selectedFile &&
              this.mode &&
              this.loadWay &&
              this.av &&
              this.encodeWay &&
              this.passBoxWay &&
              this.buildWay &&
              this.arch &&
              this.autoRun &&
              this.autoDelete &&
              this.autoDeleteTime &&
              this.yuanchengUrlOrFileName
          );
        } else {
          return (
              this.selectedFile &&
              this.mode &&
              this.loadWay &&
              this.av &&
              this.encodeWay &&
              this.passBoxWay &&
              this.buildWay &&
              this.arch &&
              this.autoRun &&
              this.autoDelete &&
              this.yuanchengUrlOrFileName
          );
        }
      }
    },
  },
  components: {
    Clock,
    UploadFilled,
    QuestionFilled,
    Refresh,
    // 注册图标组件
    'el-icon-copy-document': DocumentCopy
  },

  async mounted() {
    await this.loadPortConfig();
    // 页面加载时获取模式列表
    this.fetchModeList();
    this.fetchAvList();
    this.fetchEncodewayList();
    // 不在这里加载白程序列表，只在切换到"白加黑"模式时加载
  },
  methods: {
    // DLL 代理：选择文件
    handleDllProxyFileChange(file, fileList) {
      this.dllProxyFileList = fileList;
      this.selectedDllProxyFile = file.raw || file;
    },

    // DLL 代理：生成
    async handleGenerateDllProxy() {
      if (!this.selectedDllProxyFile) {
        this.$message.warning("请先选择 DLL 文件");
        return;
      }
      if (this.generatingDllProxy) {
        return;
      }
      if (!this.portConfig) {
        await this.loadPortConfig();
      }

      this.generatingDllProxy = true;
      try {
        const formData = new FormData();
        const dllFile = this.selectedDllProxyFile.raw || this.selectedDllProxyFile;
        formData.append("dllFile", dllFile);
        formData.append("encodeWay", this.encodeWay || "aes");
        // 添加架构选择参数
        if (this.whiteBlackArch) {
          formData.append("forceArch", this.whiteBlackArch);
        }
        // 添加自定义代理后 DLL 名称参数
        if (this.dllProxyName && this.dllProxyName.trim()) {
          formData.append("proxyDllName", this.dllProxyName.trim());
        }

        const resp = await axios.post(
          `http://127.0.0.1:${this.portConfig.mainPort}/dllproxy`,
          formData,
          { headers: { "Content-Type": "multipart/form-data" } }
        );

        const data = resp.data;
        if (!data.success) {
          this.$message.error(data.error || "生成 DLL 代理转发失败");
        } else {
          this.$message.success("DLL 代理转发生成成功");
        }
      } catch (e) {
        console.error(e);
        this.$message.error("请求 DLL 代理转发接口失败");
      } finally {
        this.generatingDllProxy = false;
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
    fetchModeList() {
      axios
        .get(`http://127.0.0.1:${this.portConfig.mainPort}/modelist`, {
          params: { targetOS: "windows" },
        })
        .then((response) => {
          this.modeList = response.data.all_modes;
          this.filteredModeList = this.modeList;
          // 白加黑专用loader编号由后端下发，避免前端硬编码模板名
          if (response.data.bai_jia_hei) {
            this.baiJiaHeiMode = response.data.bai_jia_hei;
          }
          if (this.mode && !this.modeList.includes(this.mode)) {
            this.mode = this.modeList[0] || "";
          }
        })
        .catch((error) => {
          console.error("Error fetching mode list:", error);
        });
    },
    handleTargetOSChange() {
      // 目标平台已固定为 Windows（保留方法避免误引用）
    },
    fetchAvList() {
        axios
          .get(`http://127.0.0.1:${this.portConfig.mainPort}/avlist`)
          .then((response) => {
            this.avList = response.data.av_list;
          })
          .catch((error) => {
            console.error("Error fetching AV list:", error);
          });
    },
    fetchEncodewayList() {
        axios
          .get(`http://127.0.0.1:${this.portConfig.mainPort}/encodewaylist`)
          .then((response) => {
            this.encodeWayList = response.data.encode_way_list;
          })
          .catch((error) => {
            console.error("Error fetching encodeWay list:", error);
          });
    },
    // 获取SkyShadow白程序列表
    async fetchSkyShadowPayloads() {
      if (!this.portConfig) return;
      try {
        const resp = await axios.get(
          `http://127.0.0.1:${this.portConfig.mainPort}/skyshadow/payloads`
        );
        const data = resp.data;
        if (data.success && data.payloadDetails) {
          this.skyShadowPayloads = data.payloadDetails || [];
          // 调试日志
          console.log("Debug Frontend: 获取到的白程序列表", this.skyShadowPayloads.map(p => ({
            exeName: p.exeName,
            hasDlls: !!p.dlls,
            dllsCount: p.dlls ? p.dlls.length : 0
          })));
        }
      } catch (e) {
        console.error("获取SkyShadow白程序列表失败:", e);
        this.skyShadowPayloads = [];
      }
    },
    // 处理SkyShadow白程序选择变化
    handleSkyShadowPayloadChange(payload) {
      if (payload) {
        console.log('已选择白程序:', payload);
        // 清空之前的选择
        this.selectedFunctions = [];
      } else {
        console.log('已清除白程序选择');
        this.selectedFunctions = [];
      }
    },
    // 加载模式变化处理
    handleLoadWayChange(newLoadWay) {
      if (newLoadWay === '白加黑') {
        // 设置为白加黑模式，并锁定
        this.mode = this.baiJiaHeiMode;
        this.filteredModeList = [this.baiJiaHeiMode]; // 只显示白加黑选项
        this.buildWay = 'go';
        this.buildWayList = ['go']
        this.isModeDisabled = true; // 禁用选择
        // 只在切换到白加黑模式时加载SkyShadow白程序列表
        this.fetchSkyShadowPayloads();

        console.log('已切换到白加黑模式，运行模式已锁定为', this.baiJiaHeiMode);
      } else {
        // 恢复正常的模式列表和选择能力
        this.filteredModeList = this.modeList;
        this.buildWayList = ['go','garble']
        this.isModeDisabled = false;

        // 如果当前模式是白加黑专用loader但加载模式不是白加黑，则重置模式
        if (this.mode === this.baiJiaHeiMode) {
          this.mode = '请选择运行模式';
          this.buildWay = '选择编译方式';
        }
        
        // 切换到其他模式时，清空白程序相关的数据
        this.selectedSkyShadowPayload = null;
        this.selectedFunctions = [];
      }
    },
    handleModeChange(newLoadWay) {
      if (newLoadWay === this.baiJiaHeiMode) {
        // 设置为白加黑模式，并锁定
        this.loadWay = '白加黑';
        this.loadList = ["白加黑"]; // 只显示白加黑选项
        this.buildWay = 'go';
        this.buildWayList = ['go']
        this.isLoadDisabled = true; // 禁用选择
        // 切换到白加黑模式时加载SkyShadow白程序列表
        this.fetchSkyShadowPayloads();

        console.log('已切换到白加黑模式，加载模式已锁定为白加黑');
      } else {
        // 恢复正常的模式列表和选择能力
        this.loadList = ["一体","远程","分离","白加黑"];
        this.buildWayList = ['go','garble']
        this.isLoadDisabled = false;

        // 如果当前加载模式是白加黑但运行模式已不是白加黑专用loader，则重置加载模式
        if (this.loadWay === '白加黑') {
          this.loadWay = '选择加载模式';
          this.buildWay = '选择编译方式';
        }
        
        // 切换到其他模式时，清空白程序相关的数据
        this.selectedSkyShadowPayload = null;
        this.selectedFunctions = [];
      }
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

    handleInitCheck() {
      // 防重复点击：处理中忽略后续点击
      if (this.processingInit) return;
      this.processingInit = true;
      // 显示处理中的对话框
      ElMessageBox({
        title: "正在处理",
        message: "正在处理shellcode，请稍候...",
        type: "info",
        showConfirmButton: false,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      });

      axios
          .get(`http://127.0.0.1:${this.portConfig.mainPort}/initCheck`)
          .then(() => {
            return this.handleUpload(); // 返回 Promise 链
          })
          .then(() => {
            // 上传成功后的处理
            ElMessageBox.close();
            ElMessageBox({
              title: "处理完成",
              message: "shellcode预处理完成！",
              type: "success",
            });
            console.log('shellcode预处理完成');
          })
          .catch((error) => {
            // 关闭处理中的对话框
            ElMessageBox.close();

            // 获取错误信息：优先取后端 error 字段（真实原因），其后是 message 字段。
            // 老版本 /initCheck 只回 message，前端只读 error 时拿不到内容，
            // 界面上就只剩 axios 的兜底文案 "Request failed with status code 400"。
            const errorMessage = error.response?.data?.error
                || error.response?.data?.message
                || error.message;
            const statusCode = error.response?.status;
            // 环境初始化失败（go/gcc/garble 缺失，或依赖安装失败）：
            // 后端固定以"环境初始化失败: ..."开头，据此给出可执行的排查方向
            const isEnvInitFailure = statusCode === 404
                || (typeof errorMessage === "string" && errorMessage.includes("环境初始化失败"));

            // 使用 setTimeout 确保对话框正确关闭后再显示错误
            setTimeout(() => {
              if (statusCode === 403 || statusCode === 401) {
                // 服务端拒绝处理
                this.uploadStatusIsError = true;
                this.uploadStatus = errorMessage || "服务端拒绝处理，请检查运行环境后重试";
                ElMessageBox({
                  title: "处理失败",
                  message: errorMessage || "服务端拒绝处理，请检查运行环境后重试",
                  type: "error",
                });
              } else if (isEnvInitFailure) {
                const tips = "请确认 go、gcc、garble 已安装并加入 PATH；若提示依赖安装失败，请检查网络或 GOPROXY 配置。";
                this.uploadStatusIsError = true;
                this.uploadStatus = `${errorMessage || "环境初始化失败"} ${tips}`;
                ElMessageBox({
                  title: "处理失败",
                  message: `${errorMessage || "环境初始化失败"}。${tips}`,
                  type: "error",
                });
              } else {
                // 其他错误
                this.uploadStatusIsError = true;
                this.uploadStatus = errorMessage || "shellcode预处理失败";
                ElMessageBox({
                  title: "处理失败",
                  message: errorMessage || "shellcode预处理失败",
                  type: "error",
                });
              }

              // 报错信息（含依赖/环境排查提示）保留更久，避免用户还没读完就消失
              setTimeout(() => {
                this.uploadStatus = "";
              }, 8000);
            }, 100);
          })
          .finally(() => {
            this.processingInit = false;
          });
    },

    handleFileChange(file) {
      this.selectedFile = file;
    },

    // ============ 绕过Qvm 图标处理（集成于 EXE处理「绕过Qvm查杀」） ============

    // 内置图标缩略图地址（/bypass/icon/:name，避开 /icon 静态目录路由）
    iconLibThumbUrl(name) {
      const port = this.portConfig?.mainPort;
      if (!port || !name) return "";
      return `http://127.0.0.1:${port}/bypass/icon/${encodeURIComponent(name)}`;
    },

    // 拉取内置图标库（Qvm 弹窗首次进入时加载一次）
    async loadQvmIconLibrary() {
      if (this.qvmLibIcons.length > 0 || this.qvmLibLoading) return;
      const port = this.portConfig?.mainPort;
      if (!port) return;
      this.qvmLibLoading = true;
      try {
        const resp = await axios.get(`http://127.0.0.1:${port}/bypass/iconlist`);
        const data = resp.data;
        if (data.success && Array.isArray(data.icons)) {
          this.qvmLibIcons = data.icons.map((n) => ({ name: n }));
        } else {
          this.qvmLibIcons = [];
        }
      } catch (e) {
        console.error("加载内置图标库失败:", e);
        this.qvmLibIcons = [];
        ElMessage.error("内置图标库加载失败，请检查本地服务");
      } finally {
        this.qvmLibLoading = false;
      }
    },

    // 打开 EXE处理弹窗时：Qvm 模式重置选择并预加载图标库
    openDialog(mode) {
      this.dialogMode = mode;
      if (this.dialogMode === "byPassQvm") {
        this.qvmIconMode = "none";
        this.qvmLibSel = "";
        this.qvmCustomData = "";
        this.qvmCustomLabel = "";
        this.loadQvmIconLibrary();
      }
      if (this.dialogMode === "yuanchengDeal") {
        this.fileType = "bin";
      } else {
        this.fileType = "exe";
      }
      this.dialogVisible = true;
    },

    // 指定模式下选中内置库图标（与自定义互斥）
    pickQvmLibIcon(name) {
      this.qvmLibSel = name;
      this.qvmCustomData = "";
      this.qvmCustomLabel = "";
    },

    // 指定模式下上传自定义图标（.ico/.png -> dataURL），与内置库互斥
    handleQvmIconUpload(file) {
      const raw = file.raw || file;
      if (!raw) return;
      const name = (raw.name || "").toLowerCase();
      if (!name.endsWith(".ico") && !name.endsWith(".png")) {
        ElMessage.error("仅支持 .ico / .png 图标文件");
        return;
      }
      if (raw.size > 512 * 1024) {
        ElMessage.error("图标文件不能超过 512KB");
        return;
      }
      const reader = new FileReader();
      reader.onload = (e) => {
        this.qvmCustomData = e.target?.result || "";
        this.qvmCustomLabel = raw.name;
        this.qvmLibSel = "";
      };
      reader.onerror = () => {
        ElMessage.error("读取图标文件失败");
      };
      reader.readAsDataURL(raw);
    },

    handleUpload() {
      return new Promise((resolve, reject) => {
        if (!this.isFormValid) {
          this.uploadStatusIsError = true;
          this.uploadStatus = "请完成表单后再提交。";
          setTimeout(() => {
            this.uploadStatus = "";
          }, 3000);
          reject(new Error('表单验证失败'));
          return;
        }

        const formData = new FormData();
        formData.append("file", this.selectedFile.raw);
        formData.append("loadWay", this.loadWay);
        formData.append("yuanchengUrlOrFileName", this.yuanchengUrlOrFileName);
        formData.append("autoDeleteTime", this.autoDeleteTime);
        // exe 载荷：预置运行参数（Donut），供 mimikatz 等命令行工具使用
        if (this.isExeSelected && this.exeParams && this.exeParams.trim()) {
          formData.append("params", this.exeParams.trim());
        }
        // 上传 exe 时携带架构参数，供内置 Donut 转换使用
        formData.append("arch", this.arch || 'amd64');

        axios
            .post(`http://127.0.0.1:${this.portConfig.mainPort}/upload`, formData, {
              headers: { "Content-Type": "multipart/form-data" },
            })
            .then((resp) => {
              // 展示后端转换回执（含 exe 预置参数说明）
              const note = resp?.data?.note;
              this.uploadStatusIsError = false;
              this.uploadStatus = note || "文件处理成功！";
              if (note) {
                // 若回执较长，延长展示时间
                setTimeout(() => {
                  this.uploadStatus = "";
                }, 6000);
              } else {
                setTimeout(() => {
                  this.uploadStatus = "";
                }, 3000);
              }
              // 记录本次上传所用的运行参数（防呆：参数变更需重新处理）
              this.lastUploadParams = this.isExeSelected ? ((this.exeParams || "").trim()) : "";
              resolve(); // 上传成功
            })
            .catch((error) => {
              // 如果是授权失败（403状态码），显示授权相关的错误信息
              const errorMessage = error.response?.data?.error || error.message;
              if (error.response?.status === 403 || error.response?.status === 401) {
                this.uploadStatusIsError = true;
                this.uploadStatus = errorMessage || "处理失败，请检查运行环境后重试";
              } else {
                // 优先显示后端真实原因（如 Donut 转换失败的具体报错），否则退回通用提示
                this.uploadStatus = errorMessage
                    || `文件处理失败！请上传${this.fileType === 'bin' ? 'bin' : 'exe'}文件`;
              }
              this.selectedFile = null;
              this.clearUploadRef();
              setTimeout(() => {
                this.uploadStatus = "";
              }, 3000);
              reject(new Error(errorMessage || '文件上传失败'));
            });
      });
    },

    // 防御式清空上传列表（兼容 ref 为数组/组件不存在的情况）
    clearUploadRef() {
      try {
        let u = this.$refs.upload;
        if (Array.isArray(u)) u = u[0];
        if (u && typeof u.clearFiles === 'function') u.clearFiles();
      } catch (e) {
        console.warn('clearFiles 调用失败:', e);
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
    async handleBinderSubmit() {
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

      axios.post(`http://127.0.0.1:${this.portConfig.mainPort}/binder`, formData)
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
            // 如果是授权失败（403状态码），显示授权相关的错误信息
            const errorMessage = err.response?.data?.error || err.message;
            if (err.response?.status === 403 || err.response?.status === 401) {
              this.$message.error(errorMessage || "处理失败，请检查运行环境后重试");
            } else {
              this.$message.error("捆绑文件失败");
            }
            console.error(err);
          });
    },


    // 修改生成操作的处理逻辑
    async handleGenerate() {
      // 防重复点击：处理中忽略后续点击
      if (this.generating) return;

      if (!this.isFormValid) {
        this.uploadStatusIsError = true;
        this.uploadStatus = "请完成表单后再提交。";
        setTimeout(() => {
          this.uploadStatus = "";
        }, 3000);
        return;
      }

      // 防呆：exe 运行参数修改后必须重新"处理bin/预处理"（参数在 Donut 转换阶段烘焙）
      if (this.isExeSelected && ((this.exeParams || "").trim() !== this.lastUploadParams)) {
        this.uploadStatusIsError = false;
        this.uploadStatus = "运行参数已修改：请先点击“处理bin文件/预处理”重新处理，再编译生成";
        setTimeout(() => {
          this.uploadStatus = "";
        }, 6000);
        return;
      }

      // 点击后立刻弹出处理对话框，
      // 避免等待期间界面毫无反馈，让用户误以为没点上或程序卡死
      this.generating = true;
      const message = this.buildWay === "garble"
          ? "请耐心稍等，混淆加密生成很慢~~，正在进行生成操作..."
          : "请稍等，正在进行生成操作...";
      ElMessageBox({
        title: "正在生成",
        message: message,
        type: "info",
        showConfirmButton: false,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      });

      try {
        // 如果是白加黑 + 手动 DLL 代理模式，则走 dllproxy 接口
        if (this.loadWay === "白加黑" && this.whiteBlackSource === "dllproxy") {
          try {
            if (!this.selectedDllProxyFile) {
              ElMessageBox.close();
              this.$message.warning("请先选择 DLL 文件");
              return;
            }

            const formData = new FormData();
            const dllFile = this.selectedDllProxyFile.raw || this.selectedDllProxyFile;
            formData.append("dllFile", dllFile);
            formData.append("encodeWay", this.encodeWay || "aes");
            // 添加架构选择参数
            if (this.whiteBlackArch) {
              formData.append("forceArch", this.whiteBlackArch);
            }
            // 添加自定义代理后 DLL 名称参数
            if (this.dllProxyName && this.dllProxyName.trim()) {
              formData.append("proxyDllName", this.dllProxyName.trim());
            }

            const resp = await axios.post(
                `http://127.0.0.1:${this.portConfig.mainPort}/dllproxy`,
                formData,
                {headers: {"Content-Type": "multipart/form-data"}}
            );

            ElMessageBox.close();
            const data = resp.data;
            if (!data.success) {
              this.$message.error(data.error || "生成 DLL 代理转发失败");
            } else {
              this.$message.success("DLL 代理转发生成成功，请在 EasyToolsFiles/file/dllproxy 目录查看。");
            }
          } catch (e) {
            ElMessageBox.close();
            console.error(e);
            // 如果是授权失败（403状态码），显示授权相关的错误信息
            const errorMessage = e.response?.data?.error || e.message;
            if (e.response?.status === 403 || e.response?.status === 401) {
              this.$message.error(errorMessage || "处理失败，请检查运行环境后重试");
            } else {
              this.$message.error("DLL代理转发生成失败");
            }
          }
          return;
        }

        // 默认走原有 /make 生成逻辑
        const data = {
          mode: this.mode,
          av: this.av,
          encodeWay: this.encodeWay,
          buildWay: this.buildWay,
          arch: this.loadWay === "白加黑" ? this.whiteBlackArch : this.arch,
          targetOS: this.targetOS,
          loadWay: this.loadWay,
          passBoxWay: this.passBoxWay,
          passBoxToken: this.passBoxWay === "命令行运行" ? this.passBoxToken : "",
          autoRun: this.autoRun,
          autoDelete: this.autoDelete,
          autoDeleteTime: this.autoDeleteTime,
          yuanchengUrlOrFileName: this.yuanchengUrlOrFileName,
          // exe 工具（如 mimikatz）需要保留控制台输出
          keepConsole: this.isExeSelected && this.keepConsole,
        };

        // 编译生成默认不携带/添加任何图标；需要图标请走 EXE处理「绕过Qvm查杀」

        // 如果是白加黑模式，传递架构选择和白程序信息
        if (this.loadWay === "白加黑") {
          // 传递架构选择
          data.skyShadowArch = this.whiteBlackArch || "amd64";

          // 如果选择了SkyShadow白程序，则传递白程序信息
          if (this.selectedSkyShadowPayload) {
            // 复制白程序信息并添加选中的函数名列表
            const payloadCopy = JSON.parse(JSON.stringify(this.selectedSkyShadowPayload));
            payloadCopy.selectedFunctions = this.selectedFunctions || [];
            data.skyShadowPayload = payloadCopy;

            // 调试日志
            console.log("Debug Frontend: 传递白程序数据", {
              hasDlls: !!payloadCopy.dlls,
              dllsCount: payloadCopy.dlls ? payloadCopy.dlls.length : 0,
              selectedFunctions: payloadCopy.selectedFunctions,
              allFunctions: payloadCopy.dlls ? payloadCopy.dlls.flatMap(dll => dll.functions || []) : []
            });
          }
        }

        axios
            .post(`http://127.0.0.1:${this.portConfig.mainPort}/make`, data, {
              headers: {"Content-Type": "application/json"},
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
              const errorMessage = error.response?.data?.error || error.message;
              // 如果是授权失败（403状态码），显示授权相关的错误信息
              if (error.response?.status === 403 || error.response?.status === 401) {
                ElMessage.error({
                  message: errorMessage || "处理失败，请检查运行环境后重试",
                  type: "error",
                });
              } else {
                // 透出后端真实报错（编译失败原因 / 模板不存在 / 载荷读取失败等），
                // 原实现固定显示"生成出错，请先处理bin文件进行初始化"，把真实原因丢掉了
                ElMessage.error({
                  message: errorMessage || "生成出错，请先处理bin文件进行初始化",
                  type: "error",
                  duration: 8000,
                  showClose: true,
                });
              }
            });
      } catch (error) {
        console.error('处理生成操作失败:', error);
        ElMessageBox.close();
      } finally {
        this.generating = false;
      }
    },

    // 绕过Qvm查杀：随机化版本信息 + 清理静态特征（集成图标处理：不更换/随机内置/指定）
    async handleBypassQvm() {
      if (this.qvmProcessing) return; // 防重复点击
      this.qvmProcessing = true;

      // 组装图标配置
      const payload = { iconMode: this.qvmIconMode || "none" };
      if (this.qvmIconMode === "spec") {
        if (this.qvmCustomData) {
          payload.iconData = this.qvmCustomData;
        } else if (this.qvmLibSel) {
          payload.iconName = this.qvmLibSel;
        } else {
          ElMessage.warning("指定图标模式下请选择内置图标或上传 .ico/.png");
          this.qvmProcessing = false;
          return;
        }
      }

      // 先弹处理框再校验授权，点击即有反馈
      ElMessageBox({
        title: "正在处理",
        message: "请稍等，正在执行绕过Qvm处理...",
        type: "info",
        showConfirmButton: false,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      });

      try {
        const resp = await axios.post(
            `http://127.0.0.1:${this.portConfig.mainPort}/passqvm`,
            payload,
            { headers: { "Content-Type": "application/json" } }
        );
        ElMessageBox.close();
        const data = resp.data || {};
        if (data.success === false) {
          ElMessage.error(data.error || "绕过Qvm处理失败");
        } else {
          ElMessageBox({
            title: "处理完成",
            message: "绕过Qvm处理完成，请在 EasyToolsFiles/file 目录查看。",
            type: "success",
          });
          this.dialogVisible = false;
        }
      } catch (error) {
        ElMessageBox.close();
        const errorMessage = error.response?.data?.error || error.message;
        if (error.response?.status === 403 || error.response?.status === 401) {
          ElMessage.error(errorMessage || "处理失败，请检查运行环境后重试");
        } else {
          ElMessage.error({
            message: errorMessage || "绕过Qvm处理失败，请确认已生成 exe 产物",
            type: "error",
          });
        }
      } finally {
        this.qvmProcessing = false;
      }
    },

    // 打开文件夹
    openDir() {
      axios
        .get(`http://127.0.0.1:${this.portConfig.mainPort}/opendir`)
        .catch(() => {
          ElMessage.error({
            message: "打开文件夹失败",
            type: "error",
          });
        });
    },


    async handleChangeExeSubmit() {
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
      // 先显示正在修改的提示框，点击即有反馈，再校验授权
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
            .post(`http://127.0.0.1:${this.portConfig.mainPort}/changefilesize`, {size: this.changeExeSizeForm.size}, {headers: {"Content-Type": "application/json"}})
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
              const errorMessage = error.response?.data?.error || error.message;
              // 如果是授权失败（403状态码），显示授权相关的错误信息
              if (error.response?.status === 403 || error.response?.status === 401) {
                ElMessageBox({
                  title: "处理失败",
                  message: errorMessage || "处理失败，请检查运行环境后重试",
                  type: "error",
                  showConfirmButton: false,
                  closeOnClickModal: false,
                  closeOnPressEscape: false,
                });
              } else {
                // 失败后弹出消息框
                ElMessageBox({
                  title: "文件处理失败",
                  message: "文件处理失败",
                  type: "error",
                  showConfirmButton: false,
                  closeOnClickModal: false,
                  closeOnPressEscape: false,
                });
              }

              this.selectedFile = null;
              this.clearUploadRef();
            });
      } else if (this.dialogMode === "removeUpx") {
        axios
            .get(`http://127.0.0.1:${this.portConfig.mainPort}/removeupx`)
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
              const errorMessage = error.response?.data?.error || error.message;
              ElMessage.error({
                message: "生成出错",
                type: "error",
              });
            });
      } else if (this.dialogMode === "addSignthief") {
        axios
            .get(`http://127.0.0.1:${this.portConfig.mainPort}/addSignthief`)
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
              const errorMessage = error.response?.data?.error || error.message;
              ElMessage.error({
                message: "生成出错",
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
  min-height: calc(100vh - 40px); /* 贴合视口，避免临界高度触发滚动条抖动 */
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

.auth-status {
  padding: 8px 16px;
  margin-bottom: 16px;
  border-radius: 4px;
  text-align: center;
  font-weight: inherit;
  font-size: 12px;
}

.auth-status.authorized {
  color: #03543f;
}

.auth-status.unauthorized {
  color: #981b1b;
}

.auth-status.clickable {
  cursor: pointer;
  transition: all 0.3s ease;
  user-select: none;
}

.auth-status.clickable:hover {
  background-color: rgba(152, 27, 27, 0.1);
  transform: scale(1.05);
  text-decoration: underline;
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
  flex: 0 0 calc(33.333% - 4px); /* 一行 3 个展示 */
  box-sizing: border-box; /* 包含padding和border在内的宽度 */
  margin-bottom: 5px; /* 设置子元素的底部外边距 */
}


.select-container {
  width: 45%;
}

/* 禁用状态下的样式 */
.el-select.is-disabled {
  width: 45%;
  opacity: 0.6;
  cursor: not-allowed;
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

/* 绕过Qvm：图标处理选择面板 */
.qvm-icon-panel {
  margin-top: 12px;
  padding: 10px 12px;
  background: rgba(0, 0, 0, 0.03);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  text-align: left;
}

.qvm-icon-head {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.qvm-icon-title {
  font-size: 13px;
  font-weight: 600;
  color: #4a5568;
  flex-shrink: 0;
}

.qvm-icon-spec {
  margin-top: 12px;
}

.qvm-icon-upload-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.qvm-icon-lib-title {
  margin: 12px 0 6px;
  font-size: 12px;
  color: #606266;
}

.qvm-icon-tip {
  margin-top: 10px;
  font-size: 12px;
  line-height: 1.6;
  color: #718096;
}

/* 图标选择弹窗 / 图标库选择（供 Qvm 面板复用） */
.icon-dlg-tip {
  font-size: 12px;
  color: #909399;
  margin-bottom: 10px;
}

.icon-dlg-upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.icon-dlg-label {
  font-size: 13px;
  color: #4a5568;
  font-weight: 500;
  flex-shrink: 0;
}

.icon-dlg-preview {
  width: 32px;
  height: 32px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background: #fff;
}

.icon-dlg-name {
  font-size: 12px;
  color: #334155;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.icon-dlg-name-placeholder {
  font-size: 12px;
  color: #c0c4cc;
}

.icon-dlg-divider-text {
  font-size: 12px;
  color: #909399;
}

.icon-dlg-loading {
  padding: 24px 0;
  text-align: center;
  font-size: 13px;
  color: #909399;
}

.icon-dlg-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(84px, 1fr));
  gap: 8px;
  max-height: 320px;
  overflow-y: auto;
  padding: 4px 2px;
}

.icon-dlg-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 4px 6px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  background: #fff;
}

.icon-dlg-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.18);
}

.icon-dlg-item.active {
  border-color: #409eff;
  background: #ecf5ff;
  box-shadow: 0 0 0 1px #409eff inset;
}

.icon-dlg-item img {
  width: 40px;
  height: 40px;
  object-fit: contain;
}

.icon-dlg-item-name {
  width: 100%;
  font-size: 11px;
  color: #606266;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 反沙箱校验配置卡片 */
.sandbox-config {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 14px;
  margin: 0 auto 12px;
  max-width: 720px;
  background: rgba(0, 0, 0, 0.03);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  text-align: left;
}

.sandbox-config-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 13px;
  color: #4a5568;
}

.sandbox-config-title .el-icon {
  color: #db7714;
}

.sandbox-config-desc {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 12px;
  color: #718096;
  line-height: 1.6;
}

.sandbox-config-desc .el-icon {
  margin-top: 3px;
  color: #a0aec0;
  flex-shrink: 0;
}

.sandbox-config-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sandbox-config-label {
  flex-shrink: 0;
  font-size: 12px;
  color: #4a5568;
  font-weight: 500;
}

.sandbox-config-input {
  max-width: 360px;
}

.sandbox-config-cmd {
  font-size: 12px;
  color: #db7714;
  background: rgba(219, 119, 20, 0.1);
  border-radius: 4px;
  padding: 4px 8px;
  font-family: monospace;
}

.action-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex-wrap: wrap;
}

.action-btns {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex-wrap: wrap;
}

.arch-toggle {
  display: inline-flex;
  align-items: center;
  margin: 0 10px;
  gap: 6px;
}

.arch-label {
  font-size: 13px;
  color: #606266;
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
  min-height: 150px; /* 内容自适应，不再固定高度裁剪 */
  border: 1px dashed #ddd;
  border-radius: 8px;
  padding: 20px;
  overflow: visible; /* 允许内容完整展示，避免上传列表被裁掉 */
}

/* DLL 代理文件名：长文件名截断，避免撑出横向滚动条 */
.dllproxy-file-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
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

/* exe 载荷选项（运行参数 / 保留控制台） */
.exe-option-row {
  margin: 0 auto 10px;
  max-width: 760px;
  padding: 10px 14px;
  text-align: left;
}

.exe-option-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.exe-option-params .el-input {
  max-width: 480px;
}

.exe-option-label {
  flex-shrink: 0;
  font-size: 13px;
  color: #4a5568;
  font-weight: 500;
}

.exe-option-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.6;
}

.status-message {
  margin-top: 15px;
}

/* SkyShadow 选择器样式 */
.skyshadow-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.dllproxy-row {
  flex-wrap: nowrap;
  justify-content: flex-start;
}

.dllproxy-label {
  min-width: 100px;
  color: #606266;
  font-size: 13px;
  text-align: left;
  white-space: nowrap;
}

.dllproxy-upload {
  flex-shrink: 0;
}

.skyshadow-select {
  flex: 1;
}

.skyshadow-refresh-btn {
  flex-shrink: 0;
  min-width: 36px;
  height: 32px;
  padding: 0;
  border: 1px solid #dcdfe6;
  background-color: #ffffff;
  color: #606266;
  border-radius: 4px;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.skyshadow-refresh-btn:hover {
  border-color: #409eff;
  color: #409eff;
  background-color: #ecf5ff;
  transform: rotate(180deg);
}

.skyshadow-refresh-btn:active {
  transform: rotate(360deg);
}

.skyshadow-refresh-btn .el-icon {
  font-size: 16px;
}

/* 白加黑程序下拉列表滚动样式（只保留一层滚动约束，避免原生+自定义双滚动条） */
:deep(.skyshadow-payload-dropdown .el-select-dropdown__list),
:deep(.skyshadow-functions-dropdown .el-select-dropdown__list) {
  max-height: 400px;
}

/* 多选标签折叠优化（已使用 collapse-tags，隐藏多余的原生滚动条） */
:deep(.skyshadow-select .el-select__tags) {
  max-height: 80px;
  overflow-y: auto;
  scrollbar-width: thin;
}

:deep(.skyshadow-select .el-select__tags .el-tag) {
  margin: 2px 4px 2px 0;
  max-width: 200px;
}

.footer {
  margin-top: 5px;
  font-size: 14px;
  color: #909399;
  height: auto !important; /* 覆盖 el-footer 默认 60px，避免免责声明被挤压溢出 */
  min-height: 0;
  padding: 4px 20px;
  line-height: 1.6;
}

/* 授权弹框样式 */
.glass-dialog-content {
  background: transparent;
  backdrop-filter: none;
  border-radius: 0;
  border: none;
  box-shadow: none;
  overflow: hidden;
  transition: all 0.3s ease;
}

.dialog-header {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  position: relative;
  overflow: hidden;
}

.auth-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.auth-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  position: relative;
  z-index: 2;
  transition: all 0.2s ease;
}

.auth-icon .el-icon {
  color: white;
  font-size: 20px;
}

.header-text {
  flex: 1;
  z-index: 2;
  position: relative;
}

.dialog-title {
  margin: 0;
  color: white;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.3px;
}

.auth-form-wrapper {
  padding: 0px;
  background: transparent;
}

.auth-form {
  padding: 0;
}

.auth-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.auth-form :deep(.el-form-item__label) {
  color: #666;
  font-weight: 500;
  font-size: 13px;
  padding-bottom: 6px;
}

.auth-form :deep(.el-input__wrapper) {
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  background: #fff;
}

.auth-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
}

.auth-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.25);
}

.auth-form :deep(.el-input__inner) {
  padding-left: 10px;
  font-size: 13px;
}

.auth-form :deep(.el-input__prefix) {
  left: 10px;
  color: #667eea;
  font-size: 14px;
}

.form-input {
  transition: all 0.3s ease;
}

.auth-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #fff3e0 0%, #ffe0b2 100%);
  border-left: 4px solid #ff9800;
  border-radius: 6px;
  color: #e65100;
  font-size: 13px;
  margin-top: 10px;
}

.auth-tip .el-icon {
  color: #ff9800;
  font-size: 16px;
  flex-shrink: 0;
}

.glass-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  background: transparent;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 0;
}

.auth-footer {
  background: transparent;
}

.footer-btn {
  border-radius: 6px;
  min-width: 80px;
  height: 32px;
  font-weight: 500;
  font-size: 13px;
  transition: all 0.2s ease;
  padding: 0 16px;
}

.footer-btn:active {
  transform: scale(0.98);
}

.cancel-btn {
  border: 1px solid #d9d9d9;
  color: #666;
  background: #fff;
}

.cancel-btn:hover:not(:disabled) {
  border-color: #4096ff;
  color: #4096ff;
}

.confirm-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  color: white;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.confirm-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #7c8ff0 0%, #8a5fb8 100%);
  box-shadow: 0 3px 12px rgba(102, 126, 234, 0.4);
}

.confirm-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>

<style>
/* 全局样式，美化授权弹框 */
.glass-auth-dialog .el-dialog {
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  border-radius: 20px;
  overflow: hidden;
  border: none;
}

.glass-auth-dialog .el-dialog__header {
  display: none;
}

.glass-auth-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent;
}

.glass-auth-dialog .el-dialog__footer {
  padding: 0 !important;
  background: transparent;
}

.glass-auth-dialog .el-overlay {
  backdrop-filter: blur(8px);
  background: rgba(0, 0, 0, 0.3);
}

.glass-auth-dialog .el-overlay-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 全局下拉列表滚动优化 */
.skyshadow-payload-dropdown .el-select-dropdown__list,
.skyshadow-functions-dropdown .el-select-dropdown__list {
  max-height: 400px;
  overflow-y: auto;
}

.skyshadow-payload-dropdown .el-scrollbar,
.skyshadow-functions-dropdown .el-scrollbar {
  max-height: 400px;
}

/* 优化滚动条样式 */
.skyshadow-payload-dropdown .el-scrollbar__bar,
.skyshadow-functions-dropdown .el-scrollbar__bar {
  opacity: 0.6;
}

.skyshadow-payload-dropdown .el-scrollbar__bar:hover,
.skyshadow-functions-dropdown .el-scrollbar__bar:hover {
  opacity: 1;
}
</style>
