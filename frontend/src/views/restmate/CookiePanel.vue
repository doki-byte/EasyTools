<template>
  <div>
    <div class="cookie-toolbar">
      <el-button size="small" type="primary" @click="loadCookies" :loading="loading">
        刷新
      </el-button>
      <el-button size="small" type="danger" plain @click="handleClearAll">
        清空全部
      </el-button>
    </div>
    <el-table
      :data="cookies"
      size="small"
      border
      height="360"
      v-loading="loading"
    >
      <el-table-column prop="Domain" label="域名" width="200" />
      <el-table-column prop="Name" label="名称" width="160" />
      <el-table-column prop="Value" label="值" />
      <el-table-column label="操作" width="90">
        <template #default="scope">
          <el-button type="text" size="small" @click="handleDelete(scope.row.Domain)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script>
import { ElMessage } from 'element-plus';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { GetAllCookies, ClearAllCookies, DeleteCookies } from '../../../wailsjs/go/restmate/RestMate';

export default {
  name: 'CookiePanel',
  data() {
    return {
      loading: false,
      cookies: [],
    };
  },
  created() {
    this.loadCookies();
  },
  methods: {
    async loadCookies() {
      this.loading = true;
      try {
        const res = await GetAllCookies();
        if (res.success) {
          this.cookies = res.data || [];
        } else {
          ElMessage.error(res.msg || '获取 Cookie 失败');
        }
      } catch (e) {
        ElMessage.error('获取 Cookie 失败');
      } finally {
        this.loading = false;
      }
    },
    async handleClearAll() {
      try {
        await confirmDangerousAction({
          title: '清空 Cookie',
          message: '确定清空所有 Cookie 吗？',
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        const res = await ClearAllCookies();
        if (res.success) {
          ElMessage.success(res.msg || '已清空 Cookie');
          this.loadCookies();
        } else {
          ElMessage.error(res.msg || '清空 Cookie 失败');
        }
      } catch {
        ElMessage.error('清空 Cookie 失败');
      }
    },
    async handleDelete(domain) {
      try {
        await confirmDangerousAction({
          title: '删除 Cookie',
          message: `将删除域名「${domain}」下的 Cookie。`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        const res = await DeleteCookies(domain);
        if (res.success) {
          ElMessage.success(res.msg || '删除成功');
          this.loadCookies();
        } else {
          ElMessage.error(res.msg || '删除失败');
        }
      } catch {
        ElMessage.error('删除失败');
      }
    },
  },
};
</script>

<style scoped>
.cookie-toolbar {
  margin-bottom: 8px;
  display: flex;
  gap: 8px;
}
</style>


