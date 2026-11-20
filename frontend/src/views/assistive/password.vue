<template>
  <div class="password-management">
    <div class="header">
      <div class="search-bar">
        <el-input
            placeholder="请输入查询条件"
            v-model="queryPasswordInput"
            class="input"
            clearable
        />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="success" @click="handleAdd">新增</el-button>
      </div>
    </div>

    <!-- 数据表格 -->
    <el-table
        :data="passwordData"
        border
        class="custom-table uniform-rows glass-card"
        :row-style="{ height: '45px' }"
        :cell-style="{ padding: '8px 0', lineHeight: '1.5' }"
        v-loading="loading"
        @cell-click="handleCellClick"
        style="background: rgba(255, 255, 255, 0.95);">

      <el-table-column prop="id" label="ID" width="60" align="center"/>
      <el-table-column prop="name" label="Name" :min-width="150" show-overflow-tooltip />
      <el-table-column prop="method" label="Method" :min-width="150" show-overflow-tooltip />
      <el-table-column prop="userId" label="User ID" :min-width="150" show-overflow-tooltip />
      <el-table-column prop="password" label="Password" :min-width="150" show-overflow-tooltip />
      <el-table-column prop="level" label="Level" :min-width="120" show-overflow-tooltip />

      <el-table-column label="操作" width="180" fixed="right">
        <template #default="scope">
          <el-button
              size="small"
              @click="handleEdit(scope.row)"
              type="primary"
              link
          >编辑</el-button>
          <el-button
              size="small"
              @click="handleCopyPassword(scope.row)"
              type="success"
              link
          >复制密码</el-button>
          <el-button
              size="small"
              @click="handleDelete(scope.row)"
              type="danger"
              link
          >删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="pagination-container">
      <el-pagination
          :current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          @current-change="handlePageChange"
          layout="prev, pager, next, jumper, total"
          :pager-count="5"
          class="uniform-pagination"
          background>
      </el-pagination>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
        :title="dialogTitle"
        v-model="dialogVisible"
        width="500px"
        @close="handleDialogClose">
      <el-form
          ref="formRef"
          :model="formData"
          label-width="80px">

        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入名称" />
        </el-form-item>

        <el-form-item label="方法" prop="method">
          <el-input v-model="formData.method" placeholder="请输入方法" />
        </el-form-item>

        <el-form-item label="用户ID" prop="userId">
          <el-input v-model="formData.userId" placeholder="请输入用户ID" />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
              v-model="formData.password"
              placeholder="请输入密码"
          />
        </el-form-item>

        <el-form-item label="等级" prop="level">
          <el-select v-model="formData.level" placeholder="请选择等级" style="width: 100%">
            <el-option label="低" value="low" />
            <el-option label="中" value="medium" />
            <el-option label="高" value="high" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="submitting">
            {{ submitting ? '提交中...' : '确定' }}
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {
  QueryPasswordsAPI,
  CreatePassword,
  UpdatePassword,
  DeletePassword,
  GetPasswordByID
} from "../../../wailsjs/go/assistive/Assistive";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';

export default {
  name: "password",
  data() {
    return {
      // 查询相关
      queryPasswordInput: "",
      passwordData: [],
      total: 0,
      currentPage: 1,
      pageSize: 12,
      loading: false,

      // 对话框相关
      dialogVisible: false,
      dialogTitle: "新增密码",
      submitting: false,
      isEditing: false,
      editingId: null,

      // 表单数据
      formData: {
        name: "",
        method: "",
        userId: "",
        password: "",
        level: ""
      },
    }
  },
  async mounted() {
    await this.fetchPasswords(false);
  },
  methods: {
    async fetchPasswords(isSearch = false) {
      const query = isSearch ? this.queryPasswordInput.trim() : "";
      this.loading = true;

      try {
        let response = await QueryPasswordsAPI(this.currentPage, this.pageSize, query);

        if (!response || !response.data || typeof response.total !== 'number') {
          this.$message.error("查询结果格式不正确！");
          return;
        }

        this.passwordData = response.data || [];
        this.total = response.total;

        if (this.passwordData.length === 0 && this.currentPage > 1) {
          this.currentPage = 1;
          await this.$nextTick(() => {
            this.fetchPasswords(isSearch);
          });
        }
      } catch (err) {
        console.error("查询错误：", err);
        this.$message.error(`查询失败: ${err.message || "请重试！"}`);
      } finally {
        this.loading = false;
      }
    },

    handleSearch() {
      this.currentPage = 1;
      this.fetchPasswords(true);
    },

    handleAdd() {
      this.dialogTitle = "新增密码";
      this.isEditing = false;
      this.editingId = null;
      this.dialogVisible = true;
    },

    async handleEdit(row) {
      try {
        this.loading = true;
        const data = await GetPasswordByID(row.id);

        this.formData = { ...data };
        this.dialogTitle = "编辑密码";
        this.isEditing = true;
        this.editingId = row.id;
        this.dialogVisible = true;
      } catch (err) {
        this.$message.error(`获取数据失败: ${err.message || "请重试！"}`);
      } finally {
        this.loading = false;
      }
    },

    async handleSubmit() {
      try {
        const valid = await this.$refs.formRef.validate();
        if (!valid) return;

        this.submitting = true;

        if (this.isEditing) {
          // 更新操作
          const updateData = {
            ...this.formData,
            id: this.editingId
          };
          await UpdatePassword(updateData);
          this.$message.success("更新成功！");
        } else {
          // 新增操作
          await CreatePassword(this.formData);
          this.$message.success("新增成功！");
        }

        this.dialogVisible = false;
        await this.fetchPasswords(!!this.queryPasswordInput.trim());

      } catch (err) {
        this.$message.error(`操作失败: ${err.message || "请重试！"}`);
      } finally {
        this.submitting = false;
      }
    },

    async handleDelete(row) {
      try {
        await confirmDangerousAction({
          title: '删除密码条目',
          message: `确定要删除「${row.name}」吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        });

        await DeletePassword(row.id);
        this.$message.success("删除成功！");

        // 重新加载数据
        await this.fetchPasswords(!!this.queryPasswordInput.trim());

      } catch (err) {
        if (err !== 'cancel') {
          this.$message.error(`删除失败: ${err.message || "请重试！"}`);
        }
      }
    },

    // 复制密码
    async handleCopyPassword(row) {
      try {
        await navigator.clipboard.writeText(row.password);
        this.$message.success(`已复制密码: ${row.password}`);
      } catch (error) {
        this.$message.error("复制失败，请重试");
      }
    },

    // 单元格点击复制（保留原有功能）
    async handleCellClick(row, column, cell, event) {
      // 如果点击的是操作列，不执行复制
      if (column.property) {
        const textToCopy = row[column.property];
        try {
          await navigator.clipboard.writeText(textToCopy);
          this.$message.success(`已复制: ${textToCopy}`);
        } catch (error) {
          this.$message.error("复制失败，请重试");
        }
      }
    },

    handlePageChange(newPage) {
      this.currentPage = newPage;
      this.fetchPasswords(!!this.queryPasswordInput.trim());
    },

    handleDialogClose() {
      this.$refs.formRef.resetFields();
      this.formData = {
        name: "",
        method: "",
        userId: "",
        password: "",
        level: ""
      };
    }
  }
}
</script>

<style scoped>
.password-management {
  padding: 0 8px 3px 3px;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}

.input {
  flex: 1;
  max-width: 300px;
}

/* 表格 */
.custom-table {
  flex-grow: 1;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  overflow: hidden;
  min-height: 550px;
}

.custom-table .el-table__row:hover {
  background-color: #f5f7fa !important;
}

/* 分页 */
.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  padding: 20px 0;
}

/* 自定义分页控件按钮颜色 */
.el-pagination .el-pager li {
  color: #409EFF;
}

.el-pagination .el-pager li:hover {
  background-color: #f4f4f5;
}

.el-pagination .el-button--primary {
  background-color: #409EFF;
  border-color: #409EFF;
  color: white;
}

/* 控制分页按钮的大小 */
.el-pagination button {
  padding: 5px 15px;
}

/* 添加分隔线 */
.el-pagination .el-pager li:not(:last-child) {
  border-right: 1px solid #ddd;
}

/* 操作按钮间距 */
.el-table .cell .el-button {
  margin: 0 2px;
}

/* 密码字段样式 */
.password-field {
  font-family: monospace;
  color: #333;
}
</style>