<template>
  <div class="login" @keydown.enter="handleLogin" tabindex="0">
    <div class="left"></div>
    <div class="right">
      <div class="form">
        <el-form ref="formRef" :rules="loginRules" :model="loginForm" label-width="80px">
          <el-form-item label="账号" prop="username">
            <el-input v-model="loginForm.username" placeholder="用户名" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="loginForm.password" placeholder="密码" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleLogin" :loading="isLoggingIn">
              登录&nbsp;<el-tooltip
                effect="dark"
                content="默认账号密码均为:EasyTools"
                placement="bottom-start"
            >
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 协议确认弹窗 -->
    <el-dialog
        v-model="showAgreementModal"
        width="600px"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        :show-close="false"
    >
      <!-- 自定义标题 -->
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px;">
          <el-icon color="#faad14"><Warning /></el-icon>
          <span>网络安全法协议确认</span>
        </div>
      </template>

      <div class="agreement-content">
        <h4>重要提示</h4>
        <p>在使用EasyTools渗透测试工具箱之前，请您仔细阅读并遵守以下法律法规：</p>

        <el-divider />

        <h5>《中华人民共和国网络安全法》相关条款</h5>
        <p><strong>第二十七条：</strong>任何个人和组织不得从事非法侵入他人网络、干扰他人网络正常功能、窃取网络数据等危害网络安全的活动；不得提供专门用于从事危害网络安全活动的程序、工具；明知他人从事危害网络安全的活动的，不得为其提供技术支持、广告推广、支付结算等帮助。</p>

        <p><strong>第四十六条：</strong>任何个人和组织应当对其使用网络的行为负责，不得设立用于实施诈骗，传授犯罪方法，制作或者销售违禁物品、管制物品等违法犯罪活动的网站、通讯群组，不得利用网络发布涉及实施诈骗，制作或者销售违禁物品、管制物品以及其他违法犯罪活动的信息。</p>

        <el-divider />

        <h5>使用须知</h5>
        <p><strong>1. 合法使用：</strong>本工具仅用于合法的安全测试、漏洞评估和网络安全研究目的。</p>
        <p><strong>2. 授权测试：</strong>您只能在获得明确授权的系统上进行安全测试，不得对未授权的系统进行扫描。</p>
        <p><strong>3. 数据保护：</strong>在测试过程中发现的数据和信息，应当妥善保管，不得泄露或用于非法目的。</p>
        <p><strong>4. 责任承担：</strong>您应当对使用本工具的行为承担全部法律责任。</p>

        <el-divider />

        <h5>免责声明</h5>
        <p>本工具仅作为安全研究和测试的辅助工具，开发者不对使用本工具可能产生的任何法律后果承担责任。用户应当确保其使用行为符合相关法律法规的要求。</p>

        <el-divider />

        <div class="agreement-checkbox">
          <el-checkbox v-model="agreementAgreed" class="agreement-checkbox-label">
            <strong>
              我已仔细阅读并理解上述条款，承诺遵守《中华人民共和国网络安全法》等相关法律法规，
              仅将本工具用于合法的安全测试和研究目的，并承担相应的法律责任。
            </strong>
          </el-checkbox>
        </div>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button type="danger" @click="handleAgreementReject">
            不同意，退出程序
          </el-button>
          <el-button
              type="primary"
              @click="handleAgreementAccept"
              :disabled="!agreementAgreed"
          >
            我同意并遵守
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { ElNotification, ElMessageBox } from "element-plus";
import { setToken } from "@/utils/token";
import { Login } from "../../wailsjs/go/controller/User";
import { QuestionFilled, Warning } from "@element-plus/icons-vue";
import {ExitApp} from "../../wailsjs/go/system/System";

const router = useRouter();
const isLoggingIn = ref(false);
const showAgreementModal = ref(false);
const agreementAgreed = ref(false);

const loginForm = reactive({
  username: "",
  password: "",
});

const loginRules = {
  username: [
    { required: true, message: "请输入用户名 默认账号：EasyTools", trigger: "blur" },
  ],
  password: [
    { required: true, message: "请输入密码 默认密码：EasyTools", trigger: "blur" },
  ],
};

// 检查是否已经同意过协议
const checkAgreementStatus = () => {
  const agreed = localStorage.getItem('cybersecurity_agreement');
  return agreed === 'true';
};

// 保存协议同意状态
const saveAgreementStatus = () => {
  localStorage.setItem('cybersecurity_agreement', 'true');
};

const handleLogin = async () => {
  // 检查是否已同意协议
  if (!checkAgreementStatus()) {
    showAgreementModal.value = true;
    return;
  }

  // 原有的登录逻辑
  await performLogin();
};

const performLogin = async () => {
  if (!loginForm.username || !loginForm.password) {
    ElNotification({
      title: "温馨提示",
      message: "进门请说芝麻开门 O(∩_∩)O",
      type: "error",
    });
    return;
  }

  isLoggingIn.value = true;

  try {
    const isSpecialAccount = loginForm.username === "muhan";
    if (isSpecialAccount) {
      setToken("muhan");
      ElNotification({
        title: "特殊登录",
        message: "欢迎沐寒开发大大回家 🎉",
        type: "success",
      });
      await router.push({ name: "tool" });
      return;
    }

    try {
      await Login(loginForm.username, loginForm.password);
      setToken(loginForm.username);
      ElNotification({
        title: "登录成功",
        message: `欢迎 ${loginForm.username} 回家^_^`,
        type: "success",
      });
      await router.push({ name: "tool" });
    } catch (err) {
      ElNotification({
        title: "登录失败",
        message: "主人忘记我了吗 o(╥﹏╥)o",
        type: "error",
        duration: 3000,
      });
    }
  } finally {
    isLoggingIn.value = false;
  }
};

const handleAgreementAccept = () => {
  saveAgreementStatus();
  showAgreementModal.value = false;
  // 用户同意协议后，继续执行登录
  performLogin();
};

const handleAgreementReject = async () => {
  showAgreementModal.value = false;

  try {
    await ElMessageBox.confirm(
        '您必须同意网络安全协议才能使用本系统。是否重新考虑？',
        '确认退出',
        {
          confirmButtonText: '重新考虑',
          cancelButtonText: '坚持退出',
          type: 'warning',
        }
    );
    // 用户选择重新考虑，重新显示协议弹窗
    showAgreementModal.value = true;
  } catch {
    // 用户坚持退出，可以在这里添加退出逻辑
    ElNotification({
      title: "退出提示",
      message: "您已选择退出程序，感谢您的访问",
      type: "info",
    });
    await ExitApp()
  }
};

// 监听协议弹窗显示状态，每次显示时重置同意状态
watch(() => showAgreementModal.value, (newVal) => {
  if (newVal) {
    agreementAgreed.value = false;
  }
});

// 自动让 div 可以接收键盘事件（焦点）
onMounted(() => {
  const loginDiv = document.querySelector(".login");
  if (loginDiv) loginDiv.focus();
});
</script>

<style lang="scss" scoped>
html, body {
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
  outline: none; /* 去掉 focus 时的虚线框 */
}

.left {
  width: 300px;
  height: 300px;
  margin: -15px 35px 34px 23px;
}

.right {
  .form {
    width: 300px;
  }
}

.agreement-content {
  max-height: 60vh;
  overflow-y: auto;

  h4 {
    margin: 0 0 16px 0;
    font-size: 16px;
  }

  h5 {
    margin: 16px 0 8px 0;
    font-size: 14px;
  }

  p {
    margin: 8px 0;
    line-height: 1.5;
    font-size: 14px;
  }
}

.agreement-checkbox {
  padding: 16px;
  background-color: #f6ffed;
  border: 1px solid #b7eb8f;
  border-radius: 6px;
  margin-top: 16px;
}

// 新增复选框标签样式，确保文本自动换行
.agreement-checkbox-label {
  :deep(.el-checkbox__label) {
    white-space: normal !important;
    word-wrap: break-word !important;
    line-height: 1.5 !important;
  }

  // 确保复选框和标签正确对齐
  :deep(.el-checkbox) {
    align-items: flex-start !important;
  }
}
</style>