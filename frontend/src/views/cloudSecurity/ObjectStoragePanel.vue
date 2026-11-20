<template>
  <div ref="ossRootRef" class="oss-content ct-root kam-cos" :class="{ 'is-fullscreen': isFullscreen }">
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      title="仅限用于自身业务测试，杜绝非法操作，对此产生的任何问题与作者无关"
    />
    <!-- 参考云资产管理：资产列表 + 横向厂商（箭头滚动）+ 表格同面板 -->
    <div
      v-show="!assetPanelCollapsed"
      class="kam-asset-panel"
      :class="{ 'is-with-explorer': !!(activeCredId && activeCred) }"
    >
      <div class="kam-asset-toolbar">
        <div class="kam-asset-toolbar-main">
          <div class="kam-asset-tab is-active">
            <el-icon class="kam-asset-tab-ico"><Files /></el-icon>
            <span>资产列表</span>
          </div>
          <el-dropdown trigger="click" class="kam-filter-dd" @command="onFilterVendor">
            <el-button size="small" class="kam-filter-btn">
              {{ filterLabel }}
              <el-icon class="kam-dd-ico"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="all">全部厂商</el-dropdown-item>
                <el-dropdown-item
                  v-for="(p, idx) in providerOptions"
                  :key="'f-' + p.value"
                  :command="p.value"
                  :divided="idx === 0"
                >
                  {{ p.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <div class="kam-vendor-rail">
            <button type="button" class="kam-vendor-arrow" aria-label="向左" @click="scrollVendorStrip(-1)">
              <el-icon><ArrowLeft /></el-icon>
            </button>
            <div ref="vendorStripRef" class="kam-vendor-scroll">
              <button
                v-for="p in providerOptions"
                :key="'v-' + p.value"
                type="button"
                class="kam-vendor-chip"
                @click="openAddForProvider(p.value)"
              >
                <span class="kam-vendor-logo" :class="'is-' + p.value">{{ providerShort(p.value) }}</span>
                <span class="kam-vendor-name">{{ p.tabLabel }}</span>
              </button>
            </div>
            <button type="button" class="kam-vendor-arrow" aria-label="向右" @click="scrollVendorStrip(1)">
              <el-icon><ArrowRight /></el-icon>
            </button>
          </div>
        </div>
        <div class="kam-asset-actions">
          <el-button size="small" @click="showSecrets = !showSecrets">{{ showSecrets ? '隐藏' : '显示' }}</el-button>
          <el-button type="primary" size="small" plain class="kam-btn-export" @click="exportAssets">导出资产</el-button>
        </div>
      </div>
      <div class="kam-asset-search-row">
        <el-input
          v-model="searchKw"
          class="kam-asset-search"
          size="small"
          clearable
          placeholder="搜索名称、密钥、备注、扩展 JSON…"
          @keyup.enter="noop"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      <div class="kam-asset-table-sizer">
        <div class="kam-asset-table-scroll">
          <el-table
            :data="pagedCredentials"
            border
            stripe
            size="small"
            class="cos-asset-table kam-asset-table"
            :row-class-name="credRowClassName"
            height="100%"
            empty-text="暂无凭据，点击上方厂商标签新建该云账号"
          >
        <el-table-column prop="name" label="名称" min-width="140" show-overflow-tooltip sortable>
          <template #default="{ row }">
            <span class="ct-name-cell">
              <span class="ct-prov-dot" :title="providerLabel(row.provider)">{{ providerShort(row.provider) }}</span>
              {{ row.name || '-' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="accessKey" min-width="160" show-overflow-tooltip sortable prop="access_key">
          <template #default="{ row }">{{ maskField(row.access_key) }}</template>
        </el-table-column>
        <el-table-column label="secretKey" min-width="160" show-overflow-tooltip sortable prop="secret_key">
          <template #default="{ row }">{{ maskField(row.secret_key) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="168" show-overflow-tooltip sortable prop="created_at">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="100" show-overflow-tooltip sortable />
        <el-table-column label="操作" width="200" fixed="right" align="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" class="op-link" @click.stop="openExplorer(row)">登入</el-button>
            <el-button link type="default" size="small" class="op-link op-edit" @click.stop="openCredDialog(row)">编辑</el-button>
            <el-button link type="danger" size="small" class="op-link" @click.stop="removeCred(row)">删除</el-button>
          </template>
        </el-table-column>
          </el-table>
        </div>
        <div v-if="filteredCredentials.length > 0" class="kam-asset-pager">
          <el-pagination
            v-model:current-page="credAssetPage"
            v-model:page-size="credAssetPageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="filteredCredentials.length"
            layout="total, sizes, prev, pager, next"
            size="small"
            background
            @size-change="onCredAssetPageSizeChange"
          />
        </div>
      </div>
    </div>

    <div v-if="assetPanelCollapsed && activeCredId && activeCred" class="kam-asset-panel-compact">
      <span class="kam-asset-compact-text">
        <el-icon class="kam-asset-compact-ico"><Files /></el-icon>
        {{ activeCred.name || '未命名' }} · {{ providerLabel(activeCred.provider) }}
      </span>
      <el-button size="small" text type="primary" @click="assetPanelCollapsed = false">展开凭据列表</el-button>
    </div>

    <!-- 参考云资产管理：极左图标栏 + 存储空间 + 主区（面包屑 / 统计 / 搜索与视图切换） -->
    <section v-if="activeCredId && activeCred" class="kam-explorer-panel oss-explorer-section ct-explorer-card arsa-explorer">
      <div class="ct-explorer-head kam-explorer-head">
        <span class="ct-explorer-title">{{ activeCred.name || '未命名' }} · {{ providerLabel(activeCred.provider) }}</span>
        <div class="kam-explorer-head-actions">
          <el-button
            v-if="activeCredId && activeCred && !assetPanelCollapsed"
            size="small"
            text
            type="info"
            @click="assetPanelCollapsed = true"
          >
            收起凭据
          </el-button>
          <el-button
            v-else-if="assetPanelCollapsed"
            size="small"
            text
            type="primary"
            @click="assetPanelCollapsed = false"
          >
            展开凭据
          </el-button>
          <el-input
            v-model="objectSearchKw"
            size="small"
            clearable
            placeholder="搜索名称或路径"
            class="kam-explorer-top-search"
          />
          <el-button size="small" text type="primary" @click="showExplorerHint = !showExplorerHint">填写说明</el-button>
        </div>
      </div>
      <el-alert v-if="showExplorerHint" type="info" :closable="false" class="ct-explorer-hint" @close="showExplorerHint = false">
        新建桶时若提示缺地域，在对话框填写或通过扩展 JSON 配置 region/zone；阿里云可留空由后端探测；七牛 kkfile 需 domain；青云/又拍 kkfile 请本地下载预览。
      </el-alert>
      <div class="kam-explorer-body arsa-content-row ct-explorer-body">
        <aside class="kam-icon-rail" aria-label="快捷操作">
          <el-tooltip content="新增凭据" placement="right">
            <el-button class="kam-rail-btn" circle size="small" @click="openAdd">
              <el-icon><Plus /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="刷新桶列表" placement="right">
            <el-button class="kam-rail-btn" circle size="small" :loading="loadingBuckets" @click="fetchBuckets">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </el-tooltip>
          <div class="kam-rail-spacer" />
          <el-tooltip content="kkfile 设置" placement="right">
            <el-button class="kam-rail-btn" circle size="small" @click="openKkfileConfig">
              <el-icon><Setting /></el-icon>
            </el-button>
          </el-tooltip>
        </aside>
        <aside class="ct-expl-sidebar arsa-layout-menu">
          <div class="ct-side-title">存储空间</div>
          <div class="ct-side-bucket-toolbar">
            <el-button size="small" class="ct-side-refresh" :loading="loadingBuckets" @click="fetchBuckets">刷新</el-button>
            <el-button size="small" type="primary" @click="promptCreateBucket">新建桶</el-button>
          </div>
          <ul class="ct-bucket-ul arsa-buckets-menu">
            <li
              v-for="b in buckets"
              :key="(b.name || '') + (b.tag || '')"
              :class="['ct-bucket-li', { 'is-active': isBucketSidebarActive(b) }]"
              :aria-current="isBucketSidebarActive(b) ? 'true' : undefined"
              @click="onBucketRow(b)"
              @contextmenu.prevent.stop="onBucketContextMenu($event, b)"
            >
              <span class="ct-bucket-abbr">{{ bucketAbbr(b.name) }}</span>
              <div class="ct-bucket-text">
                <div class="ct-bucket-name">{{ b.name }}</div>
                <div v-if="b.tag" class="ct-bucket-tag">{{ b.tag }}</div>
              </div>
            </li>
          </ul>
          <div v-if="!buckets.length" class="ct-bucket-empty">点击「刷新桶」加载</div>
        </aside>
        <div class="arsa-bucketpage" :class="{ 'is-idle': !currentBucket }">
          <template v-if="currentBucket">
            <div class="arsa-dir-layout kam-dir-toolbar">
              <div class="arsa-header-dir-view">
                <nav class="kam-crumb" aria-label="路径">
                  <el-icon class="kam-crumb-home"><FolderOpened /></el-icon>
                  <button type="button" class="kam-crumb-link" @click="setPrefix('')">根</button>
                  <template v-for="(seg, i) in prefixSegments" :key="'c-' + i">
                    <span class="kam-crumb-sep">&gt;</span>
                    <button type="button" class="kam-crumb-link" @click="setPrefixByIndex(i)">{{ seg }}</button>
                  </template>
                </nav>
              </div>
              <div class="arsa-header-info-view kam-file-stats">
                <span class="count">本页 <strong>{{ objectFileCount }}</strong> 个文件</span>
                <span class="size">本页 <strong>{{ objectTotalSizeLabel }}</strong> 存储量</span>
              </div>
              <div class="arsa-header-button-view kam-dir-actions">
                <el-button-group size="small" class="kam-view-toggle">
                  <el-button :type="objectViewMode === 'list' ? 'primary' : 'default'" @click="objectViewMode = 'list'">
                    <el-icon><List /></el-icon>
                  </el-button>
                  <el-button :type="objectViewMode === 'grid' ? 'primary' : 'default'" @click="objectViewMode = 'grid'">
                    <el-icon><Grid /></el-icon>
                  </el-button>
                </el-button-group>
                <el-button size="small" class="arsa-dir-btn" @click="fetchObjects(true)" :loading="loadingObjects">
                  <el-icon class="btn-ico"><Refresh /></el-icon>
                  刷新
                </el-button>
              </div>
            </div>

            <div
              class="oss-object-surface"
              :class="{ 'is-drag-over': objectDropOver }"
              @dragover.prevent="onObjectDragOver"
              @dragleave.prevent="onObjectDragLeave"
              @drop.prevent="onObjectAreaDrop"
              @contextmenu.prevent="onObjectListSurfaceContextMenu"
            >
              <div class="oss-obj-pager">
                <span class="oss-pager-label">每页</span>
                <el-select
                  v-model="objectPageSize"
                  size="small"
                  class="oss-page-size"
                  :disabled="loadingObjects"
                  @change="onObjectPageSizeChange"
                >
                  <el-option :value="50" label="50" />
                  <el-option :value="100" label="100" />
                  <el-option :value="200" label="200" />
                </el-select>
                <el-button size="small" :disabled="objectCurrentPage <= 0 || loadingObjects" @click="goObjectPrevPage">
                  上一页
                </el-button>
                <span class="oss-page-num">第 {{ objectCurrentPage + 1 }} 页</span>
                <el-button
                  size="small"
                  :disabled="!listTruncated || loadingObjects"
                  :loading="loadingObjects"
                  @click="goObjectNextPage"
                >
                  下一页
                </el-button>
              </div>
              <div
                v-if="objectViewMode === 'grid'"
                ref="objectGridRef"
                v-loading="loadingObjects"
                class="ct-file-grid arsa-file-list kam-file-grid"
              >
                <div
                  v-for="item in explorerGridItems"
                  :key="item.key"
                  class="ct-file-tile"
                  :data-oss-thumb-key="thumbObserveKey(item)"
                  @dblclick="onExplorerItemDblclick(item)"
                  @contextmenu.prevent.stop="onObjectContextMenu($event, item)"
                >
                  <div class="ct-file-icon-wrap">
                    <img
                      v-if="!item.is_dir && thumbUrlByKey[item.key]"
                      class="ct-file-thumb"
                      :src="thumbUrlByKey[item.key]"
                      alt=""
                      loading="lazy"
                      decoding="async"
                      @error="onThumbError(item.key)"
                    />
                    <div v-else class="ct-file-icon" :class="fileIconClass(item)">{{ fileIconText(item) }}</div>
                  </div>
                  <div class="ct-file-name" :title="item.display || item.key">{{ item.display || item.key }}</div>
                </div>
                <div v-if="!explorerGridItems.length && !loadingObjects" class="ct-grid-empty">当前目录无对象</div>
              </div>
              <el-table
                v-else
                :data="explorerGridItems"
                size="small"
                stripe
                height="100%"
                class="inner-table ct-obj-table arsa-file-list"
                empty-text="当前目录无对象"
                @row-contextmenu="onTableRowContextMenu"
                @row-dblclick="onObjectTableDblclick"
              >
                <el-table-column prop="display" label="名称" min-width="200" show-overflow-tooltip />
                <el-table-column prop="size" label="大小" width="100" />
                <el-table-column prop="last_modified" label="修改时间" width="168" show-overflow-tooltip />
              </el-table>

              <div class="oss-drop-hint" @contextmenu.stop>
                <el-icon class="oss-drop-hint-ico"><Upload /></el-icon>
                <span>将文件拖入上方区域或整个应用窗口将打开上传对话框；右键空白处可新建文件夹 / 文本 / 选择上传；右键文件夹可进入目录、删除或强制删除。</span>
              </div>
            </div>
          </template>
          <div v-else class="objects-placeholder">
            <el-icon class="placeholder-ico"><Box /></el-icon>
            <p>在左侧选择一个存储桶</p>
          </div>
        </div>
      </div>
    </section>

    <teleport to="body">
      <div
        v-show="ctxMenuVisible"
        class="ct-ctx-menu"
        :style="{ left: ctxMenuX + 'px', top: ctxMenuY + 'px' }"
        @click.stop
      >
        <template v-if="ctxMenuMode === 'bucket'">
          <button type="button" class="ct-ctx-item danger" @click="ctxDeleteBucket">删除存储桶</button>
        </template>
        <template v-else-if="ctxMenuMode === 'blank'">
          <button type="button" class="ct-ctx-item" @click="ctxNewFolder">新建文件夹</button>
          <button type="button" class="ct-ctx-item" @click="ctxNewTextFile">新建文本文件</button>
          <button type="button" class="ct-ctx-item" @click="ctxPickAndUpload">上传文件…</button>
        </template>
        <template v-else-if="ctxTargetRow && isObjectDirItem(ctxTargetRow)">
          <button type="button" class="ct-ctx-item" @click="ctxOpenPrefix">进入目录</button>
          <button type="button" class="ct-ctx-item danger" @click="ctxDeleteFolder">删除文件夹</button>
          <button type="button" class="ct-ctx-item danger" @click="ctxForceDeleteFolder">强制删除…</button>
        </template>
        <template v-else>
          <button type="button" class="ct-ctx-item" @click="ctxEditObject">编辑</button>
          <button type="button" class="ct-ctx-item" @click="ctxKkfilePreview">kkfile 预览</button>
          <button type="button" class="ct-ctx-item" @click="ctxDownload">下载</button>
          <button type="button" class="ct-ctx-item" @click="ctxRename">重命名</button>
          <button type="button" class="ct-ctx-item danger" @click="ctxDeleteObject">删除</button>
        </template>
      </div>
    </teleport>

    <el-dialog
      v-model="createBucketDialogVisible"
      title="新建存储桶"
      width="480px"
      destroy-on-close
      class="cos-dialog"
      @closed="resetCreateBucketForm"
    >
      <el-alert v-if="createBucketUnsupportedReason" type="warning" :closable="false" show-icon class="ct-create-bucket-warn">
        {{ createBucketUnsupportedReason }}
      </el-alert>
      <template v-else>
        <p class="ct-create-bucket-tip">
          不少厂商创建空桶时没有已选桶的「地域标签」，需在下方填写<strong>地域</strong>或<strong>可用区</strong>（也可事先写在凭据的扩展 JSON）。
        </p>
        <el-form label-position="top" class="ct-create-bucket-form" @submit.prevent>
          <el-form-item label="存储桶名称">
            <el-input v-model="createBucketForm.name" clearable placeholder="须符合云厂商命名规则" @keyup.enter="submitCreateBucket" />
          </el-form-item>
          <el-form-item :required="createBucketRegionRequired">
            <template #label>
              <span>{{ createBucketRegionLabel }}</span>
            </template>
            <el-input
              v-model="createBucketForm.regionTag"
              clearable
              :placeholder="createBucketRegionPlaceholder"
              @keyup.enter="submitCreateBucket"
            />
            <div v-if="createBucketRegionHint" class="ct-create-bucket-field-hint">{{ createBucketRegionHint }}</div>
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <el-button @click="createBucketDialogVisible = false">取消</el-button>
        <el-button
          v-if="!createBucketUnsupportedReason"
          type="primary"
          :loading="createBucketSubmitting"
          @click="submitCreateBucket"
        >
          创建
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="deleteBucketDialogVisible"
      title="删除存储桶"
      width="520px"
      destroy-on-close
      class="cos-dialog"
      :close-on-click-modal="false"
      @closed="resetDeleteBucketDialog"
    >
      <el-alert type="error" :closable="false" show-icon class="ct-delete-bucket-alert">
        <template #title>危险操作</template>
        <p class="ct-delete-bucket-alert-body">
          将删除空存储桶「<strong>{{ deleteBucketTarget?.name || '—' }}</strong>」（云侧要求桶内无对象）。此操作不可撤销。
        </p>
      </el-alert>
      <p class="ct-delete-bucket-instr">请对照下方灰色区域内的文字<strong>手动键入</strong>（不可用复制粘贴）；该区域文字不可复制。</p>
      <div class="ct-delete-bucket-phrase-wrap" @copy.prevent @cut.prevent>
        <code class="ct-delete-bucket-phrase" tabindex="-1">{{ deleteBucketRequiredPhrase }}</code>
      </div>
      <el-input
        v-model="deleteBucketConfirmInput"
        class="ct-delete-bucket-input"
        type="textarea"
        :rows="2"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        placeholder="在此输入确认语（禁止复制粘贴、拖入文本）"
        @paste.prevent
        @copy.prevent
        @cut.prevent
        @drop.prevent
      />
      <template #footer>
        <el-button @click="deleteBucketDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="deleteBucketSubmitting" @click="submitDeleteBucket">确认删除</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="kkfileConfigVisible" title="kkfile 预览地址" width="520px" destroy-on-close class="cos-dialog">
      <p class="ct-kkfile-tip">须包含占位符 <code>{base64(url)}</code>（与 OSS 文件遍历一致）。</p>
      <el-input v-model="kkfilePreviewURL" type="textarea" :rows="3" placeholder="https://file.kkview.cn/onlinePreview?url={base64(url)}&key=000" />
      <template #footer>
        <el-button @click="kkfileConfigVisible = false">取消</el-button>
        <el-button @click="resetKkfileTpl">默认</el-button>
        <el-button type="primary" @click="saveKkfileTpl">保存</el-button>
      </template>
    </el-dialog>

    <!-- 与 OssBucketPanel 一致：kkfile 在弹窗内 iframe 预览 -->
    <el-dialog
      v-model="kkfilePreviewVisible"
      width="75%"
      top="5vh"
      :close-on-click-modal="false"
      destroy-on-close
      class="cos-dialog kam-kkfile-preview-dialog"
      @closed="onKkfilePreviewClosed"
    >
      <template #header>
        <div class="kam-kkfile-preview-header">
          <span class="kam-kkfile-preview-title">文件预览</span>
          <div v-if="kkfilePreviewSrc" class="kam-kkfile-preview-actions">
            <el-button type="primary" link size="small" @click="downloadKkfilePreviewTarget">
              <el-icon class="kam-kkfile-act-ico"><Download /></el-icon>
              下载
            </el-button>
            <el-button type="primary" link size="small" @click="pickOverwriteKkfilePreviewTarget">
              <el-icon class="kam-kkfile-act-ico"><Upload /></el-icon>
              本地覆盖上传
            </el-button>
            <el-button type="primary" link size="small" @click="openKkfileInBrowser">在浏览器打开</el-button>
          </div>
        </div>
      </template>
      <div v-if="kkfilePreviewSrc" class="kam-kkfile-preview-wrap">
        <iframe :src="kkfilePreviewSrc" class="kam-kkfile-preview-iframe" title="kkfile 预览" frameborder="0" allowfullscreen />
      </div>
    </el-dialog>

    <!-- 音视频：预签名 URL 流式播放（避免整文件 Base64）；归档对象仍会 403，见 ossHumanizeObjectError -->
    <el-dialog
      v-model="objectMediaPreviewVisible"
      width="min(92vw, 820px)"
      top="6vh"
      class="cos-dialog oss-object-media-preview-dialog"
      destroy-on-close
      :close-on-click-modal="false"
      @closed="onObjectMediaPreviewClosed"
    >
      <template #header>
        <div class="oss-media-preview-head">
          <span class="oss-media-preview-title">
            {{ objectMediaPreviewKind === 'audio' ? '音频' : '视频' }}预览：{{ objectMediaPreviewTitle }}
          </span>
          <div class="oss-media-preview-actions">
            <el-button
              v-if="objectMediaPreviewSrc"
              type="primary"
              link
              size="small"
              @click="openObjectMediaInBrowser"
            >
              在浏览器打开
            </el-button>
          </div>
        </div>
      </template>
      <div v-loading="objectMediaPreviewLoading" class="oss-media-preview-body">
        <video
          v-if="objectMediaPreviewKind === 'video' && objectMediaPreviewSrc"
          class="oss-media-preview-video"
          controls
          playsinline
          preload="metadata"
          :src="objectMediaPreviewSrc"
          @error="onObjectMediaPlaybackError"
        />
        <audio
          v-else-if="objectMediaPreviewKind === 'audio' && objectMediaPreviewSrc"
          class="oss-media-preview-audio"
          controls
          preload="metadata"
          :src="objectMediaPreviewSrc"
          @error="onObjectMediaPlaybackError"
        />
        <el-empty
          v-else-if="!objectMediaPreviewLoading && !objectMediaPreviewSrc"
          description="无法生成播放地址"
        />
      </div>
    </el-dialog>

    <!-- 双击 / 编辑：默认仅文本（Monaco）；kkfile 由用户点「kkfile 预览」后嵌入上方 -->
    <el-dialog
      v-model="objectTextPreviewVisible"
      width="82%"
      top="3vh"
      class="cos-dialog oss-object-text-preview-dialog"
      destroy-on-close
      :close-on-click-modal="false"
      @closed="onObjectTextPreviewDialogClosed"
    >
      <template #header>
        <div class="oss-obj-text-preview-head">
          <span class="oss-obj-text-preview-title">编辑 / 预览：{{ objectFilePreview.name || objectFilePreview.key }}</span>
          <div class="oss-obj-text-preview-actions">
            <el-tag v-if="objectFilePreview.isText" size="small" type="info" effect="plain">
              {{ fmtSize(objectFilePreview.sizeBytes) }}
            </el-tag>
            <el-tag v-if="objectFilePreview.binaryLatin1" size="small" type="warning" effect="plain">
              二进制字节映射（保存可还原）
            </el-tag>
            <el-button
              v-if="objectFilePreview.isText"
              size="small"
              type="primary"
              :disabled="!objectFilePreview.dirty"
              @click="saveObjectTextPreview"
            >
              保存修改
            </el-button>
            <el-button
              v-if="objectFilePreview.key && !objectFilePreview.loading"
              size="small"
              text
              type="primary"
              @click="showKkfileInTextPreview"
            >
              kkfile 预览
            </el-button>
            <el-button size="small" text type="danger" @click="closeObjectTextPreview">关闭</el-button>
          </div>
        </div>
      </template>
      <div class="oss-obj-text-preview-body">
        <div v-if="objectTextPreviewKkfileSrc" class="oss-text-preview-kkfile-block">
          <div class="oss-text-preview-kkfile-bar">
            <span class="oss-text-preview-kkfile-title">kkfile 预览</span>
            <el-button type="primary" link size="small" @click="openTextPreviewKkfileInBrowser">在浏览器打开</el-button>
            <el-button type="primary" link size="small" @click="clearTextPreviewKkfile">收起</el-button>
          </div>
          <iframe
            :src="objectTextPreviewKkfileSrc"
            class="oss-text-preview-kkfile-iframe"
            title="kkfile"
            frameborder="0"
          />
        </div>
        <el-alert
          v-if="objectFilePreview.largeFileWarn && !objectFilePreview.loading"
          type="warning"
          :closable="false"
          show-icon
          class="oss-text-preview-warn"
          title="文件较大，加载或保存可能较慢。"
        />
        <div v-if="objectFilePreview.loading" class="oss-obj-text-preview-empty">
          <el-icon class="is-loading" style="margin-right: 6px"><Refresh /></el-icon>
          正在加载…
        </div>
        <div v-else-if="objectFilePreview.error" class="oss-obj-text-preview-empty">
          <el-empty :description="objectFilePreview.error" />
        </div>
        <div v-else-if="objectFilePreview.isText" class="oss-text-preview-editor-block">
          <div class="oss-text-preview-editor-bar">
            <span class="oss-text-preview-editor-title">文本内容（UTF-8）</span>
            <el-button type="primary" link size="small" @click="objectTextEditorExpanded = !objectTextEditorExpanded">
              {{ objectTextEditorExpanded ? '收起' : '展开' }}
            </el-button>
          </div>
          <div v-show="objectTextEditorExpanded" class="oss-text-preview-editor-pane">
            <VueMonacoEditor
              :key="'oss-edit-' + (objectFilePreview.key || '')"
              v-model:value="ossTextPreviewContent"
              :language="ossObjectPreviewLanguage"
              theme="vs-dark"
              :height="objectTextPreviewEditorHeight"
              class="oss-obj-text-monaco"
              :options="objectFilePreviewMonacoOptions"
            />
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 参考 ssh 文件传输：拖拽后在此确认并查看上传进度 -->
    <el-dialog
      v-model="ossUploadDialogVisible"
      title="上传到对象存储"
      width="720px"
      class="cos-dialog oss-upload-dialog"
      align-center
      :close-on-click-modal="false"
      :close-on-press-escape="!ossUploadStatus.isUploading"
      @closed="onOssUploadDialogClosed"
    >
      <div class="oss-upload-dialog-content">
        <el-form label-width="88px" label-position="left">
          <el-form-item label="目标路径">
            <el-input :value="objectPrefix || '（桶根目录）'" disabled />
          </el-form-item>
        </el-form>
        <div
          class="oss-upload-drop-zone"
          :class="{ 'is-drag-over': ossUploadDragOver }"
          @drop.prevent="onOssUploadDialogDrop"
          @dragover.prevent="ossUploadDragOver = true"
          @dragleave.prevent="ossUploadDragOver = false"
          @dragenter.prevent="ossUploadDragOver = true"
        >
          <el-icon class="oss-upload-drop-ico" :size="40"><Upload /></el-icon>
          <p class="oss-upload-drop-text">拖拽本地文件到此处，或点击下方按钮选择</p>
          <el-button type="primary" size="small" :disabled="ossUploadStatus.isUploading" @click="pickFilesForOssUpload">
            选择文件
          </el-button>
        </div>
        <div v-if="ossUploadPaths.length > 0" class="oss-upload-file-list">
          <div class="oss-upload-list-head">
            <span>待上传 {{ ossUploadPaths.length }} 个</span>
            <el-button type="danger" text size="small" :disabled="ossUploadStatus.isUploading" @click="clearOssUploadList">
              清空
            </el-button>
          </div>
          <el-scrollbar height="200px">
            <div class="oss-upload-list-items">
              <div v-for="(item, index) in ossUploadPaths" :key="item.path + '-' + index" class="oss-upload-list-item">
                <el-icon class="oss-upload-file-ico"><Document /></el-icon>
                <span class="oss-upload-file-name" :title="item.path">{{ item.name }}</span>
              </div>
            </div>
          </el-scrollbar>
        </div>
        <div v-if="ossUploadStatus.isUploading" class="oss-upload-progress-block">
          <div class="oss-upload-progress-head">
            <span class="oss-upload-status-text">
              {{ ossUploadStatus.uploadedFiles }} / {{ ossUploadStatus.totalFiles }} 个文件
              <span v-if="ossUploadStatus.currentFile" class="oss-upload-current-file">
                — 正在上传: {{ ossUploadStatus.currentFile }}
              </span>
            </span>
          </div>
          <el-progress
            :percentage="ossUploadPercentage"
            :status="ossUploadPercentage === 100 ? 'success' : ''"
            :stroke-width="12"
          />
        </div>
      </div>
      <template #footer>
        <span class="oss-upload-footer">
          <el-button :disabled="ossUploadStatus.isUploading" @click="closeOssUploadDialog">取消</el-button>
          <el-button
            type="primary"
            :disabled="ossUploadPaths.length === 0 || ossUploadStatus.isUploading"
            @click="startOssUploadFromDialog"
          >
            开始上传
          </el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog
      v-model="credDialog"
      class="cos-dialog kam-cred-dialog"
      width="680px"
      destroy-on-close
      align-center
      @closed="onDialogClosed"
    >
      <template #header>
        <span class="kam-cred-dialog-title">{{ credDialogTitle }}</span>
      </template>
      <el-form class="kam-cred-form" label-position="top" :model="credForm">
        <el-row :gutter="16" class="kam-cred-form-row">
          <el-col :span="12">
            <el-form-item v-if="!credForm.id" label="云平台" required>
              <el-select v-model="credForm.provider" filterable style="width: 100%" placeholder="选择云厂商">
                <el-option v-for="o in providerOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
            <el-form-item v-else label="云平台">
              <el-input :model-value="providerLabel(credForm.provider)" disabled />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="别名" required>
              <el-input v-model="credForm.name" maxlength="64" show-word-limit placeholder="例如 阿里云-1" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="凭据类型" class="kam-cred-form-full">
          <el-radio-group v-model="credAuthKind" size="small" class="kam-cred-auth-kind" @change="onCredAuthKindChange">
            <el-radio-button label="long">长期访问密钥</el-radio-button>
            <el-radio-button label="sts">临时凭证 (STS)</el-radio-button>
          </el-radio-group>
          <p v-if="credAuthKind === 'long'" class="kam-cred-auth-hint">
            仅 AccessKey + SecretKey。在下方密钥框粘贴 STS 返回的整段 JSON 会自动识别并切换为临时凭证。
          </p>
          <p v-else class="kam-cred-auth-hint">
            适用于阿里云 RAM/STS、腾讯云 CAM、AWS AssumeRole、七牛 S3 兼容 STS、MinIO/京东云等 S3 临时凭证；需填写会话令牌。青云/又拍云对象存储请用长期密钥，勿填 Token。
          </p>
        </el-form-item>
        <el-row :gutter="16" class="kam-cred-form-row">
          <el-col :span="12">
            <el-form-item label="ACCESS_KEY" required>
              <el-input
                v-model="credForm.access_key"
                type="password"
                show-password
                :placeholder="credAccessKeyPlaceholder"
                @paste="onCredKeyPaste"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="SECRET_KEY" required>
              <el-input
                v-model="credForm.secret_key"
                type="password"
                show-password
                :placeholder="credSecretKeyPlaceholder"
                @paste="onCredKeyPaste"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item
          v-show="credForm.provider === 'minio'"
          label="S3 API Endpoint"
          required
          class="kam-cred-form-full"
        >
          <el-input
            v-model="credForm.minio_endpoint"
            clearable
            placeholder="例如 https://minio.example.com:9000 或内网/反代地址（不含桶名路径）"
          />
          <p class="kam-cred-auth-hint">
            MinIO 的 S3 协议入口；保存时会合并进下方扩展 JSON 的 <code>endpoint</code> 字段。
          </p>
        </el-form-item>
        <el-form-item
          v-show="credAuthKind === 'sts'"
          label="会话令牌 (SessionToken)"
          required
          class="kam-cred-form-full"
        >
          <el-input
            v-model="credForm.session_token"
            type="password"
            show-password
            placeholder="SecurityToken / SessionToken / Token；也可将整段 STS JSON 粘贴到 ACCESS_KEY 或 SECRET_KEY 框"
            @paste="onCredKeyPaste"
          />
        </el-form-item>
        <el-form-item label="备注" class="kam-cred-form-full">
          <el-input
            v-model="credForm.remark"
            type="textarea"
            :rows="1"
            maxlength="100"
            show-word-limit
            placeholder="请输入备注"
          />
        </el-form-item>
        <el-collapse class="kam-cred-advanced">
          <el-collapse-item title="扩展 JSON（区域 / Endpoint 等）" name="extra">
            <el-input
              v-model="credForm.extra_json"
              type="textarea"
              :rows="4"
              :placeholder="credExtraJsonPlaceholder"
              @blur="onCredExtraJsonBlur"
            />
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <div class="kam-cred-footer">
        <p class="kam-cred-hint">
          <span class="kam-cred-hint-star">*</span>
          密钥请在各云控制台创建；临时凭证需选「临时凭证」并填写 Token。区域等可写在扩展 JSON。腾讯云列举桶仅需 AK/SK（或临时 Id/Key），选桶后自动带地域。MinIO 请填写上方「S3 API Endpoint」。
        </p>
        <div class="kam-cred-btns">
          <el-button @click="credDialog = false">取消</el-button>
          <el-button @click="resetCredDialog">重置</el-button>
          <el-button type="primary" :loading="savingCred" @click="credForm.id ? saveCred() : saveCredAndEnter()">
            {{ credForm.id ? '保存' : '设置并进入' }}
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import dayjs from 'dayjs';
import { ElMessageBox } from 'element-plus';
import {
  Search,
  FolderOpened,
  Refresh,
  Upload,
  Download,
  Document,
  Box,
  Files,
  ArrowLeft,
  ArrowRight,
  ArrowDown,
  Plus,
  List,
  Grid,
  Setting,
  Operation
} from '@element-plus/icons-vue';
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';
import { BrowserOpenURL, EventsOn } from '../../../wailsjs/runtime';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import {
  ObjectStorageCredentialsJSON,
  ObjectStorageSaveCredentialJSON,
  ObjectStorageDeleteCredential,
  ObjectStorageListBucketsJSON,
  ObjectStorageListObjectsJSON,
  ObjectStorageUploadLocalFile,
  ObjectStoragePickLocalFile,
  ObjectStorageSuggestObjectKey,
  ObjectStorageObjectPresignedURL,
  ObjectStorageGetObjectBase64,
  ObjectStorageDeleteObject,
  ObjectStorageDeletePrefixRecursive,
  ObjectStorageRenameObject,
  ObjectStorageDownloadObject,
  ObjectStoragePutObjectBase64,
  ObjectStorageCreateBucket,
  ObjectStorageDeleteBucket,
  ObjectStorageProbeCredentialJSON
} from '../../../wailsjs/go/cloudSecurity/ObjectStorage';

function ossBase64ToUint8Array(b64) {
  const bin = atob(b64);
  const u8 = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i);
  return u8;
}

function ossBytesToLatin1(u8) {
  let s = '';
  for (let i = 0; i < u8.length; i++) s += String.fromCharCode(u8[i]);
  return s;
}

function ossLatin1ToBytes(str) {
  const u8 = new Uint8Array(str.length);
  for (let i = 0; i < str.length; i++) u8[i] = str.charCodeAt(i) & 0xff;
  return u8;
}

function ossUint8ToBase64(u8) {
  let binary = '';
  const step = 0x8000;
  for (let i = 0; i < u8.length; i += step) {
    binary += String.fromCharCode.apply(null, u8.subarray(i, Math.min(i + step, u8.length)));
  }
  return btoa(binary);
}

/** 将常见对象存储 XML/文本错误转为可读说明（如归档对象未解冻） */
function ossHumanizeObjectError(msg) {
  const s = String(msg || '');
  if (/InvalidObjectState/i.test(s) || /ErrorCode\s*=\s*InvalidObjectState/i.test(s)) {
    return (
      '对象当前不可读取（InvalidObjectState）：常见于归档/冷归档/深度冷归档等，需先在云控制台对该对象解冻/恢复为标准可读状态后再预览或下载；若正在解冻中请稍后再试。'
    );
  }
  return s || '操作失败';
}

/** 从剪贴板 JSON 解析临时凭证：AWS AssumeRole、阿里云 STS、腾讯云 CAM 等 */
function parseStsCredentialJSON(raw) {
  const t = String(raw || '').trim();
  if (!t.startsWith('{')) return null;
  let j;
  try {
    j = JSON.parse(t);
  } catch {
    return null;
  }
  const pick = (o, keys) => {
    if (!o || typeof o !== 'object') return '';
    for (const k of keys) {
      if (o[k] != null && String(o[k]).trim()) return String(o[k]).trim();
    }
    return '';
  };
  let ak = pick(j, ['AccessKeyId', 'accessKeyId', 'AccessKeyID', 'TmpSecretId', 'tmpSecretId']);
  let sk = pick(j, [
    'SecretAccessKey',
    'secretAccessKey',
    'AccessKeySecret',
    'accessKeySecret',
    'TmpSecretKey',
    'tmpSecretKey'
  ]);
  let tok = pick(j, ['SessionToken', 'sessionToken', 'SecurityToken', 'securityToken', 'Token', 'token']);

  const cred = j.Credentials || j.credentials || j.Credential;
  if (cred && typeof cred === 'object') {
    if (!ak) ak = pick(cred, ['AccessKeyId', 'accessKeyId', 'TmpSecretId', 'tmpSecretId']);
    if (!sk)
      sk = pick(cred, [
        'SecretAccessKey',
        'secretAccessKey',
        'AccessKeySecret',
        'accessKeySecret',
        'TmpSecretKey',
        'tmpSecretKey'
      ]);
    if (!tok) tok = pick(cred, ['SessionToken', 'sessionToken', 'SecurityToken', 'securityToken', 'Token', 'token']);
  }

  const respC = j.Response && j.Response.Credentials;
  if (respC && typeof respC === 'object') {
    if (!ak) ak = pick(respC, ['TmpSecretId', 'AccessKeyId']);
    if (!sk) sk = pick(respC, ['TmpSecretKey', 'SecretAccessKey', 'AccessKeySecret']);
    if (!tok) tok = pick(respC, ['Token', 'SessionToken', 'SecurityToken']);
  }

  if (!ak || !sk || !tok) return null;
  return { access_key: ak, secret_key: sk, session_token: tok };
}

const providerOptions = [
  { value: 'qiniu', label: '七牛云', tabLabel: '七牛云' },
  { value: 'tencent', label: '腾讯云 COS', tabLabel: '腾讯云' },
  { value: 'aliyun', label: '阿里云 OSS', tabLabel: '阿里云' },
  { value: 'huawei', label: '华为云 OBS', tabLabel: '华为云' },
  { value: 'qingstor', label: '青云 QingStor', tabLabel: '青云' },
  { value: 'upyun', label: '又拍云', tabLabel: '又拍云' },
  { value: 'aws', label: 'AWS S3', tabLabel: 'AWS-S3' },
  { value: 'jd', label: '京东云 OSS (S3)', tabLabel: '京东云' },
  { value: 'ks3', label: '金山云 KS3 (S3)', tabLabel: 'ks3 金山云' },
  { value: 'minio', label: 'MinIO', tabLabel: 'MinIO' }
];

export default {
  name: 'ObjectStoragePanel',
  components: {
    Search,
    FolderOpened,
    Refresh,
    Upload,
    Download,
    Document,
    Box,
    Files,
    ArrowLeft,
    ArrowRight,
    ArrowDown,
    Plus,
    List,
    Grid,
    Setting,
    Operation,
    VueMonacoEditor
  },
  data() {
    return {
      providerOptions,
      listVendorFilter: 'all',
      searchKw: '',
      /** 资产列表（凭据表）前端分页 */
      credAssetPage: 1,
      credAssetPageSize: 20,
      showSecrets: false,
      credentials: [],
      activeCredId: 0,
      loadingBuckets: false,
      buckets: [],
      currentBucket: '',
      currentTag: '',
      objectPrefix: '',
      listRaw: [],
      commonPrefixes: [],
      listTruncated: false,
      listContinuation: '',
      objectPageSize: 50,
      objectCurrentPage: 0,
      objectPageSnapshots: [],
      loadingObjects: false,
      _objectListFetchGen: 0,
      _thumbObserver: null,
      uploading: false,
      objectDropOver: false,
      assetPanelCollapsed: false,
      thumbUrlByKey: {},
      credDialog: false,
      savingCred: false,
      /** long=仅 AK/SK；sts=AK/SK+SessionToken（阿里云 SecurityToken、腾讯云 Token、AWS SessionToken 等统一存后端 session_token） */
      credAuthKind: 'long',
      credForm: {
        id: 0,
        name: '',
        provider: 'qiniu',
        access_key: '',
        secret_key: '',
        session_token: '',
        extra_json: '',
        remark: '',
        /** MinIO：与 extra_json.endpoint 同步，保存时写回扩展 JSON */
        minio_endpoint: ''
      },
      pendingProvider: '',
      objectViewMode: 'grid',
      objectSearchKw: '',
      showExplorerHint: false,
      ctxMenuVisible: false,
      ctxMenuX: 0,
      ctxMenuY: 0,
      ctxMenuMode: 'item',
      ctxTargetRow: null,
      ctxBucketRow: null,
      _offGlobalFileDrop: null,
      kkfileConfigVisible: false,
      kkfilePreviewURL: 'https://file.kkview.cn/onlinePreview?url={base64(url)}&key=000',
      kkfilePreviewVisible: false,
      kkfilePreviewSrc: '',
      kkfilePreviewObjectKey: '',
      objectMediaPreviewVisible: false,
      objectMediaPreviewLoading: false,
      objectMediaPreviewKind: '',
      objectMediaPreviewTitle: '',
      objectMediaPreviewSrc: '',
      createBucketDialogVisible: false,
      createBucketSubmitting: false,
      createBucketForm: {
        name: '',
        regionTag: ''
      },
      deleteBucketDialogVisible: false,
      deleteBucketSubmitting: false,
      /** 须与用户输入完全一致（删除存储桶强校验） */
      deleteBucketRequiredPhrase: '我已知晓删除资源桶危害，确认要删除',
      deleteBucketConfirmInput: '',
      /** 待删桶 { name, tag } */
      deleteBucketTarget: null,
      ossUploadDialogVisible: false,
      ossUploadDragOver: false,
      ossUploadPaths: [],
      ossUploadPercentage: 0,
      ossUploadStatus: {
        isUploading: false,
        currentFile: '',
        uploadedFiles: 0,
        totalFiles: 0
      },
      objectTextPreviewVisible: false,
      /** 编辑弹窗内「文本内容」Monaco 区域是否展开（与 kkfile 区块各自可收起） */
      objectTextEditorExpanded: true,
      objectTextPreviewKkfileSrc: '',
      ossTextPreviewContent: '',
      ossTextPreviewBaseline: '',
      objectFilePreview: {
        key: '',
        name: '',
        sizeBytes: 0,
        loading: false,
        isText: false,
        error: '',
        dirty: false,
        suppressDirty: false,
        largeFileWarn: false,
        binaryLatin1: false
      },
      objectFilePreviewMonacoOptions: {
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
      _ctxDocClose: null,
      /** 与 unwxapp.vue 一致：窗口高度接近可用屏高时视为全屏，用于微调上下分栏比例 */
      isFullscreen: false
    };
  },
  computed: {
    filterLabel() {
      if (this.listVendorFilter === 'all') return '全部厂商';
      return this.providerLabel(this.listVendorFilter);
    },
    credDialogTitle() {
      const name = this.providerLabel(this.credForm.provider);
      return this.credForm.id ? `编辑${name}密钥` : `设置${name}密钥`;
    },
    credAccessKeyPlaceholder() {
      return this.credAuthKind === 'sts'
        ? '临时 AccessKeyId / TmpSecretId'
        : 'ACCESS_KEY；可粘贴 STS/临时密钥返回的整段 JSON';
    },
    credSecretKeyPlaceholder() {
      return this.credAuthKind === 'sts'
        ? '临时 SecretAccessKey / TmpSecretKey'
        : 'SECRET_KEY；可粘贴 STS/临时密钥返回的整段 JSON';
    },
    credExtraJsonPlaceholder() {
      if (this.credForm.provider === 'minio') {
        return '可选：{"region":"us-east-1"} 等；S3 入口请填上方「S3 API Endpoint」，保存时会写入本处 endpoint';
      }
      return '阿里云 {"region":"cn-hangzhou"}；腾讯云 {"region":"ap-guangzhou","appid":"1250000000"}（新建桶若只写短名须填纯数字 APPID；完整名形如 mybucket-1250000000 时可省略 appid）或 {"service_url":"..."}；华为云 推荐 {"endpoint":"https://obs.cn-north-4.myhuaweicloud.com"} 或 {"region":"cn-north-4"}，其余参数按默认即可；七牛 S3+STS 可不填地域时「刷新桶列表」会多地域探测并在侧栏点桶带地域；也可固定 {"s3_region":"cn-east-1"}；七牛 kkfile 另需 {"domain":"https://..."}';
    },
    activeCred() {
      return this.credentials.find((c) => c.id === this.activeCredId) || null;
    },
    /** 当前凭据扩展 JSON 解析结果（仅对象，非法 JSON 则为空对象） */
    activeCredExtraObj() {
      const c = this.activeCred;
      if (!c) return {};
      try {
        const t = String(c.extra_json || '').trim();
        if (!t) return {};
        const o = JSON.parse(t);
        if (o && typeof o === 'object' && !Array.isArray(o)) return o;
      } catch (_) {
        /* ignore */
      }
      return {};
    },
    createBucketUnsupportedReason() {
      const c = this.activeCred;
      if (!c) return '';
      if (c.provider === 'huawei') {
        return '当前未接入华为云 OBS 的创建桶 API，请至控制台创建后在此刷新列表。';
      }
      if (c.provider === 'upyun') {
        return '又拍云存储空间请在控制台创建，本工具仅支持列举与对象操作。';
      }
      return '';
    },
    createBucketPrefillRegionTag() {
      const ex = this.activeCredExtraObj;
      return String(ex.region || ex.zone || ex.s3_region || '').trim();
    },
    createBucketRegionRequired() {
      const c = this.activeCred;
      if (!c) return false;
      const ex = this.activeCredExtraObj;
      const has = (k) => String(ex[k] || '').trim() !== '';
      const pv = c.provider;
      if (pv === 'tencent') return !has('region');
      if (pv === 'qingstor') return !has('zone');
      if (pv === 'jd' || pv === 'ks3') return !has('region') && !has('endpoint');
      return false;
    },
    createBucketRegionMissingMsg() {
      const c = this.activeCred;
      if (!c) return '请填写地域或可用区';
      if (c.provider === 'qingstor') return '青云创建桶需要可用区（Zone），请填写或先在扩展 JSON 中配置 zone';
      if (c.provider === 'tencent') return '腾讯云创建桶需要地域（Region），请填写或先在扩展 JSON 中配置 region';
      if (c.provider === 'jd' || c.provider === 'ks3') {
        return '请填写地域，或先在扩展 JSON 中配置 region / endpoint';
      }
      return '请填写地域或可用区';
    },
    createBucketRegionLabel() {
      const c = this.activeCred;
      if (!c) return '地域 / 可用区';
      switch (c.provider) {
        case 'tencent':
          return '地域（Region）';
        case 'qingstor':
          return '可用区（Zone）';
        case 'aws':
          return '区域（可选，扩展未配 region 时生效）';
        case 'jd':
        case 'ks3':
          return '地域';
        case 'minio':
          return 'Region（可选）';
        default:
          return '地域 / 可用区（可选）';
      }
    },
    createBucketRegionPlaceholder() {
      const c = this.activeCred;
      if (!c) return '如 ap-guangzhou、cn-hangzhou、pek3a';
      switch (c.provider) {
        case 'tencent':
          return '例如 ap-guangzhou、ap-shanghai';
        case 'qingstor':
          return '例如 pek3a、sh1a（与控制台一致）';
        case 'jd':
          return '例如 cn-north-1';
        case 'ks3':
          return '例如 BEIJING、SHANGHAI（与金山控制台一致）';
        case 'aws':
          return '例如 ap-southeast-1（不填且扩展无 region 时默认 us-east-1）';
        case 'minio':
          return '通常与扩展 JSON 中 region 一致';
        default:
          return '与云控制台地域一致；不确定可查阅厂商文档';
      }
    },
    createBucketRegionHint() {
      const c = this.activeCred;
      if (!c) return '';
      const ex = this.activeCredExtraObj;
      const has = (k) => String(ex[k] || '').trim() !== '';
      if (c.provider === 'tencent') {
        let s =
          'COS 桶访问域名为「名称-APPID.cos.地域」；新建时请填控制台中的完整桶名，或只填短名并在扩展 JSON 配置 appid（纯数字）。';
        if (has('region')) s += ' 已从扩展 JSON 读取 region，地域栏可留空或覆盖。';
        return s;
      }
      if (c.provider === 'qingstor' && has('zone')) {
        return '已从扩展 JSON 读取 zone，可留空或覆盖。';
      }
      if ((c.provider === 'jd' || c.provider === 'ks3') && (has('region') || has('endpoint'))) {
        return '凭据已配置 region 或 endpoint 时，此处可留空。';
      }
      if (c.provider === 'aliyun') {
        return '阿里云未配扩展地域时，创建桶由后端自动探测；仍失败请在扩展 JSON 填写 region。';
      }
      if (c.provider === 'qiniu') {
        return '七牛原生密钥使用扩展 JSON 的 region；且云 S3+STS 时可填 s3_region / 或与列举桶时一致的地域。';
      }
      if (c.provider === 'minio') {
        return 'MinIO 须在扩展 JSON 填写可访问的 endpoint；region 用于部分签名场景。';
      }
      return '';
    },
    filteredCredentials() {
      let list = [...this.credentials];
      if (this.listVendorFilter !== 'all') {
        list = list.filter((c) => c.provider === this.listVendorFilter);
      }
      const k = (this.searchKw || '').trim().toLowerCase();
      if (k) {
        list = list.filter((c) => {
          const blob = [c.name, c.access_key, c.secret_key, c.remark, c.extra_json, c.provider]
            .filter(Boolean)
            .join('\n')
            .toLowerCase();
          return blob.includes(k);
        });
      }
      return list;
    },
    /** 当前页展示的凭据（资产列表分页） */
    pagedCredentials() {
      const list = this.filteredCredentials;
      const start = (this.credAssetPage - 1) * this.credAssetPageSize;
      return list.slice(start, start + this.credAssetPageSize);
    },
    prefixSegments() {
      const p = this.objectPrefix.replace(/^\/+/, '').replace(/\/+$/, '');
      if (!p) return [];
      return p.split('/').filter(Boolean);
    },
    objectRows() {
      return this.listRaw.map((o) => {
        const key = o.key || '';
        const isDir = !!o.is_dir || /\/$/.test(key);
        const display = isDir
          ? `${key.replace(/\/+$/, '').split('/').filter(Boolean).pop() || key}/`
          : key;
        return {
          ...o,
          is_dir: isDir,
          display,
          size: isDir ? '-' : this.fmtSize(o.size),
          size_bytes: isDir ? 0 : Number(o.size) || 0
        };
      });
    },
    explorerGridItems() {
      const folders = this.commonPrefixes
        .filter((p) => !this.folderKeyIsCurrentDirectory(p))
        .map((p) => ({
          key: p,
          display: p.replace(/\/+$/, '').split('/').filter(Boolean).pop() + '/',
          size: '-',
          last_modified: '',
          is_dir: true
        }));
      const files = this.objectRows
        .filter((o) => !(this.isObjectDirItem(o) && this.folderKeyIsCurrentDirectory(o.key)))
        .map((o) => ({ ...o }));
      let list = [...folders, ...files];
      const k = (this.objectSearchKw || '').trim().toLowerCase();
      if (k) {
        list = list.filter((x) => {
          const key = (x.key || '').toLowerCase();
          const disp = (x.display || '').toLowerCase();
          const base = key.replace(/\/+$/, '').split('/').pop() || '';
          return key.includes(k) || disp.includes(k) || base.includes(k);
        });
      }
      list.sort((a, b) => {
        if (!!a.is_dir !== !!b.is_dir) return a.is_dir ? -1 : 1;
        return (a.display || a.key || '').localeCompare(b.display || b.key || '', 'zh-CN');
      });
      return list;
    },
    objectFileCount() {
      return this.objectRows.filter((o) => !o.is_dir).length;
    },
    objectTotalBytes() {
      return this.objectRows.filter((o) => !o.is_dir).reduce((s, o) => s + (Number(o.size_bytes) || Number(o.size) || 0), 0);
    },
    ossObjectPreviewLanguage() {
      const name = String(this.objectFilePreview.name || '').toLowerCase();
      if (!name) return 'plaintext';
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
      if (name.endsWith('.vue')) return 'html';
      return 'plaintext';
    },
    objectTotalSizeLabel() {
      return this.fmtSize(this.objectTotalBytes);
    },
    objectTextPreviewEditorHeight() {
      return this.objectTextPreviewKkfileSrc ? '480px' : '620px';
    }
  },
  watch: {
    loadingObjects(val, old) {
      if (old === true && val === false && this.objectViewMode === 'grid') {
        this.scheduleThumbObservers();
      }
    },
    objectViewMode(val) {
      this.teardownThumbObserver();
      if (val === 'grid') this.scheduleThumbObservers();
    },
    currentBucket() {
      this.thumbUrlByKey = {};
      this.teardownThumbObserver();
    },
    objectPrefix() {
      this.thumbUrlByKey = {};
      this.teardownThumbObserver();
    },
    objectSearchKw() {
      this.$nextTick(() => this.scheduleThumbObservers());
    },
    ossTextPreviewContent: {
      handler() {
        if (this.objectFilePreview.suppressDirty) return;
        if (!this.objectFilePreview.isText || this.objectFilePreview.loading) return;
        this.objectFilePreview.dirty = this.ossTextPreviewContent !== this.ossTextPreviewBaseline;
      }
    },
    searchKw() {
      this.credAssetPage = 1;
      this.$nextTick(() => this.clampCredAssetPage());
    },
    listVendorFilter() {
      this.credAssetPage = 1;
      this.$nextTick(() => this.clampCredAssetPage());
    },
    'credentials.length'() {
      this.$nextTick(() => this.clampCredAssetPage());
    },
    'credForm.provider'(v) {
      if (!this.credDialog || v !== 'minio') return;
      const ep = this.parseEndpointFromExtraJson(this.credForm.extra_json);
      if (ep) this.credForm.minio_endpoint = ep;
    }
  },
  activated() {
    this.updateFullscreenState();
  },
  mounted() {
    this.updateFullscreenState();
    this._resizeHandler = this.updateFullscreenState.bind(this);
    window.addEventListener('resize', this._resizeHandler);

    this.loadCredentials({ silentAuthFail: true });
    this._ctxDocClose = () => {
      this.ctxMenuVisible = false;
    };
    document.addEventListener('click', this._ctxDocClose);
    this._offGlobalFileDrop = EventsOn('global-file-drop', (paths) => {
      if (!paths || !paths.length || !this.isObjectStorageDropActive()) return;
      this.openOssUploadDialog(paths);
    });
  },
  beforeUnmount() {
    if (this._resizeHandler) {
      window.removeEventListener('resize', this._resizeHandler);
      this._resizeHandler = null;
    }
    this.teardownThumbObserver();
    if (this._ctxDocClose) {
      document.removeEventListener('click', this._ctxDocClose);
    }
    if (typeof this._offGlobalFileDrop === 'function') {
      this._offGlobalFileDrop();
      this._offGlobalFileDrop = null;
    }
  },
  methods: {
    updateFullscreenState() {
      if (typeof window === 'undefined' || !window.screen) return;
      const h = window.innerHeight || 0;
      const availH = window.screen.availHeight || window.screen.height || 0;
      this.isFullscreen = h >= availH - 60;
    },
    credRowClassName({ row }) {
      return row && row.id === this.activeCredId ? 'is-cos-active-row' : '';
    },
    noop() {},
    indexMethod(i) {
      return i + 1;
    },
    sortByProvider(a, b) {
      return this.providerLabel(a.provider).localeCompare(this.providerLabel(b.provider), 'zh-CN');
    },
    providerLabel(p) {
      const o = providerOptions.find((x) => x.value === p);
      return o ? o.label : p || '-';
    },
    providerShort(p) {
      const o = providerOptions.find((x) => x.value === p);
      const t = o ? o.tabLabel : p || '?';
      return (t || '?').slice(0, 1);
    },
    bucketAbbr(name) {
      const n = (name || '').trim();
      if (!n) return '?';
      if (n.length <= 2) return n.toUpperCase();
      return n.slice(0, 2).toUpperCase();
    },
    /** 侧栏桶列表当前选中行（与 currentBucket / currentTag 一致） */
    isBucketSidebarActive(b) {
      if (!b) return false;
      const name = b.name != null ? String(b.name) : '';
      const tag = b.tag != null ? String(b.tag) : '';
      return name === (this.currentBucket || '') && tag === (this.currentTag || '');
    },
    maskField(s) {
      if (this.showSecrets) return s || '-';
      if (!s) return '-';
      if (s.length <= 6) return '••••••';
      return `${s.slice(0, 3)}***${s.slice(-3)}`;
    },
    formatTime(t) {
      if (!t) return '-';
      try {
        return dayjs(t).format('YYYY-MM-DD HH:mm:ss');
      } catch {
        return '-';
      }
    },
    fmtSize(n) {
      if (n == null || n === '') return '-';
      const x = Number(n);
      if (Number.isNaN(x)) return String(n);
      if (x < 1024) return `${x} B`;
      if (x < 1024 * 1024) return `${(x / 1024).toFixed(1)} KB`;
      if (x < 1024 * 1024 * 1024) return `${(x / 1024 / 1024).toFixed(1)} MB`;
      return `${(x / 1024 / 1024 / 1024).toFixed(2)} GB`;
    },
    async loadCredentials(opts = {}) {
      const silentAuth = opts.silentAuthFail === true;
      try {
        const s = await ObjectStorageCredentialsJSON();
        const arr = JSON.parse(s || '[]');
        this.credentials = Array.isArray(arr) ? arr : [];
      } catch (e) {
        const msg = e?.message || String(e);
        if (silentAuth && (msg.includes('授权') || msg.includes('云上安全'))) {
          this.credentials = [];
        } else {
          this.$message.error(msg);
        }
      } finally {
        this.$nextTick(() => this.clampCredAssetPage());
      }
    },
    clampCredAssetPage() {
      const total = this.filteredCredentials.length;
      if (total === 0) {
        this.credAssetPage = 1;
        return;
      }
      const maxPage = Math.max(1, Math.ceil(total / this.credAssetPageSize));
      if (this.credAssetPage > maxPage) this.credAssetPage = maxPage;
    },
    onCredAssetPageSizeChange() {
      this.credAssetPage = 1;
      this.$nextTick(() => this.clampCredAssetPage());
    },
    async openExplorer(row) {
      this.activeCredId = row.id;
      this.buckets = [];
      this.currentBucket = '';
      this.currentTag = '';
      this.loadingObjects = false;
      this._objectListFetchGen += 1;
      this.clearObjectState();
      // 不自动收起上方凭据列表：连接后保持「资产列表(上) + 资源浏览器(下)」同屏，
      // 避免用户误以为凭据被遮挡丢失；需要更大浏览空间时点顶部「收起凭据」即可。
      this.$nextTick(() => {
        document.querySelector('.oss-explorer-section')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        this.fetchBuckets();
      });
    },
    openAdd() {
      this.openAddForProvider('');
    },
    nextCredAliasForProvider(providerValue) {
      const pv = providerValue || 'qiniu';
      const opt = providerOptions.find((o) => o.value === pv);
      const vendorLabel = opt ? opt.tabLabel : pv;
      const n = this.credentials.filter((c) => c.provider === pv).length + 1;
      return `${vendorLabel}-${n}`;
    },
    openAddForProvider(provider) {
      this.pendingProvider = provider || '';
      this.resetCredForm();
      if (provider) this.credForm.provider = provider;
      this.credForm.name = this.nextCredAliasForProvider(this.credForm.provider);
      this.credDialog = true;
    },
    openCredDialog(row) {
      if (row) {
        this.credForm = {
          id: row.id,
          name: row.name || '',
          provider: row.provider || 'qiniu',
          access_key: row.access_key || '',
          secret_key: row.secret_key || '',
          session_token: row.session_token || '',
          extra_json: row.extra_json || '',
          remark: row.remark || '',
          minio_endpoint:
            row.provider === 'minio' ? this.parseEndpointFromExtraJson(row.extra_json) : ''
        };
        this.credAuthKind = String(row.session_token || '').trim() ? 'sts' : 'long';
      } else {
        this.openAdd();
        return;
      }
      this.credDialog = true;
    },
    onCredAuthKindChange(val) {
      if (val === 'long') {
        this.credForm.session_token = '';
      }
    },
    onCredKeyPaste(ev) {
      const text = ev.clipboardData?.getData('text/plain');
      const parsed = parseStsCredentialJSON(text);
      if (!parsed) return;
      ev.preventDefault();
      this.credForm.access_key = parsed.access_key;
      this.credForm.secret_key = parsed.secret_key;
      this.credForm.session_token = parsed.session_token;
      this.credAuthKind = 'sts';
      this.$message.success('已识别临时凭证 JSON，已切换为「临时凭证」模式');
    },
    onCredExtraJsonBlur() {
      if (this.credForm.provider !== 'minio') return;
      const ep = this.parseEndpointFromExtraJson(this.credForm.extra_json);
      if (ep) this.credForm.minio_endpoint = ep;
    },
    parseEndpointFromExtraJson(str) {
      try {
        const t = String(str || '').trim();
        if (!t) return '';
        const obj = JSON.parse(t);
        if (obj && typeof obj === 'object' && !Array.isArray(obj) && typeof obj.endpoint === 'string') {
          return obj.endpoint;
        }
      } catch (_) {
        /* ignore */
      }
      return '';
    },
    mergeMinioEndpointIntoExtraJson(extraJson, endpoint) {
      let o = {};
      try {
        const t = String(extraJson || '').trim();
        if (t) {
          const p = JSON.parse(t);
          if (p && typeof p === 'object' && !Array.isArray(p)) o = { ...p };
        }
      } catch (_) {
        o = {};
      }
      const ep = String(endpoint || '').trim();
      if (ep) o.endpoint = ep;
      else delete o.endpoint;
      const keys = Object.keys(o);
      if (keys.length === 0) return '';
      return JSON.stringify(o);
    },
    normalizeCredForApi() {
      const o = { ...this.credForm };
      if (this.credAuthKind === 'long') o.session_token = '';
      if (o.provider === 'minio') {
        o.extra_json = this.mergeMinioEndpointIntoExtraJson(o.extra_json, this.credForm.minio_endpoint);
      }
      delete o.minio_endpoint;
      return o;
    },
    onFilterVendor(cmd) {
      this.listVendorFilter = cmd;
    },
    scrollVendorStrip(dir) {
      const el = this.$refs.vendorStripRef;
      if (!el) return;
      el.scrollBy({ left: dir * 240, behavior: 'smooth' });
    },
    resetCredDialog() {
      if (this.credForm.id) {
        const row = this.credentials.find((c) => c.id === this.credForm.id);
        if (row) {
          this.openCredDialog(row);
          return;
        }
      }
      const p = this.pendingProvider || this.credForm.provider || 'qiniu';
      this.resetCredForm();
      this.credForm.provider = p;
      this.credForm.name = this.nextCredAliasForProvider(p);
      this.credDialog = true;
    },
    async saveCredAndEnter() {
      if (this.credAuthKind === 'sts' && !String(this.credForm.session_token || '').trim()) {
        this.$message.error('临时凭证模式下请填写「会话令牌」');
        return;
      }
      this.savingCred = true;
      try {
        const payload = JSON.stringify(this.normalizeCredForApi());
        try {
          await ObjectStorageProbeCredentialJSON(payload);
        } catch (e) {
          this.$message.error(e?.message || '凭据校验失败，请检查密钥与扩展配置');
          return;
        }
        const prevIds = new Set(this.credentials.map((c) => c.id));
        await ObjectStorageSaveCredentialJSON(payload);
        this.$message.success('已保存');
        this.credDialog = false;
        await this.loadCredentials();
        const newCred = this.credentials.find((c) => !prevIds.has(c.id));
        if (newCred) {
          this.openExplorer(newCred);
        }
      } catch (e) {
        this.$message.error(e?.message || String(e));
      } finally {
        this.savingCred = false;
      }
    },
    onDialogClosed() {
      this.pendingProvider = '';
    },
    resetCredForm() {
      this.credAuthKind = 'long';
      this.credForm = {
        id: 0,
        name: '',
        provider: 'qiniu',
        access_key: '',
        secret_key: '',
        session_token: '',
        extra_json: '',
        remark: '',
        minio_endpoint: ''
      };
    },
    async saveCred() {
      if (this.credAuthKind === 'sts' && !String(this.credForm.session_token || '').trim()) {
        this.$message.error('临时凭证模式下请填写「会话令牌」');
        return;
      }
      this.savingCred = true;
      try {
        await ObjectStorageSaveCredentialJSON(JSON.stringify(this.normalizeCredForApi()));
        this.$message.success('已保存');
        this.credDialog = false;
        await this.loadCredentials();
      } catch (e) {
        this.$message.error(e?.message || String(e));
      } finally {
        this.savingCred = false;
      }
    },
    async removeCred(row) {
      const label = String(row.name || row.id || '');
      try {
        await confirmDangerousAction({
          title: '删除凭据',
          message: `将删除凭据「${label}」，相关本地保存的密钥将不可恢复。`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        await ObjectStorageDeleteCredential(row.id);
        if (this.activeCredId === row.id) {
          this.activeCredId = 0;
          this.buckets = [];
          this.currentBucket = '';
          this.assetPanelCollapsed = false;
          this.clearObjectState();
        }
        this.$message.success('已删除');
        await this.loadCredentials();
      } catch (e) {
        this.$message.error(e?.message || String(e));
      }
    },
    exportAssets() {
      const rows = this.filteredCredentials.map((r) => ({ ...r }));
      const blob = new Blob([JSON.stringify(rows, null, 2)], { type: 'application/json;charset=utf-8' });
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = `object_storage_assets_${dayjs().format('YYYYMMDD_HHmmss')}.json`;
      a.click();
      URL.revokeObjectURL(a.href);
      this.$message.success('已导出当前列表（含密钥，请妥善保管）');
    },
    clearObjectState() {
      this.objectPrefix = '';
      this.listRaw = [];
      this.commonPrefixes = [];
      this.listTruncated = false;
      this.listContinuation = '';
      this.objectCurrentPage = 0;
      this.objectPageSnapshots = [];
      this.thumbUrlByKey = {};
      this.teardownThumbObserver();
    },
    isImageThumbItem(item) {
      if (!item || this.isObjectDirItem(item)) return false;
      const ext = this.fileExt(item);
      return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp'].includes(ext);
    },
    thumbObserveKey(item) {
      if (!item || this.isObjectDirItem(item) || !this.isImageThumbItem(item)) return undefined;
      const sz = Number(item.size_bytes);
      if (Number.isFinite(sz) && sz > 1 * 1024 * 1024) return undefined;
      return item.key || undefined;
    },
    teardownThumbObserver() {
      if (this._thumbObserver) {
        this._thumbObserver.disconnect();
        this._thumbObserver = null;
      }
    },
    scheduleThumbObservers() {
      this.teardownThumbObserver();
      if (this.objectViewMode !== 'grid' || !this.currentBucket || this.loadingObjects) return;
      this.$nextTick(() => {
        const root = this.$refs.objectGridRef;
        if (!root || this.objectViewMode !== 'grid') return;
        const obs = new IntersectionObserver(
          (entries) => {
            entries.forEach((en) => {
              if (!en.isIntersecting) return;
              const k = en.target.getAttribute('data-oss-thumb-key');
              if (!k) return;
              const item = this.explorerGridItems.find((x) => x.key === k && !x.is_dir);
              if (item) this.ensureThumbForItem(item);
              obs.unobserve(en.target);
            });
          },
          { root, rootMargin: '160px', threshold: 0.01 }
        );
        root.querySelectorAll('.ct-file-tile[data-oss-thumb-key]').forEach((el) => obs.observe(el));
        this._thumbObserver = obs;
      });
    },
    onObjectPageSizeChange() {
      if (this.currentBucket) this.fetchObjects(true);
    },
    goObjectPrevPage() {
      if (this.objectCurrentPage <= 0) return;
      this._objectListFetchGen += 1;
      this.objectCurrentPage -= 1;
      const snap = this.objectPageSnapshots[this.objectCurrentPage];
      if (!snap) return;
      this.listRaw = snap.listRaw.map((o) => ({ ...o }));
      this.commonPrefixes = [...snap.commonPrefixes];
      this.listContinuation = snap.continuationNext;
      this.listTruncated = snap.truncated;
      this.$nextTick(() => this.scheduleThumbObservers());
    },
    async goObjectNextPage() {
      if (!this.listTruncated || !this.listContinuation) return;
      const nextCont = this.listContinuation;
      const nextIdx = this.objectCurrentPage + 1;
      const ok = await this._fetchObjectListInternal(nextCont, nextIdx);
      if (ok) this.$nextTick(() => this.scheduleThumbObservers());
    },
    async ensureThumbForItem(item) {
      if (!this.isImageThumbItem(item) || !this.activeCredId || !this.currentBucket) return;
      if (this.thumbUrlByKey[item.key]) return;
      try {
        const url = await ObjectStorageObjectPresignedURL(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          item.key,
          180
        );
        if (!url) return;
        this.thumbUrlByKey = { ...this.thumbUrlByKey, [item.key]: url };
      } catch {
        /* 青云/又拍等无预签名时保留扩展名图标 */
      }
    },
    onThumbError(key) {
      if (!key) return;
      const next = { ...this.thumbUrlByKey };
      delete next[key];
      this.thumbUrlByKey = next;
    },
    isObjectStorageDropActive() {
      return (
        this.activeCredId &&
        this.currentBucket &&
        this.$el &&
        document.body.contains(this.$el) &&
        this.$el.offsetParent !== null
      );
    },
    joinObjectKey(name) {
      const base = (name || '').trim().replace(/^\/+/, '');
      const pre = (this.objectPrefix || '').replace(/\/+$/, '');
      if (!pre) return base;
      return `${pre}/${base}`;
    },
    async putObjectBytes(key, bytesUtf8, contentType) {
      const b64 = btoa(unescape(encodeURIComponent(bytesUtf8)));
      await ObjectStoragePutObjectBase64(
        this.activeCredId,
        this.currentBucket,
        this.currentTag,
        key,
        b64,
        contentType || ''
      );
    },
    async putObjectUint8(key, u8, contentType) {
      const b64 = ossUint8ToBase64(u8);
      await ObjectStoragePutObjectBase64(
        this.activeCredId,
        this.currentBucket,
        this.currentTag,
        key,
        b64,
        contentType || ''
      );
    },
    async uploadSingleLocalPath(localPath, objectKeyOverride) {
      const p = (localPath || '').trim();
      if (!p) return;
      let key = (objectKeyOverride || '').trim();
      if (!key) {
        try {
          key = await ObjectStorageSuggestObjectKey(p, this.objectPrefix.replace(/\/+$/, ''));
        } catch (e) {
          this.$message.error((e && e.message) || '需要完成授权后方可使用云上安全功能');
          return;
        }
      }
      await ObjectStorageUploadLocalFile(
        this.activeCredId,
        this.currentBucket,
        this.currentTag,
        key,
        p
      );
    },
    openOssUploadDialog(initialPaths) {
      if (!this.activeCredId || !this.currentBucket) {
        this.$message.warning('请先选择存储桶');
        return;
      }
      if (!this.ossUploadDialogVisible) {
        this.ossUploadDragOver = false;
        this.ossUploadPercentage = 0;
        this.ossUploadStatus = {
          isUploading: false,
          currentFile: '',
          uploadedFiles: 0,
          totalFiles: 0
        };
        this.ossUploadPaths = [];
      }
      this.addOssUploadPaths(initialPaths);
      this.ossUploadDialogVisible = true;
    },
    addOssUploadPaths(paths) {
      const raw = (paths || []).map((x) => String(x).trim()).filter(Boolean);
      if (!raw.length) return;
      const seen = new Set(this.ossUploadPaths.map((x) => x.path));
      for (const path of raw) {
        if (seen.has(path)) continue;
        seen.add(path);
        const name = path.replace(/\\/g, '/').split('/').filter(Boolean).pop() || path;
        this.ossUploadPaths.push({ path, name });
      }
    },
    pathsFromDataTransferFiles(files) {
      const paths = [];
      if (!files || !files.length) return paths;
      for (let i = 0; i < files.length; i += 1) {
        const f = files[i];
        if (f && f.path) paths.push(f.path);
      }
      return paths;
    },
    onOssUploadDialogDrop(e) {
      this.ossUploadDragOver = false;
      const paths = this.pathsFromDataTransferFiles(e.dataTransfer?.files);
      if (paths.length) this.addOssUploadPaths(paths);
    },
    clearOssUploadList() {
      if (this.ossUploadStatus.isUploading) return;
      this.ossUploadPaths = [];
    },
    async pickFilesForOssUpload() {
      if (this.ossUploadStatus.isUploading) return;
      try {
        const p = await ObjectStoragePickLocalFile();
        if (p) this.addOssUploadPaths([p]);
      } catch (err) {
        this.$message.error(err?.message || String(err));
      }
    },
    closeOssUploadDialog() {
      if (this.ossUploadStatus.isUploading) return;
      this.ossUploadDialogVisible = false;
    },
    onOssUploadDialogClosed() {
      this.ossUploadDragOver = false;
      if (this.ossUploadStatus.isUploading) return;
      this.ossUploadPaths = [];
      this.ossUploadPercentage = 0;
    },
    async startOssUploadFromDialog() {
      if (!this.activeCredId || !this.currentBucket || !this.ossUploadPaths.length) return;
      const list = [...this.ossUploadPaths];
      this.ossUploadStatus = {
        isUploading: true,
        currentFile: '',
        uploadedFiles: 0,
        totalFiles: list.length
      };
      this.ossUploadPercentage = 0;
      try {
        for (let i = 0; i < list.length; i += 1) {
          const { path, name } = list[i];
          this.ossUploadStatus.currentFile = name;
          await this.uploadSingleLocalPath(path, '');
          this.ossUploadStatus.uploadedFiles = i + 1;
          this.ossUploadPercentage = Math.round(((i + 1) / list.length) * 100);
        }
        this.$message.success(`已上传 ${list.length} 个文件`);
        this.ossUploadPaths = [];
        this.fetchObjects(true);
        this.ossUploadDialogVisible = false;
      } catch (e) {
        this.$message.error(e?.message || String(e));
      } finally {
        this.ossUploadStatus.isUploading = false;
        this.ossUploadStatus.currentFile = '';
        this.ossUploadPercentage = 0;
        this.ossUploadStatus.uploadedFiles = 0;
        this.ossUploadStatus.totalFiles = 0;
      }
    },
    onObjectDragOver() {
      if (!this.currentBucket) return;
      this.objectDropOver = true;
    },
    onObjectDragLeave() {
      this.objectDropOver = false;
    },
    onObjectAreaDrop(e) {
      this.objectDropOver = false;
      if (!this.activeCredId || !this.currentBucket) return;
      const paths = this.pathsFromDataTransferFiles(e.dataTransfer?.files);
      if (paths.length) this.openOssUploadDialog(paths);
    },
    onObjectListSurfaceContextMenu(e) {
      const t = e.target;
      if (t.closest('.ct-file-tile') || t.closest('.el-table__row') || t.closest('.el-table__header')) {
        return;
      }
      if (!this.currentBucket) return;
      this.ctxMenuMode = 'blank';
      this.ctxTargetRow = null;
      this.ctxBucketRow = null;
      this.ctxMenuX = e.clientX;
      this.ctxMenuY = e.clientY;
      this.ctxMenuVisible = true;
    },
    onBucketContextMenu(e, row) {
      this.ctxMenuMode = 'bucket';
      this.ctxBucketRow = row;
      this.ctxTargetRow = null;
      this.ctxMenuX = e.clientX;
      this.ctxMenuY = e.clientY;
      this.ctxMenuVisible = true;
    },
    resetCreateBucketForm() {
      this.createBucketForm.name = '';
      this.createBucketForm.regionTag = '';
      this.createBucketSubmitting = false;
    },
    async promptCreateBucket() {
      if (!this.activeCredId) {
        this.$message.warning('请先选择一条凭据');
        return;
      }
      if (this.createBucketUnsupportedReason) {
        this.$message.warning(this.createBucketUnsupportedReason);
        return;
      }
      this.createBucketForm.name = '';
      this.createBucketForm.regionTag = this.createBucketPrefillRegionTag;
      this.createBucketDialogVisible = true;
    },
    async submitCreateBucket() {
      const name = (this.createBucketForm.name || '').trim();
      if (!name) {
        this.$message.warning('请输入存储桶名称');
        return;
      }
      const tag = (this.createBucketForm.regionTag || '').trim();
      if (this.createBucketRegionRequired && !tag) {
        this.$message.warning(this.createBucketRegionMissingMsg);
        return;
      }
      if (!this.activeCredId) return;
      this.createBucketSubmitting = true;
      try {
        await ObjectStorageCreateBucket(this.activeCredId, name, tag);
        this.$message.success('已创建');
        this.createBucketDialogVisible = false;
        await this.fetchBuckets();
      } catch (err) {
        this.$message.error(err?.message || String(err));
      } finally {
        this.createBucketSubmitting = false;
      }
    },
    resetDeleteBucketDialog() {
      this.deleteBucketConfirmInput = '';
      this.deleteBucketTarget = null;
      this.deleteBucketSubmitting = false;
    },
    async submitDeleteBucket() {
      const row = this.deleteBucketTarget;
      if (!row || !this.activeCredId) return;
      const typed = (this.deleteBucketConfirmInput || '').trim();
      if (typed !== this.deleteBucketRequiredPhrase) {
        this.$message.error('须与确认语完全一致（逐字手动输入，请勿复制粘贴）');
        return;
      }
      this.deleteBucketSubmitting = true;
      try {
        await ObjectStorageDeleteBucket(this.activeCredId, row.name, row.tag || '');
        this.$message.success('已删除');
        this.deleteBucketDialogVisible = false;
        if (this.currentBucket === row.name && this.currentTag === (row.tag || '')) {
          this.currentBucket = '';
          this.currentTag = '';
          this.clearObjectState();
        }
        await this.fetchBuckets();
      } catch (e) {
        this.$message.error(e?.message || String(e));
      } finally {
        this.deleteBucketSubmitting = false;
      }
    },
    async ctxDeleteBucket() {
      const row = this.ctxBucketRow;
      this.ctxMenuVisible = false;
      if (!row || !this.activeCredId) return;
      this.deleteBucketTarget = { name: row.name, tag: row.tag || '' };
      this.deleteBucketConfirmInput = '';
      this.deleteBucketDialogVisible = true;
    },
    async ctxNewFolder() {
      this.ctxMenuVisible = false;
      if (!this.activeCredId || !this.currentBucket) return;
      try {
        const { value } = await ElMessageBox.prompt('文件夹名称（将创建为对象键后缀 /）', '新建文件夹', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputPattern: /^[^/\\]+$/,
          inputErrorMessage: '不要包含路径分隔符'
        });
        const raw = (value || '').trim();
        if (!raw) return;
        const key = `${this.joinObjectKey(raw).replace(/\/+$/, '')}/`;
        this.uploading = true;
        try {
          await ObjectStoragePutObjectBase64(
            this.activeCredId,
            this.currentBucket,
            this.currentTag,
            key,
            '',
            ''
          );
          this.$message.success('已创建');
          this.fetchObjects(true);
        } finally {
          this.uploading = false;
        }
      } catch (e) {
        if (e !== 'cancel') this.$message.error(e?.message || String(e));
      }
    },
    async ctxNewTextFile() {
      this.ctxMenuVisible = false;
      if (!this.activeCredId || !this.currentBucket) return;
      try {
        const { value: name } = await ElMessageBox.prompt('文件名（含扩展名，如 notes.txt）', '新建文本文件', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputValue: '新建.txt',
          inputPattern: /\S+/,
          inputErrorMessage: '文件名不能为空'
        });
        const fn = (name || '').trim();
        if (!fn) return;
        const key = this.joinObjectKey(fn);
        this.uploading = true;
        try {
          await this.putObjectBytes(key, '', 'text/plain; charset=utf-8');
          this.$message.success('已创建空文件');
          this.fetchObjects(true);
        } finally {
          this.uploading = false;
        }
      } catch (e) {
        if (e !== 'cancel') this.$message.error(e?.message || String(e));
      }
    },
    async ctxPickAndUpload() {
      this.ctxMenuVisible = false;
      if (!this.activeCredId || !this.currentBucket) return;
      try {
        const p = await ObjectStoragePickLocalFile();
        if (p) this.openOssUploadDialog([p]);
      } catch (e) {
        this.$message.error(e?.message || String(e));
      }
    },
    async fetchBuckets() {
      if (!this.activeCredId) {
        this.$message.warning('请先选择一条凭据');
        return;
      }
      this.loadingBuckets = true;
      try {
        const s = await ObjectStorageListBucketsJSON(this.activeCredId);
        const arr = JSON.parse(s || '[]');
        this.buckets = Array.isArray(arr) ? arr : [];
        if (this.buckets.length === 0) this.$message.info('未返回存储桶');
      } catch (e) {
        this.$message.error(e?.message || String(e));
      } finally {
        this.loadingBuckets = false;
      }
    },
    onBucketRow(row) {
      if (!row) return;
      this.currentBucket = row.name != null ? String(row.name) : '';
      this.currentTag = row.tag != null ? String(row.tag) : '';
      this.objectPrefix = '';
      this.loadingObjects = false;
      this.fetchObjects(true);
    },
    setPrefix(p) {
      this.objectPrefix = p || '';
      this.fetchObjects(true);
    },
    setPrefixByIndex(i) {
      const segs = this.prefixSegments.slice(0, i + 1);
      this.objectPrefix = segs.length ? `${segs.join('/')}/` : '';
      this.fetchObjects(true);
    },
    enterCommonPrefix(p) {
      this.objectPrefix = p.replace(/^\//, '');
      if (this.objectPrefix && !this.objectPrefix.endsWith('/')) this.objectPrefix += '/';
      this.fetchObjects(true);
    },
    async _fetchObjectListInternal(continuation, pageIndex, showLoadingOverlay = true) {
      if (!this.activeCredId || !this.currentBucket) return false;
      const gen = ++this._objectListFetchGen;
      if (showLoadingOverlay) {
        this.loadingObjects = true;
      }
      try {
        const s = await ObjectStorageListObjectsJSON(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          this.objectPrefix,
          continuation,
          this.objectPageSize
        );
        if (gen !== this._objectListFetchGen) return false;
        const data = JSON.parse(s || '{}');
        const objs = Array.isArray(data.objects) ? data.objects : [];
        const cps = Array.isArray(data.common_prefixes) ? data.common_prefixes : [];
        this.objectCurrentPage = pageIndex;
        this.listRaw = objs.map((o) => ({ ...o }));
        this.commonPrefixes = [...cps];
        this.listTruncated = !!data.truncated;
        this.listContinuation = data.next_marker || '';
        this.objectPageSnapshots[pageIndex] = {
          listRaw: this.listRaw.map((o) => ({ ...o })),
          commonPrefixes: [...this.commonPrefixes],
          continuationNext: this.listContinuation,
          truncated: this.listTruncated
        };
        return true;
      } catch (e) {
        if (gen === this._objectListFetchGen) {
          this.$message.error(e?.message || String(e));
        }
        return false;
      } finally {
        if (gen === this._objectListFetchGen && showLoadingOverlay) {
          this.loadingObjects = false;
        }
      }
    },
    async fetchObjects(reset, opts = {}) {
      if (!this.activeCredId || !this.currentBucket) return;
      if (!reset) return;
      const silent = !!opts.silent;
      this.objectCurrentPage = 0;
      this.objectPageSnapshots = [];
      const ok = await this._fetchObjectListInternal('', 0, !silent);
      if (ok) this.$nextTick(() => this.scheduleThumbObservers());
    },
    onObjectContextMenu(e, row) {
      this.ctxMenuMode = 'item';
      this.ctxBucketRow = null;
      this.ctxTargetRow = row;
      this.ctxMenuX = e.clientX;
      this.ctxMenuY = e.clientY;
      this.ctxMenuVisible = true;
    },
    ctxOpenPrefix() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      if (!row || !this.isObjectDirItem(row)) return;
      this.enterCommonPrefix(row.key);
    },
    folderObjectKeyForDelete(key) {
      let k = String(key || '').replace(/^\/+/, '');
      if (!k) return '';
      if (!k.endsWith('/')) k += '/';
      return k;
    },
    async deleteFolderByItem(item) {
      if (!item || !this.isObjectDirItem(item) || !this.activeCredId || !this.currentBucket) return;
      if (this.folderKeyIsCurrentDirectory(item.key)) return;
      const objectKey = this.folderObjectKeyForDelete(item.key);
      if (!objectKey) return;
      const label = (item.display || objectKey).trim();
      try {
        await confirmDangerousAction({
          title: '删除文件夹',
          message: `将删除文件夹「${label}」。仅当该目录下已无任何对象或子文件夹时可成功；请先清空内部文件后再删。`,
          expectedInput: DANGER_DELETE_TOKEN,
          confirmButtonText: '删除'
        });
      } catch {
        return;
      }
      try {
        await ObjectStorageDeleteObject(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          objectKey
        );
        this.$message.success('已删除');
        this.fetchObjects(true);
      } catch (e) {
        this.$message.error(e?.message || String(e));
      }
    },
    ctxDeleteFolder() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      this.deleteFolderByItem(row);
    },
    async forceDeleteFolderByItem(item) {
      if (!item || !this.isObjectDirItem(item) || !this.activeCredId || !this.currentBucket) return;
      if (this.folderKeyIsCurrentDirectory(item.key)) return;
      const objectKey = this.folderObjectKeyForDelete(item.key);
      if (!objectKey) return;
      const label = (item.display || objectKey).trim();
      try {
        await this.$confirm(
          `「${label}」下的全部对象与子文件夹将被永久删除，不可恢复。`,
          '强制删除（第 1 步）',
          { type: 'warning', confirmButtonText: '继续', cancelButtonText: '取消' }
        );
      } catch {
        return;
      }
      try {
        await confirmDangerousAction({
          title: '强制删除（第 2 步）',
          message: '此操作不可撤销，将递归删除该路径下所有文件与目录占位。',
          expectedInput: '确认强制删除',
          confirmButtonText: '确认强制删除',
          type: 'error'
        });
      } catch {
        return;
      }
      try {
        await ObjectStorageDeletePrefixRecursive(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          objectKey
        );
        this.$message.success('已强制删除该目录下全部对象');
        this.fetchObjects(true);
      } catch (e) {
        this.$message.error(e?.message || String(e));
      }
    },
    ctxForceDeleteFolder() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      this.forceDeleteFolderByItem(row);
    },
    onTableRowContextMenu(row, _column, e) {
      if (e && e.preventDefault) e.preventDefault();
      if (e && e.stopPropagation) e.stopPropagation();
      this.onObjectContextMenu(e, row);
    },
    isObjectDirItem(item) {
      if (!item) return false;
      if (item.is_dir) return true;
      const k = item.key || '';
      return /\/$/.test(k);
    },
    /** 进入某目录后，列表里不应再出现与当前 objectPrefix 相同的「本层目录」占位项 */
    folderKeyIsCurrentDirectory(key) {
      let p = (this.objectPrefix || '').replace(/^\/+/, '');
      if (p && !p.endsWith('/')) p += '/';
      let k = (key || '').replace(/^\/+/, '');
      if (k && !k.endsWith('/')) k += '/';
      if (!p || !k) return false;
      return k === p;
    },
    onExplorerItemDblclick(item) {
      if (!item) return;
      if (this.isObjectDirItem(item)) {
        this.enterCommonPrefix(item.key);
        return;
      }
      const mk = this.mediaPreviewKind(item);
      if (mk) {
        this.openObjectMediaPreviewFromItem(item);
        return;
      }
      this.openObjectTextPreviewFromItem(item);
    },
    onObjectTableDblclick(row) {
      this.onExplorerItemDblclick(row);
    },
    fileExt(item) {
      const k = (item.key || '').replace(/\/+$/, '');
      if (!k || item.is_dir) return '';
      const base = k.split('/').pop() || '';
      const i = base.lastIndexOf('.');
      if (i <= 0 || i === base.length - 1) return '';
      return base.slice(i + 1).toLowerCase();
    },
    fileIconText(item) {
      if (item.is_dir) return 'DIR';
      const ext = this.fileExt(item);
      if (!ext) return 'FILE';
      return ext.slice(0, 4).toUpperCase();
    },
    mediaPreviewKind(item) {
      if (!item || this.isObjectDirItem(item)) return '';
      const ext = this.fileExt(item);
      if (['mp4', 'webm', 'ogv', 'mov', 'm4v', 'avi', 'mkv'].includes(ext)) return 'video';
      if (['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac', 'aac', 'opus'].includes(ext)) return 'audio';
      return '';
    },
    async openObjectMediaPreviewFromItem(item) {
      const key = item?.key;
      const kind = this.mediaPreviewKind(item);
      if (!key || !kind || !this.activeCredId || !this.currentBucket) return;
      const baseName = key.replace(/\/+$/, '').split('/').filter(Boolean).pop() || key;
      this.objectMediaPreviewVisible = true;
      this.objectMediaPreviewKind = kind;
      this.objectMediaPreviewTitle = baseName;
      this.objectMediaPreviewSrc = '';
      this.objectMediaPreviewLoading = true;
      try {
        const url = await ObjectStorageObjectPresignedURL(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          key,
          3600
        );
        if (!url) {
          this.$message.warning('当前凭证不支持预签名 URL，无法在面板内播放');
          this.objectMediaPreviewVisible = false;
          return;
        }
        this.objectMediaPreviewSrc = url;
      } catch (e) {
        this.$message.error(ossHumanizeObjectError(e?.message || String(e)));
        this.objectMediaPreviewVisible = false;
      } finally {
        this.objectMediaPreviewLoading = false;
      }
    },
    onObjectMediaPreviewClosed() {
      this.objectMediaPreviewSrc = '';
      this.objectMediaPreviewKind = '';
      this.objectMediaPreviewTitle = '';
      this.objectMediaPreviewLoading = false;
    },
    openObjectMediaInBrowser() {
      if (this.objectMediaPreviewSrc) BrowserOpenURL(this.objectMediaPreviewSrc);
    },
    onObjectMediaPlaybackError() {
      this.$message.error(
        ossHumanizeObjectError(
          "InvalidObjectState: The operation is not valid for the object's state"
        )
      );
    },
    fileIconClass(item) {
      if (item.is_dir) return 'is-folder';
      const ext = this.fileExt(item);
      if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg'].includes(ext)) return 'is-img';
      if (['mp4', 'webm', 'ogv', 'mov', 'm4v', 'avi', 'mkv'].includes(ext)) return 'is-video';
      if (['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac', 'aac', 'opus'].includes(ext)) return 'is-audio';
      if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return 'is-zip';
      if (['pdf'].includes(ext)) return 'is-pdf';
      if (['doc', 'docx'].includes(ext)) return 'is-doc';
      if (['xls', 'xlsx', 'csv'].includes(ext)) return 'is-xls';
      if (['ppt', 'pptx'].includes(ext)) return 'is-ppt';
      if (['js', 'ts', 'vue', 'jsx', 'tsx'].includes(ext)) return 'is-js';
      if (['go', 'py', 'java', 'c', 'cpp', 'h'].includes(ext)) return 'is-code';
      if (['json', 'xml', 'yaml', 'yml'].includes(ext)) return 'is-data';
      return 'is-file';
    },
    openKkfileConfig() {
      const saved = localStorage.getItem('kkfilePreviewURL');
      if (saved) this.kkfilePreviewURL = saved;
      this.kkfileConfigVisible = true;
    },
    saveKkfileTpl() {
      if (!this.kkfilePreviewURL.includes('{base64(url)}')) {
        this.$message.warning('地址须包含 {base64(url)}');
        return;
      }
      localStorage.setItem('kkfilePreviewURL', this.kkfilePreviewURL);
      this.kkfileConfigVisible = false;
      this.$message.success('已保存');
    },
    resetKkfileTpl() {
      this.kkfilePreviewURL = 'https://file.kkview.cn/onlinePreview?url={base64(url)}&key=000';
    },
    async ctxKkfilePreview() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      if (!row || this.isObjectDirItem(row)) return;
      await this.openKkfilePreviewForItem(row);
    },
    kkfileTemplateOrWarn() {
      let tpl = this.kkfilePreviewURL;
      const saved = localStorage.getItem('kkfilePreviewURL');
      if (saved) tpl = saved;
      if (!tpl.includes('{base64(url)}')) {
        this.$message.warning('请先在「kkfile 设置」中配置含 {base64(url)} 的预览地址');
        return '';
      }
      return tpl;
    },
    async openKkfilePreviewForItem(item, options) {
      if (!item || this.isObjectDirItem(item) || !this.activeCredId || !this.currentBucket) return;
      await this.openKkfilePreviewForKey(item.key, options);
    },
    async openKkfilePreviewForKey(objectKey, options = {}) {
      const { cacheBust = false } = options;
      const ok = String(objectKey || '').trim();
      if (!ok || /\/$/.test(ok) || !this.activeCredId || !this.currentBucket) return;
      const tpl = this.kkfileTemplateOrWarn();
      if (!tpl) return;
      try {
        const fileUrl = await ObjectStorageObjectPresignedURL(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          ok,
          3600
        );
        const enc = btoa(unescape(encodeURIComponent(fileUrl)));
        const urlParam = encodeURIComponent(enc);
        let finalUrl = tpl.replace('{base64(url)}', urlParam);
        if (cacheBust) {
          finalUrl += (finalUrl.includes('?') ? '&' : '?') + '_oss_ts=' + Date.now();
        }
        this.kkfilePreviewObjectKey = ok;
        this.kkfilePreviewSrc = finalUrl;
        this.kkfilePreviewVisible = true;
      } catch (e) {
        this.$message.error(ossHumanizeObjectError(e?.message || String(e)));
      }
    },
    onKkfilePreviewClosed() {
      this.kkfilePreviewSrc = '';
      this.kkfilePreviewObjectKey = '';
    },
    async downloadKkfilePreviewTarget() {
      const key = this.kkfilePreviewObjectKey;
      if (!key || !this.activeCredId || !this.currentBucket) return;
      try {
        await ObjectStorageDownloadObject(this.activeCredId, this.currentBucket, this.currentTag, key);
        this.$message.success('下载完成');
      } catch (e) {
        const m = e?.message || String(e);
        if (m.includes('user_cancelled')) return;
        this.$message.error(m);
      }
    },
    async pickOverwriteKkfilePreviewTarget() {
      const key = this.kkfilePreviewObjectKey;
      if (!key || !this.activeCredId || !this.currentBucket) return;
      try {
        const p = await ObjectStoragePickLocalFile();
        if (!p) return;
        this.uploading = true;
        await ObjectStorageUploadLocalFile(this.activeCredId, this.currentBucket, this.currentTag, key, p);
        this.$message.success('已用本地文件覆盖云端对象');
        this.fetchObjects(true);
        await this.openKkfilePreviewForKey(key, { cacheBust: true });
      } catch (e) {
        this.$message.error(e?.message || String(e));
      } finally {
        this.uploading = false;
      }
    },
    openKkfileInBrowser() {
      if (this.kkfilePreviewSrc) {
        BrowserOpenURL(this.kkfilePreviewSrc);
      }
    },
    closeObjectTextPreview() {
      this.objectTextPreviewVisible = false;
    },
    onObjectTextPreviewDialogClosed() {
      this.objectTextEditorExpanded = true;
      this.objectTextPreviewKkfileSrc = '';
      this.ossTextPreviewContent = '';
      this.ossTextPreviewBaseline = '';
      this.objectFilePreview = {
        key: '',
        name: '',
        sizeBytes: 0,
        loading: false,
        isText: false,
        error: '',
        dirty: false,
        suppressDirty: false,
        largeFileWarn: false,
        binaryLatin1: false
      };
    },
    async kkfileEmbedUrlForKey(objectKey) {
      const ok = String(objectKey || '').trim();
      if (!ok || /\/$/.test(ok) || !this.activeCredId || !this.currentBucket) return '';
      let tpl = this.kkfilePreviewURL;
      const saved = localStorage.getItem('kkfilePreviewURL');
      if (saved) tpl = saved;
      if (!tpl.includes('{base64(url)}')) return '';
      try {
        const fileUrl = await ObjectStorageObjectPresignedURL(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          ok,
          3600
        );
        const enc = btoa(unescape(encodeURIComponent(fileUrl)));
        const urlParam = encodeURIComponent(enc);
        return tpl.replace('{base64(url)}', urlParam);
      } catch {
        return '';
      }
    },
    async fillTextPreviewKkfileSrc(objectKey, cacheBust = false) {
      let u = await this.kkfileEmbedUrlForKey(objectKey);
      if (this.objectFilePreview.key !== objectKey) return;
      if (u && cacheBust) {
        u += (u.includes('?') ? '&' : '?') + '_oss_ts=' + Date.now();
      }
      this.objectTextPreviewKkfileSrc = u || '';
    },
    openTextPreviewKkfileInBrowser() {
      if (this.objectTextPreviewKkfileSrc) {
        BrowserOpenURL(this.objectTextPreviewKkfileSrc);
      }
    },
    clearTextPreviewKkfile() {
      this.objectTextPreviewKkfileSrc = '';
    },
    async showKkfileInTextPreview() {
      const key = this.objectFilePreview.key;
      if (!key || !this.activeCredId || !this.currentBucket) return;
      const tpl = this.kkfileTemplateOrWarn();
      if (!tpl) return;
      await this.fillTextPreviewKkfileSrc(key);
    },
    async openObjectTextPreviewFromItem(item) {
      if (!item || this.isObjectDirItem(item) || !this.activeCredId || !this.currentBucket) return;
      const key = item.key;
      const baseName = key.replace(/\/+$/, '').split('/').filter(Boolean).pop() || key;
      const sz = Number(item.size_bytes);
      const largeWarn = Number.isFinite(sz) && sz > 5 * 1024 * 1024;
      this.objectTextEditorExpanded = true;
      this.objectTextPreviewKkfileSrc = '';
      this.ossTextPreviewContent = '';
      this.ossTextPreviewBaseline = '';
      this.objectFilePreview = {
        key,
        name: baseName,
        sizeBytes: Number.isFinite(sz) ? sz : 0,
        loading: true,
        isText: false,
        error: '',
        dirty: false,
        suppressDirty: true,
        largeFileWarn: largeWarn,
        binaryLatin1: false
      };
      this.objectTextPreviewVisible = true;
      try {
        const b64 = await ObjectStorageGetObjectBase64(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          key
        );
        const bytes = ossBase64ToUint8Array(b64);
        let text;
        let binaryLatin1 = false;
        try {
          text = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
        } catch {
          text = ossBytesToLatin1(bytes);
          binaryLatin1 = true;
        }
        if (this.objectFilePreview.key !== key) return;
        this.objectFilePreview.isText = true;
        this.objectFilePreview.binaryLatin1 = binaryLatin1;
        this.objectFilePreview.error = '';
        this.ossTextPreviewContent = text;
        this.ossTextPreviewBaseline = text;
        this.objectFilePreview.dirty = false;
        this.$nextTick(() => {
          this.objectFilePreview.suppressDirty = false;
        });
      } catch (e) {
        if (this.objectFilePreview.key === key) {
          this.objectFilePreview.error = ossHumanizeObjectError(e?.message || String(e) || '预览失败');
        }
      } finally {
        if (this.objectFilePreview.key === key) {
          this.objectFilePreview.loading = false;
        }
      }
    },
    ctxEditObject() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      if (!row || this.isObjectDirItem(row)) return;
      const mk = this.mediaPreviewKind(row);
      if (mk) {
        this.openObjectMediaPreviewFromItem(row);
        return;
      }
      this.openObjectTextPreviewFromItem(row);
    },
    async saveObjectTextPreview() {
      const key = this.objectFilePreview.key;
      if (!key || !this.objectFilePreview.isText || !this.objectFilePreview.dirty) return;
      try {
        if (this.objectFilePreview.binaryLatin1) {
          await this.putObjectUint8(key, ossLatin1ToBytes(this.ossTextPreviewContent), 'application/octet-stream');
        } else {
          await this.putObjectBytes(key, this.ossTextPreviewContent, 'text/plain; charset=utf-8');
        }
        this.$message.success('已保存');
        this.objectFilePreview.suppressDirty = true;
        this.ossTextPreviewBaseline = this.ossTextPreviewContent;
        this.objectFilePreview.dirty = false;
        this.$nextTick(() => {
          this.objectFilePreview.suppressDirty = false;
        });
        // 静默刷新列表，避免 v-loading 遮罩在编辑弹窗打开时像「空白弹框」一闪
        this.fetchObjects(true, { silent: true });
        if (this.objectTextPreviewKkfileSrc) {
          await this.fillTextPreviewKkfileSrc(key, true);
        }
      } catch (e) {
        this.$message.error(e?.message || String(e));
      }
    },
    async ctxDownload() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      if (!row || this.isObjectDirItem(row) || !this.activeCredId || !this.currentBucket) return;
      try {
        await ObjectStorageDownloadObject(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          row.key
        );
        this.$message.success('下载完成');
      } catch (e) {
        const m = e?.message || String(e);
        if (m.includes('user_cancelled')) return;
        this.$message.error(m);
      }
    },
    async ctxRename() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      if (!row || this.isObjectDirItem(row) || !this.activeCredId || !this.currentBucket) return;
      try {
        const { value } = await ElMessageBox.prompt('新的对象键（完整路径）', '重命名', {
          inputValue: row.key,
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        });
        const nk = (value || '').trim();
        if (!nk || nk === row.key) return;
        await ObjectStorageRenameObject(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          row.key,
          nk
        );
        this.$message.success('已重命名');
        this.fetchObjects(true);
      } catch (e) {
        if (e !== 'cancel') this.$message.error(e?.message || String(e));
      }
    },
    async ctxDeleteObject() {
      const row = this.ctxTargetRow;
      this.ctxMenuVisible = false;
      if (!row || this.isObjectDirItem(row) || !this.activeCredId || !this.currentBucket) return;
      try {
        await confirmDangerousAction({
          title: '删除对象',
          message: `将永久删除对象「${row.key}」。`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        await ObjectStorageDeleteObject(
          this.activeCredId,
          this.currentBucket,
          this.currentTag,
          row.key
        );
        this.$message.success('已删除');
        this.fetchObjects(true);
      } catch (e) {
        this.$message.error(e?.message || String(e));
      }
    }
  }
};
</script>

<style scoped>
/* 与 OssBucketPanel 云存储模块一致（对齐 cloudTools 说明中的云存储工具布局） */
.oss-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  min-height: 0;
  width: 100%;
  max-width: 100%;
  padding: 0 2px 0;
  box-sizing: border-box;
  background: #fff;
}

.oss-url-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.url-input-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.cos-panel-title-row {
  align-items: flex-start;
}

.url-input-header h4 {
  margin: 0 0 6px;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.panel-subtitle {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: #909399;
  max-width: 42rem;
}

.batch-stats {
  display: flex;
  gap: 20px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 6px;
  flex-wrap: wrap;
}

.cos-stats-inline {
  margin: 0;
}

.stat-item {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.stat-item strong {
  color: #409eff;
  font-weight: 600;
}

.function-select {
  margin-bottom: 0;
}

.cos-vendor-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 16px;
}

.field-label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
  flex-shrink: 0;
}

.cos-vendor-row :deep(.el-radio-group) {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.url-input-group {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  flex-wrap: wrap;
}

.url-input {
  flex: 1 1 240px;
  min-width: 0;
}

.cos-toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.search-ico {
  color: #909399;
}

.btn-ico {
  margin-right: 4px;
  vertical-align: middle;
}

.config-tip {
  margin-top: 0;
  margin-bottom: 0;
}

.cos-config-tip :deep(.el-alert__description) {
  margin-top: 0;
  line-height: 1.55;
  font-size: 13px;
}

/* 凭据列表 / 资源浏览器外层（同 OssBucket csv-table-section） */
.csv-table-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.csv-table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.csv-table-header h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.explorer-panel-header {
  align-items: flex-start;
}

.explorer-title-block {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.explorer-title-block h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.explorer-title-ico {
  font-size: 22px;
  color: #409eff;
  flex-shrink: 0;
  margin-top: 2px;
}

.explorer-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: #909399;
}

.explorer-sub .dot {
  margin: 0 6px;
  color: #c0c4cc;
}

.oss-explorer-section .explorer-grid {
  margin: 0 -20px -20px;
}

.cos-asset-table {
  width: 100%;
}

.cos-asset-table :deep(.el-table__empty-block) {
  min-height: 0 !important;
  padding: 28px 16px 32px;
}

.cos-asset-table :deep(.el-table__body-wrapper) {
  min-height: 0;
}

.cos-asset-table :deep(th.el-table__cell) {
  font-weight: 600;
  font-size: 13px;
  color: #606266;
  background: #fafafa !important;
}

.cos-asset-table :deep(.el-table__row) {
  cursor: pointer;
}

.cos-asset-table :deep(.el-table__row:hover > td) {
  background: #f5f7fa !important;
}

.cos-asset-table :deep(.el-table__row.is-cos-active-row > td) {
  background: #ecf5ff !important;
  box-shadow: inset 3px 0 0 #409eff;
}

.op-link {
  font-weight: 500;
}

.explorer-grid {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(320px, 1.6fr);
  gap: 0;
  min-height: 0;
  background: #fff;
  border-top: 1px solid #ebeef5;
}

@media (max-width: 900px) {
  .explorer-grid {
    grid-template-columns: 1fr;
  }
}

.explorer-pane {
  padding: 14px 16px 18px;
  min-height: 200px;
}

.buckets-pane {
  border-right: 1px solid #ebeef5;
  background: #fff;
}

@media (max-width: 900px) {
  .buckets-pane {
    border-right: none;
    border-bottom: 1px solid #ebeef5;
  }
}

.objects-pane {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: #fff;
}

.objects-pane.is-idle {
  justify-content: center;
  align-items: center;
  min-height: 280px;
}

.pane-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}

.pane-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.pane-meta {
  font-size: 12px;
  color: #909399;
  text-align: right;
  word-break: break-all;
}

.inner-table {
  width: 100%;
}

.inner-table :deep(th.el-table__cell) {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  background: #fafafa !important;
}

.path-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #fff;
  border: 1px solid #ebeef5;
}

.breadcrumb-wrap {
  flex: 1 1 200px;
  min-width: 0;
}

.crumb-btn {
  padding: 0 2px;
  font-weight: 500;
}

.obj-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.prefixes {
  margin-top: 4px;
}

.prefix-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: #909399;
  margin-bottom: 8px;
}

.prefix-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.prefix-chip {
  appearance: none;
  border: 1px solid #dcdfe6;
  background: #fff;
  color: #606266;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
  font-family: inherit;
}

.prefix-chip:hover {
  color: #409eff;
  border-color: #c6e2ff;
  background: #ecf5ff;
}

.objects-placeholder {
  text-align: center;
  padding: 32px 20px;
  color: #909399;
}

.objects-placeholder p {
  margin: 12px 0 0;
  font-size: 13px;
}

.placeholder-ico {
  font-size: 48px;
  color: #dcdfe6;
}

@media (max-width: 768px) {
  .cos-panel-title-row.url-input-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .csv-table-header {
    flex-direction: column;
    align-items: flex-start;
  }
}

@media (max-width: 480px) {
  .oss-url-section,
  .csv-table-section {
    padding: 16px;
  }

  .oss-explorer-section .explorer-grid {
    margin: 0 -16px -16px;
  }
}

:deep(.cos-dialog.el-dialog) {
  border-radius: 8px;
}

:deep(.cos-dialog .el-dialog__header) {
  padding: 16px 20px 12px;
  margin: 0;
  border-bottom: 1px solid #ebeef5;
}

:deep(.cos-dialog .el-dialog__title) {
  font-weight: 600;
  font-size: 16px;
  color: #303133;
}

:deep(.cos-dialog .el-dialog__body) {
  padding: 16px 20px 8px;
}

/* —— 云资产管理式布局（白底、细线边框） —— */
.ct-root {
  --ct-blue: #409eff;
  --ct-border: #e8eaec;
  --kam-text: #515a6e;
}

.kam-cos.oss-content {
  background: #fff;
  overflow: hidden;
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

/* 与资源浏览器同时出现时：凭据表格外层占可视高度一部分，表格在内部滚动 */
.kam-asset-panel.is-with-explorer {
  flex: 0 1 46%;
  max-height: 46%;
  min-height: 140px;
}

.kam-cos.is-fullscreen .kam-asset-panel.is-with-explorer {
  max-height: 52%;
}

/* 仅凭据列表时：整块铺满剩余高度 */
.kam-asset-panel:not(.is-with-explorer) {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
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
  overflow: hidden;
}

.kam-asset-pager {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  margin-top: 0;
}

.kam-asset-panel-compact {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 8px 14px;
  flex-shrink: 0;
  background: #fff;
  border: 1px solid var(--ct-border);
  border-radius: 2px;
}

.kam-asset-compact-text {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #606266;
  min-width: 0;
}

.kam-asset-compact-ico {
  flex-shrink: 0;
  color: var(--ct-blue);
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

.kam-vendor-rail {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  max-width: 100%;
}

.kam-vendor-arrow {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ct-border);
  border-radius: 2px;
  background: #fff;
  color: #606266;
  cursor: pointer;
  padding: 0;
}

.kam-vendor-arrow:hover {
  color: var(--ct-blue);
  border-color: #c6e2ff;
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
  transition: color 0.15s, border-color 0.15s, background 0.15s;
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

.kam-vendor-logo.is-qiniu {
  background: #07be6b;
}
.kam-vendor-logo.is-tencent {
  background: #006eff;
}
.kam-vendor-logo.is-aliyun {
  background: #ff6a00;
}
.kam-vendor-logo.is-huawei {
  background: #d8262c;
}
.kam-vendor-logo.is-qingstor {
  background: #00b4b4;
}
.kam-vendor-logo.is-upyun {
  background: #2e8b57;
}
.kam-vendor-logo.is-aws {
  background: #ff9900;
}
.kam-vendor-logo.is-jd {
  background: #e1251b;
}
.kam-vendor-logo.is-ks3 {
  background: #1e88e5;
}
.kam-vendor-logo.is-minio {
  background: #c72c48;
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

.kam-explorer-panel {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
  border: 1px solid var(--ct-border);
  border-radius: 2px;
  overflow: hidden;
}

.kam-explorer-head {
  padding: 10px 14px;
}

.kam-explorer-head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.kam-explorer-top-search {
  width: 200px;
  min-width: 160px;
}

.kam-explorer-top-search :deep(.el-input__wrapper) {
  width: 100%;
}

.arsa-explorer .kam-explorer-body.arsa-content-row.ct-explorer-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
  border-top: 1px solid var(--ct-border);
}

.kam-icon-rail {
  width: 52px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
  border-right: 1px solid var(--ct-border);
  background: #fff;
}

.kam-rail-spacer {
  flex: 1;
  min-height: 8px;
}

.kam-rail-btn {
  margin: 0 !important;
}

.kam-crumb {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 6px;
  min-width: 0;
  font-size: 13px;
  color: #606266;
}

.kam-crumb-home {
  flex-shrink: 0;
  color: #909399;
}

.kam-crumb-sep {
  flex-shrink: 0;
  color: #909399;
  user-select: none;
}

.kam-crumb-link {
  border: none;
  background: none;
  padding: 0 2px;
  font-size: 13px;
  color: var(--ct-blue);
  cursor: pointer;
  font-family: inherit;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kam-crumb-link:hover {
  text-decoration: underline;
}

.kam-file-stats .count {
  margin-right: 12px;
}

.kam-file-stats strong {
  margin: 0 2px;
  font-weight: 600;
  color: #303133;
}

.kam-file-grid {
  grid-template-columns: repeat(7, minmax(72px, 1fr));
}

@media (max-width: 1200px) {
  .kam-file-grid {
    grid-template-columns: repeat(5, minmax(72px, 1fr));
  }
}

@media (max-width: 900px) {
  .kam-file-grid {
    grid-template-columns: repeat(4, minmax(72px, 1fr));
  }
}

@media (max-width: 600px) {
  .kam-file-grid {
    grid-template-columns: repeat(3, minmax(64px, 1fr));
  }
}

.arsa-layout-menu .ct-bucket-li.is-active {
  background: #f0f7ff;
  border-color: transparent;
  box-shadow: inset 3px 0 0 var(--ct-blue);
}

:deep(.kam-cred-dialog.el-dialog) {
  border-radius: 4px;
}

:deep(.kam-cred-dialog .el-dialog__header) {
  padding: 20px 24px 12px;
  margin: 0;
  border-bottom: 1px solid var(--ct-border);
}

.kam-cred-dialog-title {
  font-size: 18px;
  font-weight: 600;
  color: #17233d;
}

:deep(.kam-cred-dialog .el-dialog__body) {
  padding: 16px 24px 20px;
}

.kam-cred-form :deep(.el-form-item__label) {
  color: #515a6e;
  font-weight: 500;
}

.kam-cred-form-row {
  width: 100%;
  margin-bottom: 0;
}

.kam-cred-form-row :deep(.el-col .el-form-item) {
  margin-bottom: 18px;
}

.kam-cred-form-full {
  width: 100%;
}

.kam-cred-auth-kind {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.kam-cred-auth-hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: #808695;
  line-height: 1.5;
}

.kam-cred-advanced {
  margin-top: 8px;
  border: none;
}

.kam-cred-advanced :deep(.el-collapse-item__header) {
  font-size: 13px;
  color: #606266;
  border: none;
  height: 40px;
}

.kam-cred-advanced :deep(.el-collapse-item__wrap) {
  border: none;
}

.kam-cred-footer {
  margin-top: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--ct-border);
}

.kam-cred-hint {
  margin: 0 0 14px;
  font-size: 12px;
  color: #808695;
  line-height: 1.5;
}

.kam-cred-hint-star {
  color: #ed4014;
  margin-right: 4px;
}

.kam-cred-btns {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.ct-name-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.ct-prov-dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  background: #ecf5ff;
  color: var(--ct-blue);
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
}

.op-edit {
  color: #606266 !important;
}

.ct-explorer-card {
  padding: 0;
  overflow: hidden;
  min-height: 0;
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
}

.ct-explorer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 12px 16px;
  border-bottom: 1px solid var(--ct-border);
  background: #fff;
  flex-shrink: 0;
}

.ct-explorer-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  min-width: 0;
  flex: 1 1 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ct-explorer-hint {
  margin: 0 16px 8px;
  flex-shrink: 0;
}

/* arsa：左侧窄菜单 + 右侧 bucket 区 + 顶栏单行 dir-layout（白底） */
.arsa-explorer .arsa-content-row.ct-explorer-body {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  background: #fff;
}

.ct-expl-sidebar.arsa-layout-menu {
  width: 180px;
  max-width: 180px;
  flex-shrink: 0;
  align-self: stretch;
  min-height: 0;
  border-right: 1px solid var(--ct-border);
  border-bottom-right-radius: 0;
  z-index: 1;
  background: #fff;
  padding: 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
}

.arsa-buckets-menu {
  flex: 1;
  min-height: 0;
}

.arsa-bucketpage {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
  color: #303133;
  overflow: hidden;
}

.arsa-bucketpage.is-idle {
  justify-content: center;
  align-items: center;
  min-height: 0;
}

.ct-side-bucket-toolbar {
  display: flex;
  gap: 8px;
  width: 100%;
}

.ct-side-bucket-toolbar .el-button {
  flex: 1;
  width: auto;
  margin: 0;
}

.oss-obj-pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  padding: 8px 10px 10px;
  flex-shrink: 0;
  font-size: 12px;
  color: #606266;
}

.oss-pager-label {
  flex-shrink: 0;
}

.oss-page-size {
  width: 88px;
}

.oss-page-num {
  min-width: 5em;
  text-align: center;
  color: #909399;
}

.oss-object-surface {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  margin: 0 4px 4px;
  padding: 4px;
  border: 2px dashed transparent;
  border-radius: 8px;
  overflow: hidden;
  transition:
    border-color 0.15s,
    background 0.15s;
}

.oss-object-surface.is-drag-over {
  border-color: var(--ct-blue);
  background: rgba(64, 158, 255, 0.06);
}

.oss-drop-hint {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 8px 8px 12px;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.5;
  color: #909399;
  background: #f5f7fa;
  border-radius: 6px;
  flex-shrink: 0;
}

.oss-drop-hint-ico {
  flex-shrink: 0;
  margin-top: 2px;
  color: var(--ct-blue);
}

.arsa-dir-layout {
  display: flex;
  flex-direction: row;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 12px;
  padding: 15px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--ct-border);
}

.arsa-header-dir-view {
  flex: 1 1 200px;
  min-width: 0;
  overflow-x: auto;
  margin-right: 4px;
}

.arsa-breadcrumb-wrap {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  min-width: 0;
}

.arsa-breadcrumb :deep(.el-breadcrumb__separator),
.arsa-breadcrumb :deep(.el-breadcrumb__inner) {
  color: #606266;
  font-weight: 500;
}

.arsa-header-info-view {
  display: flex;
  flex-direction: row;
  align-items: center;
  flex-shrink: 0;
  font-weight: 500;
  font-size: 12px;
  color: #606266;
}

.arsa-header-info-view .count {
  margin-right: 5px;
  padding-left: 5px;
}

.arsa-header-info-view .size {
  padding-left: 5px;
}

.arsa-header-info-view strong {
  font-weight: 600;
  color: #303133;
}

.arsa-header-button-view {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-items: center;
  flex-shrink: 0;
  gap: 8px;
}

.arsa-dir-search {
  width: 160px;
}

.arsa-header-button-view .arsa-dir-btn {
  font-size: 12px;
  font-weight: 500;
}

.arsa-breadcrumb-wrap .ct-home-ico {
  margin-right: 6px;
  color: #909399;
  flex-shrink: 0;
  vertical-align: middle;
}

.ct-file-grid.arsa-file-list {
  flex: 1;
  min-height: 0;
  margin: 6px 8px 0;
  padding: 8px 4px 12px;
  overflow: auto;
  align-content: start;
}

.ct-obj-table.arsa-file-list {
  flex: 1;
  min-height: 0;
  margin: 6px 8px 0;
  width: calc(100% - 16px) !important;
  max-width: none;
}

.ct-expl-sidebar {
  width: 200px;
  flex-shrink: 0;
  border-right: 1px solid var(--ct-border);
  background: #fff;
  padding: 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ct-side-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  padding-left: 4px;
}

.ct-side-refresh {
  width: 100%;
}

.ct-bucket-ul {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow: auto;
  flex: 1;
  min-height: 0;
}

.ct-bucket-li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 8px;
  border-radius: 6px;
  cursor: pointer;
  margin-bottom: 4px;
  border: 1px solid transparent;
}

.ct-bucket-li:hover {
  background: #fff;
}

.arsa-layout-menu .ct-bucket-li:hover {
  background: #f5f7fa;
}

/* 选中桶在鼠标仍停留在行上时保持高亮（:hover 规则需被覆盖） */
.arsa-layout-menu .ct-bucket-li.is-active:hover {
  background: #f0f7ff;
  border-color: transparent;
  box-shadow: inset 3px 0 0 var(--ct-blue);
}

.ct-bucket-li.is-active {
  background: #ecf5ff;
  border-color: #b3d8ff;
  box-shadow: inset 3px 0 0 var(--ct-blue);
}

.ct-bucket-abbr {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: #909399;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.ct-bucket-li.is-active .ct-bucket-abbr {
  background: var(--ct-blue);
}

.ct-bucket-name {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  word-break: break-all;
  line-height: 1.3;
}

.ct-bucket-tag {
  font-size: 11px;
  color: #909399;
  margin-top: 2px;
}

.ct-bucket-empty {
  font-size: 12px;
  color: #c0c4cc;
  text-align: center;
  padding: 16px 4px;
}

.ct-file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
  gap: 12px 10px;
  min-height: 120px;
  padding: 8px 0;
}

.ct-file-icon-wrap {
  width: 56px;
  height: 56px;
  margin: 0 auto 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  overflow: hidden;
  background: #f5f7fa;
}

.ct-file-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.ct-file-tile {
  text-align: center;
  cursor: default;
  padding: 6px 4px;
  border-radius: 8px;
  transition: background 0.15s;
}

.ct-file-tile:hover {
  background: #f5f7fa;
}

.ct-file-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 6px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  background: #909399;
}

.ct-file-icon.is-folder {
  background: #606266;
}

.ct-file-icon.is-pdf {
  background: #f56c6c;
}

.ct-file-icon.is-doc {
  background: #409eff;
}

.ct-file-icon.is-xls {
  background: #67c23a;
}

.ct-file-icon.is-ppt {
  background: #e6a23c;
}

.ct-file-icon.is-zip {
  background: #9b59b6;
}

.ct-file-icon.is-img {
  background: #13ce66;
}

.ct-file-icon.is-video {
  background: #e91e8c;
}

.ct-file-icon.is-audio {
  background: #7c4dff;
}

.ct-file-icon.is-js {
  background: #f7dc6f;
  color: #333;
}

.ct-file-icon.is-code {
  background: #34495e;
}

.ct-file-icon.is-data {
  background: #1abc9c;
}

.ct-file-icon.is-file {
  background: #8492a6;
}

.ct-file-name {
  font-size: 12px;
  color: #606266;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.ct-grid-empty {
  grid-column: 1 / -1;
  text-align: center;
  color: #c0c4cc;
  padding: 40px;
  font-size: 13px;
}

.ct-obj-table {
  width: 100%;
}

.ct-kkfile-tip {
  font-size: 13px;
  color: #606266;
  margin: 0 0 10px;
}

.ct-kkfile-tip code {
  background: #fff;
  border: 1px solid var(--ct-border);
  padding: 1px 6px;
  border-radius: 4px;
}

.ct-create-bucket-warn {
  margin-bottom: 12px;
}

.ct-create-bucket-tip {
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
  margin: 0 0 14px;
}

.ct-create-bucket-form {
  margin-top: 4px;
}

.ct-create-bucket-field-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 6px;
  line-height: 1.45;
}

.ct-delete-bucket-alert {
  margin-bottom: 12px;
}

.ct-delete-bucket-alert-body {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.5;
}

.ct-delete-bucket-instr {
  margin: 0 0 10px;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.ct-delete-bucket-phrase-wrap {
  margin-bottom: 12px;
  padding: 10px 12px;
  background: var(--el-fill-color-light, #f5f7fa);
  border: 1px solid var(--ct-border, #ebeef5);
  border-radius: 6px;
}

.ct-delete-bucket-phrase {
  display: block;
  font-size: 13px;
  line-height: 1.55;
  word-break: break-all;
  user-select: none;
  color: #303133;
}

.ct-delete-bucket-input {
  margin-top: 4px;
}

:deep(.kam-kkfile-preview-dialog .el-dialog__body) {
  padding: 8px 12px 16px;
}

:deep(.oss-object-text-preview-dialog .el-dialog__body) {
  padding: 4px;
}

:deep(.oss-object-media-preview-dialog .el-dialog__body) {
  padding: 12px 16px 20px;
}

.oss-media-preview-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  width: 100%;
  padding-right: 36px;
  box-sizing: border-box;
}

.oss-media-preview-title {
  font-weight: 600;
  font-size: 15px;
  color: var(--el-text-color-primary);
}

.oss-media-preview-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.oss-media-preview-body {
  min-height: 200px;
}

.oss-media-preview-video {
  display: block;
  width: 100%;
  max-height: min(70vh, 520px);
  border-radius: 8px;
  background: #000;
}

.oss-media-preview-audio {
  display: block;
  width: 100%;
  margin-top: 24px;
}

.oss-obj-text-preview-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  width: 100%;
  padding-right: 36px;
  box-sizing: border-box;
}

.oss-obj-text-preview-title {
  font-weight: 600;
  font-size: 16px;
  color: #303133;
}

.oss-obj-text-preview-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.oss-obj-text-preview-body {
  min-height: 200px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.oss-text-preview-kkfile-block {
  flex-shrink: 0;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
  background: #fafafa;
}

.oss-text-preview-kkfile-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background: #f0f2f5;
  border-bottom: 1px solid #e4e7ed;
  font-size: 13px;
}

.oss-text-preview-kkfile-title {
  font-weight: 600;
  color: #303133;
}

.oss-text-preview-kkfile-iframe {
  display: block;
  width: 100%;
  height: 50vh;
  min-height: 260px;
  max-height: 420px;
  background: #fff;
}

.oss-text-preview-warn {
  flex-shrink: 0;
}

.oss-text-preview-editor-block {
  flex-shrink: 0;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
  background: #fafafa;
}

.oss-text-preview-editor-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background: #f0f2f5;
  border-bottom: 1px solid #e4e7ed;
  font-size: 13px;
}

.oss-text-preview-editor-title {
  font-weight: 600;
  color: #303133;
}

.oss-text-preview-editor-pane {
  padding: 8px 10px 10px;
  background: #fff;
}

.oss-obj-text-preview-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  padding: 16px;
}

.oss-obj-text-monaco :deep(.monaco-editor),
.oss-obj-text-monaco :deep(.monaco-editor-background) {
  border-radius: 4px;
}

.kam-kkfile-preview-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  width: 100%;
  padding-right: 36px;
  box-sizing: border-box;
}

.kam-kkfile-preview-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 4px 4px;
}

.kam-kkfile-act-ico {
  margin-right: 2px;
  vertical-align: middle;
}

.kam-kkfile-preview-title {
  font-weight: 600;
  font-size: 16px;
  color: #303133;
}

.oss-upload-dialog-content {
  padding: 8px 0 4px;
}

.oss-upload-drop-zone {
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  text-align: center;
  background: #fafafa;
  transition: all 0.2s;
  padding: 24px 16px;
  margin: 12px 0 0;
}

.oss-upload-drop-zone.is-drag-over {
  border-color: var(--ct-blue, #409eff);
  background: #ecf5ff;
}

.oss-upload-drop-ico {
  color: #909399;
}

.oss-upload-drop-text {
  margin: 10px 0 14px;
  font-size: 14px;
  color: #606266;
}

.oss-upload-file-list {
  margin-top: 16px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
}

.oss-upload-list-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  font-size: 13px;
  color: #606266;
}

.oss-upload-list-items {
  padding: 6px 8px;
}

.oss-upload-list-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 4px;
}

.oss-upload-list-item:hover {
  background: #f5f7fa;
}

.oss-upload-file-ico {
  color: #909399;
  font-size: 16px;
  flex-shrink: 0;
}

.oss-upload-file-name {
  flex: 1;
  font-size: 13px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.oss-upload-progress-block {
  margin-top: 16px;
  padding: 14px;
  background: #f9fafb;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.oss-upload-progress-head {
  margin-bottom: 8px;
}

.oss-upload-status-text {
  font-size: 13px;
  color: #606266;
}

.oss-upload-current-file {
  color: var(--ct-blue, #409eff);
  font-weight: 500;
}

.oss-upload-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
}

.kam-kkfile-preview-wrap {
  min-height: 500px;
  width: 100%;
}

.kam-kkfile-preview-iframe {
  display: block;
  width: 100%;
  height: 70vh;
  min-height: 500px;
  max-height: 800px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
}

.ct-ctx-menu {
  position: fixed;
  z-index: 9999;
  min-width: 140px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  padding: 4px 0;
}

.ct-ctx-item {
  display: block;
  width: 100%;
  text-align: left;
  border: none;
  background: none;
  font-size: 13px;
  padding: 8px 14px;
  cursor: pointer;
  color: #606266;
  font-family: inherit;
}

.ct-ctx-item:hover {
  background: #ecf5ff;
  color: var(--ct-blue);
}

.ct-ctx-item.danger {
  color: #f56c6c;
}

.ct-ctx-item.danger:hover {
  background: #fef0f0;
  color: #f56c6c;
}

@media (max-width: 768px) {
  .arsa-explorer .kam-explorer-body.arsa-content-row.ct-explorer-body {
    flex-direction: column;
  }

  .kam-icon-rail {
    flex-direction: row;
    width: 100%;
    flex-wrap: wrap;
    justify-content: center;
    padding: 8px;
    border-right: none;
    border-bottom: 1px solid var(--ct-border);
  }

  .kam-rail-spacer {
    display: none;
  }

  .ct-expl-sidebar.arsa-layout-menu {
    width: 100%;
    max-width: none;
    max-height: 220px;
    border-right: none;
    border-bottom: 1px solid var(--ct-border);
  }

  .kam-explorer-top-search {
    width: 140px;
  }
}
</style>
