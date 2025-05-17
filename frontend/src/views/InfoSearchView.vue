<template>
    <el-container class="container">
        <!-- 标签栏 -->
        <el-tabs v-model="activeTab" class="tabs">
            <el-tab-pane label="Google语法" name="google-syntax" />
            <el-tab-pane label="默认密码查询" name="password-query" />
            <el-tab-pane label="反弹Shell" name="shell-syntax" />
            <el-tab-pane label="杀软进程查询" name="process-query" />
        </el-tabs>

        <!-- 内容区域 -->
        <el-main :class="['content', { 'shell-syntax-active': activeTab === 'shell-syntax' }]">
            <!-- 默认密码查询 -->
            <div v-if="activeTab === 'password-query'" class="tab-content">
                <div class="header">
                    <div class="search-bar">
                        <el-input placeholder="请输入查询条件" v-model="queryPasswordInput" class="input" />
                        <el-button type="primary" @click="fetchPasswords">查询</el-button>
                    </div>
                </div>
                <el-table :data="passwordData" border class="custom-table" @cell-click="handleCopeClick">
                    <el-table-column prop="name" label="Name" :min-width="190" />
                    <el-table-column prop="method" label="Method" :min-width="190" />
                    <el-table-column prop="userId" label="User ID" :min-width="190" />
                    <el-table-column prop="password" label="PassWord" :min-width="190" />
                    <el-table-column prop="level" label="Level" :min-width="190" />
                </el-table>
                <!-- 自定义分页样式 -->
                <el-pagination ref="pagination" :current-page="currentPage" :page-size="pageSize" :total="total"
                    @current-change="handlePageChange" layout="prev, pager, next, jumper, total" :pager-count="5"
                    background style="padding: 10px 0; text-align: center;">
                    <template #prev>
                        <el-button size="small" type="primary">Previous</el-button>
                    </template>
                    <template #next>
                        <el-button size="small" type="primary">Next</el-button>
                    </template>
                </el-pagination>
            </div>

            <!-- 杀软进程查询 -->
            <div v-if="activeTab === 'process-query'" class="tab-content">
                <div class="header">
                    <h4>杀软识别 tasklist /svc</h4>
                    <el-input type="textarea" v-model="tasklistInput" placeholder="请输入 tasklist /svc 输出内容" :rows="10"
                        class="process_textarea"></el-input>
                    <el-button type="primary" @click="handleAVQuery">查询</el-button>
                </div>
                <div class="result" v-if="isAvQueried">
                    <h5>识别结果：</h5>
                    <!-- 如果有结果，显示表格 -->
                    <el-table v-if="avResults.length > 0" :data="avResults" border style="width: 100%">
                        <el-table-column prop="program" label="程序名" />
                        <el-table-column prop="match" label="杀软匹配" />
                        <el-table-column prop="description" label="描述" />
                    </el-table>
                    <!-- 如果没有结果，显示提示文字 -->
                    <p v-else class="no-results">未识别到任何杀软进程。</p>
                </div>
            </div>

            <!-- Google语法查询 -->
            <div v-if="activeTab === 'google-syntax'" class="tab-content">
                <div class="google-syntax-container">
                    <h4>Google Hacking 语法查询</h4>
                    <div class="form-group">
                        <el-input v-model="googleDomainInput" placeholder="请输入域名或IP" class="input" />
                        <el-button type="primary" @click="generateGoogleQueries">生成语法</el-button>
                    </div>

                    <div v-if="googleQueries.length > 0" class="query-results">
                        <h4>查询语法：</h4>
                        <el-collapse v-model="activeCategories">
                            <el-collapse-item v-for="category in googleQueries" :key="category.category"
                                :title="category.category" :name="category.category">
                                <el-table :data="category.commands" border style="width: 100%">
                                    <el-table-column class="query-item" prop="description" label="说明"/>
                                    <el-table-column class="query-item" prop="command" label="语法">
                                        <template #default="scope">
                                            <div class="query-item">
                                                <span>{{ scope.row.command }}</span>
                                                <el-button type="text"
                                                    @click="CopyToClipboard(scope.row.command)">复制</el-button>
                                            </div>
                                        </template>
                                    </el-table-column>
                                </el-table>
                            </el-collapse-item>
                        </el-collapse>
                    </div>
                </div>
            </div>

            <!-- 反弹shell查询 -->
            <div v-if="activeTab === 'shell-syntax'" class="tab-content">
                <!-- 嵌入 public/shell/index.html -->
                <div class="iframe-container">
                    <iframe src="/Shell/index.html"></iframe>
                </div>
            </div>
        </el-main>
    </el-container>
</template>


<script>
import { QueryAntivirusProcesses, QueryGoogleQueries, QueryPasswordsAPI } from "../../wailsjs/go/controller/InfoSearch"
export default {
    data() {
        return {
            activeTab: "google-syntax",
            queryPasswordInput: "", // 查询输入框的绑定数据
            tabTitles: {
                "password-query": "默认密码查询",
                "process-query": "杀软进程查询",
                "google-syntax": "Google语法",
                "shell-syntax": "反弹Shell",
            },
            passwordData: [],
            total: 0,
            currentPage: 1,
            pageSize: 10,

            tasklistInput: "", // 用户输入的 tasklist 内容
            avResults: [], // 查询结果
            isAvQueried: false, // 是否已进行查询

            googleDomainInput: '',
            googleQueries: [],
            activeCategories: [], // 默认展开的分类

        };
    },
    mounted() {
        // 页面加载时自动加载第一页数据
        this.fetchPasswords(false);
        // 禁用右键菜单
        document.addEventListener('contextmenu', (event) => {
            event.preventDefault();
        });
    },
    beforeUnmount() {
        // 在组件卸载时移除事件监听器，避免潜在的内存泄漏
        document.removeEventListener('contextmenu', (event) => {
            event.preventDefault();
        });
    },
    methods: {
        // 加载数据（支持默认查询和按条件查询）
        async fetchPasswords(isSearch = false) {
            const query = isSearch ? this.queryPasswordInput.trim() : ""; // 根据是否是搜索查询设置条件

            try {
                // 调用后端的 QueryPasswordsAPI 方法
                let response = await QueryPasswordsAPI(this.currentPage, this.pageSize, query);

                // 检查返回数据
                if (!response || !response.data || !("total" in response)) {
                    this.$message.error("查询结果为空或格式不正确！");
                    return;
                }

                // 更新表格数据和分页总数
                this.passwordData = response.data;
                // console.log(this.passwordData)
                this.total = response.total;
            } catch (err) {
                console.error("查询错误：", err);
                this.$message.error("查询失败，请重试！");
            }
        },
        // 搜索按钮点击事件
        handlePasswdQuery() {
            this.fetchPasswords(true); // 执行按条件查询
        },
        // 分页控件页码切换事件
        handlePageChange(page) {
            this.currentPage = page;
            this.fetchPasswords(); // 根据新页码重新加载
        },

        async handleCopeClick(row, column, cell, event) {
            const textToCopy = cell.innerText; // 获取单元格内容
            try {
                await navigator.clipboard.writeText(textToCopy); // 复制到剪贴板
                this.$message.success(`已复制: ${textToCopy}`);
            } catch (error) {
                this.$message.error("复制失败，请重试");
            }
        },

        async handleAVQuery() {
            // 如果用户输入为空，返回错误提示
            if (!this.tasklistInput.trim()) {
                this.$message.error("请输入有效的 tasklist /svc 输出内容！");
                this.avResults = [];
                this.isAvQueried = false;
                return;
            }

            try {
                // 调用后端接口查询杀软进程
                const response = await QueryAntivirusProcesses(this.tasklistInput);

                this.avResults = response.map(item => ({
                    program: item.program,    // 数据库中匹配的进程名
                    match: item.match,        // 可选，显示完整的进程名
                    description: item.description         // 杀软描述
                }));

                this.isAvQueried = true; // 标记查询完成
            } catch (err) {
                this.isAvQueried = true; // 标记查询完成
            }
        },

        async generateGoogleQueries() {
            // 如果用户输入为空，返回错误提示
            if (!this.googleDomainInput.trim()) {
                this.$message.error('请输入域名或IP！');
                return;
            }

            const googleDomain = this.googleDomainInput.trim();
            try {
                // 调用后端接口查询生成的 Google 查询语法
                let response = await QueryGoogleQueries(googleDomain);

                // 如果返回的是单一对象而非数组，包装成数组
                if (!Array.isArray(response)) {
                    response = [response];
                }

                // 更新查询结果
                this.googleQueries = response.map(item => ({
                    category: item.category,           // 查询分类
                    commands: [{
                        id: "google1",                 // 查询命令的ID（根据实际数据格式修改）
                        description: item.description, // 查询命令的描述
                        command: item.command          // 生成的查询语法
                    }]
                }));

                // 设置所有分类为展开状态
                this.activeCategories = this.googleQueries.map((query) => query.category);

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
    },
};
</script>

<style scoped>
/* 页面容器 */
.container {
    height: 100vh;
    display: flex;
    margin-left: 10px;
    flex-direction: column;
    background-color: #f8f9fb;
}

/* 顶部 Tabs 样式 */
.tabs {
    background-color: #ffffff;
    box-shadow: 0px 2px 2px rgba(0, 0, 0, 0.1);
    margin-bottom: 10px;
    /* border-bottom: 2px solid #ebeef5; */
    padding-left: 10px;
    /* 增加左边距 */
    border-radius: 10px 10px 10px 10px;
}

:deep(.el-tabs__item) {
    font-weight: 700;
    /* 加粗字体 */
    font-size: 16px;
    /* 设置字体大小 */
    transition: all 0.3s;
}

.el-tabs__item:hover {
    color: #409eff !important;
}

.el-tabs__item.is-active {
    color: #ffffff !important;
    background-color: #409eff !important;
    border-radius: 5px 5px 0 0;
}

/* 内容区域 */
.content {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    padding: 20px;
    background-color: #ffffff;
    box-shadow: 0px 0px 10px rgba(0, 0, 0, 0.05);
    border-radius: 8px;
    box-sizing: border-box;
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
}

/* 表格 */
.custom-table {
    flex-grow: 1;
    border-radius: 8px;
    border: 1px solid #ebeef5;
    overflow: hidden;
}

.custom-table .el-table__row:hover {
    background-color: #f5f7fa !important;
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

/* 占位内容 */
.placeholder {
    flex-grow: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    font-size: 16px;
    color: #606266;
    background-color: #f9f9f9;
    border: 1px dashed #dcdfe6;
    border-radius: 8px;
}

/* 杀软查询输入框 */
.process_textarea {
    margin-bottom: 15px;
}

/* 精确定位 shell-syntax 的 tab-content */
.shell-syntax-active {
    padding: 0px !important;
}

/* iframe 容器样式 */
.iframe-container {
    /* margin: 0 auto; */
    max-width: 100%;
    /* 控制 iframe 的最大宽度 */
    overflow: hidden;
    border-radius: 8px;
    /* 圆角处理 */
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    /* 增加阴影效果 */
}

/* iframe 样式 */
.iframe-container iframe {
    width: 100%;
    /* 保持 iframe 宽度填充父容器 */
    height: 90vh;
    /* 高度占视口的 80% */
    border: none;
    /* 去掉默认边框 */
    border-radius: 8px;
    /* 与父容器一致的圆角 */
}

:deep(.el-textarea__inner) {
    padding-top: 15px;
    /* 增加顶部内边距，将文字往下移 */
    padding-bottom: 10px;
    /* 保持上下内边距平衡 */
    line-height: 1.5;
    /* 设置合适的行高 */
}

/* google语法 */
.google-syntax-container {
    padding: 20px;
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

.query-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    word-wrap: break-word;
    /* 自动换行 */

    overflow-wrap: break-word;
    /* 防止溢出 */
    color: #5cb85c;
    
}

.query-item span {
    max-width: 90%;
    /* 限制命令显示的最大宽度 */
    overflow: hidden;
    text-overflow: ellipsis;
    /* 超出部分用省略号显示 */
    word-wrap: break-word;

}

.query-item el-button {
    margin-left: 10px;
}

/* 蓝队封禁IP工具 */
/* 上半部分和下半部分使用 flex 布局 */
.tab-content .info {
    color: #409eff;
}

.row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 10px;
}

.column {
    width: 48%;
    /* 两个元素各占一半的宽度 */
    box-sizing: border-box;
}

.upper-row .column {
    margin-bottom: 20px;
    /* 上半部分的列之间添加空隙 */
}

.lower-row .column {
    display: flex;
    flex-direction: column;
    align-items: stretch;
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
    /* 距离容器顶部 5px */
    right: 5px;
    /* 距离容器右侧 5px */
    z-index: 10;
    /* 确保按钮位于输入框上方 */
    padding: 5px 10px;
    font-size: 12px;
    background-color: #00aaff;
    /* 按钮颜色 */
    color: #ffffff;
    border: none;
    border-radius: 3px;
    cursor: pointer;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

/* 按钮 hover 效果 */
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
    /* margin-left: 0; */
    width: 100%;
    font-size: 18px;
    text-align: center;
    background-color: #5cb85c;
}
</style>
