<template>
  <el-card shadow="always" class="map-query-card">
    <div class="map-query-container">
      <!-- 左侧：选择与参数输入 -->
      <div class="map-form">
        <el-form :model="mapForm" label-width="100px" class="inner-form">
          <el-form-item label="地图服务">
            <el-select v-model="mapForm.provider" placeholder="请选择" @change="resetMapResult">
              <el-option
                  v-for="item in mapProviders"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
              />
            </el-select>
          </el-form-item>

          <!-- 公共 Key 输入 -->
          <el-form-item label="API Key">
            <el-input v-model="mapForm.key" placeholder="输入你的 key" />
          </el-form-item>

          <!-- 动态参数区域 -->
          <template v-if="mapForm.provider === 'amap-walking'">
            <el-form-item label="起点坐标">
              <el-input v-model="mapForm.origin" placeholder="lng,lat(如：116.434307,39.90909)" />
            </el-form-item>
            <el-form-item label="终点坐标">
              <el-input v-model="mapForm.destination" placeholder="lng,lat(如：116.434446,39.90816)" />
            </el-form-item>
          </template>

          <template v-else-if="mapForm.provider === 'amap-geocode'">
            <el-form-item label="坐标">
              <el-input v-model="mapForm.location" placeholder="lng,lat(如：116.434446,39.90816)" />
            </el-form-item>
          </template>

          <template v-else-if="mapForm.provider === 'amap-mini'">
            <el-form-item label="坐标">
              <el-input v-model="mapForm.location" placeholder="lng,lat(如：116.434446,39.90816)" />
            </el-form-item>
            <el-form-item label="AppName">
              <el-input v-model="mapForm.amapMiniAppID" placeholder="如：c589cf63f592ac13bcab35f8cd18f495" />
            </el-form-item>
          </template>

          <template v-else-if="mapForm.provider === 'baidu-web' || mapForm.provider === 'baidu-ios'">
            <el-form-item label="关键字">
              <el-input v-model="mapForm.query" placeholder="如：ATM机" />
            </el-form-item>
            <el-form-item label="关键字">
              <el-input v-model="mapForm.baiduTag" placeholder="如：银行" />
            </el-form-item>
            <el-form-item label="地区">
              <el-input v-model="mapForm.region" placeholder="如：北京" />
            </el-form-item>
          </template>

          <template v-else-if="mapForm.provider === 'qq-web'">
            <el-form-item label="关键词">
              <el-input v-model="mapForm.keyword" placeholder="如：酒店" />
            </el-form-item>
            <el-form-item label="中心坐标">
              <el-input v-model="mapForm.center" placeholder="lat,lng(如：39.908491,116.374328)" />
            </el-form-item>
            <el-form-item label="半径 (m)">
              <el-input v-model="mapForm.radius" placeholder="1000" />
            </el-form-item>
          </template>

          <template v-else-if="mapForm.provider === 'google-geocode'">
            <el-form-item label="地址">
              <el-input v-model="mapForm.googleAddress" placeholder="如：天安门、1600 Amphitheatre Parkway" />
            </el-form-item>
          </template>

          <el-form-item>
            <el-button
                type="primary"
                @click="doMapQuery"
                :loading="mapLoading"
                class="submit-btn"
            >
              发起请求
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 右侧：结果展示 -->
      <div class="map-result">
        <el-card shadow="hover" class="result-card">
          <div class="result-header">
            <i class="el-icon-document"></i>
            <span>返回结果</span>
          </div>
          <div class="result-content" v-if="mapResult">
            <pre class="formatted-json">{{ formattedResult }}</pre>
          </div>
          <div v-else class="no-result">
            暂无数据，请先发起请求
          </div>
        </el-card>
      </div>
    </div>
  </el-card>
</template>

<script>
import axios from "axios";
import { GetPortConfig } from '../../../wailsjs/go/controller/Server';

export default {
  name: "mapCheck",
  data() {
    return {
      // 地图测试相关
      mapProviders: [
        { label: '高德 Walking', value: 'amap-walking' },
        { label: '高德 Geocode(JSAPI)', value: 'amap-geocode' },
        { label: '高德 小程序 RE Geocode', value: 'amap-mini' },
        { label: '百度 Web 搜索', value: 'baidu-web' },
        { label: '百度 iOS 搜索', value: 'baidu-ios' },
        { label: '腾讯 Web 搜索', value: 'qq-web' },
        { label: 'Google Geocoding (JSON)', value: 'google-geocode' },
      ],
      mapForm: {
        provider: '',
        key: '',
        origin: '116.434307,39.90909',
        destination: '116.434446,39.90816',
        location: '116.434446,39.90816',
        query: 'ATM机',
        baiduTag:"银行",
        region: '北京',
        keyword: '酒店',
        center: '39.908491,116.374328',
        radius: '1000',
        amapMiniAppID:"",
        googleAddress: '天安门',
      },
      mapResult: null,
      mapLoading: false,
      apiClient: null,
      portConfig: null,
    }
  },
  async mounted() {
    await this.initApiClient();
  },
  computed: {
    formattedResult() {
      if (!this.mapResult) {
        return '';
      }
      // 如果是字符串（错误信息），直接返回
      if (typeof this.mapResult === 'string') {
        return this.mapResult;
      }
      // 如果是对象，格式化为 JSON
      try {
        return JSON.stringify(this.mapResult, null, 2);
      } catch (e) {
        return String(this.mapResult);
      }
    }
  },
  methods: {
    async initApiClient() {
      try {
        this.portConfig = await GetPortConfig();
        const port = this.portConfig?.mainPort || 52867;
        const baseURL = `http://127.0.0.1:${port}`;
        this.apiClient = axios.create({
          baseURL: baseURL,
          timeout: 30000,
        });
        console.log('[MapCheck] API客户端初始化成功，端口:', port);
      } catch (error) {
        console.error('[MapCheck] 初始化API客户端失败:', error);
        this.$message.error('初始化API客户端失败');
      }
    },

    resetMapResult() {
      this.mapResult = null;
    },

    jsonp(url, cbName) {
      return new Promise((resolve, reject) => {
        const script = document.createElement('script');
        script.type = 'text/javascript';

        // 调试：打印 JSONP 请求
        console.log('[MapCheck][JSONP] create script', {
          cbName,
          url,
        });

        window[cbName] = (res) => {
          console.log('[MapCheck][JSONP] callback called', {
            cbName,
            hasResult: !!res,
          });
          resolve(res);
          delete window[cbName];
          if (script.parentNode) {
            document.head.removeChild(script);
          }
        };

        script.onerror = () => {
          console.error('[MapCheck][JSONP] script error', {
            cbName,
            url,
          });
          reject(new Error('JSONP 请求失败'));
          delete window[cbName];
          if (script.parentNode) {
            document.head.removeChild(script);
          }
        };

        script.src = url;
        document.head.appendChild(script);
      });
    },

    async doMapQuery() {
      if (!this.mapForm.provider || !this.mapForm.key) {
        this.$message.warning('请先选择服务并填写 API Key');
        return;
      }
      console.log('[MapCheck] doMapQuery provider =', this.mapForm.provider);
      this.mapLoading = true;
      const k = encodeURIComponent(this.mapForm.key);
      try {
        switch (this.mapForm.provider) {
          case 'amap-walking':
            this.mapResult = (
                await axios.get(
                    `https://restapi.amap.com/v3/direction/walking?origin=${this.mapForm.origin}&destination=${this.mapForm.destination}&key=${k}`
                )
            ).data;
            break;

          case 'amap-geocode':
            this.mapResult = (
                await axios.get(
                    `https://restapi.amap.com/v3/geocode/regeo?key=${k}&s=rsv3&location=${this.mapForm.location}&callback=jsonp_258885_&platform=JS`
                )
            ).data;
            break;

          case 'amap-mini':
            this.mapResult = (
                await axios.get(
                    `https://restapi.amap.com/v3/geocode/regeo?key=${k}&location=${encodeURIComponent(
                        this.mapForm.location
                    )}&extensions=all&platform=WXJS&appname=${this.mapForm.amapMiniAppID}&sdkversion=1.2.0&logversion=2.0`
                )
            ).data;
            break;

          case 'baidu-web':
          case 'baidu-ios':
          {
            // 通过后端代理请求，解决 CORS 跨域问题
            // 注意：需要在百度开放平台配置正确的浏览器 AK 和 Referer 白名单
            if (!this.apiClient) {
              await this.initApiClient();
            }
            let url = `https://api.map.baidu.com/place/v2/search?query=${encodeURIComponent(
              this.mapForm.query
            )}&tag=${encodeURIComponent(this.mapForm.baiduTag)}&region=${encodeURIComponent(
              this.mapForm.region
            )}&output=json&ak=${this.mapForm.key}`;
            if (this.mapForm.provider === 'baidu-ios') {
              // iOS 端一般还会带 mcode / os 等参数，这里简单模拟一个常见形式
              url += '&mcode=com.didapinche.taxi&os=12.5.6';
            }
            console.log('[MapCheck] baidu url (proxy) =', url);
            // 通过后端代理接口转发请求
            const response = await this.apiClient.get('/api/assistive/baidu-map/proxy', {
              params: { url: url }
            });
            this.mapResult = response.data;
          }
            break;

          case 'qq-web':
          {
            const cbName = `jsonp_qq_${Date.now()}`;
            const url = `https://apis.map.qq.com/ws/place/v1/search?keyword=${encodeURIComponent(
              this.mapForm.keyword
            )}&boundary=nearby(${this.mapForm.center},${this.mapForm.radius})&key=${k}&output=jsonp&callback=${cbName}`;
            console.log('[MapCheck] qq url =', url);
            const data = await this.jsonp(url, cbName);
            this.mapResult = data;
          }
            break;

          case 'google-geocode':
          {
            if (!this.mapForm.googleAddress || !String(this.mapForm.googleAddress).trim()) {
              this.$message.warning('请填写要地理编码的地址');
              return;
            }
            if (!this.apiClient) {
              await this.initApiClient();
            }
            const targetUrl = `https://maps.googleapis.com/maps/api/geocode/json?address=${encodeURIComponent(
              this.mapForm.googleAddress.trim()
            )}&key=${this.mapForm.key}`;
            console.log('[MapCheck] google geocode url (proxy) =', targetUrl.replace(this.mapForm.key, '***'));
            const response = await this.apiClient.get('/api/assistive/google-map/proxy', {
              params: { url: targetUrl },
            });
            this.mapResult = response.data;
          }
            break;
        }
      } catch (err) {
        console.error('[MapCheck] doMapQuery error', this.mapForm.provider, err);
        // 统一把错误也塞进结果里，方便查看
        this.mapResult = {
          error: err.toString(),
          provider: this.mapForm.provider,
        };
      } finally {
        this.mapLoading = false;
      }
    },


  }
}
</script>

<style scoped>
/* 地图测试 样式 */

.el-card{
  background: none;
}

.map-query-card {
  width: auto;

}
.map-query-container {
  display: flex;
  gap: 20px;
}
.map-form {
  flex: 1 1 auto;
  min-width: 380px;
}
.inner-form {
  background: #ffffff;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}
.submit-btn {
  width: 100%;
}
.map-result {
  flex: 0 0 520px; /* 固定右侧结果区域宽度，避免压缩左侧 */
  max-width: 520px;
}
.result-card {
  height: 100%;
  padding: 0;
}
.result-header {
  display: flex;
  align-items: center;
  background: #f0f9ff;
  padding: 12px 20px;
  font-weight: 500;
  border-bottom: 1px solid #e4e7ed;
}
.result-header i {
  margin-right: 8px;
  font-size: 16px;
  color: #409EFF;
}
.result-content {
  padding: 20px;
  max-height: 500px;
  overflow-y: auto;
  overflow-x: hidden;
  background: #fafafa;
}
.result-content pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  word-break: break-all; /* 长字段也强制换行，避免撑布局 */
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #333;
  margin: 0;
}
.formatted-json {
  background: #ffffff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 16px;
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.05);
}
.no-result {
  padding: 40px;
  text-align: center;
  color: #909399;
}
</style>