<template>
  <div class="login" @keydown.enter="handleAgreementAccept" tabindex="0">
    <div class="left"></div>
    <div class="right">
      <div class="form">
        <div class="welcome-section">
          <h2 class="welcome-title">欢迎使用EasyTools工具箱</h2>
          <p class="welcome-hint">使用免责声明与网络安全法协议每 7 个自然日需重新阅读并确认一次。</p>
          <div class="agreement-collapse">
            <el-collapse v-model="activeNames" @change="handleCollapseChange">
              <el-collapse-item name="1">
                <template #title>
                  <div class="collapse-title">
                    <el-icon>
                      <Warning />
                    </el-icon>
                    <span>使用免责声明 & 网络安全法协议</span>
                  </div>
                </template>
                <div class="agreement-content">
                  <!-- ========= 使用免责声明（放在网络安全法前面） ========= -->
                  <h4>📢 使用免责声明</h4>
                  <p><strong>重要提示：</strong>本工具（EasyTools工具箱）仅供合法授权范围内的网络安全测试、漏洞评估及安全研究使用。</p>
                  <p><strong>1.
                      使用者责任：</strong>您理解并同意，本工具仅提供基础功能支持，任何使用本工具所产生的一切行为后果（包括但不限于法律纠纷、数据泄露、系统损坏、业务中断等）均由您自行承担全部责任。</p>
                  <p><strong>2. 作者免责：</strong>本工具的作者及贡献者不对任何直接或间接损失承担责任，包括但不限于因使用本工具而导致的任何法律诉讼、行政处罚、经济损失或名誉损害。</p>
                  <p><strong>3. 合法合规：</strong>您承诺在使用本工具时严格遵守中华人民共和国及您所在国家/地区的法律法规，不得利用本工具从事任何非法活动（如未授权入侵、数据窃取、恶意破坏等）。
                  </p>
                  <p><strong>4. 风险自担：</strong>本工具按“现状”提供，不提供任何明示或默示的担保，包括但不限于适销性、特定用途适用性及无侵权担保。您应自行评估风险并采取必要的防护措施。</p>
                  <p><strong>5. 协议接受：</strong>若您不同意本免责声明，请立即停止使用并退出本工具箱。继续使用即表示您已阅读、理解并接受本声明的全部条款。</p>

                  <el-divider />

                  <h4>⚖️ 网络安全法协议</h4>
                  <h5>《中华人民共和国网络安全法》相关条款</h5>
                  <p>
                    <strong>第二十七条：</strong>任何个人和组织不得从事非法侵入他人网络、干扰他人网络正常功能、窃取网络数据等危害网络安全的活动；不得提供专门用于从事危害网络安全活动的程序、工具；明知他人从事危害网络安全的活动的，不得为其提供技术支持、广告推广、支付结算等帮助。
                  </p>
                  <p>
                    <strong>第四十六条：</strong>任何个人和组织应当对其使用网络的行为负责，不得设立用于实施诈骗，传授犯罪方法，制作或者销售违禁物品、管制物品等违法犯罪活动的网站、通讯群组，不得利用网络发布涉及实施诈骗，制作或者销售违禁物品、管制物品以及其他违法犯罪活动的信息。
                  </p>

                  <el-divider />

                  <div class="agreement-checkbox">
                    <el-checkbox v-model="agreementAgreed" class="agreement-checkbox-label">
                      <strong>
                        我已仔细阅读并理解上述“使用免责声明”及《网络安全法》相关条款，
                        承诺遵守法律法规，自愿承担使用本工具的一切责任与后果。
                      </strong>
                    </el-checkbox>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
          <div class="action-buttons">
            <el-button type="danger" @click="handleAgreementReject" size="large">
              不同意，退出程序
            </el-button>
            <el-button type="primary" @click="handleAgreementAccept" :disabled="!agreementAgreed" size="large">
              我同意并遵守
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { ElNotification, ElMessageBox } from "element-plus";
import { Warning } from "@element-plus/icons-vue";
import { ExitApp } from "../../wailsjs/go/system/System";
import { applyThemeToApp } from "@/utils/themeBackground";
import { getFirstActiveMenuName } from "@/utils/menuConfig";
import { clearCloudServerTerminalStorageOnExit } from "@/utils/cloudServerTerminalStorage";
import {
  isCybersecurityAgreementValid,
  saveCybersecurityAgreementTimestamp,
} from "@/utils/cybersecurityAgreement";

const router = useRouter();
const activeNames = ref([]); // 默认收缩，不展开
const agreementAgreed = ref(false);

const checkAgreementStatus = () => isCybersecurityAgreementValid();

const saveAgreementStatus = () => saveCybersecurityAgreementTimestamp();

// ---------- 进入主页 ----------
async function proceedToApp() {
  try {
    const firstActiveMenu = await getFirstActiveMenuName();
    await router.push({ name: firstActiveMenu });
    setTimeout(async () => {
      await applyThemeToApp(true);
    }, 100);
  } catch (error) {
    console.error("跳转失败:", error);
    ElNotification.error({
      title: "进入失败",
      message: error.message || "无法进入主页，请联系管理员",
      duration: 0,
    });
  }
}

// ---------- 折叠面板变化（保留）----------
const handleCollapseChange = (val) => {
  // 当展开时，可以在这里添加逻辑
};

// ---------- 同意按钮点击 ----------
const handleAgreementAccept = async () => {
  if (!agreementAgreed.value) {
    ElNotification({
      title: "提示",
      message: "请先阅读并同意使用免责声明及网络安全法协议",
      type: "warning",
    });
    return;
  }
  saveAgreementStatus();
  await proceedToApp();
};

// ---------- 不同意退出 ----------
const handleAgreementReject = async () => {
  try {
    await ElMessageBox.confirm(
      '您必须同意使用免责声明及网络安全法协议才能使用本系统。是否重新考虑？',
      '确认退出',
      {
        confirmButtonText: '重新考虑',
        cancelButtonText: '坚持退出',
        type: 'warning',
      }
    );
    // 用户选择重新考虑，不做任何操作
  } catch {
    ElNotification({
      title: "退出提示",
      message: "您已选择退出程序，感谢您的访问",
      type: "info",
    });
    clearCloudServerTerminalStorageOnExit();
    await ExitApp();
  }
};

// ---------- 组件挂载：如果已经同意过，直接跳转主页 ----------
onMounted(async () => {
  const loginDiv = document.querySelector(".login");
  if (loginDiv) loginDiv.focus();

  if (checkAgreementStatus()) {
    agreementAgreed.value = true;
    await proceedToApp();
  }
});
</script>

<style lang="scss" scoped>
html,
body {
  height: 100%;
  margin: 0;
  padding: 0;
  overflow: hidden;
}

.login {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ebf4ff url("/assets/system/loginBackGround.jpg") no-repeat center / cover;
  outline: none;
}

.left {
  width: 300px;
  height: 300px;
  margin: -15px 35px 34px 23px;
}

.right {
  .form {
    width: 500px;
    background: rgba(255, 255, 255);
    padding: 20px;
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }
}

.welcome-section {
  width: 100%;
}

.welcome-title {
  text-align: center;
  color: #333;
  margin-bottom: 8px;
  font-size: 24px;
  font-weight: 600;
}

.welcome-hint {
  text-align: center;
  color: #888;
  font-size: 13px;
  line-height: 1.5;
  margin: 0 0 20px 0;
}

.agreement-collapse {
  margin-bottom: 20px;
}

.collapse-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: #333;
}

.agreement-content {
  max-height: 60vh;
  overflow-y: auto;
  padding: 10px 0;

  h4 {
    margin: 0 0 16px 0;
    font-size: 16px;
    color: #333;
  }

  h5 {
    margin: 16px 0 8px 0;
    font-size: 14px;
    color: #333;
  }

  p {
    margin: 8px 0;
    line-height: 1.6;
    font-size: 14px;
    color: #666;
  }
}

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 15px;
  margin-top: 20px;
}

:deep(.el-form-item) {
  margin-bottom: 20px;
}

:deep(.el-form-item__label) {
  color: #333;
  font-weight: 500;
}

:deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.8) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1) !important;
  border: 1px solid rgba(0, 0, 0, 0.1) !important;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15) !important;
  border-color: rgba(64, 150, 255, 0.5) !important;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 2px 8px rgba(64, 150, 255, 0.3) !important;
  border-color: #4096ff !important;
}

:deep(.el-input__inner) {
  background-color: transparent !important;
  color: #333 !important;
}

:deep(.el-input__inner::placeholder) {
  color: rgba(0, 0, 0, 0.5) !important;
}

.agreement-checkbox {
  padding: 16px;
  background-color: #f6ffed;
  border: 1px solid #b7eb8f;
  border-radius: 6px;
  margin-top: 16px;
}

.agreement-checkbox-label {
  :deep(.el-checkbox__label) {
    white-space: normal !important;
    word-wrap: break-word !important;
    line-height: 1.5 !important;
  }

  :deep(.el-checkbox) {
    align-items: flex-start !important;
  }
}
</style>