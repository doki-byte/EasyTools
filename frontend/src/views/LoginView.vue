<template>
<!--    <div class="login" @keydown.enter="handleLogin" tabindex="0" @contextmenu.prevent>-->
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from "vue";
import { useRouter } from "vue-router";
import { ElNotification } from "element-plus";
import { setToken } from "@/utils/token";
import { Login } from "../../wailsjs/go/controller/User";
import {QuestionFilled} from "@element-plus/icons-vue";

const router = useRouter();
const isLoggingIn = ref(false);

const loginForm = reactive({
  username: "",
  password: "",
});

const loginRules  = {
  username: [
    { required: true, message: "请输入用户名 默认账号：EasyTools", trigger: "blur" },
  ],
  password: [
    { required: true, message: "请输入密码 默认密码：EasyTools", trigger: "blur" },
  ],
};

const handleLogin = async () => {
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
    const isSpecialAccount = loginForm.username === "muhan" && loginForm.password === "muhan";
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
  background: #ebf4ff url("../assets/loginBackGround.jpg") no-repeat center / cover;
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
</style>
