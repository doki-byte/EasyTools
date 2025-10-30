<template>
  <el-container class="container">
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
    <el-main>
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
            <el-button type="primary" @click="processFile">处理文件</el-button>
            <el-button type="success" @click="openExcelPreview" v-if="isDealFile">
              预览 Excel 文件
            </el-button>
            <p v-if="isDealFile">文件保存位置：{{ excelFilePath }}</p>
          </div>

          <!-- Excel 预览 -->
          <div v-if="sheetsData.length" class="fscan-preview-card">
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

      <!--oss存储桶遍历-->
      <div v-if="activeTab === 'oss-list'" class="tab-content">
        <div class="oss-content">
          <div class="oss-header-card">
            <h4>选择 Oss资源桶 文件</h4>
            <div class="form-group">
              <el-input v-model="OsslistInput" placeholder="请输入Oss资源桶链接" class="input" />
              <el-button type="primary" @click="generateOssListQueries">获取数据</el-button>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="oss-result-card" v-if="OssListSuccess">
            <p> 文件保存位置：
              <span style="color: #4dcd31">{{ OssListSavePath }}</span>
            </p>
            <el-button type="success" @click="openXlsxFileDir">
              打开文件夹
            </el-button>
          </div>
        </div>
      </div>

      <!--wx小程序反编译-->
      <div v-if="activeTab === 'unwxapp'" class="tab-content">
        <div class="unwxapp-wrapper">
          <!-- 配置区域 -->
          <el-card shadow="hover" class="config-card">
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
                      <el-input
                          v-model="appletPath"
                          readonly
                          placeholder="请选择微信小程序存储路径"
                          class="path-input"
                      />
                      <el-button
                          type="primary"
                          @click="selectAppletPath"
                          class="select-btn"
                      >
                        选择路径
                      </el-button>
                      <el-tooltip
                          effect="dark"
                          placement="top"
                      >
                        <template #content>
                          <div style="text-align: left; line-height: 1.5;">
                            <div>• Win v3: C:\Users\用户名\Documents\WeChat Files\Applet (手工定位：PC微信->设置->文件管理)</div>
                            <div>• Win v4: C:\Users\用户名\AppData\Roaming\Tencent\xwechat\radium\Applet\packages</div>
                            <div>• Mac v3: /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/.wxapplet/packages</div>
                            <div>• Mac v4: /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages</div>
                          </div>
                        </template>
                        <el-icon class="help-icon"><QuestionFilled /></el-icon>
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
                    <el-space>
                      <el-button
                          :type="autoDecompile ? 'success' : 'primary'"
                          @click="toggleAutoDecompile"
                          size="small"
                      >
                        {{ autoDecompile ? '停用自动反编译' : '启用自动反编译' }}
                      </el-button>

                      <el-popconfirm
                          title="确认删除"
                          description="确认删除Applet目录下的所有文件吗？(会同时删除反编译后的所有文件)"
                          @confirm="clearApplet"
                      >
                        <template #reference>
                          <el-button size="small">清空Applet目录</el-button>
                        </template>
                      </el-popconfirm>

                      <el-popconfirm
                          title="确认删除"
                          description="确认删除所有反编译后的文件吗？"
                          @confirm="clearDecompiled"
                      >
                        <template #reference>
                          <el-button size="small">清空反编译文件</el-button>
                        </template>
                      </el-popconfirm>
                    </el-space>
                  </el-form-item>
                </el-form>
              </el-col>

              <el-col :span="12">
                <el-form label-width="120px" label-position="left">
                  <el-row :gutter="10">
                    <el-col :span="12">
                      <el-form-item label="信息提取规则">
                        <el-button
                            size="small"
                            type="success"
                            @click="showRulesDialog = true"
                        >
                          编辑规则
                        </el-button>
                      </el-form-item>
                    </el-col>
                    <el-col :span="12">
                      <el-form-item label="格式化代码">
                        <el-switch v-model="wxformat" />
                      </el-form-item>
                    </el-col>
                  </el-row>
                </el-form>
              </el-col>
            </el-row>
          </el-card>

          <!-- 主内容区域 -->
          <el-row :gutter="10" class="main-content">
            <!-- 左侧小程序列表 -->
            <el-col :span="11">
              <el-card class="applet-list-panel">
                <template #header>
                  <div class="panel-header">
                    <span>
                      小程序列表
                      <el-button
                          type="primary"
                          @click="loadAppletList"
                          size="small"
                          :loading="refreshing"
                      >
                        <el-icon><Refresh /></el-icon>
                        {{ refreshing ? '刷新中...' : '刷新列表' }}
                      </el-button>
                    </span>

                    <span class="total-count">共 {{ appletList.length }} 个</span>
                  </div>
                </template>

                <div class="applet-list-container">
                  <div
                      v-for="app in appletList"
                      :key="app.AppID"
                      class="applet-item"
                  >
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
                      <div class="applet-info">
                        <el-tag
                            v-if="app.Info && app.Info.nickname && app.Info.nickname !== ''"
                            type="danger"
                            size="small"
                            class="nickname-tag"
                            :title="app.Info.nickname"
                        >
                          {{ app.Info.nickname }}
                        </el-tag>
                        <el-tag
                            v-if="app.Info && app.Info.username && app.Info.username !== ''"
                            type="info"
                            size="small"
                        >
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
                      <div
                          v-for="version in app.Versions"
                          :key="`${app.AppID}-${version.Number}`"
                          class="version-item"
                      >
                        <div class="version-info">
                          <el-tag
                              :type="getVersionTagType(version)"
                              class="version-tag"
                          >
                            <el-icon class="version-icon"><Files /></el-icon>
                            {{ version.Number }}
                          </el-tag>

                          <el-space class="version-actions" :size="4">
                            <!-- 反编译按钮 -->
                            <el-tooltip content="反编译">
                              <el-button
                                  :disabled="version.DecompileStatus === 'Running'"
                                  :loading="version.DecompileStatus === 'Running'"
                                  size="small"
                                  @click="decompileApplet(app, version)"
                              >
                                <template #icon>
                                  <el-icon><MagicStick /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>

                            <!-- 提取敏感信息按钮 -->
                            <el-tooltip content="提取敏感信息">
                              <el-button
                                  :disabled="version.MatchStatus === 'Running'"
                                  :loading="version.MatchStatus === 'Running'"
                                  size="small"
                                  @click="extractSensitiveInfo(app, version)"
                              >
                                <template #icon>
                                  <el-icon><Search /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>

                            <el-tooltip content="打开文件夹">
                              <el-button
                                  size="small"
                                  @click="openAppletFolder(app, version)"
                              >
                                <template #icon>
                                  <el-icon><FolderOpened /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>

                            <el-tooltip content="敏感信息">
                              <el-button
                                  size="small"
                                  @click="showMatchedInfo(app, version)"
                              >
                                <template #icon>
                                  <el-icon><Document /></el-icon>
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

                <el-input
                    v-model="matchedResult"
                    type="textarea"
                    placeholder="选中版本的反编译结果和敏感信息将显示在这里"
                    :rows="25"
                    class="result-textarea"
                    resize="none"
                />
              </el-card>
            </el-col>
          </el-row>

          <!-- 规则编辑对话框 -->
          <el-dialog
              v-model="showRulesDialog"
              title="信息提取规则"
              width="700px"
              destroy-on-close
          >
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

              <el-input
                  v-model="rulesText"
                  type="textarea"
                  placeholder="每行一条正则表达式规则"
                  :rows="15"
                  class="rules-textarea"
                  resize="none"
              />
            </div>
          </el-dialog>
        </div>
      </div>

      <!--jwt密钥爆破-->
      <div v-if="activeTab === 'jwt_crack'" class="jwt-content">
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
                      <el-icon><CopyDocument /></el-icon></el-button>
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
        </el-row>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import {DealOssList, FscanResultDeal, GetExcelContent, UploadFile} from "../../wailsjs/go/controller/InfoDeal";
import {AutoDecompile, ClearApplet, ClearDecompiled, Decompile, ExtractSensitiveInfo, GetAllMiniApp, GetAppletPath, GetMatchedString, GetWechatRules, InitCheck, SaveWechatRules, SelectDirectory, SetAppletPath} from "../../wailsjs/go/controller/UnWxapp";
import {ElMessage, ElMessageBox} from "element-plus";
import * as XLSX from "xlsx";
import {GetConfigDir, OpenPath} from "../../wailsjs/go/controller/System";
import {BruteForceJWT, ChooseJwtFile, DecodeJWTWithAlg, EncodeJWTWithAlg, GetDefaultDictPath} from "../../wailsjs/go/controller/JwtCrackController";
import {BrowserOpenURL} from "../../wailsjs/runtime";
import { loadMenuOrder, moduleTabsConfig } from '@/utils/menuConfig';
import {
  CopyDocument,
  Delete,
  Close,
  Collection,
  MagicStick,
  QuestionFilled,
  Refresh,
  Search,
  Files,
  FolderOpened,
  Unlock, Edit, Document
} from '@element-plus/icons-vue'


export default {
  name: "InfoDealView",
  components: {
    Edit,
    Unlock,
    Collection,
    Close,
    Delete,
    CopyDocument,
    Search,
    Refresh,
    QuestionFilled,
    Files,
    MagicStick,
    FolderOpened,
    Document
  },
  data() {
    return {
      activeTab: "", // 初始为空，等配置加载后设置
      moduleTabs: [], // 模块标签页配置
      tabsKey: Date.now(),

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

      OsslistInput: "", //oss资源桶路径
      OssListSuccess: false, //oss处理结果
      OssListSavePath: "", //oss处理结果文件保存位置

      // 小程序反编译配置相关
      appletPath: "",
      autoDecompile: false,
      wxformat: true,
      rulesText: "",
      showRulesDialog: false,

      // 数据相关
      appletList: [],
      selectedApplet: null,
      matchedResult: "",

      // 状态相关
      refreshing: false,
      dataCachePath: "",
      pollInterval: null,
      pollingTasks: new Set(), // 正在轮询的任务集合

      //jwt相关
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
    },

    activeSheet() {
      this.paginatedData = this.getActiveSheetData(); // 更新分页数据
      this.currentPage = 1; // 每次切换表格时重置为第一页
    }
  },

  async mounted() {
    // 加载小程序反编译配置
    await this.initUnwxAppData();

    // 加载标签页配置
    await this.loadTabsConfig();
    this.configLoaded = true;

    // 监听菜单更新事件
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
  },
  beforeUnmount() {
    // 清理所有轮询
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
    this.pollingTasks.clear();

    // 在组件卸载时移除事件监听器
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
        const infoDealTabs = savedTabsOrder.infoDeal || [];

        // 获取默认配置
        const defaultTabs = moduleTabsConfig.infoDeal || [];

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
        infoDealTabs.forEach(savedTab => {
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
      this.moduleTabs = moduleTabsConfig.infoDeal.map(tab => ({
        ...tab,
        order: tab.defaultOrder
      }));
      this.setDefaultActiveTab();
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
    async openXlsxFileDir() {
      const baseDir = await GetConfigDir()
      const fileDir = baseDir + "/file"; // 拼接file子目录
      await OpenPath(fileDir)
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
        this.appletList = (data || []).map(app => {
          return {
            AppID: app.AppID || app.app_id || app.AppId || '',
            UpdateDate: app.UpdateDate || app.update_date || app.updateDate || '',
            Info: app.Info || app.info || {},
            Versions: (app.Versions || app.versions || app.Version || []).map(version => ({
              Number: version.Number || version.number || version.Version || version.version || '',
              DecompileStatus: version.DecompileStatus || version.decompileStatus || version.status || 'Waiting',
              MatchStatus: version.MatchStatus || version.matchStatus || 'Waiting',
              Message: version.Message || version.message || ''
            }))
          };
        });

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
        await ClearDecompiled();
        ElMessage.success("反编译文件已清空");
        // 立即刷新列表
        await this.loadAppletList();
      } catch (error) {
        ElMessage.error("清空失败: " + error);
      }
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
          await this.loadAppletList(); // 刷新列表

          // 检查任务是否完成
          const task = this.findTask(appID, versionNumber);
          if (task &&
              task.DecompileStatus !== 'Running' &&
              task.MatchStatus !== 'Running') {
            // 任务完成，停止轮询
            clearInterval(pollInterval);
            this.pollingTasks.delete(taskKey);
            console.log(`停止轮询任务: ${taskKey}`);
          }
        } catch (error) {
          console.error('轮询任务状态失败:', error);
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

        // 构建包路径
        const packagePath = `${this.appletPath}\\${app.AppID}\\${version.Number}`;

        // 构建输出路径
        const isWin = navigator.platform.toLowerCase().startsWith('win');
        const sep = isWin ? '\\' : '/';
        const outputPath = `${packagePath}`;

        // console.log("打开文件夹:", outputPath);

        // 调用打开路径的方法
        await OpenPath(outputPath);

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
          const formattedResult = result.map(line => {
            if (line.includes(" -> ")) {
              const [file, content] = line.split(" -> ");
              return `📄 ${file}\n   🔍 ${content}`;
            }
            return line;
          }).join("\n\n");

          this.matchedResult = formattedResult;
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
        }
        if (this.jwtpayload) {
          const payloadObj = JSON.parse(this.jwtpayload);
          this.jwtpayload = JSON.stringify(payloadObj, null, 2);
        }
        ElMessage.success('JSON格式化完成');
      } catch (error) {
        ElMessage.error('JSON格式错误，无法格式化');
      }
    },

    // 清空所有内容
    clearAll() {
      this.$confirm('确定要清空所有JWT相关内容吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.jwtInput = '';
        this.jwtheader = '';
        this.jwtpayload = '';
        this.jwtsecret = ''; // 保留默认密钥
        this.jwtsignature = '';
        this.dictPath = '选择jwt字典路径';
        this.progress = 0;
        ElMessage.success('已清空所有内容');
      });
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

      // 如果还没有验证过秘钥，先验证
      if (!this.jwtverify) {
        this.$confirm('尚未验证秘钥的正确性，是否先测试秘钥？', '提示', {
          confirmButtonText: '先测试秘钥',
          cancelButtonText: '直接编码',
          type: 'warning'
        }).then(() => {
          this.testSecret().then(() => {
            if (this.jwtverify) {
              this.doEncode();
            }
          });
        }).catch(() => {
          // 用户选择直接编码，但仍然给出警告
          this.$confirm('直接编码可能使用错误的秘钥，是否继续？', '警告', {
            confirmButtonText: '继续',
            cancelButtonText: '取消',
            type: 'error'
          }).then(() => {
            this.doEncode();
          });
        });
        return;
      }

      // 已经验证过秘钥，直接编码
      this.doEncode();
    },

    // 执行编码
    async doEncode() {
      let headerObj = {};
      let payloadObj = {};

      try {
        // 解析header
        if (this.jwtheader && this.jwtheader.trim()) {
          headerObj = JSON.parse(this.jwtheader);
        } else {
          ElMessage.warning('请提供有效的Header');
          return;
        }

        // 解析payload
        if (this.jwtpayload && this.jwtpayload.trim()) {
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
          if (result.error === 'token signature is invalid: signature is invalid') {
            ElMessage.error("签名无效");
          }else {
            ElMessage.error(result.error);
          }
        }
      } catch (error) {
        ElMessage.error("JWT编码失败");
      }
    },

    // 修改解析方法，默认使用空秘钥进行测试
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

          // 根据验证结果设置状态
          this.jwtverify = result.valid;

          if (result.valid) {
            ElMessage.success("JWT解析成功 - 无签名或签名验证通过");
            // 如果是无签名验证通过，可能是None算法
            if (this.value === 'None' || (result.header && result.header.alg === 'none')) {
              this.jwtsecret = ''; // 清空秘钥
              ElMessage.info("检测到无签名JWT (None算法)");
            }
          } else {
            ElMessage.warning("JWT解析完成，但签名验证失败 - 需要正确秘钥");
            this.jwtsecret = ''; // 清空当前秘钥，提示用户需要输入正确秘钥
          }
        } else {
          // 解析出错，可能是格式错误
          ElMessage.error("JWT解析失败: " + result.error);
          this.jwtverify = false;
        }
      } catch (error) {
        ElMessage.error(error.message || "JWT 解析失败");
        this.jwtverify = false;
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
  }
};
</script>

<style scoped>
/* 页面容器 - 修复撑满屏幕问题 */
.container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #f8f9fb;
  padding: 0 10px;
  box-sizing: border-box;
}

/* 顶部 Tabs 样式 */
.tabs {
  background-color: #ffffff;
  box-shadow: 0 2px 2px rgba(0, 0, 0, 0.1);
  margin-bottom: 5px;
  border-radius: 10px 10px 10px 10px;
  padding: 0 10px;
  flex-shrink: 0;
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
  margin: 10px auto;
  padding: 8px 16px;
  margin-top: 25px;
  border-radius: 5px;
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
  flex-wrap: wrap;
  gap: 10px;
}

/* ============ 小程序反编译样式 ============ */
.unwxapp-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 8px 0;
  min-height: 0;
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

.main-content {
  flex: 1;
  display: flex;
  gap: 5px;
}

.applet-list-panel,
.detail-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 300px;
  margin-left: 5px;
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
  max-height: 450px;
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
}

.appid-tag,
.date-tag {
  font-weight: bold;
}

.nickname-tag {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-list {
  margin-top: 8px;
  max-height: 200px;
  overflow-y: auto;
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
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.result-textarea {
  flex: 1;
  border: none;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  min-height: 200px;
  resize: vertical;
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

.rules-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
}

/* ============ 其他界面统一应用卡片样式 ============ */

/* Fscan结果处理 */
.fscan-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.fscan-header-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.fscan-preview-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

/* 蓝队IP封禁 */
.ip-ban-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.ip-ban-info-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.ip-ban-input-card {
  background: white;
  border-radius: 6px;
  padding: 5px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* OSS存储桶遍历 */
.oss-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.oss-header-card {
  background: white;
  border-radius: 6px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.oss-result-card {
  background: white;
  border-radius: 6px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex: 1;
}

/* JWT 专用样式 */
.jwt-content {
  flex: 1;
  padding: 10px 0;
  min-height: 0;
}

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
}

.section-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
}

.config-item {
  margin-bottom: 16px;
}

.config-item:last-child {
  margin-bottom: 0;
}

.config-label {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
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
  font-size: 12px;
  color: #909399;
}

/* 响应式设计 */
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
}

@media (max-width: 480px) {
  .jwt-card {
    margin-bottom: 15px;
  }
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

/* 响应式调整 */
@media (max-width: 768px) {
  .file-selector {
    flex-direction: column;
    align-items: stretch;
  }

  .select-btn {
    width: 100%;
    margin-bottom: 8px;
  }
}

/* 响应式设计 */


@media (max-width: 768px) {
  .container {
    padding: 0 5px;
  }

  .row {
    flex-direction: column;
  }

  .column {
    width: 100%;
    min-width: unset;
    padding: 0 5px;
  }

  .main-content {
    flex-direction: column;
  }

  .applet-list-panel,
  .detail-panel {
    height: 350px;
  }

  /* 在小屏幕上调整按钮布局 */
  .actions {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .jwt-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .jwt-actions .el-button {
    margin-bottom: 8px;
  }

  .version-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .fscan-header-card,
  .ip-ban-info-card,
  .oss-header-card,
  .jwt-card {
    padding: 15px;
  }
}

@media (max-width: 480px) {
  .tabs {
    padding: 0 5px;
  }

  .tab-content {
    gap: 8px;
    padding: 4px 0;
  }

  .config-row .el-col {
    margin-bottom: 10px;
  }

  .path-selector {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .path-input {
    width: 100% !important;
  }

  .form-group {
    flex-direction: column;
    align-items: stretch;
  }

  .form-group .el-input {
    margin-bottom: 10px;
  }

  .applet-basic {
    flex-direction: column;
    align-items: flex-start;
  }

  .applet-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .rules-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .rules-toolbar .el-button {
    margin-bottom: 8px;
  }
}

/* 确保所有flex容器都能正确收缩 */
.flex-container {
  min-height: 0;
}

/* 修复版本列表的滚动 */
.version-list {
  margin-top: 8px;
  max-height: 150px;
  overflow-y: auto;
  flex-shrink: 0;
}

/* 修复上传区域 */
.upload-demo {
  width: 100%;
}

/* 确保文本区域可以自适应 */
.el-textarea__inner {
  resize: vertical;
  min-height: 80px;
}

/* 修复进度条布局 */
.jwt-progress {
  margin-top: 20px;
  width: 100%;
}

/* 修复操作按钮在小屏幕上的布局 */
@media (max-width: 576px) {
  .version-actions {
    flex-wrap: wrap;
    justify-content: flex-start;
    width: 100%;
  }

  .version-actions .el-button {
    margin: 2px;
    flex: 1;
    min-width: 40px;
  }

  .applet-basic {
    flex-direction: column;
    align-items: flex-start;
  }

  .applet-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .ip-copy-button {
    position: relative;
    top: 0;
    right: 0;
    margin-top: 5px;
    width: 100%;
  }

  .ip-input-container {
    display: flex;
    flex-direction: column;
  }
}

/* 分页组件响应式 */
@media (max-width: 768px) {
  .el-pagination {
    justify-content: center;
  }

  .el-pagination .btn-prev,
  .el-pagination .btn-next,
  .el-pagination .number {
    margin: 0 2px;
    min-width: 32px;
  }
}

/* 表格在小屏幕上的响应式 */
@media (max-width: 768px) {
  .excel-preview-table {
    font-size: 12px;
  }

  .excel-preview-table th,
  .excel-preview-table td {
    padding: 4px 6px;
  }
}

/* 确保标签在小屏幕上不换行 */
.el-tag {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
</style>
