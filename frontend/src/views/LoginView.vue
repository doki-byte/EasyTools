<template>
  <div class="login" @contextmenu.prevent>
<!--    <div class="login">-->
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
              登录
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from "vue";
import { useRouter } from "vue-router";
import { ElNotification } from "element-plus";
import { setToken } from "@/utils/token";
import { Login } from "../../wailsjs/go/controller/User";

const router = useRouter();

const isLoggingIn = ref(false); // 防止重复点击

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

// 登录处理逻辑
const handleLogin = async () => {
  try {
    if (loginForm.username === "" || loginForm.password === ""){
      ElNotification({
        title: "温馨提示",
        message: "进门请说芝麻开门 O(∩_∩)O",
        type: "error",
      });
      return;
    }
    isLoggingIn.value = true;

    // 特殊账户硬编码验证
    const isSpecialAccount =
        loginForm.username === "muhan" &&
        loginForm.password === "muhan";

    if (isSpecialAccount) {
      // 特殊账户直接登录
      setToken("muhan");
      ElNotification({
        title: "特殊登录",
        message: "欢迎沐寒开发大大回家 🎉",
        type: "success",
      });
      await router.push({ name: "tool" });
      return; // 提前返回避免执行后续代码
    }
    try{
      await Login(
          loginForm.username,
          loginForm.password
      );
      // 普通账户登录成功处理
      setToken(loginForm.username);
      ElNotification({
        title: "登录成功",
        message: `欢迎 ${loginForm.username} 回家^_^`,
        type: "success",
      });
      await router.push({ name: "tool" });
    } catch (err){
      // 统一错误提示（包含所有异常情况）
      ElNotification({
        title: "登录失败",
        message: "主人忘记我了吗 o(╥﹏╥)o",
        type: "error",
        duration: 3000
      })
    }

  } catch (err) {
  } finally {
    isLoggingIn.value = false;
  }
};

</script>


<style lang="scss" scoped>
/* 确保 html 和 body 填满整个窗口 */
html,
body {
  height: 100%;
  /* 确保高度是100% */
  margin: 0;
  /* 去除默认的外边距 */
  padding: 0;
  /* 去除默认的内边距 */
  overflow: hidden;
  /* 禁止滚动条 */
}

.login {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ebf4ff;
  background: url("../assets/loginBackGround.jpg") no-repeat center / cover;
}

.left {
  width: 300px;
  height: 300px;
  // background: url("../assets/loginBg.png") no-repeat center / cover;
  margin: -15px 35px 34px 23px;
}

.right {
  .form {
    width: 300px;
  }
}
</style>
