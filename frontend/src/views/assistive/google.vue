<template>
  <div class="google-syntax-container">
    <h4>Google Hacking 语法查询</h4>
    <div class="form-group">
      <el-input v-model="googleDomainInput" placeholder="请输入域名或IP" class="input" />
      <el-button type="primary" @click="generateGoogleQueries">生成语法</el-button>
      <el-button type="success" @click="showManageView = !showManageView">
        {{ showManageView ? '返回查询' : '管理语法' }}
      </el-button>
    </div>

    <!-- 查询结果区域 -->
    <div v-if="!showManageView">
      <!-- 生成语法后的搜索和分页功能 -->
      <div v-if="googleQueries.length > 0" class="result-controls">
      <div class="search-section">
        <el-input
            v-model="searchKeyword"
            placeholder="搜索分类、说明或命令..."
            clearable
            style="width: 300px;"
            @input="filterQueries"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 生成结果分页 -->
      <div class="result-pagination">
        <el-pagination
            v-model:current-page="resultCurrentPage"
            v-model:page-size="resultPageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="filteredQueries.length"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleResultSizeChange"
            @current-change="handleResultCurrentChange"
        />
      </div>
      </div>

      <!-- 生成结果表格 -->
      <div v-if="pagedQueries.length > 0" class="query-results">
        <h4>查询语法：</h4>
        <el-table :data="pagedQueries" border style="width: 100%" class="glass-card">
          <el-table-column prop="category" label="分类" width="150" />
          <el-table-column prop="description" label="说明" />
          <el-table-column prop="command" label="语法命令">
            <template #default="scope">
              <div class="command-cell">
                <span class="command-text">{{ scope.row.command }}</span>
                <div class="action-buttons">
                  <el-button type="primary" size="small" @click="CopyToClipboard(scope.row.command)">
                    复制
                  </el-button>
                  <el-button type="success" size="small" @click="openInBrowser(scope.row.command)">
                    浏览器搜索
                  </el-button>
                </div>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div v-else-if="googleQueries.length > 0 && filteredQueries.length === 0" class="no-results">
        <el-empty description="未找到匹配的语法" />
      </div>
    </div>

    <!-- 管理语法区域 -->
    <div v-if="showManageView" class="manage-section">
      <h4>管理Google语法</h4>
      <div class="manage-toolbar">
        <el-button type="primary" @click="handleAdd">新增语法</el-button>
        <div style="flex: 1"></div>
        <el-input
            v-model="manageSearchKeyword"
            placeholder="搜索分类、说明或语法..."
            clearable
            style="width: 300px;"
            @input="handleManageSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <el-table :data="pagedTableData" border style="width: 100%" class="glass-card">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="category" label="分类" width="150">
          <template #default="scope">
            <span v-if="!scope.row.editing">{{ scope.row.category }}</span>
            <el-input v-else v-model="scope.row.editCategory" />
          </template>
        </el-table-column>
        <el-table-column prop="description" label="说明">
          <template #default="scope">
            <span v-if="!scope.row.editing">{{ scope.row.description }}</span>
            <el-input v-else v-model="scope.row.editDescription" />
          </template>
        </el-table-column>
        <el-table-column prop="command" label="语法">
          <template #default="scope">
            <span v-if="!scope.row.editing">{{ scope.row.command }}</span>
            <el-input v-else v-model="scope.row.editCommand" type="textarea" :rows="2" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="scope">
            <div v-if="!scope.row.editing">
              <el-button type="primary" size="small" @click="handleEdit(scope.row)">编辑</el-button>
              <el-button type="danger" size="small" @click="handleDelete(scope.row)">删除</el-button>
            </div>
            <div v-else>
              <el-button type="success" size="small" @click="handleSave(scope.row)">保存</el-button>
              <el-button type="default" size="small" @click="handleCancel(scope.row)">取消</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 管理页面分页组件 -->
      <div class="pagination-container">
        <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="filteredManageData.length"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
        />
      </div>
    </div>

    <!-- 新增语法对话框 -->
    <el-dialog v-model="showAddDialog" title="新增语法" width="600px">
      <el-form :model="newQuery" label-width="80px">
        <el-form-item label="分类">
          <el-input v-model="newQuery.category" placeholder="请输入分类名称" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="newQuery.description" placeholder="请输入语法说明" />
        </el-form-item>
        <el-form-item label="语法">
          <el-input v-model="newQuery.command" type="textarea" :rows="3"
                    placeholder="请输入语法命令，使用 ${googleDomain} 作为域名占位符" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAddConfirm">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {Search} from '@element-plus/icons-vue'
import {QueryGoogleQueries, GetAllGoogleQueries, AddGoogleQuery, UpdateGoogleQuery, DeleteGoogleQuery} from "../../../wailsjs/go/assistive/Assistive";
import {BrowserOpenURL} from "../../../wailsjs/runtime";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';

export default {
  name: "google",
  components: {
    Search
  },
  data(){
    return {
      // google 语法查询
      googleDomainInput: '',
      googleQueries: [],
      searchKeyword: '',
      filteredQueries: [],

      // 生成结果分页相关
      resultCurrentPage: 1,
      resultPageSize: 10,

      // 管理相关
      showManageView: false,
      showAddDialog: false,
      allGoogleQueries: [],
      newQuery: {
        category: '',
        description: '',
        command: ''
      },

      // 管理页面分页相关
      currentPage: 1,
      pageSize: 10,

      // 搜索相关
      manageSearchKeyword: '',
      filteredManageData: []
    }
  },
  computed: {
    // 管理页面分页数据
    pagedTableData() {
      const start = (this.currentPage - 1) * this.pageSize;
      const end = start + this.pageSize;
      return this.filteredManageData.slice(start, end);
    },

    // 生成结果分页数据
    pagedQueries() {
      const start = (this.resultCurrentPage - 1) * this.resultPageSize;
      const end = start + this.resultPageSize;
      return this.filteredQueries.slice(start, end);
    }
  },
  methods:{
    async generateGoogleQueries() {
      if (!this.googleDomainInput.trim()) {
        this.$message.warning('请输入域名或IP！');
        return;
      }

      const googleDomain = this.googleDomainInput.trim();
      try {
        let response = await QueryGoogleQueries(googleDomain);

        if (!Array.isArray(response)) {
          response = [response];
        }

        // 将数据展平为普通表格格式
        this.googleQueries = response.flatMap(item => ({
          category: item.category,
          description: item.description,
          command: item.command
        }));

        this.filteredQueries = [...this.googleQueries];
        this.searchKeyword = ''; // 重置搜索关键词
        this.resultCurrentPage = 1; // 重置到第一页

        // 如果当前在管理界面，自动跳转到查询界面
        if (this.showManageView) {
          this.showManageView = false;
        }

      } catch (err) {
        this.$message.error("查询失败，请重试！");
      }
    },

    CopyToClipboard(query) {
      navigator.clipboard.writeText(query).then(() => {
        this.$message.success('复制成功！');
      }).catch(() => {
        this.$message.error('复制失败！');
      });
    },

    // 浏览器搜索功能
    openInBrowser(query) {
      const searchUrl = `https://www.google.com/search?q=${encodeURIComponent(query)}`;
      BrowserOpenURL(searchUrl);
    },

    // 搜索生成的语法
    filterQueries() {
      if (!this.searchKeyword.trim()) {
        this.filteredQueries = [...this.googleQueries];
      } else {
        const keyword = this.searchKeyword.toLowerCase();
        this.filteredQueries = this.googleQueries.filter(query =>
            query.category.toLowerCase().includes(keyword) ||
            query.description.toLowerCase().includes(keyword) ||
            query.command.toLowerCase().includes(keyword)
        );
      }

      this.resultCurrentPage = 1; // 搜索后回到第一页
    },

    // 管理相关方法
    async loadAllQueries() {
      try {
        const response = await GetAllGoogleQueries();
        this.allGoogleQueries = response.map(item => ({
          ...item,
          editing: false,
          editCategory: item.category,
          editDescription: item.description,
          editCommand: item.command
        }));
        this.filteredManageData = [...this.allGoogleQueries];
        this.currentPage = 1; // 重置到第一页
      } catch (err) {
        this.$message.error("加载语法列表失败！");
      }
    },

    // 管理页面搜索
    handleManageSearch() {
      if (!this.manageSearchKeyword.trim()) {
        this.filteredManageData = [...this.allGoogleQueries];
        this.currentPage = 1;
        return;
      }

      const keyword = this.manageSearchKeyword.toLowerCase();
      this.filteredManageData = this.allGoogleQueries.filter(item =>
          item.category.toLowerCase().includes(keyword) ||
          item.description.toLowerCase().includes(keyword) ||
          item.command.toLowerCase().includes(keyword)
      );
      this.currentPage = 1; // 搜索后回到第一页
    },

    handleAdd() {
      this.newQuery = {
        category: '',
        description: '',
        command: ''
      };
      this.showAddDialog = true;
    },

    async handleAddConfirm() {
      if (!this.newQuery.category || !this.newQuery.description || !this.newQuery.command) {
        this.$message.warning('请填写完整信息！');
        return;
      }

      try {
        await AddGoogleQuery(this.newQuery);
        this.$message.success('新增成功！');
        this.showAddDialog = false;
        await this.loadAllQueries();
      } catch (err) {
        this.$message.error('新增失败！');
      }
    },

    handleEdit(row) {
      row.editing = true;
      // 初始化编辑数据
      row.editCategory = row.category;
      row.editDescription = row.description;
      row.editCommand = row.command;
    },

    async handleSave(row) {
      if (!row.editCategory || !row.editDescription || !row.editCommand) {
        this.$message.warning('请填写完整信息！');
        return;
      }

      try {
        const updateData = {
          id: row.id,
          category: row.editCategory,
          description: row.editDescription,
          command: row.editCommand
        };
        await UpdateGoogleQuery(updateData);

        // 更新本地数据
        row.category = row.editCategory;
        row.description = row.editDescription;
        row.command = row.editCommand;
        row.editing = false;

        this.$message.success('保存成功！');
      } catch (err) {
        this.$message.error('保存失败！');
      }
    },

    handleCancel(row) {
      row.editing = false;
    },

    async handleDelete(row) {
      try {
        await confirmDangerousAction({
          title: '删除语法',
          message: '确定要删除这条语法吗？',
          expectedInput: DANGER_DELETE_TOKEN
        });

        await DeleteGoogleQuery(row.id);
        this.$message.success('删除成功！');
        await this.loadAllQueries();
      } catch (err) {
        if (err !== 'cancel') {
          this.$message.error('删除失败！');
        }
      }
    },

    // 管理页面分页相关方法
    handleSizeChange(newSize) {
      this.pageSize = newSize;
      this.currentPage = 1;
    },

    handleCurrentChange(newPage) {
      this.currentPage = newPage;
    },

    // 生成结果分页相关方法
    handleResultSizeChange(newSize) {
      this.resultPageSize = newSize;
      this.resultCurrentPage = 1;
    },

    handleResultCurrentChange(newPage) {
      this.resultCurrentPage = newPage;
    }
  },
  watch: {
    showManageView(val) {
      if (val) {
        this.loadAllQueries();
        this.manageSearchKeyword = ''; // 重置搜索
      }
    }
  }
}
</script>

<style scoped>
/* google语法 */
.google-syntax-container {
  padding: 0 10px;
}

.form-group {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.input {
  margin-right: 10px;
  flex: 1;
}

.query-results {
  margin-top: 20px;
  color: #444744;
}

.command-cell {
  display: flex;
  justify-content: space-between;
  align-items: center;
  word-wrap: break-word;
  overflow-wrap: break-word;
}

.command-text {
  flex: 1;
  max-width: 70%;
  overflow: hidden;
  text-overflow: ellipsis;
  word-wrap: break-word;
  color: #5cb85c;
}

.action-buttons {
  display: flex;
  gap: 8px;
  margin-left: 10px;
}

.manage-section {
  margin-top: 20px;
}

.manage-section h4 {
  margin-bottom: 15px;
  color: #444744;
}

.manage-toolbar {
  margin-bottom: 15px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.pagination-container {
  margin-top: 15px;
  display: flex;
  justify-content: flex-end;
}

.result-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
  margin-bottom: 15px;
}

.result-pagination {
  display: flex;
  align-items: center;
}

.search-section {
  display: flex;
  align-items: center;
}

.no-results {
  margin-top: 20px;
  text-align: center;
}
</style>