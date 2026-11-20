<template>
  <div class="oss-content ct-root kam-cos csp-root">
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      title="仅限用于自身业务测试，杜绝非法操作，对此产生的任何问题与作者无关"
    />
    <div class="kam-asset-panel csp-kam-panel">
          <div class="kam-asset-toolbar">
            <div class="kam-asset-toolbar-main">
              <div class="kam-asset-tab is-active">
                <el-icon class="kam-asset-tab-ico"><Monitor /></el-icon>
                <span>资产列表</span>
              </div>
              <el-dropdown trigger="click" class="kam-filter-dd" @command="onCspFilterVendor">
                <el-button size="small" class="kam-filter-btn">
                  {{ cspFilterLabel }}
                  <el-icon class="kam-dd-ico"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="all">全部厂商</el-dropdown-item>
                    <el-dropdown-item command="aliyun" divided>阿里云</el-dropdown-item>
                    <el-dropdown-item command="huawei">华为云</el-dropdown-item>
                    <el-dropdown-item command="ucloud">UCloud</el-dropdown-item>
                    <el-dropdown-item command="tencent">腾讯云</el-dropdown-item>
                    <el-dropdown-item command="aws">AWS</el-dropdown-item>
                    <el-dropdown-item command="baidu">百度云</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <div class="kam-vendor-scroll csp-vendor-strip">
                <button type="button" class="kam-vendor-chip" @click="openVendorDialog('aliyun')">
                  <span class="kam-vendor-logo is-aliyun">阿</span>
                  <span class="kam-vendor-name">阿里云</span>
                </button>
                <button type="button" class="kam-vendor-chip" @click="openVendorDialog('huawei')">
                  <span class="kam-vendor-logo is-huawei">华</span>
                  <span class="kam-vendor-name">华为云</span>
                </button>
                <button type="button" class="kam-vendor-chip" @click="openVendorDialog('ucloud')">
                  <span class="kam-vendor-logo is-ucloud">U</span>
                  <span class="kam-vendor-name">UCloud</span>
                </button>
                <button type="button" class="kam-vendor-chip" @click="openVendorDialog('tencent')">
                  <span class="kam-vendor-logo is-tencent">腾</span>
                  <span class="kam-vendor-name">腾讯云</span>
                </button>
                <button type="button" class="kam-vendor-chip" @click="openVendorDialog('aws')">
                  <span class="kam-vendor-logo is-aws">A</span>
                  <span class="kam-vendor-name">AWS</span>
                </button>
                <button type="button" class="kam-vendor-chip" @click="openVendorDialog('baidu')">
                  <span class="kam-vendor-logo is-baidu">度</span>
                  <span class="kam-vendor-name">百度云</span>
                </button>
              </div>
            </div>
            <div class="kam-asset-actions">
              <el-button type="primary" size="small" @click="openVendorDialog()">
                <el-icon class="csp-btn-ico"><Plus /></el-icon>
                添加凭据
              </el-button>
              <el-button size="small" @click="showSecrets = !showSecrets">
                {{ showSecrets ? '隐藏密钥' : '显示密钥' }}
              </el-button>
              <el-button
                type="primary"
                size="small"
                plain
                class="kam-btn-export"
                :loading="refreshAllLoading"
                @click="refreshAllInstances"
              >
                刷新全部实例
              </el-button>
            </div>
          </div>
          <div class="kam-asset-search-row csp-search-row">
            <el-input
              v-model="searchKw"
              class="kam-asset-search csp-search-input"
              size="small"
              clearable
              placeholder="搜索别名、备注…"
              @keyup.enter="applyFilter"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            <el-button type="primary" size="small" @click="applyFilter">搜索</el-button>
          </div>
          <div class="kam-asset-table-sizer csp-table-sizer">
            <div class="kam-asset-table-scroll">
              <div class="kam-asset-table-main">
                <el-table
                class="kam-asset-table cos-asset-table"
                :data="pagedAccounts"
                border
                stripe
                size="small"
                height="100%"
                row-key="id"
                :expand-row-keys="expandedKeys"
                empty-text="暂无凭据，请点击「添加凭据」或上方厂商快捷入口新建 AK/SK"
                @expand-change="onExpandChange"
              >
            <el-table-column type="expand">
              <template #default="props">
                <div v-for="reg in props.row.regions" :key="reg.regionId" class="cst-region-block">
                  <div class="cst-region-title">区域名称：{{ reg.regionName }}</div>
                  <el-table :data="reg.instances" size="small" border class="cst-inner-table">
                    <el-table-column prop="instanceId" label="实例 id" min-width="140" show-overflow-tooltip />
                    <el-table-column prop="hostName" label="主机名称" min-width="120" />
                    <el-table-column prop="publicIp" label="公网 ip" width="120" />
                    <el-table-column prop="config" label="主机配置" min-width="100" show-overflow-tooltip />
                    <el-table-column prop="os" label="系统版本" min-width="160" show-overflow-tooltip />
                    <el-table-column prop="status" label="运行状态" width="90" />
                    <el-table-column prop="createdAt" label="创建时间" width="150" />
                    <el-table-column prop="expiredAt" label="结束时间" width="150" />
                    <el-table-column label="操作" width="100" fixed="right">
                      <template #default="scope">
                        <el-button type="primary" link size="small" @click="showInstanceDetail(props.row, reg, scope.row)">
                          详细信息
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </template>
            </el-table-column>
            <el-table-column type="index" label="No" width="50" />
            <el-table-column prop="alias" label="别名" min-width="110" />
            <el-table-column label="主机数量" width="90" sortable :sort-method="sortHostCount">
              <template #default="{ row }">{{ hostCount(row) }}</template>
            </el-table-column>
            <el-table-column label="归属" width="100" sortable :sort-method="sortVendor">
              <template #default="{ row }">{{ vendorLabel(row.vendor) }}</template>
            </el-table-column>
            <el-table-column label="accessKey" min-width="120">
              <template #default="{ row }">{{ maskSecret(row.accessKey, showSecrets) }}</template>
            </el-table-column>
            <el-table-column label="secretKey" min-width="120">
              <template #default="{ row }">{{ maskSecret(row.secretKey, showSecrets) }}</template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" min-width="80" show-overflow-tooltip />
            <el-table-column label="同步" width="82" fixed="right" align="center">
              <template #default="{ row }">
                <el-button
                  type="primary"
                  link
                  size="small"
                  :loading="!!instancesLoading[row.id]"
                  @click="refreshInstances(row)"
                >
                  刷新实例
                </el-button>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="320" fixed="right">
              <template #default="{ row }">
                <template v-if="row.vendor === 'aliyun'">
                  <el-button type="primary" link size="small" @click="openVendorDialogEdit(row)">编辑</el-button>
                  <el-button type="primary" link size="small" class="cst-link-db" @click="openRds(row)">云数据库管理</el-button>
                  <el-button type="success" link size="small" class="cst-link-alarm" @click="openAlarm(row)">告警管理</el-button>
                  <el-button type="warning" link size="small" class="cst-link-warn" @click="genericTakeover(row)">接管</el-button>
                  <el-button type="primary" link size="small" @click="openTakeoverRecords(row)">操作记录</el-button>
                  <el-button type="primary" link size="small" @click="viewAccount(row)">查看</el-button>
                  <el-button type="danger" link size="small" @click="removeAccount(row)">删除</el-button>
                </template>
                <template v-else-if="row.vendor === 'huawei'">
                  <el-button type="primary" link size="small" @click="openVendorDialogEdit(row)">编辑</el-button>
                  <el-button type="warning" link size="small" class="cst-link-warn" @click="huaweiTakeover(row)">接管</el-button>
                  <el-button type="primary" link size="small" @click="openTakeoverRecords(row)">操作记录</el-button>
                  <el-button type="danger" link size="small" @click="removeAccount(row)">删除</el-button>
                </template>
                <template v-else-if="row.vendor === 'ucloud'">
                  <el-button type="primary" link size="small" @click="openVendorDialogEdit(row)">编辑</el-button>
                  <el-button type="warning" link size="small" class="cst-link-warn" @click="ucloudTakeover(row)">接管</el-button>
                  <el-button type="primary" link size="small" @click="openUcloudOrder(row)">订单查看</el-button>
                  <el-button type="primary" link size="small" @click="openTakeoverRecords(row)">操作记录</el-button>
                  <el-button type="danger" link size="small" @click="removeAccount(row)">删除</el-button>
                </template>
                <template v-else-if="row.vendor === 'tencent' || row.vendor === 'aws'">
                  <el-button type="primary" link size="small" @click="openVendorDialogEdit(row)">编辑</el-button>
                  <el-button type="primary" link size="small" @click="openTakeoverRecords(row)">操作记录</el-button>
                  <el-button type="primary" link size="small" @click="viewAccount(row)">查看</el-button>
                  <el-button type="danger" link size="small" @click="removeAccount(row)">删除</el-button>
                </template>
                <template v-else-if="row.vendor === 'baidu'">
                  <el-button type="primary" link size="small" @click="openVendorDialogEdit(row)">编辑</el-button>
                  <el-button type="primary" link size="small" @click="openTakeoverRecords(row)">操作记录</el-button>
                  <el-button type="primary" link size="small" @click="viewAccount(row)">查看</el-button>
                  <el-button type="danger" link size="small" @click="removeAccount(row)">删除</el-button>
                </template>
              </template>
            </el-table-column>
                </el-table>
              </div>
              <div class="kam-asset-pager">
                <el-pagination
                  :current-page="currentPage"
                  :page-size="pageSize"
                  :total="filteredAccounts.length"
                  layout="prev, pager, next, total"
                  small
                  @current-change="onPageChange"
                />
              </div>
            </div>
          </div>

          <div class="csp-cmd-block">
            <el-collapse v-model="cmdCollapseActive" class="csp-cmd-collapse" :border="false">
              <el-collapse-item title="命令执行区" name="cmd">
            <el-form ref="cmdFormRef" :model="cmdForm" :rules="cmdRules" label-width="120px" size="small">
              <el-row :gutter="12">
                <el-col :span="8">
                  <el-form-item label="服务器" prop="instanceKey">
                    <el-select
                      v-model="cmdForm.instanceKey"
                      filterable
                      placeholder="请选择实例"
                      style="width: 100%"
                      @change="onCmdInstanceChange"
                    >
                      <el-option-group v-for="grp in instanceGroups" :key="grp.label" :label="grp.label">
                        <el-option
                          v-for="opt in grp.options"
                          :key="opt.value"
                          :label="opt.label"
                          :value="opt.value"
                        />
                      </el-option-group>
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="系统" prop="systemType">
                    <el-select v-model="cmdForm.systemType" placeholder="请选择" style="width: 100%">
                      <el-option label="Linux Shell Script" value="linux" />
                      <el-option label="Windows CMD" value="win_cmd" />
                      <el-option label="PowerShell" value="powershell" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="终端输出">
                    <el-radio-group v-model="cmdForm.echoYes">
                      <el-radio :label="true">追加（连续会话）</el-radio>
                      <el-radio :label="false">仅本次（清空后只显示当前命令）</el-radio>
                    </el-radio-group>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="12">
                <el-col :span="8">
                  <el-form-item label="等待时间(秒)">
                    <el-input-number v-model="cmdForm.waitSec" :min="1" :max="600" size="small" />
                    <div class="cst-muted-inline">云助手轮询上限，过短会拿不到输出</div>
                  </el-form-item>
                </el-col>
                <el-col :span="10">
                  <el-form-item label="命令" prop="command">
                    <el-input v-model="cmdForm.command" placeholder="命令" clearable />
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label=" ">
                    <el-button type="primary" :loading="cmdRunning" @click="runCommand" style="width: 100%">
                      执行命令
                    </el-button>
                  </el-form-item>
                </el-col>
              </el-row>
            </el-form>
            <div class="cst-cmd-terminal-wrap">
              <div class="cst-cmd-terminal-bar">
                <span class="cst-cmd-terminal-title">终端</span>
                <el-button type="primary" link size="small" @click="clearCommandTerminal">清空</el-button>
              </div>
              <div class="cst-cmd-terminal-outer">
                <pre ref="cmdTerminalEl" class="cst-cmd-terminal" tabindex="0">{{ cmdTerminalText }}</pre>
                <div v-if="!cmdTerminalText" class="cst-cmd-terminal-placeholder">执行命令后输出将显示在此处…</div>
              </div>
            </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>

    <el-dialog
        v-model="vendorDialogVisible"
        :title="vendorFormEditingId ? '编辑云厂商凭据' : '添加云厂商凭据'"
        destroy-on-close
        class="csp-vendor-dlg"
        @closed="onVendorDialogClosed"
      >
        <el-form
          ref="vendorFormRef"
          :model="vendorForm"
          :rules="vendorRules"
          label-width="120px"
          class="cst-vendor-form"
        >
          <el-form-item label="云厂商" prop="vendor">
            <el-select v-model="vendorForm.vendor" placeholder="请选择" style="width: 100%; max-width: 320px">
              <el-option label="阿里云" value="aliyun" />
              <el-option label="华为云" value="huawei" />
              <el-option label="UCloud" value="ucloud" />
              <el-option label="腾讯云" value="tencent" />
              <el-option label="AWS" value="aws" />
              <el-option label="百度云" value="baidu" />
            </el-select>
          </el-form-item>
          <el-form-item label="别名" prop="alias">
            <el-input v-model="vendorForm.alias" placeholder="用于区分多套凭据" style="width: 100%; max-width: 320px" />
          </el-form-item>
          <el-form-item label="ACCESS_KEY" prop="accessKey">
            <el-input v-model="vendorForm.accessKey" placeholder="请填入你的ACCESS_KEY" style="width: 100%; max-width: 480px" />
          </el-form-item>
          <el-form-item label="SECRET_KEY" prop="secretKey">
            <el-input
              v-model="vendorForm.secretKey"
              type="password"
              show-password
              placeholder="请填入你的SECRET_KEY"
              style="width: 100%; max-width: 480px"
            />
          </el-form-item>
          <el-form-item label="备注">
            <el-input
              v-model="vendorForm.remark"
              type="textarea"
              :rows="3"
              maxlength="100"
              show-word-limit
              placeholder="备注"
              style="width: 100%; max-width: 480px"
            />
          </el-form-item>
          <el-form-item label="扩展 JSON">
            <el-input
              v-model="vendorForm.extraJson"
              type="textarea"
              :rows="4"
              placeholder='可选。regions 地域 JSON 数组；UCloud 强烈建议填 ucloud_project_id（控制台项目 ID），未填时将自动 GetProjectList 并多次地域查询。刷新实例会拉取 UHost、轻量 ULHost、物理云 PHost（与 https://docs.ucloud.cn/api 计算类产品对齐）；百度云使用 BCC AK/SK，轻量应用服务器 LS 请使用同一账号，在 BCC 控制台与 LS 控制台均可看到轻量主机详情（API 以 BCC 为准，详见 https://cloud.baidu.com/doc/BCC/s/Kjwvyofgz ）'
              style="width: 100%"
            />
          </el-form-item>
        </el-form>
        <p class="cst-hint">
          阿里云支持云助手 RunCommand、RDS 列举、云安全中心告警与通知；UCloud
          订单查询仅 UCloud。命令执行：阿里云云助手、腾讯云 TAT、AWS SSM、华为云 COC（默认自动识别项目ID，无需手填扩展 JSON；识别失败时可补 huawei_project_id，实例需已安装 UniAgent 并具备执行权限）、百度云 OOS/云助手（默认按 region 自动选择 endpoint，无需手填扩展 JSON）。接管：阿里云 RAM、华为 IAM（需 IAM 建用户权限）、UCloud CreateUser；腾讯云/AWS 请用控制台 CAM/IAM。
        </p>
        <template #footer>
          <el-button @click="vendorDialogVisible = false">取 消</el-button>
          <el-button type="primary" :loading="vendorSaving" @click="saveVendor">保存并查找</el-button>
        </template>
      </el-dialog>

      <el-dialog
        v-model="takeoverVisible"
        title="接管成功"
        width="480px"
        destroy-on-close
        @closed="takeoverPayload = null"
      >
        <el-alert
          v-if="takeoverPayload"
          title="接管成功，正在保存接管信息"
          type="success"
          :closable="false"
          style="margin-bottom: 12px"
        />
        <el-alert
          v-if="takeoverPayload?.isDemo"
          type="warning"
          :closable="false"
          title="以下为前端占位演示数据，未调用华为云 API 创建 IAM 用户，无法在控制台登录。"
          description="真实登录请使用华为账号（手机号/邮箱）及密码，或由主账号在 IAM 中创建可登录控制台的子用户并授予权限；AccessKey 不能直接作为控制台用户名密码使用。"
          style="margin-bottom: 12px"
        />
        <template v-if="takeoverPayload && !takeoverPayload.isDemo">
          <p v-if="takeoverPayload.console_login_name">
            <strong>控制台登录名</strong>：{{ takeoverPayload.console_login_name }}
          </p>
          <p v-if="takeoverPayload.username"><strong>用户名</strong>：{{ takeoverPayload.username }}</p>
          <p v-if="takeoverPayload.password"><strong>密码</strong>：{{ takeoverPayload.password }}</p>
          <p v-if="takeoverPayload.vendor === 'aliyun'">
            <strong>系统策略</strong>：{{ takeoverPayload.policy_attached || '未附加' }}
          </p>
          <p v-if="takeoverPayload.account_alias"><strong>账号别名</strong>：{{ takeoverPayload.account_alias }}</p>
          <p v-if="takeoverPayload.account_id"><strong>账号 ID</strong>：{{ takeoverPayload.account_id }}</p>
          <p v-if="takeoverPayload.sts_error" class="cst-muted-inline">STS：{{ takeoverPayload.sts_error }}</p>
          <p v-if="takeoverPayload.domain_name"><strong>华为账号名</strong>：{{ takeoverPayload.domain_name }}</p>
          <p v-if="takeoverPayload.domain_id"><strong>domain_id</strong>：{{ takeoverPayload.domain_id }}</p>
          <el-alert
            v-if="takeoverPayload.iam_login_hint"
            type="info"
            :closable="false"
            :title="String(takeoverPayload.iam_login_hint)"
            style="margin: 8px 0"
          />
          <el-alert
            v-if="takeoverPayload.permission_notice"
            type="warning"
            :closable="false"
            :title="String(takeoverPayload.permission_notice)"
            style="margin-bottom: 8px"
          />
          <p v-if="takeoverLoginUrl(takeoverPayload)"><strong>登录地址</strong>：</p>
          <el-input
            v-if="takeoverLoginUrl(takeoverPayload)"
            :model-value="takeoverLoginUrl(takeoverPayload)"
            readonly
            type="textarea"
            :rows="2"
          />
        </template>
        <template v-else-if="takeoverPayload">
          <p><strong>用户名</strong>：{{ takeoverPayload.username }}</p>
          <p><strong>密码</strong>：{{ takeoverPayload.password }}</p>
          <p><strong>登录地址</strong>：</p>
          <el-input :model-value="takeoverLoginUrl(takeoverPayload)" readonly type="textarea" :rows="2" />
        </template>
        <template #footer>
          <el-button @click="takeoverVisible = false">取 消</el-button>
          <el-button type="primary" @click="takeoverVisible = false">确 定</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="ucloudUserVisible" title="UCloud 新建子用户（API）" width="560px" destroy-on-close @closed="resetUcloudUser">
        <el-alert
          type="success"
          :closable="false"
          title="已通过 CreateUser 创建 UCloud 子账号；请前往该子账号邮箱查收激活邮件，按邮件链接完成账号激活及控制台密码设置。下方为子账号 API 密钥。"
        />
        <el-alert type="info" :closable="false" title="请立即复制保存 PrivateKey（Secret）；关闭窗口后可在下方「备忘」再写一条本地记录。" style="margin-top: 8px" />
        <el-form v-if="ucloudUserDraft" label-width="108px" size="small" style="margin-top: 12px">
          <el-form-item label="账号"><span>{{ ucloudUserDraft.account }}</span></el-form-item>
          <el-form-item label="邮箱"><span>{{ ucloudUserDraft.email }}</span></el-form-item>
          <el-form-item label="登录地址"><span>{{ ucloudUserDraft.loginUrl }}</span></el-form-item>
          <el-form-item label="API Public"><code>{{ ucloudUserDraft.publicKey }}</code></el-form-item>
          <el-form-item label="API Private"><code>{{ ucloudUserDraft.privateKey }}</code></el-form-item>
          <el-form-item label="备忘">
            <el-input v-model="ucloudUserPassword" placeholder="可选：仅在本机记录一条备忘说明（如初始密码/备注）" show-password />
            <span class="cst-muted-inline">仅写入本机「操作记录」，不参与云端。</span>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="ucloudUserVisible = false">取 消</el-button>
          <el-button type="primary" @click="saveUcloudUserNote">保 存</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="ucloudOrderVisible" title="ucloud订单查看" width="900px" top="5vh" destroy-on-close>
        <div class="cst-order-bar">
          <el-select v-model="ucloudOrderFilter.status" size="small" style="width: 140px">
            <el-option label="已完成" value="done" />
            <el-option label="进行中" value="active" />
          </el-select>
          <el-date-picker
            v-model="ucloudOrderFilter.range"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始"
            end-placeholder="结束"
            size="small"
            style="margin-left: 8px; width: 360px"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
          <el-button type="primary" size="small" style="margin-left: 8px" :loading="ucloudOrderLoading" @click="fetchUcloudOrders">
            <el-icon style="vertical-align: middle; margin-right: 4px"><Search /></el-icon>
            查询
          </el-button>
        </div>
        <el-table v-loading="ucloudOrderLoading" :data="ucloudOrderRows" border size="small" max-height="420">
          <el-table-column prop="orderId" label="订单编号" min-width="120" />
          <el-table-column prop="resourceId" label="资源 id" min-width="120" />
          <el-table-column prop="resourceType" label="资源类型" width="90" />
          <el-table-column prop="start" label="开始时间" width="155" />
          <el-table-column prop="end" label="结束时间" width="155" />
          <el-table-column prop="product" label="产品类型" min-width="200" show-overflow-tooltip />
          <el-table-column prop="billing" label="计费方式" width="90" />
          <el-table-column prop="amount" label="金额" width="80" />
          <el-table-column prop="orderState" label="订单状态" width="100" show-overflow-tooltip />
        </el-table>
        <template #footer>
          <el-button @click="ucloudOrderVisible = false">取 消</el-button>
          <el-button type="primary" @click="ucloudOrderVisible = false">确 定</el-button>
        </template>
      </el-dialog>

      <el-dialog
        v-model="rdsVisible"
        :title="'云数据库 - ' + (rdsAccount && rdsAccount.alias) + ' - 阿里云'"
        width="960px"
        top="4vh"
        class="cst-rds-dlg"
        destroy-on-close
      >
        <el-table v-loading="rdsLoading" :data="rdsRegionRows" border size="small" row-key="regionId" :expand-row-keys="rdsExpanded">
          <el-table-column type="expand">
            <template #default="props">
              <div class="cst-region-title">区域名称：{{ props.row.regionName }}</div>
              <el-table :data="props.row.instances" border size="small" class="cst-inner-table">
                <el-table-column prop="engine" label="数据库类型" width="90" />
                <el-table-column prop="version" label="数据库版本" width="90" />
                <el-table-column prop="conn" label="连接地址" min-width="160" show-overflow-tooltip />
                <el-table-column prop="status" label="状态" width="80" />
                <el-table-column prop="mode" label="访问模式" width="100" />
                <el-table-column prop="created" label="创建日期" width="110" />
                <el-table-column prop="expire" label="到期时间" width="110" />
                <el-table-column label="锁定状态" min-width="128">
                  <template #default="{ row: ins }">
                    <div class="cst-rds-lock-cell">
                      <span
                        class="cst-rds-lock-text"
                        :class="{ 'cst-rds-lock-warn': !isRdsUnlock(ins) }"
                        :title="ins.lockReason || ''"
                      >{{ rdsLockModeLabel(ins.lockMode) }}</span>
                      <el-button
                        type="primary"
                        link
                        size="small"
                        :loading="rdsOpBusy(ins, 'attr')"
                        @click="openRdsLockInfo(rdsAccount, props.row, ins)"
                      >详情</el-button>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="操作" min-width="520">
                  <template #default="{ row: ins }">
                    <div class="cst-rds-ops">
                      <div><b>实例信息</b></div>
                      <div>
                        <el-button size="small" type="warning" plain :loading="rdsOpBusy(ins, 'acc_add')" @click="rdsAddAccount(rdsAccount, props.row, ins)">添加账号</el-button>
                        <el-button size="small" type="danger" plain :loading="rdsOpBusy(ins, 'acc_del')" @click="rdsDeleteAccount(rdsAccount, props.row, ins)">删除账号</el-button>
                        <el-button type="primary" link size="small" :loading="rdsOpBusy(ins, 'acc_list')" @click="rdsListAccounts(rdsAccount, props.row, ins)">账号列表</el-button>
                        <el-button type="primary" link size="small" :loading="rdsOpBusy(ins, 'acc_rec')" @click="rdsRecordAccounts(rdsAccount, props.row, ins)">记录</el-button>
                      </div>
                      <div>
                        <el-button size="small" type="warning" plain :loading="rdsOpBusy(ins, 'pub_alloc')" @click="rdsAllocatePublic(rdsAccount, props.row, ins)">申请外网地址</el-button>
                        <el-button size="small" type="danger" plain :loading="rdsOpBusy(ins, 'pub_rel')" @click="rdsReleasePublic(rdsAccount, props.row, ins)">删除外网地址</el-button>
                        <el-button type="primary" link size="small" :loading="rdsOpBusy(ins, 'net_list')" @click="rdsListPublicNet(rdsAccount, props.row, ins)">外网地址列表</el-button>
                        <el-button type="primary" link size="small" :loading="rdsOpBusy(ins, 'net_rec')" @click="rdsRecordPublicNet(rdsAccount, props.row, ins)">记录</el-button>
                      </div>
                      <div>
                        <el-button size="small" type="warning" plain :loading="rdsOpBusy(ins, 'wl_add')" @click="rdsAddWhitelist(rdsAccount, props.row, ins)">添加白名单</el-button>
                        <el-button size="small" type="danger" plain :loading="rdsOpBusy(ins, 'wl_del')" @click="rdsDeleteWhitelist(rdsAccount, props.row, ins)">删除白名单</el-button>
                        <el-button type="primary" link size="small" :loading="rdsOpBusy(ins, 'wl_list')" @click="rdsListWhitelist(rdsAccount, props.row, ins)">白名单列表</el-button>
                        <el-button type="primary" link size="small" :loading="rdsOpBusy(ins, 'wl_rec')" @click="rdsRecordWhitelist(rdsAccount, props.row, ins)">记录</el-button>
                      </div>
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </template>
          </el-table-column>
          <el-table-column prop="alias" label="别名" min-width="100" />
          <el-table-column prop="count" label="资产数量" width="90" />
          <el-table-column prop="vendor" label="归属" width="90" />
          <el-table-column prop="zoneId" label="可用区 id" width="100" />
          <el-table-column prop="zoneName" label="可用区名称" min-width="120" />
          <el-table-column prop="regionId" label="区域 id" width="100" />
        </el-table>
        <template #footer>
          <el-button @click="rdsVisible = false">取 消</el-button>
          <el-button type="primary" @click="rdsVisible = false">确 定</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="rdsDetailVisible" :title="rdsDetailTitle" width="720px" top="6vh" destroy-on-close class="cst-rds-dlg" append-to-body>
        <pre class="cst-json" style="max-height: 56vh; overflow: auto; margin: 0">{{ rdsDetailJson }}</pre>
        <template #footer>
          <el-button type="primary" @click="rdsDetailVisible = false">关 闭</el-button>
        </template>
      </el-dialog>

      <el-dialog
        v-model="alarmVisible"
        :title="'告警管理 - ' + (alarmAccount && alarmAccount.alias) + ' - 阿里云'"
        width="920px"
        top="4vh"
        destroy-on-close
        @closed="alarmHandleRow = null"
      >
        <el-tabs v-model="alarmSubTab">
          <el-tab-pane label="告警处理" name="deal">
            <div class="cst-alarm-bar">
              <el-select v-model="alarmStatusFilter" clearable placeholder="处理状态" size="small" style="width: 140px">
                <el-option label="待处理" value="待处理" />
                <el-option label="处理完毕" value="处理完毕" />
                <el-option label="已忽略" value="已忽略" />
              </el-select>
            </div>
            <el-table v-loading="securityLoading" :data="filteredAlarms" border size="small" max-height="380">
              <el-table-column prop="machine" label="机器名称" width="100" />
              <el-table-column prop="name" label="事件名称" min-width="160" show-overflow-tooltip />
              <el-table-column prop="type" label="事件类型" width="130" />
              <el-table-column prop="level" label="告警等级" width="90">
                <template #default="{ row }">
                  <span :class="alarmLevelClass(row.level)">{{ row.level }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="detail" label="告警详情" min-width="200" show-overflow-tooltip />
              <el-table-column prop="status" label="处理状态" width="100" />
              <el-table-column prop="time" label="发生时间" width="155" />
              <el-table-column label="操作" width="200" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="openAlarmHandle(row)">告警处理</el-button>
                  <el-button type="primary" link size="small" @click="showAlarmDetail(row)">告警事件详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="告警通知配置" name="notify">
            <el-table v-loading="securityLoading" :data="alarmNotifyRows" border size="small" max-height="380">
              <el-table-column prop="code" label="项目标识(code)" min-width="160" show-overflow-tooltip />
              <el-table-column prop="title" label="通知名称" min-width="160" />
              <el-table-column prop="method" label="通知方式" min-width="200" />
              <el-table-column prop="limit" label="通知时间限制" width="200" />
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="openNotifyEdit(row)">通知修改</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
        <template #footer>
          <el-button @click="alarmVisible = false">关 闭</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="alarmHandleVisible" title="处理告警" width="420px" append-to-body destroy-on-close>
        <el-form label-width="100px" size="small">
          <el-form-item label="处理方式">
            <el-select v-model="alarmHandleAction" filterable placeholder="请选择" style="width: 100%">
              <el-option label="阻断" value="block" />
              <el-option label="加白名单" value="whitelist" />
              <el-option label="忽略" value="ignore" />
              <el-option label="我已手工处理" value="manual" />
              <el-option label="结束进程" value="kill" />
              <el-option label="深度查杀" value="deep" />
              <el-option label="病毒查杀" value="virus" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="alarmHandleVisible = false">取 消</el-button>
          <el-button type="primary" @click="confirmAlarmHandle">确 定</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="notifyEditVisible" title="告警通知修改" width="440px" append-to-body destroy-on-close>
        <el-form v-if="notifyEditRow" label-width="120px" size="small">
          <el-form-item label="项目名称"><span>{{ notifyEditRow.code }}</span></el-form-item>
          <el-form-item label="当前策略"><span class="cst-muted-inline">{{ notifyEditRow.method }}</span></el-form-item>
          <el-alert type="info" :closable="false" show-icon title="通知通道（Route）沿用云端当前值，此处仅同步修改「通知时间限制」。" style="margin-bottom: 12px" />
          <el-form-item label="通知时间限制">
            <el-input-number v-model="notifyTimeLimit" :min="0" :max="1" />
            <div class="cst-muted-inline">0：无限制。1：08:00-22:00 内发送。</div>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="notifyEditVisible = false">取 消</el-button>
          <el-button type="primary" @click="saveNotifyEdit">确 定</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="recordsVisible" title="操作记录" width="640px" top="6vh" destroy-on-close class="cst-takeover-records-dlg">
        <template v-if="takeoverRecords.length">
          <div class="cst-takeover-records-meta">共 {{ takeoverRecords.length }} 条</div>
          <div class="cst-takeover-records-scroll">
            <el-timeline>
              <el-timeline-item v-for="(r, i) in takeoverRecords" :key="i" :timestamp="r.at">
                {{ r.text }}
                <pre v-if="r.extra" class="cst-echo-out cst-takeover-records-pre">{{ r.extra }}</pre>
              </el-timeline-item>
            </el-timeline>
          </div>
        </template>
        <el-empty v-else description="暂无记录" />
        <template #footer>
          <el-button type="primary" @click="recordsVisible = false">关 闭</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="viewVisible" title="账号信息" width="520px" destroy-on-close>
        <pre class="cst-json">{{ viewJson }}</pre>
        <template #footer>
          <el-button type="primary" @click="viewVisible = false">关 闭</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="instanceDetailVisible" title="主机详细信息" width="720px" top="6vh" destroy-on-close>
        <pre class="cst-json">{{ instanceDetailJson }}</pre>
        <template #footer>
          <el-button type="primary" @click="instanceDetailVisible = false">关 闭</el-button>
        </template>
      </el-dialog>
  </div>
</template>

<script>
import { Search, Monitor, ArrowDown, Plus } from '@element-plus/icons-vue';
import { ElLoading, ElMessage, ElMessageBox } from 'element-plus';
import {
  CloudServerCredentialsJSON,
  CloudServerSaveCredentialJSON,
  CloudServerDeleteCredential,
  CloudServerListInstancesJSON,
  CloudServerRunCommandJSON,
  CloudServerListRDSJSON,
  CloudServerAliyunRDSActionJSON,
  CloudServerListAliyunSecurityJSON,
  CloudServerAliyunHandleSecurityEventJSON,
  CloudServerAliyunSaveNoticeJSON,
  CloudServerListUcloudOrdersJSON,
  CloudServerTakeoverConsoleJSON
} from '../../../wailsjs/go/cloudSecurity/CloudServer';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import {
  CLOUD_SERVER_TERMINAL_SESSION_KEY as STORAGE_TERMINAL,
  CLOUD_SERVER_ECHO_LOCAL_KEY as STORAGE_ECHO,
  CLOUD_SERVER_ECHO_LEGACY_KEY
} from '@/utils/cloudServerTerminalStorage';

const STORAGE_RECORDS = 'easytools:cloud_server:takeover_records_v1';
const LEGACY = {
  echo: CLOUD_SERVER_ECHO_LEGACY_KEY,
  records: 'king_cloud_server_takeover_records_v1'
};

const TERMINAL_MAX_CHARS = 150000;

function migrateLegacy(key, legacyKey) {
  try {
    if (!localStorage.getItem(key) && localStorage.getItem(legacyKey)) {
      localStorage.setItem(key, localStorage.getItem(legacyKey));
    }
  } catch (e) {
    /* ignore */
  }
}

export default {
  name: 'CloudServerPanel',
  components: { Search, Monitor, ArrowDown, Plus },
  data() {
    return {
      vendorDialogVisible: false,
      cmdCollapseActive: [],
      listVendorFilter: '',
      searchKw: '',
      searchApplied: '',
      showSecrets: false,
      expandedKeys: [],
      currentPage: 1,
      pageSize: 10,
      accounts: [],
      vendorSaving: false,
      vendorFormEditingId: 0,
      vendorForm: {
        vendor: 'aliyun',
        alias: '',
        accessKey: '',
        secretKey: '',
        remark: '',
        extraJson: '{}'
      },
      refreshAllLoading: false,
      rdsLoading: false,
      securityLoading: false,
      ucloudOrderLoading: false,
      ucloudOrderAccId: null,
      instancesLoading: {},
      vendorRules: {
        vendor: [{ required: true, message: '请选择云厂商', trigger: 'change' }],
        alias: [{ required: true, message: '请输入别名', trigger: 'blur' }],
        accessKey: [{ required: true, message: '请输入 ACCESS_KEY', trigger: 'blur' }],
        secretKey: [{ required: true, message: '请输入 SECRET_KEY', trigger: 'blur' }]
      },
      cmdForm: {
        instanceKey: '',
        systemType: 'linux',
        command: '',
        echoYes: true,
        waitSec: 60
      },
      cmdRules: {
        instanceKey: [{ required: true, message: '请选择服务器', trigger: 'change' }],
        systemType: [{ required: true, message: '请选择系统', trigger: 'change' }],
        command: [{ required: true, message: '请输入命令', trigger: 'blur' }]
      },
      cmdRunning: false,
      // 每个实例独立的终端输出：键为 instanceKey（accountId|regionId|instanceId）
      cmdTerminalMap: {},
      // 当前展示的终端文本（与模板绑定）
      cmdTerminalText: '',
      takeoverVisible: false,
      takeoverPayload: null,
      ucloudUserVisible: false,
      ucloudUserDraft: null,
      ucloudUserPassword: '',
      ucloudOrderVisible: false,
      ucloudOrderFilter: { status: 'done', range: [] },
      ucloudOrderRows: [],
      rdsVisible: false,
      rdsAccount: null,
      rdsRegionRows: [],
      rdsExpanded: [],
      rdsActionBusyKey: null,
      rdsDetailVisible: false,
      rdsDetailTitle: '',
      rdsDetailJson: '',
      alarmVisible: false,
      alarmAccount: null,
      alarmSubTab: 'deal',
      alarmRows: [],
      alarmStatusFilter: '',
      alarmHandleVisible: false,
      alarmHandleRow: null,
      alarmHandleAction: 'manual',
      alarmNotifyRows: [],
      notifyEditVisible: false,
      notifyEditRow: null,
      notifyTimeLimit: 0,
      recordsVisible: false,
      takeoverRecords: [],
      viewVisible: false,
      viewJson: '',
      instanceDetailVisible: false,
      instanceDetailJson: '',
      ucloudTakeoverAccId: null
    };
  },
  computed: {
    filteredAccounts() {
      let list = this.accounts;
      if (this.listVendorFilter) {
        list = list.filter((a) => a.vendor === this.listVendorFilter);
      }
      const kw = (this.searchApplied || '').trim().toLowerCase();
      if (kw) {
        list = list.filter(
          (a) =>
            (a.alias && a.alias.toLowerCase().includes(kw)) ||
            (a.remark && String(a.remark).toLowerCase().includes(kw))
        );
      }
      return list;
    },
    pagedAccounts() {
      const start = (this.currentPage - 1) * this.pageSize;
      const end = start + this.pageSize;
      return this.filteredAccounts.slice(start, end);
    },
    cspFilterLabel() {
      if (!this.listVendorFilter) return '全部厂商';
      return this.vendorLabel(this.listVendorFilter);
    },
    instanceGroups() {
      const groups = [];
      for (const acc of this.accounts) {
        const opts = [];
        for (const reg of acc.regions || []) {
          for (const ins of reg.instances || []) {
            opts.push({
              value: `${acc.id}|${reg.regionId}|${ins.instanceId}`,
              label: `${acc.alias} / ${reg.regionName} / ${ins.hostName || ins.instanceId} (${ins.publicIp || ins.privateIp || '-'})`
            });
          }
        }
        if (opts.length) {
          groups.push({ label: this.vendorLabel(acc.vendor) + ' · ' + acc.alias, options: opts });
        }
      }
      return groups;
    },
    filteredAlarms() {
      if (!this.alarmStatusFilter) return this.alarmRows;
      return this.alarmRows.filter((r) => r.status === this.alarmStatusFilter);
    }
  },
  mounted() {
    migrateLegacy(STORAGE_ECHO, LEGACY.echo);
    migrateLegacy(STORAGE_RECORDS, LEGACY.records);
    this.loadTerminalFromStorage();
    this.loadAll();
  },
  methods: {
    ensureCloudServer() {
      return typeof window !== 'undefined' && window.go?.cloudSecurity?.CloudServer;
    },
    applyFilter() {
      this.searchApplied = this.searchKw;
      this.currentPage = 1;
    },
    onCspFilterVendor(command) {
      this.listVendorFilter = command === 'all' ? '' : command;
      this.applyFilter();
    },
    openVendorDialog(presetVendor) {
      this.vendorFormEditingId = 0;
      this.vendorDialogVisible = true;
      this.$nextTick(() => {
        Object.assign(this.vendorForm, {
          vendor: presetVendor || 'aliyun',
          alias: '',
          accessKey: '',
          secretKey: '',
          remark: '',
          extraJson: '{}'
        });
        this.$refs.vendorFormRef?.clearValidate?.();
      });
    },
    openVendorDialogEdit(row) {
      if (!row?.id) return;
      this.vendorFormEditingId = row.id;
      this.vendorDialogVisible = true;
      this.$nextTick(() => {
        const ex = row.extra_json != null ? row.extra_json : row.extraJson;
        let extraJson = '{}';
        if (typeof ex === 'string' && ex.trim()) {
          extraJson = ex;
        } else if (ex && typeof ex === 'object') {
          try {
            extraJson = JSON.stringify(ex, null, 2);
          } catch (e) {
            extraJson = '{}';
          }
        }
        Object.assign(this.vendorForm, {
          vendor: row.vendor || 'aliyun',
          alias: row.alias || row.name || '',
          accessKey: row.accessKey || row.access_key || '',
          secretKey: row.secretKey || row.secret_key || '',
          remark: row.remark || '',
          extraJson
        });
        this.$refs.vendorFormRef?.clearValidate?.();
      });
    },
    onVendorDialogClosed() {
      this.vendorFormEditingId = 0;
      Object.assign(this.vendorForm, {
        vendor: 'aliyun',
        alias: '',
        accessKey: '',
        secretKey: '',
        remark: '',
        extraJson: '{}'
      });
    },
    hostCount(row) {
      return (row.regions || []).reduce((n, r) => n + (r.instances || []).length, 0);
    },
    sortHostCount(a, b) {
      return this.hostCount(a) - this.hostCount(b);
    },
    sortVendor(a, b) {
      return this.vendorLabel(a.vendor).localeCompare(this.vendorLabel(b.vendor), 'zh-CN');
    },
    maskSecret(v, show) {
      if (!v) return '—';
      if (show) return v;
      return '********************';
    },
    onExpandChange(row, expandedRows) {
      this.expandedKeys = expandedRows.map((r) => r.id);
      if (expandedRows.some((r) => r.id === row.id)) {
        this.refreshInstances(row);
      }
    },
    loadTerminalFromStorage() {
      try {
        // 终端输出改为会话级存储：优先从 sessionStorage 读取，
        // 若旧版本曾写入 localStorage，则仅用于一次性迁移后不再继续写入。
        const t = typeof sessionStorage !== 'undefined' ? sessionStorage.getItem(STORAGE_TERMINAL) : null;
        if (t != null && t !== '') {
          // 新版本：尝试按实例存储的 JSON 结构
          try {
            const parsed = JSON.parse(t);
            if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
              this.cmdTerminalMap = parsed;
              // 初始时根据当前选择的实例刷新显示
              const key = this.cmdForm.instanceKey;
              this.cmdTerminalText = (key && this.cmdTerminalMap[key]) || '';
            } else {
              // 旧版本：纯字符串，按单一输出处理
              this.cmdTerminalMap = { __global__: String(t) };
              this.cmdTerminalText = String(t);
            }
          } catch (e) {
            // 解析失败也按旧版本字符串处理
            this.cmdTerminalMap = { __global__: String(t) };
            this.cmdTerminalText = String(t);
          }
          return;
        }
        const leg = localStorage.getItem(STORAGE_ECHO);
        if (!leg) return;
        const arr = JSON.parse(leg);
        if (!Array.isArray(arr) || !arr.length) return;
        const lines = [];
        for (const item of arr) {
          lines.push(this.formatCommandTerminalBlockText(item));
        }
        const legacyText = lines.join('\n').slice(-TERMINAL_MAX_CHARS);
        // 旧版本本地 echo 只能整合为一份全局输出
        this.cmdTerminalMap = { __global__: legacyText };
        this.cmdTerminalText = legacyText;
        this.persistTerminal();
      } catch (e) {
        /* ignore */
      }
    },
    formatCommandTerminalBlockText(item) {
      const cmd = item.command || '';
      const region = item.regionName || '';
      const at = item.at || '';
      const out = [];
      out.push('');
      out.push(`--- ${at}  ${region} ---`);
      out.push(`$ ${cmd}`);
      if (item.stdout) out.push(String(item.stdout));
      if (item.stderr) out.push(`[stderr]\n${String(item.stderr)}`);
      if (item.invokeId) out.push(`# InvokeId: ${item.invokeId}`);
      if (item.hint) out.push(`# ${item.hint}`);
      return out.join('\n');
    },
    persistTerminal() {
      try {
        if (typeof sessionStorage !== 'undefined') {
          const payload = JSON.stringify(this.cmdTerminalMap || {});
          sessionStorage.setItem(STORAGE_TERMINAL, payload);
        }
      } catch (e) {
        /* ignore */
      }
    },
    clearCommandTerminal() {
      const key = this.cmdForm.instanceKey;
      if (key) {
        this.cmdTerminalMap = {
          ...this.cmdTerminalMap,
          [key]: ''
        };
      } else {
        this.cmdTerminalMap = {};
      }
      this.cmdTerminalText = '';
      this.persistTerminal();
    },
    scrollCommandTerminalToBottom() {
      this.$nextTick(() => {
        const el = this.$refs.cmdTerminalEl;
        if (el && typeof el.scrollTop === 'number') {
          el.scrollTop = el.scrollHeight;
        }
      });
    },
    appendCommandTerminal(acc, reg, ins, cmd, o) {
      const stdout = o.stdout != null ? String(o.stdout) : '';
      const stderr = o.stderr != null ? String(o.stderr) : '';
      const invokeId = o.invoke_id || '';
      const at = new Date().toLocaleString('zh-CN');
      const hint =
        !o.error && !stdout && !stderr
          ? '若命令已执行但无输出，多为等待时间不足或云助手未回传 Output，可调大「等待时间」后重试。'
          : '';
      const lines = [];
      lines.push('');
      lines.push(`--- ${at}  ${acc.alias || acc.name || ''} / ${reg.regionName} / ${ins.instanceId} ---`);
      lines.push(`$ ${cmd}`);
      if (o.error) {
        lines.push(`[错误] ${o.error}`);
      } else {
        if (stdout) lines.push(stdout);
        if (stderr) lines.push(`[stderr]\n${stderr}`);
        if (!stdout && !stderr) lines.push(hint || '(无标准输出与错误流)');
      }
      if (invokeId) lines.push(`# InvokeId: ${invokeId}`);
      const chunk = lines.join('\n');
      // 以实例为维度，分别维护终端输出
      const key =
        this.cmdForm.instanceKey ||
        [acc.id, reg.regionId, ins.instanceId].map((v) => String(v || '')).join('|');
      const prev = this.cmdTerminalMap[key] || '';
      const next = this.cmdForm.echoYes
        ? (prev + chunk).slice(-TERMINAL_MAX_CHARS)
        : chunk.slice(-TERMINAL_MAX_CHARS);
      this.cmdTerminalMap = {
        ...this.cmdTerminalMap,
        [key]: next
      };
      // 同步当前展示内容
      this.cmdTerminalText = next;
      this.persistTerminal();
      this.scrollCommandTerminalToBottom();
    },
    onPageChange(page) {
      this.currentPage = page;
    },
    onCmdInstanceChange(val) {
      const key = val;
      this.cmdTerminalText = (key && this.cmdTerminalMap[key]) || '';
      this.$nextTick(() => {
        this.scrollCommandTerminalToBottom();
      });
    },
    async loadAll() {
      if (!this.ensureCloudServer()) {
        ElMessage.error('未找到 Wails 云服务器后端绑定，请在桌面版中打开本页。');
        this.accounts = [];
        return;
      }
      await this.refreshCredentialsFromBackend({ silentAuthFail: true });
    },
    async refreshCredentialsFromBackend(opts = {}) {
      const silentAuth = opts.silentAuthFail === true;
      if (!silentAuth) {
      }
      try {
        const s = await CloudServerCredentialsJSON();
        const arr = JSON.parse(s || '[]');
        this.accounts = (Array.isArray(arr) ? arr : []).map((r) => ({
          ...r,
          alias: r.alias || r.name,
          regions: Array.isArray(r.regions) ? r.regions : []
        }));
        this.currentPage = 1;
      } catch (e) {
        const msg = e?.message || String(e);
        if (silentAuth && (msg.includes('授权') || msg.includes('云上安全'))) {
          this.accounts = [];
        } else {
          ElMessage.error(msg);
          this.accounts = [];
        }
      }
    },
    async refreshInstances(row) {
      if (!row?.id) return;
      if (this.instancesLoading[row.id]) return;
      this.instancesLoading[row.id] = true;
      try {
        const j = await CloudServerListInstancesJSON(row.id);
        const data = JSON.parse(j || '{}');
        row.regions = data.regions || [];
        if (!row.regions.length) {
          ElMessage.warning('未拉到实例（请检查密钥权限、地域列表或扩展 JSON 中的 regions）');
        }
      } catch (e) {
        ElMessage.error(this.briefError(e, '刷新实例失败'));
      } finally {
        this.instancesLoading[row.id] = false;
      }
    },
    async refreshAllInstances() {
      this.refreshAllLoading = true;
      try {
        for (const row of this.accounts) {
          await this.refreshInstances(row);
        }
        ElMessage.success('已尝试刷新全部账号实例');
      } finally {
        this.refreshAllLoading = false;
      }
    },
    loadRecordsFor(accId) {
      try {
        const all = JSON.parse(localStorage.getItem(STORAGE_RECORDS) || '[]');
        this.takeoverRecords = all.filter((r) => String(r.accountId) === String(accId));
      } catch (e) {
        this.takeoverRecords = [];
      }
    },
    pushRecord(accId, text, extra) {
      let all = [];
      try {
        all = JSON.parse(localStorage.getItem(STORAGE_RECORDS) || '[]');
      } catch (e) {
        all = [];
      }
      all.push({
        accountId: accId,
        text,
        extra: extra || '',
        at: new Date().toLocaleString('zh-CN')
      });
      localStorage.setItem(STORAGE_RECORDS, JSON.stringify(all.slice(-200)));
    },
    saveVendor() {
      const form = this.$refs.vendorFormRef;
      if (!form) return;
      form.validate((ok) => {
        if (!ok) return;
        this.runSaveVendor();
      });
    },
    async runSaveVendor() {
      this.vendorSaving = true;
      try {
        const extraRaw = (this.vendorForm.extraJson || '').trim() || '{}';
        try {
          JSON.parse(extraRaw);
        } catch (e) {
          ElMessage.error('扩展 JSON 不是合法 JSON');
          return;
        }
        const payload = {
          id: this.vendorFormEditingId || 0,
          vendor: this.vendorForm.vendor,
          name: this.vendorForm.alias,
          access_key: this.vendorForm.accessKey,
          secret_key: this.vendorForm.secretKey,
          extra_json: extraRaw,
          remark: this.vendorForm.remark || ''
        };
        const ret = await CloudServerSaveCredentialJSON(JSON.stringify(payload));
        const { id } = JSON.parse(ret || '{}');
        ElMessage.success('已保存，正在拉取实例列表…');
        await this.refreshCredentialsFromBackend({ silentAuthFail: true });
        const acc = this.accounts.find((a) => a.id === id);
        if (acc) await this.refreshInstances(acc);
        this.vendorDialogVisible = false;
      } catch (e) {
        ElMessage.error(this.briefError(e, '保存失败'));
      } finally {
        this.vendorSaving = false;
      }
    },
    async removeAccount(row) {
      try {
        await confirmDangerousAction({
          title: '删除凭据',
          message: '将删除该云服务器凭据及其本地保存信息，且不可恢复。',
          expectedInput: DANGER_DELETE_TOKEN
        });
        await CloudServerDeleteCredential(row.id);
        await this.refreshCredentialsFromBackend({ silentAuthFail: true });
        ElMessage.success('已删除');
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(this.briefError(e, '删除凭据失败'));
      }
    },
    takeoverLoginUrl(p) {
      if (!p) return '';
      return p.login_url || p.loginUrl || '';
    },
    async huaweiTakeover(row) {
      if (!this.ensureCloudServer()) {
        ElMessage.error('请在桌面版中打开本页。');
        return;
      }
      try {
        await ElMessageBox.confirm(
          '将调用华为云 IAM「创建用户」接口生成可登录控制台的用户与密码。新建用户默认无任何策略，需在控制台为其授权。须使用具备 iam:users:create 的主账号或已授权的 IAM 用户 AK/SK（子用户未授权时会 403）；若自动获取 domain 失败，请在凭据扩展 JSON 中增加 huawei_domain_id。',
          '华为云接管',
          { type: 'warning', confirmButtonText: '执行', cancelButtonText: '取消' }
        );
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(this.briefError(e, '确认接管失败'));
        return;
      }
      const loading = ElLoading.service({ fullscreen: true, text: '正在调用 IAM API…' });
      try {
        const raw = await CloudServerTakeoverConsoleJSON(JSON.stringify({ credential_id: row.id }));
        const data = JSON.parse(raw || '{}');
        if (data.error) throw new Error(data.error);
        row.takeover = { ...data, savedAt: Date.now() };
        this.takeoverPayload = { ...data, isDemo: false };
        this.takeoverVisible = true;
        ElMessage.success('已创建 IAM 用户，请妥善保管密码');
        this.pushRecord(row.id, '华为云 IAM 接管', JSON.stringify(data, null, 2));
      } catch (e) {
        ElMessage.error(this.briefError(e, '华为云接管失败'));
      } finally {
        loading.close();
      }
    },
    async ucloudTakeover(row) {
      if (!this.ensureCloudServer()) {
        ElMessage.error('请在桌面版中打开本页。');
        return;
      }
      let email = '';
      let displayName = '';
      try {
        const r1 = await ElMessageBox.prompt('子用户邮箱（CreateUser 必填，用于控制台）', 'UCloud 接管', {
          confirmButtonText: '执行',
          cancelButtonText: '取消',
          inputPlaceholder: 'name@example.com',
          inputPattern: /\S+@\S+\.\S+/,
          inputErrorMessage: '请输入有效邮箱'
        });
        email = (r1 && r1.value && String(r1.value).trim()) || '';
        if (!email) return;
        try {
          const r2 = await ElMessageBox.prompt('显示名称（可选）', 'UCloud 接管', {
            confirmButtonText: '确定',
            cancelButtonText: '不设显示名',
            inputPlaceholder: '可留空'
          });
          if (r2 && r2.value) displayName = String(r2.value).trim();
        } catch (e2) {
          if (e2 !== 'cancel') {
            ElMessage.error(this.briefError(e2, '输入显示名称失败'));
            return;
          }
        }
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(this.briefError(e, '输入接管参数失败'));
        return;
      }
      this.ucloudTakeoverAccId = row.id;
      const loading = ElLoading.service({ fullscreen: true, text: '正在调用 CreateUser…' });
      try {
        const raw = await CloudServerTakeoverConsoleJSON(
          JSON.stringify({
            credential_id: row.id,
            email,
            display_name: displayName
          })
        );
        const data = JSON.parse(raw || '{}');
        if (data.error) throw new Error(data.error);
        const draft = {
          account: data.username,
          email: data.email || email,
          loginUrl: data.login_url || 'https://console.ucloud.cn/',
          publicKey: data.access_key_id || '',
          privateKey: data.access_key_secret || '',
          consolePassword: data.console_password || '',
          consolePasswordErr: data.console_password_err || ''
        };
        row.ucloudKeys = { publicKey: draft.publicKey, privateKey: draft.privateKey };
        row.takeover = { username: draft.account, savedAt: Date.now() };
        this.ucloudUserDraft = draft;
        // 不再自动设置或展示控制台密码，备忘默认为空，由用户自行填写
        this.ucloudUserPassword = '';
        this.ucloudUserVisible = true;
        ElMessage.success('已创建 IAM 子用户');
        this.pushRecord(
          row.id,
          'UCloud CreateUser',
          JSON.stringify(
            { ...data, access_key_secret: data.access_key_secret ? '***' : '', console_password: data.console_password ? '***' : '' },
            null,
            2
          )
        );
      } catch (e) {
        ElMessage.error(this.briefError(e, 'UCloud 接管失败'));
      } finally {
        loading.close();
      }
    },
    async genericTakeover(row) {
      if (!this.ensureCloudServer()) {
        ElMessage.error('请在桌面版中打开本页。');
        return;
      }
      try {
        await ElMessageBox.confirm(
          '将调用阿里云 RAM：创建子用户、控制台登录密码；并可附加系统策略（默认 AdministratorAccess）。需要当前 AK 具备 RAM 管理权限，且会在账号内创建真实资源，请谨慎操作。',
          '阿里云接管',
          { type: 'warning', confirmButtonText: '继续', cancelButtonText: '取消' }
        );
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(this.briefError(e, '确认接管失败'));
        return;
      }
      let attach_policy = true;
      let policy_name = '';
      try {
        const { value: polIn } = await ElMessageBox.prompt(
          '系统策略：留空表示附加 AdministratorAccess；输入 none 表示不附加任何系统策略（子用户无权限）',
          'RAM 策略',
          {
            confirmButtonText: '执行',
            cancelButtonText: '取消',
            inputPlaceholder: '留空 或 none'
          }
        );
        const t = (polIn || '').trim().toLowerCase();
        if (t === 'none') attach_policy = false;
        else policy_name = (polIn || '').trim();
      } catch (e) {
        if (e === 'cancel') return;
        ElMessage.error(this.briefError(e, '读取策略输入失败'));
        return;
      }
      const loading = ElLoading.service({ fullscreen: true, text: '正在调用 RAM API…' });
      try {
        const raw = await CloudServerTakeoverConsoleJSON(
          JSON.stringify({
            credential_id: row.id,
            attach_policy,
            policy_name
          })
        );
        const data = JSON.parse(raw || '{}');
        if (data.error) throw new Error(data.error);
        row.takeover = { ...data, savedAt: Date.now() };
        this.takeoverPayload = { ...data, isDemo: false };
        this.takeoverVisible = true;
        ElMessage.success('RAM 用户已创建');
        this.pushRecord(row.id, '阿里云 RAM 接管', JSON.stringify(data, null, 2));
      } catch (e) {
        ElMessage.error(this.briefError(e, '阿里云接管失败'));
      } finally {
        loading.close();
      }
    },
    openTakeoverRecords(row) {
      this.loadRecordsFor(row.id);
      this.recordsVisible = true;
    },
    saveUcloudUserNote() {
      if (!this.ucloudUserDraft) {
        this.ucloudUserVisible = false;
        return;
      }
      const acc = this.accounts.find((a) => String(a.id) === String(this.ucloudTakeoverAccId));
      if (acc) {
        acc.ucloudNote = { passwordNote: this.ucloudUserPassword };
        this.pushRecord(acc.id, '保存 UCloud 子用户备忘记录', this.ucloudUserPassword ? '***' : '');
      }
      ElMessage.success('已保存');
      this.ucloudUserVisible = false;
    },
    resetUcloudUser() {
      this.ucloudUserDraft = null;
      this.ucloudUserPassword = '';
      this.ucloudTakeoverAccId = null;
    },
    openUcloudOrder(row) {
      this.ucloudOrderAccId = row.id;
      this.ucloudOrderVisible = true;
      this.fetchUcloudOrders();
    },
    async fetchUcloudOrders() {
      if (this.ucloudOrderAccId == null) return;
      this.ucloudOrderLoading = true;
      let begin = 0;
      let end = 0;
      const r = this.ucloudOrderFilter.range;
      if (r && r.length === 2 && r[0] && r[1]) {
        begin = Math.floor(new Date(String(r[0]).replace(/-/g, '/')).getTime() / 1000);
        end = Math.floor(new Date(String(r[1]).replace(/-/g, '/')).getTime() / 1000);
      } else {
        end = Math.floor(Date.now() / 1000);
        begin = end - 90 * 24 * 3600;
      }
      const acc = this.accounts.find((a) => String(a.id) === String(this.ucloudOrderAccId));
      const region = (acc?.regions && acc.regions[0]?.regionId) || '';
      try {
        const s = await CloudServerListUcloudOrdersJSON(
          JSON.stringify({
            credential_id: this.ucloudOrderAccId,
            begin_time: begin,
            end_time: end,
            region
          })
        );
        const data = JSON.parse(s || '{}');
        let rows = data.orders || [];
        const st = (this.ucloudOrderFilter.status || '').trim();
        if (st === 'done') {
          rows = rows.filter((o) => /成功|完成|付款|paid|success|paidoff/i.test(String(o.orderState || '')));
        } else if (st === 'active') {
          rows = rows.filter((o) => !/成功|完成|付款|paid|success|paidoff/i.test(String(o.orderState || '')));
        }
        this.ucloudOrderRows = rows;
        if (!rows.length) ElMessage.info('该时间范围内无订单或账号无 UBILL 权限');
      } catch (e) {
        ElMessage.error(this.briefError(e, '查询订单失败'));
        this.ucloudOrderRows = [];
      } finally {
        this.ucloudOrderLoading = false;
      }
    },
    viewAccount(row) {
      const copy = { ...row };
      const sk = copy.secretKey;
      copy.secretKey = '***';
      copy._note = sk ? '敏感字段已掩码' : '';
      this.viewJson = JSON.stringify(copy, null, 2);
      this.viewVisible = true;
    },
    showInstanceDetail(acc, reg, ins) {
      this.instanceDetailJson = JSON.stringify(
        {
          account: acc.alias,
          vendor: this.vendorLabel(acc.vendor),
          region: reg.regionName,
          regionId: reg.regionId,
          instance: ins
        },
        null,
        2
      );
      this.instanceDetailVisible = true;
    },
    async openRds(row) {
      if (row.vendor !== 'aliyun') {
        ElMessage.warning('仅阿里云支持 RDS OpenAPI 列举');
        return;
      }
      this.rdsAccount = row;
      this.rdsVisible = true;
      this.rdsLoading = true;
      this.rdsRegionRows = [];
      try {
        const s = await CloudServerListRDSJSON(row.id);
        const data = JSON.parse(s || '{}');
        const regions = data.regions || [];
        this.rdsRegionRows = regions;
        this.rdsExpanded = regions.length ? [regions[0].regionId] : [];
        if (!regions.length) ElMessage.info('未查询到 RDS 实例（请检查 AK 权限与地域配置）');
      } catch (e) {
        ElMessage.error(this.briefError(e, '加载 RDS 列表失败'));
      } finally {
        this.rdsLoading = false;
      }
    },
    isElMsgBoxCancel(e) {
      return e === 'cancel' || e === 'close';
    },
    showRdsDetail(title, obj) {
      this.rdsDetailTitle = title;
      this.rdsDetailJson = typeof obj === 'string' ? obj : JSON.stringify(obj, null, 2);
      this.rdsDetailVisible = true;
    },
    rdsOpBusy(ins, op) {
      const id = ins && ins.dbInstanceId;
      if (!id || !op) return false;
      return this.rdsActionBusyKey === `${id}:${op}`;
    },
    isRdsUnlock(ins) {
      const m = ins && ins.lockMode;
      return !m || String(m).toLowerCase() === 'unlock';
    },
    rdsLockModeLabel(mode) {
      if (mode == null || mode === '') return '—';
      const k = String(mode).toLowerCase().replace(/\s/g, '');
      const map = {
        unlock: '未锁定',
        manuallock: '手动锁定',
        lockbyexpiration: '到期锁定',
        lockbyrestoration: '回滚锁定',
        lockbydiskquota: '空间满锁定',
        released: '已释放',
        transforming: '变配中'
      };
      return map[k] || String(mode);
    },
    async rdsAliyunApi(acc, reg, ins, action, extra = {}, busyOp = null) {
      if (!acc?.id || !reg?.regionId || !ins?.dbInstanceId) {
        throw new Error('缺少凭据、地域或实例 ID');
      }
      const payload = {
        credential_id: acc.id,
        region_id: reg.regionId,
        db_instance_id: ins.dbInstanceId,
        action,
        ...extra
      };
      const key = busyOp ? `${ins.dbInstanceId}:${busyOp}` : null;
      if (key) this.rdsActionBusyKey = key;
      try {
        const s = await CloudServerAliyunRDSActionJSON(JSON.stringify(payload));
        return JSON.parse(s || '{}');
      } finally {
        if (key) this.rdsActionBusyKey = null;
      }
    },
    async openRdsLockInfo(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_db_instance_attribute', {}, 'attr');
        this.showRdsDetail('RDS 实例属性 / 锁定（DescribeDBInstanceAttribute）', data);
        if (data.lockMode && String(data.lockMode).toLowerCase() !== 'unlock') {
          ElMessage.warning('实例处于非 Unlock 状态时，请根据 lockReason 在控制台或工单处理。');
        }
      } catch (e) {
        ElMessage.error(this.briefError(e, '查询实例属性失败'));
      }
    },
    async rdsListWhitelist(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_ip_whitelist', {}, 'wl_list');
        this.showRdsDetail('RDS 白名单分组（DescribeDBInstanceIPArrayList）', data);
      } catch (e) {
        ElMessage.error(this.briefError(e, '查询白名单失败'));
      }
    },
    async rdsRecordWhitelist(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_ip_whitelist', {}, 'wl_rec');
        this.showRdsDetail('RDS 白名单分组（DescribeDBInstanceIPArrayList）', data);
        this.pushRecord(acc.id, 'RDS 白名单列表', JSON.stringify(data, null, 2));
        ElMessage.success('已展示并写入操作记录');
      } catch (e) {
        ElMessage.error(this.briefError(e, '记录白名单失败'));
      }
    },
    async rdsAddWhitelist(acc, reg, ins) {
      try {
        const { value: ips } = await ElMessageBox.prompt('IP 或 CIDR，多个用英文逗号分隔', '添加白名单（ModifySecurityIps Append）', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputPlaceholder: '如 1.2.3.4,10.0.0.0/24'
        });
        if (!ips || !String(ips).trim()) return;
        let group = '';
        try {
          const r2 = await ElMessageBox.prompt(
            '分组名须与控制台一致；留空则自动选用实例上的默认分组（可能不是 Default）',
            '分组',
            {
              confirmButtonText: '确定',
              cancelButtonText: '跳过',
              inputPlaceholder: '留空自动解析'
            }
          );
          if (r2 && r2.value && String(r2.value).trim()) group = String(r2.value).trim();
        } catch (e2) {
          if (!this.isElMsgBoxCancel(e2)) throw e2;
        }
        const data = await this.rdsAliyunApi(
          acc,
          reg,
          ins,
          'append_security_ips',
          {
            security_ips: String(ips).trim(),
            ip_array_name: group
          },
          'wl_add'
        );
        ElMessage.success('已提交添加白名单');
        this.showRdsDetail('添加白名单结果', data);
      } catch (e) {
        if (!this.isElMsgBoxCancel(e)) ElMessage.error(this.briefError(e, '添加白名单失败'));
      }
    },
    async rdsDeleteWhitelist(acc, reg, ins) {
      try {
        const { value: ips } = await ElMessageBox.prompt('要移除的 IP 或 CIDR，多个用英文逗号分隔', '删除白名单（ModifySecurityIps Delete）', {
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        });
        if (!ips || !String(ips).trim()) return;
        let group = '';
        try {
          const r2 = await ElMessageBox.prompt(
            '分组名须与控制台一致；留空则自动选用实例上的默认分组（可能不是 Default）',
            '分组',
            {
              confirmButtonText: '确定',
              cancelButtonText: '跳过',
              inputPlaceholder: '留空自动解析'
            }
          );
          if (r2 && r2.value && String(r2.value).trim()) group = String(r2.value).trim();
        } catch (e2) {
          if (!this.isElMsgBoxCancel(e2)) throw e2;
        }
        const data = await this.rdsAliyunApi(
          acc,
          reg,
          ins,
          'delete_security_ips',
          {
            security_ips: String(ips).trim(),
            ip_array_name: group
          },
          'wl_del'
        );
        ElMessage.success('已提交删除白名单');
        this.showRdsDetail('删除白名单结果', data);
      } catch (e) {
        if (!this.isElMsgBoxCancel(e)) ElMessage.error(this.briefError(e, '删除白名单失败'));
      }
    },
    async rdsListAccounts(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_accounts', {}, 'acc_list');
        this.showRdsDetail('RDS 账号列表（DescribeAccounts）', data);
      } catch (e) {
        ElMessage.error(this.briefError(e, '查询账号列表失败'));
      }
    },
    async rdsRecordAccounts(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_accounts', {}, 'acc_rec');
        this.showRdsDetail('RDS 账号列表（DescribeAccounts）', data);
        this.pushRecord(acc.id, 'RDS 账号列表', JSON.stringify(data, null, 2));
        ElMessage.success('已展示并写入操作记录');
      } catch (e) {
        ElMessage.error(this.briefError(e, '记录账号列表失败'));
      }
    },
    async rdsAddAccount(acc, reg, ins) {
      try {
        const { value: name } = await ElMessageBox.prompt(
          '数据库账号名（建议小写字母开头，2~16 位，仅字母数字下划线，与阿里云 MySQL 规范一致）',
          '添加账号（CreateAccount）',
          {
            confirmButtonText: '下一步',
            cancelButtonText: '取消'
          }
        );
        const nameTrim = name != null ? String(name).trim() : '';
        if (!nameTrim) {
          ElMessage.warning('请输入账号名');
          return;
        }
        const { value: pass } = await ElMessageBox.prompt('账号密码（须符合实例引擎的复杂度要求）', '密码', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputType: 'password'
        });
        const passStr = pass != null ? String(pass) : '';
        if (!passStr) {
          ElMessage.warning('请输入密码');
          return;
        }
        let accType = 'Normal';
        try {
          const r3 = await ElMessageBox.prompt('账号类型：留空或填 Normal（普通）/ Super（高权限）', '类型', {
            confirmButtonText: '确定',
            cancelButtonText: '跳过',
            inputPlaceholder: 'Normal 或 Super'
          });
          if (r3 && r3.value && String(r3.value).trim()) accType = String(r3.value).trim();
        } catch (e3) {
          if (!this.isElMsgBoxCancel(e3)) throw e3;
        }
        const data = await this.rdsAliyunApi(
          acc,
          reg,
          ins,
          'create_account',
          {
            account_name: nameTrim,
            account_password: passStr,
            account_type: accType
          },
          'acc_add'
        );
        ElMessage.success('已提交创建账号');
        this.showRdsDetail('创建账号结果', data);
      } catch (e) {
        if (!this.isElMsgBoxCancel(e)) ElMessage.error(this.briefError(e, '添加账号失败'));
      }
    },
    async rdsDeleteAccount(acc, reg, ins) {
      try {
        const { value: name } = await ElMessageBox.prompt('要删除的数据库账号名', '删除账号（DeleteAccount）', {
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        });
        if (!name || !String(name).trim()) return;
        const nameTrim = String(name).trim();
        await confirmDangerousAction({
          title: '删除数据库账号',
          message: `将提交删除 RDS 账号「${nameTrim}」，请确认实例与账号名无误。`,
          expectedInput: nameTrim
        });
        const data = await this.rdsAliyunApi(acc, reg, ins, 'delete_account', {
          account_name: nameTrim
        }, 'acc_del');
        ElMessage.success('已提交删除账号');
        this.showRdsDetail('删除账号结果', data);
      } catch (e) {
        if (!this.isElMsgBoxCancel(e)) ElMessage.error(this.briefError(e, '删除账号失败'));
      }
    },
    async rdsListPublicNet(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_net_info', {}, 'net_list');
        this.showRdsDetail('RDS 网络信息（DescribeDBInstanceNetInfo，含外网连接串）', data);
      } catch (e) {
        ElMessage.error(this.briefError(e, '查询网络信息失败'));
      }
    },
    async rdsRecordPublicNet(acc, reg, ins) {
      try {
        const data = await this.rdsAliyunApi(acc, reg, ins, 'describe_net_info', {}, 'net_rec');
        this.showRdsDetail('RDS 网络信息（DescribeDBInstanceNetInfo，含外网连接串）', data);
        this.pushRecord(acc.id, 'RDS 外网/网络信息', JSON.stringify(data, null, 2));
        ElMessage.success('已展示并写入操作记录');
      } catch (e) {
        ElMessage.error(this.briefError(e, '记录网络信息失败'));
      }
    },
    async rdsAllocatePublic(acc, reg, ins) {
      try {
        await ElMessageBox.confirm(
          '将按控制台习惯自动：根据实例内网端口申请外网、生成连接串前缀，并向实例默认白名单分组追加 0.0.0.0/0（分组名按实例自动解析；若已存在可能提示失败，以返回详情为准）。',
          '申请外网地址',
          { type: 'info', confirmButtonText: '确定申请', cancelButtonText: '取消' }
        );
        const data = await this.rdsAliyunApi(acc, reg, ins, 'allocate_public_connection', {}, 'pub_alloc');
        if (data.whitelistAppendError) {
          ElMessage.warning(`外网已申请，但追加白名单失败：${data.whitelistAppendError}`);
        } else {
          ElMessage.success('已提交申请外网连接');
        }
        this.showRdsDetail('申请外网地址结果', data);
      } catch (e) {
        if (!this.isElMsgBoxCancel(e)) ElMessage.error(this.briefError(e, '申请外网失败'));
      }
    },
    async rdsReleasePublic(acc, reg, ins) {
      try {
        await confirmDangerousAction({
          title: '删除外网地址',
          message:
            '将自动从实例网络信息中解析当前外网连接串并释放（与控制台「关闭外网」一致，无需手动输入）。外网访问将中断。',
          expectedInput: DANGER_DELETE_TOKEN,
          confirmButtonText: '确定释放'
        });
        const data = await this.rdsAliyunApi(acc, reg, ins, 'release_public_connection', {}, 'pub_rel');
        ElMessage.success('已提交释放外网连接');
        this.showRdsDetail('释放外网结果', data);
      } catch (e) {
        if (!this.isElMsgBoxCancel(e)) ElMessage.error(this.briefError(e, '释放外网失败'));
      }
    },
    async openAlarm(row) {
      if (row.vendor !== 'aliyun') {
        ElMessage.warning('仅阿里云支持云安全中心 OpenAPI');
        return;
      }
      this.alarmAccount = row;
      this.alarmSubTab = 'deal';
      this.alarmVisible = true;
      this.securityLoading = true;
      this.alarmRows = [];
      this.alarmNotifyRows = [];
      try {
        const s = await CloudServerListAliyunSecurityJSON(row.id);
        const data = JSON.parse(s || '{}');
        this.alarmRows = data.alarms || [];
        this.alarmNotifyRows = data.notify_rows || [];
        if (data.notify_error) ElMessage.warning(data.notify_error);
        if (!this.alarmRows.length) {
          ElMessage.info('未查询到可疑事件（需开通云安全中心并存在告警数据）');
        }
      } catch (e) {
        ElMessage.error(this.briefError(e, '加载告警失败'));
      } finally {
        this.securityLoading = false;
      }
    },
    alarmLevelClass(level) {
      if (level === '紧急') return 'cst-level-urgent';
      if (level === '提醒') return 'cst-level-warn';
      return '';
    },
    openAlarmHandle(row) {
      this.alarmHandleRow = row;
      this.alarmHandleAction = 'manual';
      this.alarmHandleVisible = true;
    },
    showAlarmDetail(row) {
      ElMessage.info(row.detail || '无详情');
    },
    async confirmAlarmHandle() {
      if (!this.alarmHandleRow || !this.alarmAccount) {
        this.alarmHandleVisible = false;
        return;
      }
      if (!this.alarmHandleRow.securityEventIds) {
        ElMessage.warning('该事件缺少 SecurityEventIds，无法调用处置 API');
        this.alarmHandleVisible = false;
        return;
      }
      try {
        const s = await CloudServerAliyunHandleSecurityEventJSON(
          JSON.stringify({
            credential_id: this.alarmAccount.id,
            security_event_ids: this.alarmHandleRow.securityEventIds,
            action: this.alarmHandleAction
          })
        );
        const o = JSON.parse(s || '{}');
        if (o.error) {
          ElMessage.error(o.error);
        } else {
          this.alarmHandleRow.status = '处理完毕';
          ElMessage.success('处置请求已提交');
        }
      } catch (e) {
        ElMessage.error(this.briefError(e, '处置告警失败'));
      }
      this.alarmHandleVisible = false;
    },
    openNotifyEdit(row) {
      if (typeof row.route !== 'number') {
        ElMessage.warning('缺少云端 Route 字段，无法调用 ModifyNoticeConfig');
        return;
      }
      this.notifyEditRow = row;
      this.notifyTimeLimit = row.limit && String(row.limit).indexOf('08:00') >= 0 ? 1 : 0;
      this.notifyEditVisible = true;
    },
    async saveNotifyEdit() {
      if (!this.notifyEditRow || !this.alarmAccount) {
        this.notifyEditVisible = false;
        return;
      }
      try {
        const s = await CloudServerAliyunSaveNoticeJSON(
          JSON.stringify({
            credential_id: this.alarmAccount.id,
            project: this.notifyEditRow.code,
            route: this.notifyEditRow.route,
            time_limit: this.notifyTimeLimit
          })
        );
        const o = JSON.parse(s || '{}');
        if (o.error) {
          ElMessage.error(o.error);
        } else {
          this.notifyEditRow.limit =
            this.notifyTimeLimit === 1 ? '08:00-22:00 时间段内发送通知' : '无限制';
          ElMessage.success('已更新通知时间限制');
        }
      } catch (e) {
        ElMessage.error(this.briefError(e, '保存通知配置失败'));
      }
      this.notifyEditVisible = false;
    },
    resolveInstance(key) {
      if (!key) return null;
      const [aid, rid, iid] = key.split('|');
      const acc = this.accounts.find((a) => String(a.id) === String(aid));
      if (!acc) return null;
      const reg = (acc.regions || []).find((r) => r.regionId === rid);
      if (!reg) return null;
      const ins = (reg.instances || []).find((i) => i.instanceId === iid);
      if (!ins) return null;
      return { acc, reg, ins };
    },
    runCommand() {
      this.$refs.cmdFormRef.validate((ok) => {
        if (!ok) return;
        this.runCommandExec();
      });
    },
    async runCommandExec() {
      this.cmdRunning = true;
      try {
      const resolved = this.resolveInstance(this.cmdForm.instanceKey);
      if (!resolved) {
        ElMessage.error('实例不存在');
        return;
      }
      const { acc, reg, ins } = resolved;
      const cmd = this.cmdForm.command.trim();
        const body = JSON.stringify({
          credential_id: acc.id,
          region_id: reg.regionId,
          instance_id: ins.instanceId,
          command: cmd,
          system_type: this.cmdForm.systemType,
          wait_sec: Math.min(Math.max(this.cmdForm.waitSec || 60, 1), 600)
        });
        const r = await CloudServerRunCommandJSON(body);
        const o = JSON.parse(r || '{}');
        if (o.error) {
          ElMessage.warning(this.briefError({ message: o.error }, '命令执行失败'));
        } else {
          ElMessage.success('命令已下发并完成轮询');
        }
        this.appendCommandTerminal(acc, reg, ins, cmd, o);
      } catch (e) {
        ElMessage.error(this.briefError(e, '命令执行失败'));
      } finally {
        this.cmdRunning = false;
      }
    },
    briefError(e, fallback) {
      let msg = (e && e.message) ? String(e.message) : String(e || '');
      msg = msg.replace(/\s+/g, ' ').trim();
      if (!msg) return fallback || '操作失败';
      if (msg.length > 120) msg = msg.slice(0, 120) + '...';
      return msg;
    },
    vendorLabel(v) {
      if (v === 'aliyun') return '阿里云';
      if (v === 'huawei') return '华为云';
      if (v === 'ucloud') return 'UCloud';
      if (v === 'tencent') return '腾讯云';
      if (v === 'aws') return 'AWS';
      if (v === 'baidu') return '百度云';
      return v;
    }
  }
};
</script>

<style scoped>
/* 与 ObjectStoragePanel 云资产管理布局对齐 */
.ct-root {
  --ct-blue: #409eff;
  --ct-border: #e8eaec;
  --kam-text: #515a6e;
}

.oss-content.csp-root {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  min-height: 0;
  width: 100%;
  max-width: 100%;
  padding: 0 2px 8px;
  box-sizing: border-box;
  background: #fff;
  overflow: hidden;
}

.kam-cos.oss-content {
  background: #fff;
}

.csp-root > .csp-kam-panel {
  flex: 1;
  min-height: 0;
}

.csp-btn-ico {
  margin-right: 4px;
  vertical-align: middle;
}

.csp-cmd-collapse {
  border: none;
  --el-collapse-border-color: transparent;
}

.csp-cmd-collapse :deep(.el-collapse-item__header) {
  height: auto;
  line-height: 1.4;
  padding: 10px 0;
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--ct-border);
}

:deep(.csp-vendor-dlg){
  margin-top: 20px;
  width: 720px;
}

.csp-cmd-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.csp-cmd-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 12px;
}

.kam-asset-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #fff;
  border: 1px solid var(--ct-border);
  border-radius: 2px;
  overflow: hidden;
}

.kam-asset-panel:not(.is-with-explorer) {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
}

.csp-kam-panel {
  flex: 1;
  min-height: 0;
}

.kam-asset-table-sizer {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.kam-asset-table-scroll {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.kam-asset-table-main {
  flex: 1;
  min-height: 0;
}

.kam-asset-pager {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  padding-bottom: 2px;
}

.kam-asset-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px 16px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--ct-border);
}

.kam-asset-toolbar-main {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
  flex: 1;
  min-width: 0;
}

.kam-asset-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  padding: 4px 0;
  border-bottom: 2px solid transparent;
  flex-shrink: 0;
}

.kam-asset-tab.is-active {
  color: var(--ct-blue);
  border-bottom-color: var(--ct-blue);
}

.kam-asset-tab-ico {
  font-size: 18px;
  color: var(--ct-blue);
}

.kam-filter-btn {
  color: var(--kam-text);
}

.kam-dd-ico {
  margin-left: 4px;
  vertical-align: middle;
}

.kam-vendor-scroll {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: thin;
  padding: 2px 0;
}

.kam-vendor-scroll::-webkit-scrollbar {
  height: 4px;
}

.kam-vendor-scroll::-webkit-scrollbar-thumb {
  background: #dcdfe6;
  border-radius: 2px;
}

.kam-vendor-chip {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border: 1px solid var(--ct-border);
  border-radius: 2px;
  background: #fff;
  cursor: pointer;
  font-size: 13px;
  color: var(--kam-text);
  font-family: inherit;
  transition:
    color 0.15s,
    border-color 0.15s,
    background 0.15s;
}

.kam-vendor-chip:hover {
  color: var(--ct-blue);
  border-color: #c6e2ff;
  background: #ecf5ff;
}

.kam-vendor-logo {
  width: 22px;
  height: 22px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #fff;
  background: #909399;
}

.kam-vendor-logo.is-aliyun {
  background: #ff6a00;
}

.kam-vendor-logo.is-huawei {
  background: #c7000b;
}

.kam-vendor-logo.is-ucloud {
  background: #0068b7;
}

.kam-vendor-logo.is-tencent {
  background: #0052d9;
}

.kam-vendor-logo.is-aws {
  background: #232f3e;
}

.kam-vendor-logo.is-baidu {
  background: #2932e1;
}

.kam-vendor-name {
  white-space: nowrap;
}

.kam-asset-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.kam-btn-export {
  font-weight: 500;
}

.kam-asset-search-row {
  padding: 8px 14px 10px;
  border-bottom: 1px solid var(--ct-border);
}

.csp-search-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.csp-search-input {
  flex: 1;
  min-width: 160px;
  max-width: 520px;
}

.kam-asset-search {
  max-width: 420px;
}

.kam-asset-table {
  width: 100%;
}

.kam-asset-table :deep(.el-table__header th) {
  background: #fafafa !important;
  color: #515a6e;
  font-weight: 600;
}

.kam-asset-table :deep(.el-table__cell) {
  border-color: var(--ct-border) !important;
}

.csp-cmd-block {
  flex-shrink: 0;
  margin-top: 0;
  padding: 4px 14px 4px;
  border-top: 1px solid var(--ct-border);
  background: #fafbfc;
}

.cst-region-block {
  padding: 8px 4px 12px;
  background: var(--el-fill-color-lighter);
}
.cst-region-title {
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}
.cst-inner-table {
  margin-bottom: 8px;
}
.cst-vendor-form {
  max-width: 640px;
  padding: 8px 0;
}
.cst-hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  max-width: 720px;
}
.cst-takeover-records-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 10px;
}
.cst-takeover-records-scroll {
  max-height: min(58vh, 480px);
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 6px;
  margin-right: -4px;
  overscroll-behavior: contain;
}
.cst-takeover-records-pre {
  max-height: 220px;
  overflow: auto;
  margin-top: 6px;
}
.cst-link-db {
  padding: 0 4px;
}
.cst-link-alarm {
  padding: 0 4px;
}
.cst-link-warn {
  padding: 0 4px;
}
.cst-cmd-terminal-wrap {
  margin-top: 5px;
}
.cst-cmd-terminal-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.cst-cmd-terminal-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.cst-cmd-terminal-outer {
  position: relative;
  display: flex;
  flex-direction: column;
  border-radius: 6px;
  border: 1px solid var(--el-border-color-darker);
  background: #0d0d0d;
  min-height: 160px;
  max-height: min(360px, 38vh);
  overflow: hidden;
}
.cst-cmd-terminal {
  margin: 0;
  flex: 1 1 auto;
  min-height: 0;
  padding: 10px 12px 16px;
  max-height: none;
  overflow: auto;
  box-sizing: border-box;
  font-family: ui-monospace, 'Cascadia Code', 'Consolas', 'SF Mono', Menlo, monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-word;
  tab-size: 4;
}
.cst-cmd-terminal-placeholder {
  position: absolute;
  left: 14px;
  top: 12px;
  font-size: 12px;
  color: #6e6e6e;
  pointer-events: none;
  font-family: ui-monospace, 'Consolas', monospace;
}
.cst-echo-out {
  margin: 0;
  padding: 8px;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 4px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.cst-order-bar {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.cst-rds-lock-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.cst-rds-lock-text {
  font-size: 12px;
}
.cst-rds-lock-warn {
  color: var(--el-color-warning);
  font-weight: 600;
}
.cst-rds-ops {
  font-size: 12px;
  line-height: 1.8;
}
.cst-rds-ops .el-button {
  margin-right: 4px;
  margin-bottom: 4px;
}
.cst-alarm-bar {
  margin-bottom: 8px;
}
.cst-level-urgent {
  color: var(--el-color-danger);
  font-weight: 600;
}
.cst-level-warn {
  color: var(--el-color-warning);
  font-weight: 600;
}
.cst-muted-inline {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-top: 4px;
}
.cst-ucloud-secret {
  display: inline-block;
  max-width: 100%;
  word-break: break-all;
  font-size: 13px;
  padding: 4px 8px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
}
.cst-json {
  margin: 0;
  font-size: 12px;
  max-height: 400px;
  overflow: auto;
  background: var(--el-fill-color-light);
  padding: 10px;
  border-radius: 4px;
}
</style>
