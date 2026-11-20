<template>
  <el-container class="ssh-page">
    <el-header style="
        text-align: left;
        height: 24px;
        padding-left: 0px;
        padding-right: 0px;
      ">
      <el-row class="action-row">
        <el-col :span="14">
          <el-button-group>
            <!-- 打开已存在主机配置 -->
            <el-button type="primary" size="small" :icon="Menu" @click="handleOpenHostList">打开</el-button>

            <el-button type="primary" size="small" @click="newHost" :icon="CirclePlus">新建</el-button>
            <!-- 执行命令及收藏 -->
            <el-button type="primary" size="small" :icon="Edit" @click="openCmdDialog">执行命令</el-button>

            <!-- 命令收藏列表 -->
            <el-button type="primary" size="small" :icon="Star" @click="dialogCmdFavVisible = true">命令收藏</el-button>

            <!-- 主机列表对话框 -->
            <el-dialog v-model="dialogHostListVisible" title="主机列表" width="70%" top="50px" class="ssh-dialog"
              align-center>
              <div class="dialog-toolbar">
                <el-input v-model="searchHost" size="small" placeholder="名称及主机搜索" clearable />
                <el-tag type="info" effect="plain">共 {{ filterHostTable.length }} 条</el-tag>
              </div>
              <el-table v-if="filterHostTable.length > 0" :data="pagedHostTable" height="360" size="small"
                :show-overflow-tooltip="true" :fit="false" class="glass-card">
                <el-table-column sortable fixed="left" width="140" property="name" label="名称"
                  :resizable="true"></el-table-column>
                <el-table-column sortable min-width="140" property="address" label="主机"
                  :resizable="true"></el-table-column>
                <el-table-column sortable width="90" property="user" label="用户" :resizable="true"></el-table-column>
                <el-table-column sortable width="70" property="port" label="端口" :resizable="true"></el-table-column>
                <el-table-column label="操作" fixed="right" width="210" :resizable="true">
                  <template #default="scope">
                    <el-button size="small" @click="editHost(scope.row)">编辑</el-button>
                    <el-button size="small" type="danger" @click="deleteHost(scope.row)">删除</el-button>
                    <el-button size="small" type="primary" @click="connectHost(scope.row)">连接</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div v-if="filterHostTable.length > 0"
                style="display:flex;justify-content:flex-end;margin-top:8px;">
                <el-pagination v-model:current-page="hostPagination.currentPage"
                  v-model:page-size="hostPagination.pageSize" :page-sizes="[10, 20, 50, 100]"
                  layout="total, sizes, prev, pager, next, jumper" :total="filterHostTable.length" small />
              </div>
              <div v-else-if="filterHostTable.length === 0" class="empty-host-list">
                <el-empty description="暂无主机配置">
                  <el-button type="primary" :icon="CirclePlus" @click="handleNewHostFromEmpty">新建主机</el-button>
                </el-empty>
              </div>
            </el-dialog>

            <!-- 执行命令对话框 -->
            <el-dialog v-model="dialogCmdExecVisible" width="64%" top="40px" title="执行命令" class="cmd-dialog ssh-dialog"
              align-center :close-on-click-modal="false" :close-on-press-escape="false" :destroy-on-close="false">
              <el-form :model="cmd" label-width="80px" size="small">
                <el-form-item label="执行命令">
                  <el-input v-model="cmd.data" type="textarea" :rows="4" autocomplete="off" placeholder="命令或脚本" />
                </el-form-item>
                <el-row :gutter="8">
                  <el-col :span="12">
                    <el-form-item label="会话选择">
                      <el-radio-group v-model="cmd.node">
                        <el-radio value="current">当前会话</el-radio>
                        <el-radio value="all">所有会话</el-radio>
                      </el-radio-group>
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="收藏名称">
                      <el-input v-model="cmd.name" maxlength="32" show-word-limit placeholder="需要收藏时填写">
                        <template #append>
                          <el-button-group style="display: flex; flex-wrap: nowrap;">
                            <el-button @click="addCmdNote" style="color: #138dda">收藏</el-button>
                            <el-button @click="execCmd" style="color: orange">执行</el-button>
                          </el-button-group>
                        </template>
                      </el-input>
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-form>
            </el-dialog>

            <!-- 命令收藏对话框 -->
            <el-dialog v-model="dialogCmdFavVisible" title="命令收藏" width="70%" top="60px" class="ssh-dialog"
              align-center>
              <div class="dialog-toolbar">
                <el-input v-model="searchCmdNote" size="small" placeholder="名称搜索" clearable />
                <el-tag type="info" effect="plain">共 {{ filterCmdNoteTable.length }} 条</el-tag>
              </div>
              <el-table :data="filterCmdNoteTable" height="360" size="small" :fit="false" class="glass-card">
                <el-table-column sortable width="180" :show-overflow-tooltip="true" property="cmd_name" label="名称"
                  :resizable="true"></el-table-column>
                <el-table-column sortable property="cmd_data" label="命令" :resizable="true">
                  <template #default="scope">
                    <el-popover effect="light" trigger="hover" placement="right" width="auto">
                      <template #default>
                        <div>命令详情</div>
                        <div>
                          <el-input v-model="scope.row.cmd_data" style="width: 450px"
                            :autosize="{ minRows: 4, maxRows: 20 }" type="textarea" :disabled="true" />
                        </div>
                        <div>
                          </br>
                          <el-button-group>
                            <el-tooltip effect="dark" content="执行命令,发送到所有会话" placement="top-start">
                              <el-button type="warning" @click="execCmdAllSession(scope.row)">发送所有会话</el-button>
                            </el-tooltip>
                            <el-tooltip effect="dark" content="执行命令,发送到当前会话" placement="top-start">
                              <el-button type="primary" @click="execCmdCurrentSession(scope.row)">发送当前会话</el-button>
                            </el-tooltip>
                          </el-button-group>
                        </div>
                      </template>
                      <template #reference>
                        {{ scope.row.cmd_data.substring(0, 15) + "..." }}
                      </template>
                    </el-popover>
                  </template>
                </el-table-column>
                <el-table-column label="操作" fixed="right" width="260">
                  <template #default="scope">
                    <el-button-group>
                      <el-button type="danger" @click="delCmdNote(scope.row.id)">删除</el-button>
                      <el-tooltip effect="dark" content="执行命令,发送到所有会话" placement="top-start">
                        <el-button type="warning" @click="execCmdAllSession(scope.row)">发送所有会话</el-button>
                      </el-tooltip>
                      <el-tooltip effect="dark" content="执行命令,发送到当前会话" placement="top-start">
                        <el-button type="primary" @click="execCmdCurrentSession(scope.row)">发送当前会话</el-button>
                      </el-tooltip>
                    </el-button-group>
                  </template>
                </el-table-column>
              </el-table>
            </el-dialog>

            <!-- SSH主机配置弹窗 -->
            <el-dialog :title="data.mode == 0 ? '新增主机' : '更新主机'" v-model="data.host_dialog_visible" width="70%"
              top="40px" class="ssh-dialog" align-center>
              <el-form label-width="80px" ref="host_from">
                <el-collapse v-model="data.host_config_collapse">
                  <el-collapse-item title="基础配置" name="1">
                    <el-row>
                      <el-col :span="16">
                        <el-form-item label="名称" prop="name">
                          <el-input v-model.trim="data.name" minlength="1" maxlength="30" show-word-limit
                            placeholder="请输入名称"></el-input>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="16">
                        <el-form-item label="主机" prop="address">
                          <el-input v-model.trim="data.address" minlength="1" maxlength="60" show-word-limit
                            placeholder="请输入主机地址"></el-input>
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="网络" prop="net_type">
                          <el-radio-group v-model="data.net_type">
                            <el-radio value="tcp4">IPv4</el-radio>
                            <el-radio value="tcp6">IPv6</el-radio>
                          </el-radio-group>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="16">
                        <el-form-item label="用户" prop="user">
                          <el-input minlength="1" maxlength="60" v-model.trim="data.user" show-word-limit
                            placeholder="请输入用户名"></el-input>
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="端口" prop="port">
                          <el-input-number v-model="data.port" :min="1" :max="65535"></el-input-number>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-form-item label="认证方式">
                        <el-radio-group v-model="data.auth_type">
                          <el-radio value="pwd">密码</el-radio>
                          <el-radio value="cert">证书</el-radio>
                        </el-radio-group>
                      </el-form-item>
                    </el-row>
                    <el-row v-if="data.auth_type === 'cert'">
                      <el-col :span="16">
                        <el-form-item label="证书">
                          <el-input v-model="data.cert_data" type="textarea" placeholder="请输入证书内容或上传" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="上传">
                          <el-button type="primary" @click="addCertFile">上传证书文件</el-button>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="16">
                        <el-form-item v-if="data.auth_type === 'cert'" label="证书密码" prop="cert_pwd">
                          <el-input minlength="" maxlength="60" v-model.trim="data.cert_pwd" type="password"
                            show-password show-word-limit placeholder="证书密码"></el-input>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="16">
                        <el-form-item v-if="data.auth_type === 'pwd'" label="SSH密码" prop="pwd">
                          <el-input minlength="1" maxlength="60" v-model.trim="data.pwd" type="password" show-password
                            show-word-limit placeholder="SSH密码"></el-input>
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </el-collapse-item>

                  <el-collapse-item title="高级配置" name="2">
                    <el-row>
                      <el-col :span="9">
                        <el-form-item label="终端类型" prop="pty_type">
                          <el-select style="width: 130px" v-model="data.pty_type" placeholder="请选择终端类型">
                            <el-option label="xterm-256color" value="xterm-256color" />
                            <el-option label="linux" value="linux" />
                            <el-option label="xtrem" value="xtrem" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="字体颜色" prop="foreground">
                          <el-color-picker v-model="data.foreground"></el-color-picker>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="背景颜色" prop="background">
                          <el-color-picker v-model="data.background"></el-color-picker>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="光标颜色" prop="cursor_color">
                          <el-color-picker v-model="data.cursor_color"></el-color-picker>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="9">
                        <el-form-item label="字体">
                          <el-select style="width: 130px" v-model="data.font_family" placeholder="请选择字体">
                            <el-option label="Courier" value="Courier" />
                            <el-option label="Courier New" value="Courier New" />
                            <el-option label="Menlo" value="Menlo" />
                            <el-option label="Monaco" value="Monaco" />
                            <el-option label="monospace" value="monospace" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="字体大小">
                          <el-select v-model.number="data.font_size" placeholder="请选择字体大小">
                            <el-option label="8" value="8" />
                            <el-option label="12" value="12" />
                            <el-option label="14" value="14" />
                            <el-option label="16" value="16" />
                            <el-option label="18" value="18" />
                            <el-option label="20" value="20" />
                            <el-option label="22" value="22" />
                            <el-option label="24" value="24" />
                            <el-option label="26" value="26" />
                            <el-option label="28" value="28" />
                            <el-option label="30" value="30" />
                            <el-option label="32" value="32" />
                            <el-option label="34" value="34" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="光标样式">
                          <el-select v-model="data.cursor_style" placeholder="请选择光标样式">
                            <el-option label="块状" value="block" />
                            <el-option label="下划线" value="underline" />
                            <el-option label="竖线" value="bar" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="Shell">
                          <el-select v-model="data.shell" placeholder="请选择Shell">
                            <el-option label="/bin/sh" value="/bin/sh" />
                            <el-option label="bash" value="bash" />
                            <el-option label="csh" value="csh" />
                            <el-option label="zsh" value="zsh" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="24">
                        <el-form-item label="连接命令">
                          <el-input v-model="data.init_cmd" type="textarea" :rows="1" placeholder="请输入连接后执行命令" />
                        </el-form-item>
                      </el-col>
                    </el-row>
                    <el-row>
                      <el-col :span="24">
                        <el-form-item label="连接横幅">
                          <el-input v-model="data.init_banner" type="textarea" :rows="1" placeholder="请输入连接后提示横幅" />
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </el-collapse-item>
                </el-collapse>
              </el-form>
              <template #footer>
                <span class="dialog-footer">
                  <el-button @click="data.host_dialog_visible = false">取消</el-button>
                  <el-button type="success" @click="connect">连接</el-button>
                </span>
                &nbsp;&nbsp;&nbsp;&nbsp;
                <span v-if="data.mode == 0" class="dialog-footer">
                  <el-button type="primary" @click="createHost(false)">保存</el-button>
                  <el-button type="primary" @click="createHost(true)">连接并保存</el-button>
                </span>
                <span v-if="data.mode == 1" class="dialog-footer">
                  <el-button type="primary" @click="updateHost(false)">更新</el-button>
                  <el-button type="primary" @click="updateHost(true)">连接并更新</el-button>
                </span>
              </template>
            </el-dialog>

            <!-- SSH 文件管理侧边栏（替代原来的弹窗） -->
            <el-drawer v-model="data.file_dialog_visible" size="52%" direction="rtl" :with-header="false"
              class="file-drawer">
              <div class="file-drawer-header">
                <div class="file-drawer-title" v-html="title"></div>
              </div>

              <div class="file-drawer-body" :class="{ 'has-preview': filePreview.file }">
                <!-- 目录 & 文件列表 + 预览：上下布局 -->
                <div class="file-pane">
                  <el-form-item style="margin-top: 8px; margin-bottom: 8px">
                    <el-input
                      v-model="data.sftp_current_dir"
                      placeholder="点击前缀路径快速跳转，或在此输入完整路径"
                      class="input-with-select"
                      size="small"
                      style="min-width: 360px; max-width: 100%;"
                      @keyup.enter="handlePathInputEnter"
                    >
                      <template #prepend>
                        <el-button-group class="path-breadcrumbs">
                          <el-button v-for="(item, index) in data.dir_info.paths.slice(-3)" :key="item.dir || index"
                            @click="listDir(item.dir, data.current_host)" size="small" text>
                            {{ item.name }}
                          </el-button>
                        </el-button-group>
                      </template>
                      <template #append>
                        <el-button-group class="path-actions">
                          <el-tooltip content="返回上一级" placement="top">
                            <el-button circle size="small" @click="goParentDir" :icon="Back">
                            </el-button>
                          </el-tooltip>
                          <el-tooltip content="上传文件" placement="top">
                            <el-button circle size="small" @click="openUploadDialog" :icon="Upload" />
                          </el-tooltip>
                          <el-tooltip content="创建目录" placement="top">
                            <el-button circle size="small"
                              @click="createDir(data.sftp_current_dir || '/', data.current_host)"
                              :icon="FolderOpened" />
                          </el-tooltip>
                          <el-tooltip content="刷新列表" placement="top">
                            <el-button circle size="small" @click="listDir(null, data.current_host)" :icon="Refresh" />
                          </el-tooltip>
                        </el-button-group>
                      </template>
                    </el-input>
                  </el-form-item>

                  <el-row v-if="data.sftp_upload_status.isUploading">
                    <el-col :span="24">
                      <div class="upload-progress-container">
                        <div class="upload-progress-header">
                          <span class="upload-status-text">
                            {{ data.sftp_upload_status.uploadedFiles }} / {{ data.sftp_upload_status.totalFiles }} 个文件
                            <span v-if="data.sftp_upload_status.currentFile" class="current-file-name">
                              - 正在上传: {{ data.sftp_upload_status.currentFile }}
                            </span>
                          </span>
                          <!-- <span class="upload-percentage">{{ data.sftp_upload_percentage }}%</span> -->
                          <el-button
                            v-if="data.sftp_upload_status.isUploading"
                            type="danger"
                            text
                            size="small"
                            @click="requestCancelUpload"
                            style="margin-left: 12px;"
                          >
                            取消上传
                          </el-button>
                        </div>
                        <el-progress
                          :percentage="data.sftp_upload_percentage"
                          :status="data.sftp_upload_percentage === 100 ? 'success' : ''"
                          :stroke-width="12"
                        />
                        <div v-if="data.sftp_upload_status.totalSize > 0" class="upload-size-info">
                          {{ formatFileSize(data.sftp_upload_status.uploadedSize) }} /
                          {{ formatFileSize(data.sftp_upload_status.totalSize) }}
                        </div>
                      </div>
                    </el-col>
                  </el-row>

                  <div class="file-table-wrapper" @drop.prevent="handleFileAreaDrop" @dragover.prevent
                    @dragenter.prevent>
                    <el-table
                      ref="fileTable"
                      :data="data.dir_info.files"
                      size="small"
                      :show-overflow-tooltip="true"
                      :fit="true"
                      border
                      style="width: 100%"
                      @row-contextmenu="handleFileRowContextMenu"
                      class="file-table">
                      <el-table-column
                        prop="name"
                        label="文件名"
                        fixed="left"
                        sortable
                        :show-overflow-tooltip="false"
                        min-width="220"
                        :resizable="true">
                        <template #default="scope">
                          <div class="file-name-cell" @dblclick="handleFileNameDblClick(scope.row)">
                            <el-icon v-if="scope.row.type === 'd'" class="file-icon dir-icon">
                              <FolderOpened />
                            </el-icon>
                            <el-icon v-else class="file-icon file-icon-normal">
                              <Files />
                            </el-icon>
                            <el-tooltip :content="scope.row.name" placement="top-start" effect="dark"
                              :disabled="scope.row.name.length <= 40">
                              <span class="file-name-text">
                                {{ scope.row.name }}
                              </span>
                            </el-tooltip>
                          </div>
                        </template>
                      </el-table-column>
                      <el-table-column
                        prop="size"
                        label="大小"
                        min-width="90"
                        sortable
                        align="right"
                        :resizable="true">
                        <template #default="scope">
                          {{ formatFileSize(scope.row.size) }}
                        </template>
                      </el-table-column>
                      <el-table-column prop="mode" label="权限" min-width="90" sortable align="center" :resizable="true" />
                      <el-table-column prop="mod_time" label="修改日期" min-width="130" sortable :resizable="true" />
                    </el-table>
                  </div>

                  <!-- 下方：文件预览 / 编辑（仅在选择文件时显示） -->
                  <div class="preview-pane" v-if="filePreview.file">
                    <div class="preview-header">
                      <div class="preview-title">
                        <span>预览：{{ filePreview.file.name }}</span>
                      </div>
                      <div class="preview-actions">
                        <el-tag v-if="filePreview.isText" size="small" type="info" effect="plain">
                          {{ formatFileSize(filePreview.file.size) }}
                        </el-tag>
                        <el-button v-if="filePreview.isText" size="small" type="primary" :disabled="!filePreview.dirty"
                          @click="saveFileEdit">
                          保存修改
                        </el-button>
                        <el-button size="small" text type="danger" class="preview-close-btn" @click="closeFilePreview">
                          关闭预览
                        </el-button>
                      </div>
                    </div>

                    <div class="preview-body">
                      <el-scrollbar class="preview-scroll">
                        <div v-if="filePreview.loading" class="preview-empty">
                          <el-icon class="is-loading" style="margin-right: 6px">
                            <Refresh />
                          </el-icon>
                          正在加载文件内容...
                        </div>
                        <div v-else-if="filePreview.error" class="preview-empty">
                          <el-empty :description="filePreview.error" />
                        </div>
                        <div v-else-if="filePreview.isText" class="preview-editor-wrapper">
                          <VueMonacoEditor v-model:value="filePreview.editContent" :language="filePreviewLanguage"
                            theme="vs-dark" :height="'250px'" class="code-editor" :options="filePreviewMonacoOptions" />
                        </div>
                        <div v-else class="preview-empty">
                          <el-empty description="当前文件类型暂不支持在线预览，请使用下载操作" />
                        </div>
                      </el-scrollbar>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 创建目录对话框（保留弹窗形式，更聚焦） -->
              <el-dialog v-model="data.create_dir_dialog_visible" title="创建目录" width="500px"
                :close-on-click-modal="false" class="ssh-dialog">
                <el-form label-width="100px" label-position="left">
                  <el-form-item label="目录名称" required>
                    <el-input v-model="data.new_dir_name" placeholder="请输入目录名称" clearable
                      @keyup.enter="confirmCreateDir" />
                  </el-form-item>
                  <el-form-item label="当前路径">
                    <el-input :value="data.sftp_current_dir || '/'" disabled style="background-color: #f5f7fa;" />
                  </el-form-item>
                </el-form>
                <template #footer>
                  <span class="dialog-footer">
                    <el-button @click="data.create_dir_dialog_visible = false; data.new_dir_name = ''">取消</el-button>
                    <el-button type="primary" @click="confirmCreateDir">确认</el-button>
                  </span>
                </template>
              </el-dialog>

              <!-- 上传文件/文件夹对话框 -->
              <el-dialog v-model="data.upload_dialog_visible" title="上传文件/文件夹" width="720px" margin-top="5vh"
                :close-on-click-modal="false" class="ssh-dialog"
                :close-on-press-escape="!data.sftp_upload_status.isUploading">
                <div class="upload-dialog-content">
                  <el-form label-width="100px" label-position="left">
                    <el-form-item label="目标路径">
                      <el-input :value="data.sftp_current_dir || '/'" disabled style="background-color: #f5f7fa;" />
                    </el-form-item>
                  </el-form>

                  <!-- 拖拽上传区域 -->
                  <div class="upload-drop-zone" :class="{ 'drag-over': data.upload_drag_over }"
                    @drop.prevent="handleDrop" @dragover.prevent="data.upload_drag_over = true"
                    @dragleave.prevent="data.upload_drag_over = false"
                    @dragenter.prevent="data.upload_drag_over = true">
                    <el-icon class="upload-icon" :size="40">
                      <Upload />
                    </el-icon>
                    <p class="upload-text">拖拽文件/文件夹到此处，或</p>
                    <div class="upload-buttons">
                      <el-button type="primary" @click="triggerFileSelect">选择文件</el-button>
                    </div>
                    <input ref="fileInputRef" type="file" multiple style="display: none" @change="handleFileSelect" />
                  </div>

                  <!-- 文件列表 -->
                  <div v-if="data.upload_file_list.length > 0" class="upload-file-list">
                    <div class="upload-list-header">
                      <span>已选择 {{ data.upload_file_list.length }} 个文件</span>
                      <el-button type="danger" text size="small" @click="clearFileList">清空</el-button>
                    </div>
                    <el-scrollbar height="200px">
                      <div class="file-list-items">
                        <div v-for="(item, index) in data.upload_file_list" :key="index" class="file-list-item">
                          <el-icon class="file-icon">
                            <Files v-if="!item.isDirectory" />
                            <FolderOpened v-else />
                          </el-icon>
                          <span class="file-name">{{ item.name || item.relativePath }}</span>
                          <span class="file-size">{{ formatFileSize(item.size) }}</span>
                        </div>
                      </div>
                    </el-scrollbar>
                  </div>

                  <!-- 上传进度 -->
                  <div v-if="data.sftp_upload_status.isUploading" class="upload-progress-section">
                    <div class="upload-progress-header">
                      <span class="upload-status-text">
                        {{ data.sftp_upload_status.uploadedFiles }} / {{ data.sftp_upload_status.totalFiles }} 个文件
                        <span v-if="data.sftp_upload_status.currentFile" class="current-file-name">
                          - 正在上传: {{ data.sftp_upload_status.currentFile }}
                        </span>
                      </span>
                    </div>
                    <el-progress :percentage="data.sftp_upload_percentage"
                      :status="data.sftp_upload_percentage === 100 ? 'success' : ''" :stroke-width="12" />
                    <div v-if="data.sftp_upload_status.totalSize > 0" class="upload-size-info">
                      {{ formatFileSize(data.sftp_upload_status.uploadedSize) }} / {{
                        formatFileSize(data.sftp_upload_status.totalSize) }}
                    </div>
                  </div>
                </div>
                <template #footer>
                  <span class="dialog-footer"
                    style="display:flex;align-items:center;justify-content:space-between;width:100%;">
                    <div style="display:flex;align-items:center;gap:12px;">
                      <!-- 取消 / 取消上传 按钮 -->
                      <el-button @click="closeUploadDialog">
                        {{ data.sftp_upload_status.isUploading ? '取消上传' : '取消' }}
                      </el-button>
                      <!-- 开始上传按钮：上传中禁用 -->
                      <el-button type="primary" @click="startUpload"
                        :disabled="data.upload_file_list.length === 0 || data.sftp_upload_status.isUploading">
                        开始上传
                      </el-button>
                    </div>
                    <!-- 冲突处理策略选项：放在底部右侧，更显眼 -->
                    <el-checkbox v-model="data.rememberUploadStrategy">
                      对后续同名文件自动使用相同处理（覆盖 / 跳过）
                    </el-checkbox>
                  </span>
                </template>
              </el-dialog>
            </el-drawer>
          </el-button-group>
        </el-col>
        <el-col :span="10" class="status-col">
          <el-space wrap>
            <el-tag type="success" effect="dark">会话 {{ data.host_tabs.length }}</el-tag>
            <el-tag type="info" effect="dark">在线 {{ onlineClients.length }}</el-tag>
            <el-popover placement="bottom-end" trigger="click" :width="520" @show="startOnlineClientStream">
              <template #reference>
                <el-button type="primary" plain size="small" :icon="Monitor">在线会话</el-button>
              </template>
              <el-table :data="onlineClients" size="small" height="220" empty-text="暂无在线会话" :fit="false">
                <el-table-column prop="name" label="名称" width="120" :resizable="true" />
                <el-table-column prop="address" label="主机" width="120" :resizable="true" />
                <el-table-column prop="user" label="用户" width="90" :resizable="true" />
                <el-table-column prop="client_ip" label="来源" width="100" :resizable="true">
                  <template #default="scope">
                    {{ scope.row.client_ip || scope.row.ClientIP || '-' }}
                  </template>
                </el-table-column>
                <el-table-column prop="last_active_time" label="最后活跃" />
                <el-table-column label="操作" width="90" fixed="right">
                  <template #default="scope">
                    <el-button size="small" link type="primary" @click="focusOnlineSession(scope.row)">定位</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-popover>
            <el-tooltip content="重新读取本地端口配置" placement="bottom">
              <el-button circle plain size="small" :icon="Refresh" @click="initClient" />
            </el-tooltip>
          </el-space>
        </el-col>
      </el-row>
    </el-header>
    <div class="tabs-area">
      <el-tabs v-if="data.host_tabs.length" v-model="data.current_host.session_id" type="card" closable
        @tab-remove="removeTab" @tab-click="selectTab" @contextmenu.prevent="handleTabsContextMenu">
        <template #extra>
          <el-button-group size="small">
            <el-button :disabled="!data.current_host.session_id" @click="closeOtherTabs">关闭其他</el-button>
            <el-button :disabled="!data.host_tabs.length" @click="closeAllTabs">关闭全部</el-button>
          </el-button-group>
        </template>
        <el-tab-pane v-for="item in data.host_tabs" :key="item.session_id" :label="item.name" :name="item.session_id">
          <template #label>
            <el-button-group style="width: auto; display: flex; flex-wrap: nowrap; overflow-x: auto">
              <el-popover placement="bottom" :width="260" trigger="hover">
                <template #reference>
                  <el-button :type="item.session_id === data.current_host.session_id ? 'primary' : 'info'" size="small">
                    <span v-if="item.is_close" style="color: red">{{ item.name }}</span>
                    <span v-else="item.is_close" style="color: white">{{ item.name }}</span>
                  </el-button>
                </template>
                <div>
                  <div style="padding-top: 5px">
                    <el-button-group>
                      <el-button type="primary" @click="connectHost(item, true)">重连</el-button>
                      <el-button type="primary" @click="item.term.clear()">清空缓冲区</el-button>
                    </el-button-group>
                  </div>
                  <div style="padding-top: 5px">
                    <div>
                      <el-input disabled v-model="item.session_id">
                        <template #prepend>会话</template>
                      </el-input>
                    </div>
                    <div>
                      <el-input disabled v-model="item.address">
                        <template #prepend>主机</template>
                      </el-input>
                    </div>
                    <div>
                      <el-input disabled v-model="item.user">
                        <template #prepend>用户</template>
                      </el-input>
                    </div>
                    <div>
                      <el-input disabled v-model="item.port">
                        <template #prepend>端口</template>
                      </el-input>
                    </div>
                  </div>
                </div>
              </el-popover>

              <el-tooltip class="item" effect="dark" content="文件传输" placement="top">
                <el-button :type="item.session_id === data.current_host.session_id ? 'primary' : 'info'" size="small"
                  @click="listDir(null, item, true)" :icon="Sort"></el-button>
              </el-tooltip>
            </el-button-group>
          </template>
          <template #default>
            <div class="terminal-wrap">
              <div :id="item.session_id" class="terminal"></div>
            </div>
          </template>
        </el-tab-pane>
      </el-tabs>
      <div v-else class="empty-connection">
        <el-empty description="暂无连接，请先创建或打开主机会话" />
      </div>
    </div>
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>
    <!-- 文件列表右键菜单 -->
    <Teleport to="body">
      <ul v-if="fileContextMenuVisible" class="file-context-menu" :style="{
        top: fileContextMenuPosition.y + 'px',
        left: fileContextMenuPosition.x + 'px',
        position: 'fixed'
      }">
        <li @click="handleFileMenuOpen">
          {{ fileContextMenuRow && fileContextMenuRow.type === 'd' ? '打开目录' : '预览 / 打开' }}
        </li>
        <li @click="handleFileMenuDownload">
          {{ fileContextMenuRow && fileContextMenuRow.type === 'd' ? '下载文件夹' : '下载' }}
        </li>
        <li v-if="fileContextMenuRow && fileContextMenuRow.type === 'd'" @click="handleFileMenuUploadHere">
          上传到此目录
        </li>
        <li class="danger" @click="handleFileMenuDelete">
          删除
        </li>
      </ul>
    </Teleport>
  </el-container>
</template>

<script>
import { markRaw, nextTick } from 'vue';
import axios from 'axios';
import { ElMessage, ElNotification, ElMessageBox } from 'element-plus';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { GetPortConfig } from '../../../wailsjs/go/controller/Server';
import { getToken } from '@/utils/token';
import { BrowserOpenURL, OnFileDrop } from '../../../wailsjs/runtime';
import { UploadLocalPaths } from '../../../wailsjs/go/service/SshDesktopBridge';
import { Terminal } from '@xterm/xterm';
import { AttachAddon } from '@xterm/addon-attach';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';
import {
  FolderOpened,
  Files,
  Bottom,
  Top,
  Upload,
  Menu,
  CirclePlus,
  Sort,
  InfoFilled,
  Edit,
  Star,
  Monitor,
  Refresh,
  Back
} from '@element-plus/icons-vue';

const Mode = {
  create: 0,
  update: 1
};

export default {
  name: 'SSH',
  components: {
    VueMonacoEditor
  },
  data() {
    return {
      fileInputRef: null,
      Menu: markRaw(Menu),
      CirclePlus: markRaw(CirclePlus),
      Edit: markRaw(Edit),
      Star: markRaw(Star),
      FolderOpened: markRaw(FolderOpened),
      Files: markRaw(Files),
      Bottom: markRaw(Bottom),
      Top: markRaw(Top),
      Upload: markRaw(Upload),
      Sort: markRaw(Sort),
      InfoFilled: markRaw(InfoFilled),
      Monitor: markRaw(Monitor),
      Refresh: markRaw(Refresh),
      Back: markRaw(Back),
      portConfig: null,
      apiClient: null,
      apiBase: '',
      statusSetInterval: null,
      resizeHandler: null,
      onlineEventSource: null,
      onlineClients: [],
      dialogHostListVisible: false,
      dialogCmdExecVisible: false,
      dialogCmdFavVisible: false,
      cmd: { name: '', data: '', node: 'current' },
      cmdNotes: [],
      searchHost: '',
      searchCmdNote: '',
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      fileContextMenuVisible: false,
      fileContextMenuPosition: { x: 0, y: 0 },
      fileContextMenuRow: null,
      filePreview: {
        file: null,
        loading: false,
        isText: false,
        error: '',
        content: '',
        editContent: '',
        dirty: false
      },
      filePreviewMonacoOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        lineNumbers: 'on',
        readOnly: false,
        tabSize: 2,
        insertSpaces: true
      },
      // 主机列表分页
      hostPagination: {
        currentPage: 1,
        pageSize: 10
      },
      // Wails 原生拖拽得到的本地绝对路径（用于 SSH 上传）
      wailsDropSshPaths: [],
      // 文件上传策略：'ask' 询问, 'overwrite' 覆盖, 'skip' 跳过
      uploadStrategy: 'ask',
      data: {
        mode: Mode.create,
        id: 0,
        name: '',
        address: '',
        user: '',
        auth_type: 'pwd',
        net_type: 'tcp4',
        cert_data: '',
        cert_pwd: '',
        pwd: '',
        port: 22,
        h: 20,
        w: 80,
        session_id: '',
        background: '#000000',
        foreground: '#FFFFFF',
        cursor_color: '#FFFFFF',
        font_family: 'Courier',
        font_size: 16,
        cursor_style: 'block',
        shell: 'bash',
        pty_type: 'xterm-256color',
        init_cmd: '',
        init_banner: '',
        upload_path: '',
        download_path: '',
        host_list: [],
        host_tabs: [],
        current_host: { session_id: '', name: '', address: '' },
        host_config_collapse: ['1'],
        host_dialog_visible: false,
        file_dialog_visible: false,
        dir_info: { paths: [], files: [] },
        sftp_current_dir: '',
        sftp_last_dir: {},
        sftp_upload_percentage: 0,
        sftp_upload_status: {
          totalFiles: 0,
          uploadedFiles: 0,
          currentFile: '',
          currentFileProgress: 0,
          totalSize: 0,
          uploadedSize: 0,
          isUploading: false,
          // 是否已请求取消当前上传任务
          cancelRequested: false
        },
        // 是否对后续同名文件使用相同处理（覆盖/跳过）
        rememberUploadStrategy: false,
        new_pwd_one: '',
        new_pwd_two: '',
        create_dir_dialog_visible: false,
        new_dir_name: '',
        upload_dialog_visible: false,
        upload_drag_over: false,
        upload_file_list: [],
        // 当前批量 SFTP 上传的进度 ID，用于服务端进度/取消
        currentUploadId: ''
      }
    };
  },
  computed: {
    filterHostTable() {
      return this.data.host_list.filter(
        (i) =>
          !this.searchHost ||
          i.name.toLowerCase().includes(this.searchHost.toLowerCase()) ||
          i.address.toLowerCase().includes(this.searchHost.toLowerCase())
      );
    },
    pagedHostTable() {
      const start = (this.hostPagination.currentPage - 1) * this.hostPagination.pageSize;
      const end = start + this.hostPagination.pageSize;
      return this.filterHostTable.slice(start, end);
    },
    filterCmdNoteTable() {
      return this.cmdNotes.filter(
        (i) =>
          !this.searchCmdNote ||
          i.cmd_name.toLowerCase().includes(this.searchCmdNote.toLowerCase())
      );
    },
    title() {
      return `<span style="color:red;">当前名称:${this.data.current_host.name} &nbsp;&nbsp;&nbsp;当前主机:${this.data.current_host.address}</span>`;
    },
    tabsContextMenuStyle() {
      const { x, y } = this.tabsContextMenuPosition
      return {
        top: `${y}px`,
        left: `${x}px`,
        position: 'fixed'
      }
    },
    filePreviewLanguage() {
      const file = this.filePreview.file;
      if (!file || !file.name) return 'plaintext';
      const name = String(file.name).toLowerCase();
      if (name.endsWith('.sh')) return 'shell';
      if (name.endsWith('.js')) return 'javascript';
      if (name.endsWith('.ts')) return 'typescript';
      if (name.endsWith('.json')) return 'json';
      if (name.endsWith('.yml') || name.endsWith('.yaml')) return 'yaml';
      if (name.endsWith('.xml')) return 'xml';
      if (name.endsWith('.py')) return 'python';
      if (name.endsWith('.go')) return 'go';
      if (name.endsWith('.java')) return 'java';
      if (name.endsWith('.php')) return 'php';
      if (name.endsWith('.html') || name.endsWith('.htm')) return 'html';
      if (name.endsWith('.css') || name.endsWith('.scss') || name.endsWith('.less')) return 'css';
      if (name.endsWith('.md')) return 'markdown';
      return 'plaintext';
    },
    fileTableHeight() {
      // 不再限制表格固定高度，统一交给外层 flex 布局自适应
      return null;
    }
  },
  methods: {
    handlePathInputEnter() {
      if (!this.ensureClientReady()) return;
      const raw = (this.data.sftp_current_dir || '').trim();
      const target = raw === '' ? '/' : raw;
      this.listDir(target, this.data.current_host);
    },
    // 注册 SSH 全局拖拽监听（支持桌面拖拽绝对路径）
    setupSshFileDropListener() {
      OnFileDrop((x, y, paths) => {
        if (paths && paths.length > 0) {
          this.wailsDropSshPaths = paths;
          this.startNativeSshUpload(paths);
        }
      });
    },
    // 检查 SFTP 文件是否存在
    async checkSftpFileExists(remotePath) {
      if (!this.ensureClientReady()) return false;
      try {
        // 获取文件所在目录
        const lastSlashIndex = remotePath.lastIndexOf('/');
        const dirPath = lastSlashIndex > 0 ? remotePath.substring(0, lastSlashIndex) : '/';
        const fileName = lastSlashIndex >= 0 ? remotePath.substring(lastSlashIndex + 1) : remotePath;

        const formData = new FormData();
        formData.append('session_id', this.data.current_host.session_id);
        formData.append('path', dirPath);

        const ret = await this.apiClient.post('/api/sftp/list', formData);
        if (ret.data.code === 0) {
          const files = ret.data.data?.files || [];
          return files.some(f => f.name === fileName && f.type === 'f');
        }
        return false;
      } catch (err) {
        console.error('检查文件是否存在失败:', err);
        return false;
      }
    },
    // 处理文件存在的情况
    async handleSftpFileExists(fileName, remotePath) {
      if (this.uploadStrategy === 'overwrite') {
        return 'overwrite';
      }
      if (this.uploadStrategy === 'skip') {
        return 'skip';
      }

      // 询问用户
      try {
        await ElMessageBox.confirm(
          `文件 "${fileName}" 已存在，请选择操作：`,
          '文件已存在',
          {
            distinguishCancelAndClose: true,
            confirmButtonText: '覆盖',
            cancelButtonText: '跳过',
            type: 'warning',
            autofocus: false
          }
        );
        // 用户点击“覆盖”
        if (this.data.rememberUploadStrategy) {
          // 记住本次选择，后续同名文件自动覆盖
          this.uploadStrategy = 'overwrite';
        }
        return 'overwrite';
      } catch (action) {
        if (action === 'cancel') {
          // 用户点击“跳过”
          if (this.data.rememberUploadStrategy) {
            // 记住本次选择，后续同名文件自动跳过
            this.uploadStrategy = 'skip';
          }
          return 'skip';
        }
        return 'cancel'; // 用户关闭对话框，取消上传
      }
    },

    // 使用 Wails 原生路径上传本地文件/文件夹到当前 SFTP 目录
    async startNativeSshUpload(paths) {
      if (!this.ensureClientReady()) return;
      if (!paths || paths.length === 0) return;

      const host = this.data.current_host;
      if (!host || !host.session_id) {
        ElMessage.error('当前没有有效的 SSH 会话');
        return;
      }

      const targetDir = this.data.sftp_current_dir || '/';

      try {
        this.data.sftp_upload_status.isUploading = true;
        this.data.sftp_upload_status.currentFile = '';
        this.data.sftp_upload_percentage = 0;

        const msg = await UploadLocalPaths(host.session_id, targetDir, paths);
        ElMessage.success(msg || '本地文件上传完成');

        // 刷新目录
        this.listDir(targetDir, host);
      } catch (err) {
        const errMsg = err?.message || String(err || '');
        ElMessage.error(errMsg || '本地路径上传失败');
      } finally {
        this.data.sftp_upload_status.isUploading = false;
        this.data.sftp_upload_percentage = 0;
        this.wailsDropSshPaths = [];
      }
    },
    async initClient() {
      await this.loadPortConfig();
      this.createHttpClient();
    },
    async loadPortConfig() {
      try {
        const config = await GetPortConfig();
        if (config && (config.sshPort || config.ftpPort)) {
          this.portConfig = config;
        } else {
          console.warn('获取的端口配置无效，使用默认配置');
          this.portConfig = {
            sshPort: 52868,
            ftpPort: 52869
          };
        }
      } catch (error) {
        console.error('获取端口配置失败:', error);
        this.portConfig = {
          sshPort: 52868,
          ftpPort: 52869
        };
      }
    },
    createHttpClient() {
      const port = this.portConfig?.sshPort || 52868;
      const host = '127.0.0.1';
      this.apiBase = `http://${host}:${port}`;
      const client = axios.create({
        baseURL: this.apiBase,
        // 上传大文件可能耗时较长，这里适当放宽超时时间
        timeout: 600000
      });
      client.interceptors.request.use(
        (req) => {
          req.headers.Time = String(Date.now());
          const token = getToken();
          if (token) {
            req.headers.Authorization = token;
          }
          return req;
        },
        (err) => Promise.reject(err)
      );
      client.interceptors.response.use(
        (res) => res,
        (err) => {
          if (err.response && err.response.status === 401) {
            this.$router.replace({ name: 'login' });
          }
          return Promise.reject(err);
        }
      );
      this.apiClient = client;
    },
    normalizeOnlineData(list = []) {
      return list
        .map((item) => ({
          session_id: item.session_id || item.SessionId || item.SessionID || '',
          name: item.name || item.Name || '',
          address: item.address || item.Address || '',
          user: item.user || item.User || '',
          port: item.port || item.Port || '',
          client_ip: item.client_ip || item.ClientIP || '',
          start_time: item.start_time || item.StartTime || '',
          last_active_time: item.last_active_time || item.LastActiveTime || ''
        }))
        .filter((i) => i.session_id);
    },
    startOnlineClientStream() {
      if (!this.ensureClientReady()) return;
      this.stopOnlineClientStream();
      try {
        // EventSource 无法自定义请求头，本地服务防护 Token 只能通过 query 携带
        const localToken = window.__LOCAL_SERVICE_TOKEN__ || '';
        const url = `${this.apiBase}/api/conn_manage/online_client?token=${encodeURIComponent(localToken)}`;
        const es = new EventSource(url);
        es.onmessage = (evt) => {
          try {
            const payload = JSON.parse(evt.data || '{}');
            if (payload.code === 0) {
              this.onlineClients = this.normalizeOnlineData(payload.data || []);
            }
          } catch (error) {
            console.log('在线会话解析错误', error);
          }
        };
        es.onerror = () => {
          this.stopOnlineClientStream();
        };
        this.onlineEventSource = es;
      } catch (error) {
        console.log('启动在线会话流失败', error);
      }
    },
    stopOnlineClientStream() {
      if (this.onlineEventSource) {
        try {
          this.onlineEventSource.close();
        } catch (error) {
          console.log(error);
        }
        this.onlineEventSource = null;
      }
    },
    focusOnlineSession(row) {
      const sessionId = row.session_id || row.SessionId || row.SessionID;
      if (!sessionId) {
        return;
      }
      const exist = this.data.host_tabs.find((h) => h.session_id === sessionId);
      if (exist) {
        this.setCurrentAcitveHost(sessionId);
        ElMessage.success('已定位到会话');
      } else {
        ElMessage.warning('该会话未在当前页面打开');
      }
    },
    ensureClientReady() {
      if (this.apiClient) return true;
      // 如果端口配置已存在，尝试自动创建客户端
      if (this.portConfig) {
        try {
          this.createHttpClient();
          if (this.apiClient) {
            return true;
          }
        } catch (error) {
          console.error('创建SSH客户端失败:', error);
        }
      }
      ElMessage.error('SSH服务尚未初始化，请点击刷新按钮或刷新页面');
      return false;
    },
    execCmd() {
      if (this.cmd.node === 'current') {
        this.execCmdCurrentSession({ id: 0, cmd_name: '', cmd_data: this.cmd.data });
      }
      if (this.cmd.node === 'all') {
        this.execCmdAllSession({ id: 0, cmd_name: '', cmd_data: this.cmd.data });
      }
    },
    addCmdNote() {
      if (!this.ensureClientReady()) return;
      if (this.cmd.data.trim().length === 0) {
        ElMessage.warning('收藏的命令不能为空');
        return;
      }
      if (this.cmd.name.trim().length === 0) {
        ElMessage.warning('如果收藏命令,必须输入收藏名称');
        return;
      }
      // 避免 Gin 的自动去除尾斜杠导致 307 重定向(会丢失 CORS 头)
      this.apiClient.post('/api/cmd_note', { cmd_name: this.cmd.name, cmd_data: this.cmd.data }).then((ret) => {
        if (ret.data.code === 0) {
          ElMessage.success('收藏成功');
          this.getAllCmdNote();
        } else {
          ElMessage.error('收藏命令出错了');
        }
      });
    },
    async delCmdNote(id) {
      if (!this.ensureClientReady()) return;
      try {
        await confirmDangerousAction({
          title: '删除收藏命令',
          message: '将删除该条命令收藏，不可恢复。',
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      this.apiClient.delete(`/api/cmd_note/${id}`).then((ret) => {
        if (ret.data.code === 0) {
          this.cmdNotes = ret.data.data;
          ElMessage.success('删除成功');
        } else {
          ElMessage.error('删除命令收藏出错了');
        }
      });
    },
    getAllCmdNote() {
      if (!this.ensureClientReady()) return;
      this.apiClient.get('/api/cmd_note').then((ret) => {
        if (ret.data.code === 0) {
          this.cmdNotes = ret.data.data;
        } else {
          ElMessage.error('获取主机列表错误');
        }
      });
    },
    execCmdCurrentSession(row) {
      try {
        this.data.current_host.ws.send(row.cmd_data + '\n');
      } catch (e) {
        ElMessage.error('当前会话执行命令失败');
      }
    },
    execCmdAllSession(row) {
      try {
        if (this.data.host_tabs.length === 0) {
          ElMessage.error('没有连接会话');
          return;
        }
        this.data.host_tabs.forEach((h) => {
          h.ws.send(row.cmd_data + '\n');
        });
      } catch (e) {
        ElMessage.error('执行命令失败');
      }
    },
    openCmdDialog() {
      this.dialogCmdExecVisible = true;
    },
    addCertFile() {
      const input = document.createElement('input');
      input.type = 'file';
      input.addEventListener('change', (event) => {
        const files = event.target.files;
        if (files && files.length > 0) {
          const certFile = files[0];
          const isLt1M = certFile.size / 1024 / 1024 < 1;
          if (!isLt1M) {
            ElMessage.error('上传文件大小不能超过 1MB!');
            return;
          }
          const reader = new FileReader();
          reader.onload = (e) => {
            this.data.cert_data = e.target.result;
          };
          reader.readAsText(certFile);
        }
      });
      input.click();
    },
    verifyFrom() {
      const verifyFromData = {
        host: {},
        is_success: false
      };
      if (this.data.name.length === 0) {
        ElMessage.error('名称不能为空');
        return verifyFromData;
      }
      if (this.data.name.length > 30) {
        ElMessage.error('名称不能大于30个字符');
        return verifyFromData;
      }
      if (this.data.address.length === 0) {
        ElMessage.error('主机不能为空');
        return verifyFromData;
      }
      if (this.data.address.length > 60) {
        ElMessage.error('主机不能大于60个字符');
        return verifyFromData;
      }
      if (this.data.user.length === 0) {
        ElMessage.error('用户名不能为空');
        return verifyFromData;
      }
      if (this.data.user.length > 60) {
        ElMessage.error('用户名不能大于60个字符');
        return verifyFromData;
      }
      if (this.data.auth_type === 'pwd' && this.data.pwd.length === 0) {
        ElMessage.error('密码不能为空');
        return verifyFromData;
      }
      if (this.data.pwd.length > 60) {
        ElMessage.error('密码不能大于60个字符');
        return verifyFromData;
      }
      if (!this.data.port || this.data.port < 1 || this.data.port > 65535) {
        ElMessage.error('端口范围错误,必须是1-65535');
        return verifyFromData;
      }
      if (this.data.auth_type === 'cert' && this.data.cert_data === '') {
        ElMessage.error('使用证书登陆,证书内容不能为空');
        return verifyFromData;
      }
      const h = {
        id: this.data.id,
        name: this.data.name,
        address: this.data.address,
        user: this.data.user,
        auth_type: this.data.auth_type,
        net_type: this.data.net_type,
        cert_data: this.data.cert_data,
        cert_pwd: this.data.cert_pwd,
        pwd: this.data.pwd,
        port: this.data.port,
        session_id: this.data.session_id,
        background: this.data.background,
        foreground: this.data.foreground,
        cursor_color: this.data.cursor_color,
        font_family: this.data.font_family,
        font_size: this.data.font_size,
        cursor_style: this.data.cursor_style,
        shell: this.data.shell,
        pty_type: this.data.pty_type,
        init_cmd: this.data.init_cmd,
        init_banner: this.data.init_banner
      };
      return {
        host: h,
        is_success: true
      };
    },
    cleanFrom() {
      this.data.id = 0;
      this.data.name = '';
      this.data.address = '';
      this.data.user = '';
      this.data.pwd = '';
      this.data.auth_type = 'pwd';
      this.data.net_type = 'tcp4';
      this.data.cert_data = '';
      this.data.cert_pwd = '';
      this.data.port = 22;
      this.data.session_id = '';
      this.data.background = '#000000';
      this.data.foreground = '#FFFFFF';
      this.data.cursor_color = '#FFFFFF';
      this.data.font_family = 'Courier';
      this.data.font_size = 16;
      this.data.cursor_style = 'block';
      this.data.shell = 'bash';
      this.data.pty_type = 'xterm-256color';
      this.data.init_cmd = '';
      this.data.init_banner = '';
      this.data.host_config_collapse = ['1'];
    },
    connect() {
      const result = this.verifyFrom();
      if (!result.is_success) {
        return;
      }
      this.connectHost(result.host);
    },
    goParentDir() {
      if (!this.data.sftp_current_dir) {
        this.listDir('/', this.data.current_host);
        return;
      }
      const raw = String(this.data.sftp_current_dir);
      if (raw === '/' || raw.trim() === '') {
        this.listDir('/', this.data.current_host);
        return;
      }
      // 去掉末尾的 /，再向上一级
      const trimmed = raw.replace(/\/+$/, '');
      const parts = trimmed.split('/').filter(Boolean);
      parts.pop();
      const parent = `/${parts.join('/')}`;
      this.listDir(parent === '' ? '/' : parent, this.data.current_host);
    },
    listDir(dir, h, useWorkDir = false) {
      if (!this.ensureClientReady()) return;
      if (h) {
        this.setCurrentAcitveHost(h.session_id);
      }
      const host = { ...this.data.current_host };
      if (!Object.prototype.hasOwnProperty.call(host, 'session_id') || !host.session_id) {
        return;
      }
      const remembered = this.data.sftp_last_dir[host.session_id];
      let targetDir = '';
      if (useWorkDir) {
        // 这里故意传空字符串，让后端走 Getwd() 逻辑，使用当前会话工作目录
        targetDir = '';
      } else {
        targetDir =
          dir !== undefined && dir !== null ? dir : remembered ?? this.data.sftp_current_dir ?? '';
        // 保证首次打开或路径为空时有一个稳定的默认值
        if (!targetDir || String(targetDir).trim().length === 0) {
          targetDir = '/';
        }
      }
      this.data.file_dialog_visible = true;
      const formData = new FormData();
      formData.append('session_id', host.session_id);
      formData.append('path', targetDir);
      this.apiClient.post('/api/sftp/list', formData).then((ret) => {
        if (ret.data.code === 0) {
          const payload = ret.data.data || {};
          const currentDir = payload.current_dir || targetDir;
          this.data.dir_info = payload;
          this.data.sftp_current_dir = currentDir;
          this.data.sftp_last_dir = {
            ...this.data.sftp_last_dir,
            [host.session_id]: currentDir
          };
          // 列表渲染完成后，手动让表格根据当前可用宽度重新布局一次
          this.$nextTick(() => {
            const fileTable = this.$refs.fileTable;
            if (fileTable && typeof fileTable.doLayout === 'function') {
              fileTable.doLayout();
            }
          });
        } else {
          ElMessage.error('获取文件列表错误');
        }
      });
    },
    async openFilePreview(file) {
      if (!this.ensureClientReady()) return;
      if (!file || file.type !== 'f') {
        return;
      }

      // 文件过大直接建议下载
      const size = typeof file.size === 'string' ? parseFloat(file.size) : file.size;
      if (size && size > 5 * 1024 * 1024) {
        this.filePreview = {
          ...this.filePreview,
          file,
          loading: false,
          isText: false,
          error: '文件大于 5MB，暂不支持在线预览，请使用下载功能',
          content: '',
          editContent: '',
          dirty: false
        };
        return;
      }

      this.filePreview.file = file;
      this.filePreview.loading = true;
      this.filePreview.error = '';
      this.filePreview.isText = false;
      this.filePreview.content = '';
      this.filePreview.editContent = '';
      this.filePreview.dirty = false;

      try {
        const params = {
          session_id: this.data.current_host.session_id,
          path: file.path
        };
        const res = await this.apiClient.get('/api/sftp/download', {
          params,
          responseType: 'arraybuffer'
        });
        const buffer = res.data instanceof ArrayBuffer ? res.data : res.data?.data;
        if (!buffer) {
          this.filePreview.error = '无法读取文件内容';
          return;
        }
        const bytes = new Uint8Array(buffer);
        const decoder = new TextDecoder('utf-8');
        const text = decoder.decode(bytes);

        // 简单判断是否为文本：控制字符比例过高则认为是二进制
        let controlCount = 0;
        const sampleLen = Math.min(bytes.length, 2048);
        for (let i = 0; i < sampleLen; i += 1) {
          const ch = bytes[i];
          if (ch === 9 || ch === 10 || ch === 13) continue;
          if (ch < 32 || ch === 127) controlCount += 1;
        }
        const isBinary = sampleLen > 0 && controlCount / sampleLen > 0.1;

        if (isBinary) {
          this.filePreview.isText = false;
          this.filePreview.error = '当前文件看起来是二进制文件，暂不支持在线预览';
          this.filePreview.content = '';
          this.filePreview.editContent = '';
          this.filePreview.dirty = false;
        } else {
          this.filePreview.isText = true;
          this.filePreview.content = text;
          this.filePreview.editContent = text;
          this.filePreview.dirty = false;
        }
      } catch (err) {
        console.error('预览文件失败', err);
        this.filePreview.error = '预览文件失败，请稍后重试或使用下载功能';
      } finally {
        this.filePreview.loading = false;
      }
    },
    closeFilePreview() {
      this.filePreview = {
        file: null,
        loading: false,
        isText: false,
        error: '',
        content: '',
        editContent: '',
        dirty: false
      };
    },
    async saveFileEdit() {
      if (!this.ensureClientReady()) return;
      if (!this.filePreview.file || !this.filePreview.isText) return;
      if (!this.filePreview.dirty) return;

      try {
        const fullPath = this.filePreview.file.path || '';
        if (!fullPath) {
          ElMessage.error('无法识别文件路径');
          return;
        }
        const lastSlash = fullPath.lastIndexOf('/');
        const dir = lastSlash > 0 ? fullPath.substring(0, lastSlash) : '/';
        const filename = lastSlash >= 0 ? fullPath.substring(lastSlash + 1) : fullPath;

        const blob = new Blob([this.filePreview.editContent], {
          type: 'text/plain;charset=utf-8'
        });
        const uploadFile = new File([blob], filename, {
          type: 'text/plain;charset=utf-8'
        });

        const formData = new FormData();
        formData.append('session_id', this.data.current_host.session_id);
        formData.append('path', dir || '/');
        formData.append('files', uploadFile);

        const ret = await this.apiClient.put('/api/sftp/upload', formData);
        if (ret.data.code === 0) {
          ElMessage.success('保存成功');
          this.filePreview.dirty = false;
          // 重新刷新当前目录，反映最新修改时间
          this.listDir(dir || '/', this.data.current_host);
        } else {
          ElMessage.error(ret.data.msg || '保存失败');
        }
      } catch (err) {
        console.error('保存文件失败', err);
        ElMessage.error('保存文件失败，请稍后重试');
      }
    },
    handleFileNameDblClick(row) {
      if (!row) return;
      if (row.type === 'd') {
        this.listDir(row.path, this.data.current_host);
      } else if (row.type === 'f') {
        this.openFilePreview(row);
      }
    },
    handleFileRowContextMenu(row, column, event) {
      event.preventDefault();
      event.stopPropagation();
      this.fileContextMenuRow = row;
      const x = event.clientX || 0;
      const y = event.clientY || 0;
      this.fileContextMenuPosition = { x, y };
      this.fileContextMenuVisible = true;

      const hideMenu = (e) => {
        // 无论点击哪里都关闭右键菜单，保持交互简单直观
        this.hideFileContextMenu();
        document.removeEventListener('click', hideMenu);
        document.removeEventListener('contextmenu', hideMenu);
      };
      setTimeout(() => {
        document.addEventListener('click', hideMenu);
        document.addEventListener('contextmenu', hideMenu);
      }, 50);
    },
    hideFileContextMenu() {
      this.fileContextMenuVisible = false;
      this.fileContextMenuRow = null;
    },
    handleFileMenuOpen() {
      const row = this.fileContextMenuRow;
      if (!row) return;
      if (row.type === 'd') {
        this.listDir(row.path, this.data.current_host);
      } else {
        this.openFilePreview(row);
      }
      this.hideFileContextMenu();
    },
    handleFileMenuDownload() {
      const row = this.fileContextMenuRow;
      if (!row) return;
      // 支持文件和文件夹下载
      this.downloadFile(row);
      this.hideFileContextMenu();
    },
    handleFileMenuUploadHere() {
      const row = this.fileContextMenuRow;
      if (!row || row.type !== 'd') return;
      // 切换到目标目录并打开上传对话框
      this.listDir(row.path, this.data.current_host);
      this.openUploadDialog();
      this.hideFileContextMenu();
    },
    async handleFileMenuDelete() {
      const row = this.fileContextMenuRow;
      if (!row) return;
      try {
        await confirmDangerousAction({
          title: '删除确认',
          message: `确定要删除「${row.name}」吗？此操作不可恢复。`,
          expectedInput: DANGER_DELETE_TOKEN,
          confirmButtonText: '删除'
        });
      } catch {
        this.hideFileContextMenu();
        return;
      }
      this.deleteFile(row);
      this.hideFileContextMenu();
    },
    // 处理文件列表，支持文件夹结构
    async processFileList(fileList, isDirectory = false) {
      const files = [];

      const getRelativePath = (file) => {
        // 优先使用我们自己挂载的 relativePath 字段，其次 webkitRelativePath
        if (file.relativePath && typeof file.relativePath === 'string') {
          return file.relativePath;
        }
        if (file.webkitRelativePath && typeof file.webkitRelativePath === 'string') {
          return file.webkitRelativePath;
        }
        return file.name;
      };

      if (isDirectory) {
        // 处理文件夹上传：使用相对路径属性
        for (let i = 0; i < fileList.length; i += 1) {
          const file = fileList[i];
          files.push({
            file,
            relativePath: getRelativePath(file)
          });
        }
      } else {
        // 处理普通文件上传
        for (let i = 0; i < fileList.length; i += 1) {
          const file = fileList[i];
          files.push({
            file,
            relativePath: getRelativePath(file)
          });
        }
      }
      return files;
    },
    // 创建目录（如果不存在），支持递归创建父目录
    async ensureDirectoryExists(dirPath) {
      try {
        const body = {
          session_id: this.data.current_host.session_id,
          path: dirPath
        };
        const ret = await this.apiClient.post('/api/sftp/create_dir', body);
        // 即使返回错误，也可能是目录已存在，继续执行
        if (ret.data.code !== 0 && !ret.data.msg.includes('已存在')) {
          console.warn('创建目录可能失败:', dirPath, ret.data.msg);
        }
      } catch (error) {
        // 目录可能已存在，忽略错误
        console.log('目录可能已存在或创建失败:', dirPath, error.message);
      }
    },
    // 上传单个文件（带相对路径）
    async uploadSingleFile(fileItem, targetPath, fileIndex, totalFiles) {
      return new Promise(async (resolve, reject) => {
        const formData = new FormData();
        formData.append('session_id', this.data.current_host.session_id);

        // 构建完整的目标路径，保持文件夹结构
        const relativePath = fileItem.relativePath;
        const pathParts = relativePath.split('/').filter(p => p); // 过滤空字符串
        const fileName = pathParts.pop() || fileItem.file.name;
        const subDir = pathParts.length > 0 ? pathParts.join('/') : '';

        // 如果文件在子目录中，需要在目标路径下创建相同的目录结构
        // 确保路径格式正确（去除多余的斜杠）
        let dirPath = targetPath;
        if (subDir) {
          // 构建包含子目录的完整目录路径
          dirPath = targetPath.endsWith('/')
            ? `${targetPath}${subDir}`.replace(/\/+/g, '/')
            : `${targetPath}/${subDir}`.replace(/\/+/g, '/');
          // 确保目录路径以/结尾（如果是根目录则保持）
          if (!dirPath.endsWith('/') && dirPath !== '/') {
            dirPath += '/';
          }
        } else {
          // 确保目标路径格式正确
          if (!dirPath.endsWith('/') && dirPath !== '/') {
            dirPath += '/';
          }
        }

        // 目录已在上传前批量创建，这里不需要再创建

        formData.append('path', dirPath);
        // 使用文件名（不包含路径）
        formData.append('files', fileItem.file, fileName);

        // 为本次批量上传传递统一的 uploadId（如果上层生成了）
        if (this.currentUploadId) {
          formData.append('uploadId', this.currentUploadId);
        }

        // 更新当前上传文件信息
        this.data.sftp_upload_status.currentFile = relativePath;
        this.data.sftp_upload_status.currentFileProgress = 0;

        this.apiClient
          .put('/api/sftp/upload', formData)
          .then((ret) => {
            if (ret.data.code === 0) {
              this.data.sftp_upload_status.uploadedFiles += 1;
              this.data.sftp_upload_status.currentFileLoaded = 0;
              resolve(ret);
            } else {
              reject(new Error(ret.data.msg || '上传失败'));
            }
          })
          .catch((err) => {
            this.data.sftp_upload_status.currentFileLoaded = 0;
            reject(err);
          });
      });
    },
    // 打开上传对话框
    openUploadDialog() {
      if (!this.ensureClientReady()) return;
      this.data.upload_dialog_visible = true;
      this.data.upload_file_list = [];
      this.data.upload_drag_over = false;
      // 重置上传状态
      this.data.sftp_upload_percentage = 0;
      this.data.sftp_upload_status = {
        totalFiles: 0,
        uploadedFiles: 0,
        currentFile: '',
        currentFileProgress: 0,
        totalSize: 0,
        uploadedSize: 0,
        isUploading: false,
        currentFileLoaded: 0,
        cancelRequested: false
      };
      this.currentUploadId = '';
    },
    // 关闭上传对话框 / 取消上传
    closeUploadDialog() {
      // 如果正在上传，则视为“取消上传”请求，而不是直接关闭对话框
      if (this.data.sftp_upload_status.isUploading) {
        this.requestCancelUpload();
        return;
      }
      // 未在上传时正常关闭对话框
      this.data.upload_dialog_visible = false;
      this.data.upload_file_list = [];
      this.data.upload_drag_over = false;
    },
    // 公共的“取消上传”动作（顶部按钮 + 对话框按钮共用）
    requestCancelUpload() {
      if (!this.data.sftp_upload_status.isUploading) return;
      if (!this.data.sftp_upload_status.cancelRequested) {
        this.data.sftp_upload_status.cancelRequested = true;
        // 通知后端根据当前 uploadId 取消 SFTP 上传
        if (this.currentUploadId && this.apiClient) {
          this.apiClient
            .get('/api/sftp/upload/cancel', { params: { id: this.currentUploadId } })
            .catch(() => {});
        }
        ElMessage.info('已请求取消上传，将立即停止当前文件并不再上传剩余文件');
      }
    },
    // 触发文件选择
    triggerFileSelect() {
      this.$refs.fileInputRef?.click();
    },
    // 处理文件选择
    handleFileSelect(event) {
      const fileList = event.target.files;
      if (!fileList || !fileList.length) {
        return;
      }
      this.addFilesToList(Array.from(fileList));
      // 清空input，以便可以重复选择相同文件
      event.target.value = '';
    },
    // 处理上传对话框中的拖拽放置（支持文件夹）
    handleDrop(event) {
      this.data.upload_drag_over = false;
      const items = event.dataTransfer.items;
      const files = event.dataTransfer.files;
      if ((!items || items.length === 0) && (!files || files.length === 0)) {
        return;
      }

      // 优先使用 DataTransferItem，保留文件夹结构（webkitRelativePath / 自定义 relativePath）
      if (items && items.length > 0 && items[0].webkitGetAsEntry) {
        const collectedFiles = [];

        const processEntry = (entry, parentPath = '') => {
          return new Promise((resolve) => {
            if (entry.isFile) {
              entry.file((file) => {
                // 手动补充相对路径，保留目录结构
                const relativePath = parentPath ? `${parentPath}/${file.name}` : file.name;
                // 直接挂一个自定义字段，避免在某些浏览器上修改只读属性失败
                // 后续统一通过 file.relativePath 读取
                // eslint-disable-next-line no-param-reassign
                file.relativePath = relativePath;
                collectedFiles.push(file);
                resolve();
              });
            } else if (entry.isDirectory) {
              const reader = entry.createReader();
              const dirPath = parentPath ? `${parentPath}/${entry.name}` : entry.name;
              const readEntries = () => {
                reader.readEntries((entries) => {
                  if (!entries || entries.length === 0) {
                    resolve();
                  } else {
                    Promise.all(entries.map((e) => processEntry(e, dirPath))).then(() => {
                      readEntries();
                    });
                  }
                });
              };
              readEntries();
            } else {
              resolve();
            }
          });
        };

        const tasks = [];
        for (let i = 0; i < items.length; i += 1) {
          const item = items[i];
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry && item.webkitGetAsEntry();
            if (entry) {
              tasks.push(processEntry(entry));
            } else {
              const file = item.getAsFile && item.getAsFile();
              if (file) {
                collectedFiles.push(file);
              }
            }
          }
        }

        Promise.all(tasks).then(() => {
          if (collectedFiles.length > 0) {
            this.addFilesToList(collectedFiles);
          }
        });
      } else if (files && files.length > 0) {
        // 回退逻辑：直接使用 files 列表（不保留目录结构）
        // 某些浏览器/场景下，把整个文件夹当成一个“文件”拖拽进来，
        // 这类对象通常 size 为 0 且没有扩展名，这里直接给出友好提示，避免无意义的失败上传。
        const fileArray = Array.from(files);
        const allMaybeFolderPlaceholder = fileArray.length === 1
          && fileArray[0].size === 0
          && !fileArray[0].type
          && !fileArray[0].name.includes('.');

        if (allMaybeFolderPlaceholder) {
          ElMessage.warning('当前浏览器对“直接拖拽整个文件夹”支持不佳，请使用“选择文件夹”按钮进行上传');
          return;
        }

        this.addFilesToList(fileArray);
      }
    },
    // 处理文件列表区域的拖拽上传（支持文件和文件夹）
    async handleFileAreaDrop(event) {
      if (!this.ensureClientReady()) return;

      const items = event.dataTransfer?.items;
      const fileList = event.dataTransfer?.files;
      if ((!items || items.length === 0) && (!fileList || fileList.length === 0)) {
        return;
      }

      // 收集文件（包含相对路径，用于还原文件夹结构）
      const collectedFiles = [];
       // 额外收集目录路径，用于处理“空文件夹”场景
       const collectedDirs = new Set();

      const processEntry = (entry, parentPath = '') => {
        return new Promise((resolve) => {
          if (entry.isFile) {
            entry.file((file) => {
              const relativePath = parentPath ? `${parentPath}/${file.name}` : file.name;
              // 自定义相对路径字段
              // eslint-disable-next-line no-param-reassign
              file.relativePath = relativePath;
              collectedFiles.push(file);
              resolve();
            });
          } else if (entry.isDirectory) {
            const reader = entry.createReader();
            const dirPath = parentPath ? `${parentPath}/${entry.name}` : entry.name;
             // 记录目录路径，即使该目录下没有任何文件
             if (dirPath) {
               collectedDirs.add(dirPath);
             }
            const readEntries = () => {
              reader.readEntries((entries) => {
                if (!entries || entries.length === 0) {
                  resolve();
                } else {
                  Promise.all(entries.map((e) => processEntry(e, dirPath))).then(() => {
                    readEntries();
                  });
                }
              });
            };
            readEntries();
          } else {
            resolve();
          }
        });
      };

      const tasks = [];
      if (items && items.length > 0 && items[0].webkitGetAsEntry) {
        for (let i = 0; i < items.length; i += 1) {
          const item = items[i];
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry && item.webkitGetAsEntry();
            if (entry) {
              tasks.push(processEntry(entry));
            } else {
              const file = item.getAsFile && item.getAsFile();
              if (file) {
                collectedFiles.push(file);
              }
            }
          }
        }
      } else if (fileList && fileList.length > 0) {
        const files = Array.from(fileList);
        const allMaybeFolderPlaceholder =
          files.length === 1 &&
          files[0].size === 0 &&
          !files[0].type &&
          !files[0].name.includes('.');

        if (allMaybeFolderPlaceholder) {
          ElMessage.warning('当前浏览器对"直接拖拽整个文件夹"支持不佳，请尝试在支持的浏览器中使用或使用上传对话框');
          return;
        }

        files.forEach(f => collectedFiles.push(f));
      }

      if (tasks.length > 0) {
        await Promise.all(tasks);
      }

      // 如果没有实际文件但包含目录，则认为是拖拽“空文件夹”，直接在当前目录创建这些空目录
      if (collectedFiles.length === 0 && collectedDirs.size > 0) {
        try {
          const targetDir = this.data.sftp_current_dir || '/';
          const dirs = Array.from(collectedDirs).sort();
          for (const dir of dirs) {
            let remoteDir = targetDir.endsWith('/') ? targetDir.slice(0, -1) : targetDir;
            remoteDir = remoteDir === '/' ? `/${dir}` : `${remoteDir}/${dir}`;
            remoteDir = remoteDir.replace(/\/+/g, '/');
            await this.ensureDirectoryExists(remoteDir);
          }
          ElMessage.success('已创建空文件夹');
          this.listDir(this.data.sftp_current_dir || '/', this.data.current_host);
        } catch (e) {
          ElMessage.error('创建空文件夹失败：' + (e.message || e));
        }
        return;
      }

      if (collectedFiles.length === 0) {
        return;
      }

      // 标记是否包含目录结构
      const hasDirectory = collectedFiles.some(f => f.relativePath && f.relativePath.includes('/'));

      // 若上传对话框未打开，先打开
      if (!this.data.upload_dialog_visible) {
        this.openUploadDialog();
      }

      // 添加文件到上传列表
      this.addFilesToList(collectedFiles);
    },
    // 添加文件到列表
    addFilesToList(fileList) {
      const newFiles = [];
      fileList.forEach((file) => {
        // 检查是否已存在
        const exists = this.data.upload_file_list.some(
          item => item.name === file.name && item.size === file.size && item.lastModified === file.lastModified
        );
        if (!exists) {
          const relativePath = file.relativePath || file.webkitRelativePath || file.name;
          newFiles.push({
            file,
            name: file.name,
            size: file.size,
            relativePath,
            isDirectory: relativePath && relativePath.includes('/'),
            lastModified: file.lastModified
          });
        }
      });
      this.data.upload_file_list.push(...newFiles);
      if (newFiles.length > 0) {
        ElMessage.success(`已添加 ${newFiles.length} 个文件`);
      }
    },
    // 清空文件列表
    clearFileList() {
      if (this.data.sftp_upload_status.isUploading) {
        ElMessage.warning('正在上传中，无法清空');
        return;
      }
      this.data.upload_file_list = [];
    },
    // 开始上传
    async startUpload() {
      if (!this.ensureClientReady()) return;
      if (this.data.upload_file_list.length === 0) {
        ElMessage.warning('请先选择要上传的文件');
        return;
      }

      const targetPath = this.data.sftp_current_dir || '/';

      try {
        // 判断是文件夹还是文件
        const isDirectory = this.data.upload_file_list.some(item => item.isDirectory);

        // 处理文件列表
        const files = await this.processFileList(
          this.data.upload_file_list.map(item => item.file),
          isDirectory
        );

        if (files.length === 0) {
          ElMessage.warning('没有可上传的文件');
          return;
        }

        // 计算总大小
        let totalSize = 0;
        files.forEach((item) => {
          totalSize += item.file.size || 0;
        });

        // 收集所有需要创建的目录（在上传前批量创建）
        const dirsToCreate = new Set();
        files.forEach((item) => {
          const relativePath = item.relativePath;
          const pathParts = relativePath.split('/').filter(p => p);
          if (pathParts.length > 1) {
            // 构建所有需要创建的父目录
            let currentPath = targetPath.endsWith('/') ? targetPath.slice(0, -1) : targetPath;
            for (let i = 0; i < pathParts.length - 1; i++) {
              currentPath = currentPath === '/'
                ? `/${pathParts[i]}`
                : `${currentPath}/${pathParts[i]}`;
              dirsToCreate.add(currentPath);
            }
          }
        });

        // 批量创建所有需要的目录
        if (dirsToCreate.size > 0) {
          const dirArray = Array.from(dirsToCreate).sort(); // 排序确保父目录先创建
          for (const dir of dirArray) {
            await this.ensureDirectoryExists(dir);
          }
        }

        // 初始化上传状态
        this.data.sftp_upload_status.isUploading = true;
        this.data.sftp_upload_status.totalFiles = files.length;
        this.data.sftp_upload_status.uploadedFiles = 0;
        this.data.sftp_upload_status.totalSize = totalSize;
        this.data.sftp_upload_status.uploadedSize = 0;
        this.data.sftp_upload_status.cancelRequested = false;
        this.uploadStrategy = 'ask'; // 重置策略

        // 生成 uploadId，用于后端统计本次上传进度
        const uploadId = `sftp-${Date.now()}-${Math.random().toString(16).slice(2)}`;
        this.currentUploadId = uploadId;

        // 启动轮询后端真实进度（多文件上传：后端返回的是“已上传总字节数”）
        const pollProgress = () => {
          // 如果已停止上传且未处于取消中，则不再轮询
          if (!this.data.sftp_upload_status.isUploading && !this.data.sftp_upload_status.cancelRequested) {
            return;
          }
          this.apiClient
            .get('/api/sftp/upload/progress', { params: { id: uploadId } })
            .then((res) => {
              const data = res.data || {};
              if (data.code !== 0 || !data.exists) {
                // 如果进度不存在且已取消，停止轮询
                if (this.data.sftp_upload_status.cancelRequested) {
                  return;
                }
                // 否则稍后再试
                setTimeout(pollProgress, 500);
                return;
              }

              // 后端返回：当前 uploadId 已上传的总字节数
              const backendTotal = Number(data.total || 0);
              const uploadedSize = Number(data.uploaded || 0);
              // totalSize 使用前端预先统计的“本次批量上传总大小”，后端 Total 仅作兜底
              const total =
                this.data.sftp_upload_status.totalSize > 0
                  ? this.data.sftp_upload_status.totalSize
                  : backendTotal;

              this.data.sftp_upload_status.uploadedSize = uploadedSize;

              let percent = 0;
              if (total > 0) {
                percent = Math.floor((uploadedSize / total) * 100);
              }
              if (percent < 0) percent = 0;
              if (percent > 100) percent = 100;
              this.data.sftp_upload_percentage = percent;

              // 检查是否有错误信息（如取消）
              if (data.error) {
                // 如果后端返回错误（如"上传已取消"），停止轮询
                return;
              }

              // 如果已完成或已取消，停止轮询
              if (data.done || this.data.sftp_upload_status.cancelRequested) {
                return;
              }
              // 继续轮询
              setTimeout(pollProgress, 500);
            })
            .catch(() => {
              // 如果已取消，停止轮询
              if (this.data.sftp_upload_status.cancelRequested) {
                return;
              }
              // 忽略单次进度查询错误，继续轮询
              setTimeout(pollProgress, 800);
            });
        };
        setTimeout(pollProgress, 500);

        // 逐个上传文件（保持文件夹结构）
        const uploadResults = [];
        let skipped = 0;
        for (let i = 0; i < files.length; i += 1) {
          // 如果用户请求取消上传，则中断后续文件
          if (this.data.sftp_upload_status.cancelRequested) {
            console.warn('用户取消上传，停止剩余文件上传');
            break;
          }
          try {
            // 构建远程文件路径
            const relativePath = files[i].relativePath;
            const pathParts = relativePath.split('/').filter(p => p);
            const fileName = pathParts.pop() || files[i].file.name;
            const subDir = pathParts.length > 0 ? pathParts.join('/') : '';

            let remotePath = targetPath.endsWith('/') ? targetPath.slice(0, -1) : targetPath;
            if (subDir) {
              remotePath = remotePath === '/' ? `/${subDir}/${fileName}` : `${remotePath}/${subDir}/${fileName}`;
            } else {
              remotePath = remotePath === '/' ? `/${fileName}` : `${remotePath}/${fileName}`;
            }
            remotePath = remotePath.replace(/\/+/g, '/');

            // 检查文件是否存在
            const exists = await this.checkSftpFileExists(remotePath);
            if (exists) {
              const action = await this.handleSftpFileExists(fileName, remotePath);
              if (action === 'cancel') {
                // 用户取消，停止上传
                break;
              } else if (action === 'skip') {
                skipped++;
                uploadResults.push({
                  file: files[i].relativePath,
                  success: false,
                  skipped: true
                });
                continue;
              }
              // action === 'overwrite' 继续上传
            }

            const result = await this.uploadSingleFile(files[i], targetPath, i, files.length);
            uploadResults.push({
              file: files[i].relativePath,
              success: true
            });
          } catch (error) {
            console.error(`上传文件 ${files[i].relativePath} 失败:`, error);
            uploadResults.push({
              file: files[i].relativePath,
              success: false,
              error: error.message
            });
          }
        }

        // 上传完成
        this.data.sftp_upload_status.isUploading = false;
        
        // 检查是否被取消
        const wasCanceled = this.data.sftp_upload_status.cancelRequested;
        
        if (wasCanceled) {
          // 用户取消了上传，显示取消消息
          ElNotification({
            type: 'info',
            duration: 5000,
            title: '上传已取消',
            message: '文件上传已取消'
          });
          // 延迟关闭对话框
          setTimeout(() => {
            this.closeUploadDialog();
          }, 1500);
        } else {
          // 正常完成，更新进度条到100%
          if (this.data.sftp_upload_percentage < 100) {
            this.data.sftp_upload_percentage = 100;
          }

          // 刷新文件列表
          this.listDir(this.data.sftp_current_dir || '/', this.data.current_host);

          // 显示结果通知
          const successCount = uploadResults.filter(r => r.success).length;
          const skippedCount = uploadResults.filter(r => r.skipped).length;
          const failCount = uploadResults.filter(r => !r.success && !r.skipped).length;

          if (failCount === 0 && skippedCount === 0) {
            ElNotification({
              type: 'success',
              duration: 5000,
              title: '上传完成',
              message: `成功上传 ${successCount} 个文件到 ${targetPath}`
            });
            // 延迟关闭对话框
            setTimeout(() => {
              this.closeUploadDialog();
            }, 1500);
          } else {
            let errorMsg = `成功: ${successCount} 个`;
            if (skippedCount > 0) {
              errorMsg += `，跳过: ${skippedCount} 个`;
            }
            if (failCount > 0) {
              errorMsg += `，失败: ${failCount} 个`;
            }
            const failedFiles = uploadResults.filter(r => !r.success && !r.skipped).slice(0, 5);
            if (failedFiles.length > 0) {
              errorMsg += '<br/>失败文件：<br/>' + failedFiles.map(f => `• ${f.file}`).join('<br/>');
            }
            ElNotification({
              type: 'warning',
              duration: 8000,
              title: '上传完成',
              message: errorMsg,
              dangerouslyUseHTMLString: true
            });
          }
        }

        // 延迟重置进度条和清空文件列表
        setTimeout(() => {
          this.data.sftp_upload_percentage = 0;
          this.data.sftp_upload_status = {
            totalFiles: 0,
            uploadedFiles: 0,
            currentFile: '',
            currentFileProgress: 0,
            totalSize: 0,
            uploadedSize: 0,
            isUploading: false,
            currentFileLoaded: 0
          };
          if (failCount === 0) {
            this.data.upload_file_list = [];
          }
        }, 2000);
      } catch (error) {
        console.error('上传过程出错:', error);
        ElMessage.error('上传过程出错: ' + error.message);
        this.data.sftp_upload_status.isUploading = false;
      }
    },
    // 上传文件或文件夹（已改为打开对话框，保留此方法以兼容可能的旧调用）
    async uploadFile(path) {
      // 如果指定了路径，先切换到该路径
      if (path && path !== this.data.sftp_current_dir) {
        this.listDir(path, this.data.current_host);
      }
      // 打开上传对话框
      this.openUploadDialog();
    },
    // 旧的上传逻辑（已废弃，保留以防万一）
    async uploadFileOld(path) {
      if (!this.ensureClientReady()) return;

      // 重置上传状态
      this.data.sftp_upload_percentage = 0;
      this.data.sftp_upload_status = {
        totalFiles: 0,
        uploadedFiles: 0,
        currentFile: '',
        currentFileProgress: 0,
        totalSize: 0,
        uploadedSize: 0,
        isUploading: false,
        currentFileLoaded: 0
      };

      const targetPath = path || this.data.sftp_current_dir || '/';

      // 创建文件选择器
      const fileInput = document.createElement('input');
      fileInput.type = 'file';
      fileInput.multiple = true;
      fileInput.setAttribute('webkitdirectory', ''); // 支持文件夹选择

      fileInput.onchange = async (event) => {
        const fileList = event.target.files;
        if (!fileList || !fileList.length) {
          return;
        }

        try {
          // 判断是文件夹还是文件（通过检查是否有 webkitRelativePath 属性）
          const fileArray = Array.from(fileList);
          const isDirectory = fileArray.some(file => file.webkitRelativePath && file.webkitRelativePath.includes('/'));

          // 处理文件列表
          const files = await this.processFileList(Array.from(fileList), isDirectory);

          if (files.length === 0) {
            ElMessage.warning('没有可上传的文件');
            return;
          }

          // 计算总大小和统计信息
          let totalSize = 0;
          files.forEach((item) => {
            totalSize += item.file.size || 0;
          });

          // 格式化文件大小
          const formatSize = (bytes) => {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
          };

          // 统计文件夹结构
          const folderSet = new Set();
          files.forEach((item) => {
            const pathParts = item.relativePath.split('/').filter(p => p);
            if (pathParts.length > 1) {
              // 提取所有父目录
              for (let i = 1; i < pathParts.length; i++) {
                folderSet.add(pathParts.slice(0, i).join('/'));
              }
            }
          });

          // 使用Element Plus MessageBox显示上传确认对话框
          const uploadType = isDirectory ? '文件夹' : '文件';
          const folderCount = folderSet.size;
          const message = isDirectory
            ? `准备上传文件夹到：<strong>${targetPath}</strong><br/><br/>` +
            `• 文件数量：${files.length} 个<br/>` +
            `• 总大小：${formatSize(totalSize)}<br/>` +
            `• 将创建 ${folderCount} 个子目录<br/><br/>` +
            `是否开始上传？`
            : `准备上传 ${files.length} 个文件到：<strong>${targetPath}</strong><br/><br/>` +
            `• 总大小：${formatSize(totalSize)}<br/><br/>` +
            `是否开始上传？`;

          try {
            await ElMessageBox.confirm(message, `上传${uploadType}确认`, {
              confirmButtonText: '开始上传',
              cancelButtonText: '取消',
              type: 'info',
              dangerouslyUseHTMLString: true,
              autofocus: false
            });
          } catch {
            // 用户取消上传
            return;
          }

          // 初始化上传状态
          this.data.sftp_upload_status.isUploading = true;
          this.data.sftp_upload_status.totalFiles = files.length;
          this.data.sftp_upload_status.uploadedFiles = 0;
          this.data.sftp_upload_status.totalSize = totalSize;
          this.data.sftp_upload_status.uploadedSize = 0;

          // 逐个上传文件（保持文件夹结构）
          const uploadResults = [];
          for (let i = 0; i < files.length; i += 1) {
            try {
              const result = await this.uploadSingleFile(files[i], targetPath, i, files.length);
              uploadResults.push({
                file: files[i].relativePath,
                success: true
              });
            } catch (error) {
              console.error(`上传文件 ${files[i].relativePath} 失败:`, error);
              uploadResults.push({
                file: files[i].relativePath,
                success: false,
                error: error.message
              });
            }
          }

          // 上传完成
          this.data.sftp_upload_status.isUploading = false;
          this.data.sftp_upload_percentage = 100;

          // 刷新文件列表
          this.listDir(this.data.sftp_current_dir || '/', this.data.current_host);

          // 显示结果通知
          const successCount = uploadResults.filter(r => r.success).length;
          const failCount = uploadResults.length - successCount;

          if (failCount === 0) {
            ElNotification({
              type: 'success',
              duration: 5000,
              title: '上传完成',
              message: `成功上传 ${successCount} 个文件到 ${targetPath}`
            });
          } else {
            let errorMsg = `成功: ${successCount} 个，失败: ${failCount} 个`;
            const failedFiles = uploadResults.filter(r => !r.success).slice(0, 5);
            if (failedFiles.length > 0) {
              errorMsg += '<br/>失败文件：<br/>' + failedFiles.map(f => `• ${f.file}`).join('<br/>');
            }
            ElNotification({
              type: 'warning',
              duration: 8000,
              title: '上传完成（部分失败）',
              message: errorMsg,
              dangerouslyUseHTMLString: true
            });
          }

          // 延迟重置进度条
          setTimeout(() => {
            this.data.sftp_upload_percentage = 0;
            this.data.sftp_upload_status = {
              totalFiles: 0,
              uploadedFiles: 0,
              currentFile: '',
              currentFileProgress: 0,
              totalSize: 0,
              uploadedSize: 0,
              isUploading: false,
              currentFileLoaded: 0
            };
          }, 2000);
        } catch (error) {
          console.error('上传过程出错:', error);
          ElMessage.error('上传过程出错: ' + error.message);
          this.data.sftp_upload_status.isUploading = false;
        }
      };

      fileInput.click();
    },
    downloadFile(file) {
      if (!this.ensureClientReady()) return;

      // 如果是文件夹，调用目录打包下载接口（单次 zip 下载）
      if (file.type === 'd') {
        this.downloadDirectory(file);
        return;
      }

      // 单个文件下载
      const token = getToken() || '';
      const reqUrl = `${this.apiBase}/api/sftp/download?Authorization=${encodeURIComponent(
        token
      )}&session_id=${this.data.current_host.session_id}&path=${encodeURIComponent(file.path).replace(
        /%/g,
        '%25'
      )}`;
      BrowserOpenURL(reqUrl);
    },
    // 递归下载文件夹
    async downloadDirectory(dir) {
      if (!this.ensureClientReady()) return;
      const token = getToken() || '';
      const reqUrl = `${this.apiBase}/api/sftp/download_dir?Authorization=${encodeURIComponent(
        token
      )}&session_id=${this.data.current_host.session_id}&path=${encodeURIComponent(dir.path).replace(
        /%/g,
        '%25'
      )}`;

      // 直接触发一次打包下载（zip），交给浏览器 / 系统默认下载器处理
      BrowserOpenURL(reqUrl);
    },
    deleteFile(file) {
      if (!this.ensureClientReady()) return;
      const body = {
        session_id: this.data.current_host.session_id,
        path: file.path
      };
      this.apiClient.delete('/api/sftp/delete', { data: body }).then((ret) => {
        if (ret.data.code === 0) {
          this.listDir(this.data.sftp_current_dir, this.data.current_host);
          ElMessage.success('删除文件成功');
        } else {
          ElMessage.error('删除文件出错了');
        }
      });
    },
    createDir(dir, h) {
      // 打开创建目录对话框
      this.data.create_dir_dialog_visible = true;
      this.data.new_dir_name = '';
    },
    confirmCreateDir() {
      if (!this.ensureClientReady()) return;

      // 验证目录名称
      const dirName = (this.data.new_dir_name || '').trim();
      if (!dirName) {
        ElMessage.warning('请输入目录名称');
        return;
      }

      // 验证目录名称合法性（不允许包含 / 等特殊字符，除非是相对路径）
      if (dirName.includes('/') || dirName.includes('\\')) {
        ElMessage.warning('目录名称不能包含路径分隔符');
        return;
      }

      // 构建完整路径
      const currentDir = this.data.sftp_current_dir || '/';
      const fullPath = currentDir.endsWith('/')
        ? `${currentDir}${dirName}`
        : `${currentDir}/${dirName}`;

      const body = {
        session_id: this.data.current_host.session_id,
        path: fullPath
      };

      this.apiClient.post('/api/sftp/create_dir', body).then((ret) => {
        if (ret.data.code === 0) {
          this.listDir(this.data.sftp_current_dir, this.data.current_host);
          ElMessage.success('创建目录成功');
          this.data.create_dir_dialog_visible = false;
          this.data.new_dir_name = '';
        } else {
          ElMessage.error(ret.data.msg || '创建目录出错了');
        }
      }).catch((err) => {
        ElMessage.error('创建目录失败: ' + (err.message || '未知错误'));
      });
    },
    getAllHost() {
      if (!this.ensureClientReady()) return;
      this.apiClient.get('/api/conn_conf').then((ret) => {
        if (ret.data.code === 0) {
          this.data.host_list = ret.data.data;
        } else {
          ElMessage.error('获取主机列表错误');
        }
      });
    },
    handleOpenHostList() {
      this.dialogHostListVisible = true;
    },
    handleNewHostFromEmpty() {
      this.dialogHostListVisible = false;
      this.newHost();
    },
    createOrUpdateHost(host, m) {
      if (!this.ensureClientReady()) return;
      if (m === Mode.create) {
        for (let i = 0; i < this.data.host_list.length; i += 1) {
          const item = this.data.host_list[i];
          if (item.name === host.name) {
            ElMessage.error('名称已经存在,请修改');
            return;
          }
        }
      }
      this.data.host_dialog_visible = false;
      if (m === Mode.create) {
        this.apiClient.post('/api/conn_conf', host).then((ret) => {
          if (ret.data.code === 0) {
            this.data.host_list = ret.data.data;
            this.cleanFrom();
          } else {
            ElMessage.error('新增出错了');
          }
        });
      } else {
        this.apiClient.put(`/api/conn_conf`, host).then((ret) => {
          if (ret.data.code === 0) {
            this.data.host_list = ret.data.data;
            this.cleanFrom();
          } else {
            ElMessage.error('更新出错了');
          }
        });
      }
    },
    newHost() {
      this.cleanFrom();
      this.data.host_dialog_visible = true;
      this.data.mode = Mode.create;
    },
    createHost(isConnect = false) {
      this.data.mode = Mode.create;
      const result = this.verifyFrom();
      if (!result.is_success) {
        return;
      }
      this.createOrUpdateHost(result.host, Mode.create);
      if (isConnect) {
        this.connectHost(result.host);
      }
    },
    editHost(row) {
      this.data.host_dialog_visible = true;
      this.data.mode = Mode.update;
      this.data.id = row.id;
      this.data.address = row.address;
      this.data.name = row.name;
      this.data.user = row.user;
      this.data.auth_type = row.auth_type;
      this.data.net_type = row.net_type;
      this.data.cert_data = row.cert_data;
      this.data.cert_pwd = row.cert_pwd;
      this.data.pwd = row.pwd;
      this.data.port = row.port;
      this.data.background = row.background;
      this.data.foreground = row.foreground;
      this.data.cursor_color = row.cursor_color;
      this.data.font_family = row.font_family;
      this.data.font_size = row.font_size;
      this.data.cursor_style = row.cursor_style;
      this.data.shell = row.shell;
      this.data.pty_type = row.pty_type;
      this.data.init_cmd = row.init_cmd;
      this.data.init_banner = row.init_banner;
      this.data.host_config_collapse = ['1'];
    },
    updateHost(isConnect = false) {
      const result = this.verifyFrom();
      if (!result.is_success) {
        return;
      }
      this.createOrUpdateHost(result.host, Mode.update);
      if (isConnect) {
        this.connectHost(result.host);
      }
    },
    async deleteHost(row) {
      if (!this.ensureClientReady()) return;
      try {
        await confirmDangerousAction({
          title: '删除主机',
          message: `将删除主机配置「${row.name || row.address || row.id}」，不可恢复。`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      this.apiClient.delete(`/api/conn_conf/${row.id}`).then((ret) => {
        if (ret.data.code === 0) {
          this.data.host_list = ret.data.data;
          this.cleanFrom();
        } else {
          ElMessage.error('删除主机出错了');
        }
      });
    },
    getHost(data) {
      if (data.term) {
        try {
          data.fit.dispose();
          data.term.dispose();
          data.ws.close();
        } catch (err) {
          console.log(`清理资源错误:${err}`);
        }
      }
      const connectTabElement = document.getElementById(data.session_id);
      if (connectTabElement) {
        // 清理右键菜单事件监听器
        if (connectTabElement._terminalContextMenuCleanup) {
          connectTabElement._terminalContextMenuCleanup();
          delete connectTabElement._terminalContextMenuCleanup;
        }
        connectTabElement.innerHTML = '';
      }
      return {
        id: data.id,
        name: data.name,
        address: data.address,
        user: data.user,
        auth_type: data.auth_type,
        net_type: data.net_type,
        cert_data: data.cert_data,
        cert_pwd: data.cert_pwd,
        pwd: data.pwd,
        port: data.port,
        session_id: data.session_id,
        background: data.background,
        foreground: data.foreground,
        cursor_color: data.cursor_color,
        font_family: data.font_family,
        font_size: data.font_size,
        cursor_style: data.cursor_style,
        shell: data.shell,
        pty_type: data.pty_type,
        init_cmd: data.init_cmd,
        init_banner: data.init_banner
      };
    },
    async connectHost(host, isReconnect = false) {
      if (!this.ensureClientReady()) return;
      const infoMessage = ElMessage.info('正在尝试连接，请耐心等待~~~');
      this.data.host_dialog_visible = false;
      this.dialogHostListVisible = false;
      let requestUrl = '/api/ssh/create_session';
      if (isReconnect) {
        requestUrl += `?session_id=${host.session_id}`;
      }
      const connHost = this.getHost(host);
      try {
        const ret = await this.apiClient.post(requestUrl, connHost);
        if (ret.data.code !== 0) {
          infoMessage.close?.();
          ElMessage.error('创建连接出错了');
          return;
        }
        const session_id = ret.data.data;
        connHost.session_id = session_id;
        connHost.fit = new FitAddon();
        connHost.term = new Terminal({
          cursorBlink: true,
          theme: {
            background: connHost.background,
            foreground: connHost.foreground,
            cursor: connHost.cursor_color
          },
          fontSize: connHost.font_size,
          fontFamily: connHost.font_family,
          cursorStyle: connHost.cursor_style
        });
        connHost.term.loadAddon(connHost.fit);
        if (isReconnect) {
          for (const [index, h] of this.data.host_tabs.entries()) {
            if (h.session_id === session_id) {
              connHost.is_close = false;
              this.data.host_tabs[index] = connHost;
              break;
            }
          }
        } else {
          this.data.host_tabs.push(connHost);
        }
        await nextTick();
        const connectTabElement = document.getElementById(connHost.session_id);
        if (connectTabElement === null) {
          infoMessage.close?.();
          ElMessage.error('创建连接获取dom为空!');
          return;
        }
        connectTabElement.style.height = `${Math.floor(window.innerHeight - 52)}px`;
        connHost.term.open(connectTabElement);
        connHost.fit.fit();
        // 添加右键复制功能
        this.setupTerminalContextMenu(connHost.term, connectTabElement);
        const safeSize = this.ensureTerminalSize(connHost);
        const token = getToken() || '';
        // 本地服务防护 Token（Wails 页面 Origin 为 null 时校验）
        const localToken = (window.__LOCAL_SERVICE_TOKEN__ || '');
        // 在Electron/Wails环境中，直接使用127.0.0.1，避免window.location.hostname在Linux/Mac上返回空值
        const host = '127.0.0.1';
        const protocol = 'ws:';
        const sockUrl = `${protocol}//${host}:${this.portConfig?.sshPort || 52868}/api/ssh/conn?h=${safeSize.rows}&w=${safeSize.cols}&session_id=${connHost.session_id}&Authorization=${encodeURIComponent(
          token
        )}&token=${encodeURIComponent(localToken)}`;
        const ws = new WebSocket(sockUrl);
        ws.onopen = () => {
          try {
            const bannerStr = (connHost.init_banner || '').trim();
            if (bannerStr !== '') {
              connHost.term.writeln(bannerStr);
            }
            this.windowResize();
            const cmdStr = (connHost.init_cmd || '').trim();
            if (cmdStr !== '') {
              ws.send(`${cmdStr}\n`);
            }
          } catch (err) {
            console.log(err);
          }
        };
        ws.onerror = () => {
          console.log('WebSocket error');
          infoMessage.close?.();
          connHost.term.writeln('##  连接出错,请重连!  ##');
        };
        ws.onclose = () => {
          console.log(`WebSocket close:${connHost.session_id}`);
          connHost.term.writeln('##  连接关闭,请重连!  ##');
          connHost.is_close = true;
          if (this.data.current_host.session_id === session_id) {
            this.data.current_host.is_close = true;
          }
        };
        infoMessage.close?.();
        ElMessage.success('连接成功');
        connHost.term.loadAddon(new AttachAddon(ws));
        connHost.ws = ws;
        connHost.is_close = false;
        connHost.term.focus();
        this.cleanFrom();
        this.data.current_host = { ...connHost };
      } catch (err) {
        infoMessage.close?.();
        ElMessage.error('创建会话出错了');
        console.log(err);
      }
    },
    removeTab(tabId) {
      if (!this.apiClient) return;
      try {
        this.apiClient.post(`/api/ssh/disconnect?session_id=${tabId}`);
      } catch (error) {
        console.log(error);
      }
      let removeIndex = 0;
      for (const [index, h] of this.data.host_tabs.entries()) {
        if (h.session_id === String(tabId)) {
          removeIndex = index;
          break;
        }
      }
      if (this.data.host_tabs[removeIndex]) {
        this.data.host_tabs[removeIndex].fit.dispose();
        this.data.host_tabs[removeIndex].term.dispose();
        this.data.host_tabs[removeIndex].ws.close();
      }
      this.data.host_tabs.splice(removeIndex, 1);
      if (this.data.host_tabs.length === 0) {
        return;
      }
      if (this.data.host_tabs.length === 1) {
        const activeHost = { ...this.data.host_tabs[0] };
        this.setCurrentAcitveHost(activeHost.session_id);
        return;
      }
      const nextIndex = removeIndex > 0 ? removeIndex - 1 : 0;
      const activeHost = { ...this.data.host_tabs[nextIndex] };
      this.setCurrentAcitveHost(activeHost.session_id);
    },
    ensureTerminalSize(host) {
      const minCols = 80;
      const minRows = 24;
      if (!host?.term) {
        return { rows: minRows, cols: minCols };
      }
      host.fit?.fit();
      let cols = Math.max(host.term.cols || 0, minCols);
      let rows = Math.max(host.term.rows || 0, minRows);
      // 如果 fit 结果过小，强制给默认尺寸，避免首连时宽高为 0 触发后端错误
      if (host.term.cols !== cols || host.term.rows !== rows) {
        host.term.resize(cols, rows);
      }
      this.data.h = rows;
      this.data.w = cols;
      return { rows, cols };
    },
    setupTerminalContextMenu(term, element) {
      // 右键点击直接复制选中文本到剪贴板
      const handleContextMenu = async (e) => {
        e.preventDefault();

        // 获取终端中选中的文本
        const selectedText = term.getSelection();

        if (selectedText && selectedText.trim()) {
          try {
            // 优先使用现代 Clipboard API
            await navigator.clipboard.writeText(selectedText);
            ElMessage.success('已复制到剪贴板');
          } catch (err) {
            // 降级方案：使用传统方法
            try {
              const textArea = document.createElement('textarea');
              textArea.value = selectedText;
              textArea.style.position = 'fixed';
              textArea.style.left = '-9999px';
              textArea.style.top = '0';
              textArea.style.opacity = '0';
              document.body.appendChild(textArea);
              textArea.focus();
              textArea.select();

              const successful = document.execCommand('copy');
              document.body.removeChild(textArea);

              if (successful) {
                ElMessage.success('已复制到剪贴板');
              }
            } catch (fallbackErr) {
              console.error('复制失败:', fallbackErr);
            }
          }
        } else {
          // ElMessage.warning('没有选中内容');
        }
      };

      element.addEventListener('contextmenu', handleContextMenu);

      // 存储清理函数到终端元素，以便在销毁时清理
      if (!element._terminalContextMenuCleanup) {
        element._terminalContextMenuCleanup = () => {
          element.removeEventListener('contextmenu', handleContextMenu);
        };
      }
    },
    closeOtherTabs() {
      const currentId = this.data.current_host.session_id;
      if (!currentId) return;
      const otherIds = this.data.host_tabs.filter((h) => h.session_id !== currentId).map((h) => h.session_id);
      otherIds.forEach((id) => this.removeTab(id));
    },
    closeAllTabs() {
      const ids = this.data.host_tabs.map((h) => h.session_id);
      ids.forEach((id) => this.removeTab(id));
    },
    // 标签页右键菜单处理
    handleTabsContextMenu(event) {
      event.preventDefault()
      event.stopPropagation()

      // 检查是否点击在标签页上
      const tabItem = event.target.closest('.el-tabs__item')
      if (!tabItem) {
        this.hideTabsContextMenu()
        return
      }

      const tabIdAttr = tabItem.id || tabItem.getAttribute('aria-controls')
      this.tabsContextMenuTabId = tabIdAttr
        ? tabIdAttr.replace(/^tab-/, '').replace(/^pane-/, '')
        : this.data.current_host.session_id

      // 直接从事件获取坐标，优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const x = event.clientX || 0
      const y = event.clientY || 0

      // 直接设置坐标，确保菜单显示在鼠标点击位置
      this.tabsContextMenuPosition = { x, y }
      this.tabsContextMenuVisible = true

      // 点击其他地方时隐藏菜单
      const hideMenu = () => {
        this.hideTabsContextMenu()
        document.removeEventListener('click', hideMenu)
        document.removeEventListener('contextmenu', hideMenu)
      }
      setTimeout(() => {
        document.addEventListener('click', hideMenu)
        document.addEventListener('contextmenu', hideMenu)
      }, 100)
    },
    hideTabsContextMenu() {
      this.tabsContextMenuVisible = false
    },
    // 关闭其他标签页（右键菜单）
    handleCloseOtherTabs() {
      this.hideTabsContextMenu()
      const targetId = this.tabsContextMenuTabId || this.data.current_host.session_id
      if (!targetId) return
      const otherIds = this.data.host_tabs.filter((h) => h.session_id !== targetId).map((h) => h.session_id)
      otherIds.forEach((id) => this.removeTab(id))
    },
    // 关闭所有标签页（右键菜单）
    handleCloseAllTabs() {
      this.hideTabsContextMenu()
      const ids = this.data.host_tabs.map((h) => h.session_id)
      ids.forEach((id) => this.removeTab(id))
    },
    selectTab(tab) {
      const sessionId = tab.props.name;
      if (this.data.current_host.session_id === sessionId) {
        return;
      }
      this.setCurrentAcitveHost(sessionId);
    },
    setCurrentAcitveHost(sessionId) {
      for (const host of this.data.host_tabs) {
        if (host.session_id === sessionId) {
          this.data.current_host = { ...host };
          this.data.sftp_current_dir =
            this.data.sftp_last_dir[sessionId] || this.data.sftp_current_dir || '/';
          break;
        }
      }
      this.windowResize();
    },
    windowResize() {
      const currentHost = this.data.current_host;
      if (!currentHost.session_id) {
        return;
      }
      if (this.$route.name !== 'connect') {
        return;
      }
      nextTick(() => {
        const connectTabElement = document.getElementById(currentHost.session_id);
        if (connectTabElement === null) {
          console.log('调整窗口大小,没有获取到dom');
        } else {
          connectTabElement.style.height = `${Math.floor(window.innerHeight - 58)}px`;
          currentHost.fit?.fit();
          const safeSize = this.ensureTerminalSize(currentHost);
          const url = `/api/ssh/conn?w=${safeSize.cols}&h=${safeSize.rows}&session_id=${currentHost.session_id}`;
          if (this.apiClient) {
            this.apiClient.patch(url);
          }
          this.data.h = safeSize.rows;
          this.data.w = safeSize.cols;
        }

        // 同步刷新 SFTP 文件表格布局，防止窗口变大后表格宽度不更新
        const fileTable = this.$refs.fileTable;
        if (fileTable && typeof fileTable.doLayout === 'function') {
          fileTable.doLayout();
        }
      });
    },
    reportConnectStatus() {
      if (this.statusSetInterval) {
        clearInterval(this.statusSetInterval);
      }
      this.statusSetInterval = setInterval(() => {
        if (!this.ensureClientReady()) return;
        if (!this.data.host_tabs.length) return;
        const fm = new FormData();
        this.data.host_tabs.forEach((hont) => {
          fm.append('ids', hont.session_id);
        });
        this.apiClient.put('/api/conn_manage/refresh_conn_time', fm).then((res) => {
          if (res.data.code !== 0) {
            console.log('刷新失败');
          }
        });
      }, 10000);
    },
    debounce(fn, delay) {
      let timer = null;
      return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => {
          fn.apply(this, args);
        }, delay);
      };
    },
    throttle(fn, delay) {
      let record = Date.now();
      return (...args) => {
        const now = Date.now();
        if (now - record > delay) {
          fn.apply(this, args);
          record = now;
        }
      };
    },
    disconnectAllSession() {
      if (!this.apiClient) return;
      this.data.host_tabs.forEach((host) => {
        try {
          this.apiClient.post(`/api/ssh/disconnect?session_id=${host.session_id}`);
        } catch (error) {
          console.log(error);
        }
      });
    },
    // 清理事件监听器
    cleanup() {
      if (this.resizeHandler) {
        window.removeEventListener('resize', this.resizeHandler);
      }
      window.removeEventListener('fill-connect-form', this.handleFillConnectForm);
    },
    async initPage() {
      await this.initClient();
      this.startOnlineClientStream();
      this.reportConnectStatus();
      this.getAllHost();
      this.getAllCmdNote();
      this.resizeHandler = this.debounce(() => this.windowResize(), 200);
      window.addEventListener('resize', this.resizeHandler);
      // 监听填充连接表单事件
      window.addEventListener('fill-connect-form', this.handleFillConnectForm);
      this.windowResize();
    },
    // 处理填充连接表单事件
    handleFillConnectForm(event) {
      const { module, connectionInfo } = event.detail || {};
      // 只处理SSH模块的请求
      if (module !== 'ssh' || !connectionInfo) {
        return;
      }

      // 先填充表单数据（在打开对话框之前）
      if (connectionInfo.host) {
        this.data.address = connectionInfo.host;
      }
      if (connectionInfo.port) {
        this.data.port = parseInt(connectionInfo.port) || 22;
      }
      // 确保账号密码能够正确填充
      if (connectionInfo.username !== undefined) {
        this.data.user = connectionInfo.username;
      }
      if (connectionInfo.password !== undefined) {
        this.data.pwd = connectionInfo.password;
        this.data.auth_type = 'pwd';
      }
      if (connectionInfo.name) {
        this.data.name = connectionInfo.name;
      } else if (connectionInfo.host && connectionInfo.port) {
        // 如果没有名称，自动生成
        this.data.name = `SSH - ${connectionInfo.host}:${connectionInfo.port}`;
      }

      // 打开对话框
      this.newHost();

      // 等待对话框打开后，再次确认数据（防止cleanFrom重置）
      this.$nextTick(() => {
        setTimeout(() => {
          if (connectionInfo.host) {
            this.data.address = connectionInfo.host;
          }
          if (connectionInfo.port) {
            this.data.port = parseInt(connectionInfo.port) || 22;
          }
          // 重要：账号密码需要明确设置
          if (connectionInfo.username !== undefined) {
            this.data.user = connectionInfo.username;
          }
          if (connectionInfo.password !== undefined) {
            this.data.pwd = connectionInfo.password;
            this.data.auth_type = 'pwd';
          }
          if (connectionInfo.name) {
            this.data.name = connectionInfo.name;
          } else if (connectionInfo.host && connectionInfo.port) {
            this.data.name = `SSH - ${connectionInfo.host}:${connectionInfo.port}`;
          }
        }, 150);
      });
    },
    // 格式化文件大小显示
    formatFileSize(size) {
      if (!size && size !== 0) {
        return '-';
      }
      // 如果size是字符串，尝试转换为数字
      const numSize = typeof size === 'string' ? parseFloat(size) : size;
      if (isNaN(numSize) || numSize < 0) {
        return '-';
      }

      // 小于1KB，显示字节
      if (numSize < 1024) {
        return `${numSize} B`;
      }

      // 小于1MB，显示KB
      if (numSize < 1024 * 1024) {
        const kb = (numSize / 1024).toFixed(2);
        return `${parseFloat(kb)} KB`;
      }

      // 小于1GB，显示MB
      if (numSize < 1024 * 1024 * 1024) {
        const mb = (numSize / (1024 * 1024)).toFixed(2);
        return `${parseFloat(mb)} MB`;
      }

      // 大于等于1GB，显示GB
      const gb = (numSize / (1024 * 1024 * 1024)).toFixed(2);
      return `${parseFloat(gb)} GB`;
    }
  },
  watch: {
    'filePreview.editContent'(val, oldVal) {
      if (val !== oldVal && this.filePreview.isText) {
        this.filePreview.dirty = true;
      }
    }
  },
  async mounted() {
    await this.initPage();
    // 注册 SSH 文件拖拽（Wails 原生绝对路径）
    this.setupSshFileDropListener();
  },
  beforeUnmount() {
    this.cleanup();
    this.stopOnlineClientStream();
    if (this.statusSetInterval) {
      clearInterval(this.statusSetInterval);
    }
    this.disconnectAllSession();
  }
};
</script>

<style scoped>
.ssh-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.el-header {
  border-bottom: 1px solid #e5e7eb;
  padding: 6px 10px;
  margin-bottom: 6px;
  flex-shrink: 0;
}

.action-row {
  align-items: center;
}

.status-col {
  display: flex;
  justify-content: flex-end;
}

.file-dialog {
  margin-top: 0px;
}

.el-tabs {
  --el-tabs-header-height: 28px;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.el-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.el-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tabs-area {
  margin-top: 4px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.empty-connection {
  margin-top: 12px;
  padding: 24px 0;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  height: 84vh;
}

.terminal-wrap {
  margin: 1px;
  width: 100%;
  height: 100%;
  flex: 1;
  background: #0b1021;
  border-radius: 6px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.terminal {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.terminal :deep(.xterm-screen) {
  overflow: hidden;
}

.terminal :deep(.xterm-viewport) {
  overflow-y: auto !important;
  overflow-x: hidden !important;
}

/* 自定义终端滚动条样式，仅作用于 ssh 终端区域 */
.terminal :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}

.terminal :deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: rgba(15, 23, 42, 0.7);
}

.terminal :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: rgba(148, 163, 184, 0.8);
  border-radius: 999px;
  border: 1px solid rgba(15, 23, 42, 0.9);
}

.terminal :deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: rgba(226, 232, 240, 0.9);
}

.ssh-page :deep(.el-button) {
  padding: 6px 10px;
  font-size: 13px;
}

.ssh-page :deep(.el-input__wrapper),
.ssh-page :deep(.el-select .el-input__wrapper) {
  min-height: 32px;
}

.ssh-page :deep(.el-dialog__body) {
  background: #f9fafb;
}

.cmd-dialog :deep(.el-textarea__inner) {
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
  min-height: 110px;
}

.cmd-dialog :deep(.el-input-group__append .el-button) {
  padding: 6px 8px;
}

.ssh-dialog :deep(.el-dialog__header) {
  padding: 12px 18px;
  background: linear-gradient(135deg, #0ea5e9, #6366f1);
  color: #ffffff;
  border-bottom: 1px solid #e5e7eb;
}

.ssh-dialog :deep(.el-dialog__title) {
  color: #ffffff;
  font-weight: 600;
  letter-spacing: 0.3px;
}

.ssh-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #ffffff;
}

.ssh-dialog :deep(.el-dialog__body) {
  background: #f9fafb;
}

.ssh-dialog :deep(.el-dialog__footer) {
  background: #f9fafb;
  padding: 10px 18px 16px;
}

.dialog-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.empty-host-list {
  min-height: 360px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ssh-dialog :deep(.el-table) {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.06);
}

.tabs-context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 2px 0;
  margin: 0;
  list-style: none;
  z-index: 3000;
  min-width: 85px;
}

.tabs-context-menu li {
  padding: 6px 12px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  transition: background-color 0.2s;
}

.tabs-context-menu li:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  cursor: default;
}

.file-icon {
  font-size: 16px;
}

.dir-icon {
  color: #2563eb;
}

.file-icon-normal {
  color: #16a34a;
}

.file-name-text {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
}

.path-actions :deep(.el-button) {
  border-color: transparent;
}

.file-table :deep(.el-table__header) {
  background: #f3f4f6;
}

.preview-empty-text {
  color: #e5e7eb;
  font-size: 13px;
}

.file-table :deep(.el-table__header) {
  background: #f3f4f6;
}

.file-table :deep(.el-table__row) {
  cursor: default;
}

.file-table :deep(.el-table__row:hover) {
  background-color: #eef2ff;
}

.file-drawer {
  --el-drawer-bg-color: #f9fafb;
}

:deep(.el-drawer__body) {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.file-drawer-header {
  padding: 10px 16px 6px;
  border-bottom: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #73c1e4, #bebef0);
  color: #fff;
}

.file-drawer-title {
  font-size: 13px;
}

.file-drawer-body {
  padding: 4px 10px 10px;
  display: flex;
  gap: 10px;
  height: calc(100% - 46px);
  box-sizing: border-box;
  width: 100%;
  overflow-x: auto;
  overflow-y: auto;
}

.file-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #ffffff;
  border-radius: 8px;
  padding: 0px;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.06);
  min-height: 0;
  overflow: hidden;
}

.file-table-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: auto;
  box-sizing: border-box;
  padding-bottom: 4px;
  /* 防止最后一行被下边缘遮挡 */
  width: 100%;
}

.file-drawer :deep(.el-drawer__body) {
  padding: 0;
}

.preview-pane {
  flex: 1.2;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  border-radius: 8px;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.14);
  color: #e5e7eb;
  overflow: hidden;
}

.file-pane,
.preview-pane {
  min-width: 0;
  max-width: 100%;
}

/* 当没有预览时，文件面板占满空间 */
.file-drawer-body:has(.file-pane:only-child) .file-pane {
  flex: 1;
}

/* 当没有预览时，文件面板占满空间 */
.file-drawer-body:not(.has-preview) .file-pane {
  flex: 1;
}

.file-drawer-body.has-preview .file-pane {
  flex: 1.1;
}

.file-drawer-body.has-preview .preview-pane {
  flex: 1.2;
}

/* 窗口较窄时（例如非全屏），自动改为上下布局以避免横向滚动条 */
@media (max-width: 1400px) {
  .file-drawer-body {
    flex-direction: column;
  }

  .file-pane,
  .preview-pane {
    width: 100%;
  }

  .preview-pane {
    margin-top: 8px;
  }
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.4);
  background: radial-gradient(circle at top left, #1e293b, #020617);
}

.preview-title {
  font-size: 13px;
  color: #e5e7eb;
}

.preview-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-close-btn {
  color: #fca5a5;
}

.preview-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #1f1f21;
  min-height: 0;
  overflow: hidden;
}

.preview-scroll {
  height: 100%;
  max-height: 100%;
  overflow: hidden;
}

.preview-editor-wrapper {
  height: 100%;
  padding: 0px;
  box-sizing: border-box;
}

.file-context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 2px 0;
  margin: 0;
  list-style: none;
  z-index: 3000;
  min-width: 120px;
}

.file-context-menu li {
  padding: 6px 14px;
  cursor: pointer;
  font-size: 13px;
  color: #606266;
  transition: background-color 0.2s;
}

.file-context-menu li:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.file-context-menu li.danger {
  color: #f56c6c;
}

.preview-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 230px;
  padding: 20px;
  box-sizing: border-box;
}

.preview-body>.preview-empty {
  /* 当没有文件时，空预览在 preview-body 内，限制高度避免滚动条 */
  max-height: 250px;
  overflow: hidden;
}

.preview-empty-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.preview-textarea :deep(textarea) {
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.upload-progress-container {
  padding: 8px 0;
  background: #f9fafb;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
}

.upload-progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.upload-status-text {
  font-size: 13px;
  color: #606266;
  flex: 1;
}

.current-file-name {
  color: #409eff;
  font-weight: 500;
  margin-left: 8px;
}

.upload-percentage {
  font-size: 14px;
  font-weight: 600;
  color: #409eff;
  min-width: 50px;
  text-align: right;
}

.upload-size-info {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
  text-align: right;
}

/* 上传对话框样式 */
.upload-dialog-content {
  padding: 10px 0;
}

.upload-drop-zone {
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  text-align: center;
  background-color: #fafafa;
  transition: all 0.3s;
  cursor: pointer;
  margin: 20px 0;
}

.upload-drop-zone:hover {
  border-color: #409eff;
  background-color: #f0f9ff;
}

.upload-drop-zone.drag-over {
  border-color: #409eff;
  background-color: #ecf5ff;
  border-style: solid;
}

.upload-icon {
  color: #909399;
  margin-bottom: 2px;
}

.upload-drop-zone.drag-over .upload-icon {
  color: #409eff;
}

.upload-text {
  margin: 12px 0;
  color: #606266;
  font-size: 14px;
}

.upload-buttons {
  margin: 12px 0;
}

.upload-buttons .el-button {
  margin: 0 5px;
}

.upload-tip {
  margin-top: 12px;
  font-size: 12px;
  color: #909399;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.upload-tip .el-icon {
  color: #409eff;
  font-size: 14px;
}

.upload-file-list {
  margin-top: 20px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
}

.upload-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
  background-color: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  font-size: 14px;
  color: #606266;
}

.file-list-items {
  padding: 8px;
}

.file-list-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.file-list-item:hover {
  background-color: #f5f7fa;
}

.file-list-item .file-icon {
  margin-right: 8px;
  color: #909399;
  font-size: 16px;
}

.file-list-item .file-name {
  flex: 1;
  font-size: 13px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 12px;
}

.file-list-item .file-size {
  font-size: 12px;
  color: #909399;
  min-width: 60px;
  text-align: right;
}

.upload-progress-section {
  margin-top: 20px;
  padding: 15px;
  background: #f9fafb;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}
</style>