<template>
  <div class="header">
    <h4>杀软识别 tasklist /svc</h4>
    <el-input type="textarea" v-model="tasklistInput" placeholder="请输入 tasklist /svc 输出内容" :rows="10"
              class="process_textarea"></el-input>
    <el-button type="primary" @click="handleAVQuery">查询</el-button>
  </div>
  <div class="result" v-if="isAvQueried">
    <h4>识别结果：</h4>
    <!-- 如果有结果，显示表格 -->
    <el-table v-if="avResults.length > 0" :data="avResults" border style="width: 100%">
      <el-table-column prop="program" label="程序名" />
      <el-table-column prop="match" label="杀软匹配" />
      <el-table-column prop="description" label="描述" />
    </el-table>
    <!-- 如果没有结果，显示提示文字 -->
    <p v-else class="no-results">未识别到任何杀软进程。</p>
  </div>
</template>

<script>
import {QueryAntivirusProcesses} from "../../../wailsjs/go/assistive/Assistive";

export default {
  name: "process",
  data() {
    return {
      // tasklist 查询
      tasklistInput: "",
      avResults: [],
      isAvQueried: false,
    }
  },
  methods: {
    async handleAVQuery() {
      if (!this.tasklistInput.trim()) {
        this.$message.error("请输入有效的 tasklist /svc 输出内容！");
        this.avResults = [];
        this.isAvQueried = false;
        return;
      }

      try {
        const response = await QueryAntivirusProcesses(this.tasklistInput);

        this.avResults = response.map(item => ({
          program: item.program,
          match: item.match,
          description: item.description
        }));

        this.isAvQueried = true;
      } catch (err) {
        this.isAvQueried = true;
      }
    },
  }

}

</script>

<style scoped>
/* 杀软查询输入框 */
.process_textarea {
  margin-bottom: 15px;
}

/* 精确定位 shell-syntax 的 tab-content */
.shell-syntax-active {
  padding: 0 8px 3px 3px;
}


</style>