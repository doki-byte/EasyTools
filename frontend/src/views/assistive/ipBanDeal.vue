<template>
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
</template>

<script>
export default {
  name: "ipBanDeal",
  data(){
    return {
      maliciousIPInput: "",
      whiteListIPInput: "",
      uniqueIPs: [],
      duplicateIPs: [],
    }
  },
  methods: {
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


  }
}
</script>


<style scoped>
/* 蓝队封禁IP工具 */
.ip-ban-content{
  padding: 0 8px 3px 3px;
}

.ip-ban-info-card .info{
  color: #ed9108;
}

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
  padding: 8px 16px;
  margin: 25px auto 10px;
  border-radius: 5px;
  width: 100%;
  font-size: 18px;
  text-align: center;
  background-color: #5cb85c;
}

</style>