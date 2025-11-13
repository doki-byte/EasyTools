<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Config,useConfigStore } from "./store/types";
import { Notification } from "@arco-design/web-vue";
import { IconEye, IconEyeInvisible } from '@arco-design/web-vue/es/icon';

const configState = useConfigStore();
const formData = ref({
  Code: 0,
  Error: "",
  FilePath: computed({
    get: () => configState.getFilePath(),
    set: (value) => configState.setFilePath(value),
  }),
  CoroutineCount: computed({
    get: () => configState.getCoroutineCount(),
    set: (value) => configState.setCoroutineCount(value),
  }),
  Timeout: computed({
    get: () => configState.getTimeout(),
    set: (value) => configState.setTimeout(value),
  }),
  SocksAddress: computed({
    get: () => configState.getSocksAddress(),
    set: (value) => configState.setSocksAddress(value),
  }),

  Status: 0,

  Email: computed({
    get: () => configState.getEmail(),
    set: (value) => configState.setEmail(value),
  }),
  FofaKey: computed({
    get: () => configState.getFofaKey(),
    set: (value) => configState.setFofaKey(value),
  }),
  HunterKey: computed({
    get: () => configState.getHunterKey(),
    set: (value) => configState.setHunterKey(value),
  }),
  QuakeKey: computed({
    get: () => configState.getQuakeKey(),
    set: (value) => configState.setQuakeKey(value),
  }),
  CheckTimeOut: computed({
    get: () => configState.getCheckTimeout(),
    set: (value) => configState.setCheckTimeout(value),
  }),
  MaxPage: computed({
    get: () => configState.getMaxpage(),
    set: (value) => configState.setMaxpage(value),
  }),
  Country:computed({
    get: () => configState.getCountry(),
    set: (value) => configState.setCountry(value),
  }),
  Xui: computed({
    get: () => configState.getXui(),
    set: (value) => configState.setXui(value),
  })

});

const passwordVisibility = ref({
  FofaEmail: false,
  FofaKey: false,
  QuakeKey: false,
  HunterKey: false,
});

const rules = {
  CoroutineCount: [
    { required: true, message: "请输入协程数量", trigger: "blur" },
  ],
  Timeout: [
    { required: true, message: "请指定超时时长", trigger: "blur" },
  ],
  FofaEmail: [
    { required: true, message: "请输入Fofa邮箱", trigger: "blur" },
  ],
  FofaKey: [
    { required: true, message: "请输入Fofa秘钥", trigger: "blur" },
  ],
  QuakeKey: [
    { required: true, message: "请输入Quake秘钥", trigger: "blur" },
  ],
  HunterKey: [
    { required: true, message: "请输入Hunter秘钥", trigger: "blur" },
  ],
  TimeOut: [
    { required: true, message: "请输入采集超时时间", trigger: "blur" },
  ],
  MaxPage: [
    { required: true, message: "请输入采集最大页数", trigger: "blur" },
  ],
};

const disabled = ref(false);

function saveConfig() {
  disabled.value = true;

  const configData: Config = {
    Code: formData.value.Code,
    Country: formData.value.Country,
    Error: formData.value.Error,
    FilePath: formData.value.FilePath,
    CoroutineCount: formData.value.CoroutineCount,
    Timeout: formData.value.Timeout.toString(),
    SocksAddress: formData.value.SocksAddress,
    Status: formData.value.Status,
    Email: formData.value.Email,
    FofaKey: formData.value.FofaKey,
    HunterKey: formData.value.HunterKey,
    QuakeKey: formData.value.QuakeKey,
    Maxpage: formData.value.MaxPage.toString(),
    Xui: formData.value.Xui,
    LiveProxies: configState.getLiveProxies(),
    AllProxies: configState.getAllProxies(),
    LiveProxyLists: [] as string[],  // 初始化为字符串数组
  };

  // 调用 store 的 saveConfig 方法
  configState.saveConfig(configData)
      .then(() => {
        Notification.success({ title: "成功", content: "配置已保存！" });
      })
      .catch((err) => {
        Notification.error({ title: "保存失败", content: err.message });
      })
      .finally(() => {
        disabled.value = false;
      });
}

onMounted(() => {
  configState.getProfile();
});
</script>

<template>
  <a-row :gutter="12">
    <a-col :span="18">
      <a-input
          :readonly="true"
          placeholder="配置文件在当前文件运行路径下的config.db文件proxyconfig库中"
      ></a-input>
    </a-col>
    <a-col :span="6">
      <a-button
          type="outline"
          size="medium" long
          @click="saveConfig"
      >保存配置</a-button>

    </a-col>
  </a-row>

  <br />
  <p class="config-viewer-title">秘钥配置：</p>
  <a-form ref="formRef" :model="formData" :rules="rules" :layout="'vertical'">
    <a-row :gutter="[24, 12]">
      <a-col :span="12">
        <a-form-item label="Fofa邮箱" name="Email">
          <a-input
              v-model="formData.Email"
              placeholder="请输入Fofa邮箱"
              :type="passwordVisibility.FofaEmail ? 'text' : 'password'"
          >
            <template #suffix>
              <a-button
                  type="text"
                  @click.stop="passwordVisibility.FofaEmail = !passwordVisibility.FofaEmail"
              >
                <icon-eye v-if="passwordVisibility.FofaEmail" />
                <icon-eye-invisible v-else />
              </a-button>
            </template>
          </a-input>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item label="Fofa秘钥" name="FofaKey">
          <a-input
              v-model="formData.FofaKey"
              placeholder="请输入Fofa秘钥"
              :type="passwordVisibility.FofaKey ? 'text' : 'password'"
          >
            <template #suffix>
              <a-button
                  type="text"
                  @click.stop="passwordVisibility.FofaKey = !passwordVisibility.FofaKey"
              >
                <icon-eye v-if="passwordVisibility.FofaKey" />
                <icon-eye-invisible v-else />
              </a-button>
            </template>
          </a-input>
        </a-form-item>
      </a-col>
    </a-row>

    <a-row :gutter="[24, 12]">
      <a-col :span="12">
        <a-form-item label="Quake秘钥" name="QuakeKey">
          <a-input
              v-model="formData.QuakeKey"
              placeholder="请输入Quake秘钥"
              :type="passwordVisibility.QuakeKey ? 'text' : 'password'"
          >
            <template #suffix>
              <a-button
                  type="text"
                  @click.stop="passwordVisibility.QuakeKey = !passwordVisibility.QuakeKey"
              >
                <icon-eye v-if="passwordVisibility.QuakeKey" />
                <icon-eye-invisible v-else />
              </a-button>
            </template>
          </a-input>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item label="Hunter秘钥" name="HunterKey">
          <a-input
              v-model="formData.HunterKey"
              placeholder="请输入Hunter秘钥"
              :type="passwordVisibility.HunterKey ? 'text' : 'password'"
          >
            <template #suffix>
              <a-button
                  type="text"
                  @click.stop="passwordVisibility.HunterKey = !passwordVisibility.HunterKey"
              >
                <icon-eye v-if="passwordVisibility.HunterKey" />
                <icon-eye-invisible v-else />
              </a-button>
            </template>
          </a-input>
        </a-form-item>
      </a-col>
    </a-row>

    <a-row :gutter="[24, 12]">
      <a-col :span="12">
        <a-form-item label="超时时间" name="TimeOut">
          <a-input-number
              v-model="formData.Timeout"
              :mode="'button'"
          ></a-input-number>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item label="采集最大页数" name="MaxPage">
          <a-input-number
              v-model="formData.MaxPage"
              :mode="'button'"
          ></a-input-number>
        </a-form-item>
      </a-col>
    </a-row>
    <a-row :gutter="[24, 12]">
      <a-col :span="12">
        <a-form-item label="协程数量" name="CoroutineCount">
          <a-input-number
              v-model="formData.CoroutineCount"
              :mode="'button'"
          ></a-input-number>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item label="代理地区" name="Country">
          <a-select
              v-model="formData.Country"
              placeholder="请选择代理地区"
              hide-on-select
              allow-clear
          >
            <a-option value="0">所有</a-option>
            <a-option value="1">国内</a-option>
            <a-option value="2">国外</a-option>
          </a-select>
        </a-form-item>
      </a-col>
    </a-row>
    <a-row :gutter="[24, 12]">
      <a-col :span="12">
        <a-form-item label="白嫖Xui(开启后只会抓取xui数据,点击使用只会保存excel,请手动打开excel进行调用)" name="Xui">
          <a-select
              v-model="formData.Xui"
              placeholder="请选择是否开启Xui抓取"
              hide-on-select
              allow-clear
          >
            <a-option value="0">否</a-option>
            <a-option value="1">是（不添加http代理）</a-option>
            <a-option value="2">是（添加http代理）</a-option>
          </a-select>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item label="Socks地址" name="SocksAddress">
          <a-input v-model="formData.SocksAddress"></a-input>
        </a-form-item>
      </a-col>
    </a-row>
  </a-form>
</template>

<style scoped>
.config-viewer-title{
  text-align: left;
  color: #000000;
  font-weight: bold;
  font-size: 16px;
}
</style>