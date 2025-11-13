<template>
  <div class="id-generator" @contextmenu.prevent>
    <header class="header">
      <h1>信息生成</h1>
      <span style="color: yellow">以下信息均为随机生成,与现实世界无关。</span>
    </header>

    <main class="main-content">
      <!-- 左侧表单 -->
      <section class="form-section">
        <el-card shadow="always" class="form-card">
          <el-tabs v-model="activeTab" type="card">
            <!-- 密码生成 -->
            <el-tab-pane label="密码生成" name="password">
              <h2 class="section-title">密码设置</h2>
              <el-form :model="formPassword" label-width="100px">
                <el-form-item label="密码长度">
                  <el-input-number v-model="formPassword.length" :min="6" :max="65"
                                   style="width: 100%;"></el-input-number>
                </el-form-item>

                <el-form-item label="包含字符">
                  <el-checkbox-group v-model="formPassword.includeChars">
                    <el-checkbox :label="'数字 (0-9)'" :value="'数字 (0-9)'"></el-checkbox>
                    <el-checkbox :label="'小写字母 (a-z)'" :value="'小写字母 (a-z)'"></el-checkbox>
                    <el-checkbox :label="'大写字母 (A-Z)'" :value="'大写字母 (A-Z)'"></el-checkbox>
                    <el-checkbox :label="'特殊字符 (!@#$%^&*()...)'"
                                 :value="'特殊字符 (!@#$%^&*()...)'"></el-checkbox>
                  </el-checkbox-group>
                </el-form-item>

                <el-form-item label="排除字符">
                  <el-input v-model="formPassword.excludeChars"
                            placeholder="例如：O, 0, I, l"></el-input>
                </el-form-item>

                <el-form-item label="生成数量">
                  <el-input-number v-model="formPassword.generateCount" :min="1" :max="100000"
                                   style="width: 100%;"></el-input-number>
                </el-form-item>

                <el-button type="primary" @click="generateMultiplePasswords"
                           class="generate-button">生成密码</el-button>
              </el-form>
            </el-tab-pane>

            <!-- 手机号生成 -->
            <el-tab-pane label="手机号生成" name="phone">
              <h2 class="section-title">手机号设置</h2>
              <el-form :model="formPhone" label-width="100px">
                <el-form-item label="省">
                  <el-select v-model="formPhone.province" placeholder="请选择省"
                             @change="updatePhoneCities" :loading="loadingPhoneProvince">
                    <el-option v-for="(cities, province) in phoneProvinceMap" :key="province"
                               :label="province" :value="province"></el-option>
                    <el-option v-if="loadingPhoneProvince" :key="'loading-phone-province'"
                               :label="'数据正在加载...'" :value="true" disabled></el-option>
                  </el-select>
                </el-form-item>

                <el-form-item label="市">
                  <el-select v-model="formPhone.city" placeholder="请选择市"
                             @change="updatePhoneCounties">
                    <el-option v-for="(countyData, city) in phoneCityMap" :key="city" :label="city"
                               :value="city"></el-option>
                  </el-select>
                </el-form-item>

                <el-form-item label="运营商">
                  <el-select v-model="formPhone.county" placeholder="运营商">
                    <el-option v-for="(phoneNumbers, county) in phoneCountyMap" :key="county"
                               :label="county" :value="county"></el-option>
                  </el-select>
                </el-form-item>

                <el-form-item label="生成数量">
                  <el-input-number v-model="formPhone.generateCount" :min="1" :max="100000"
                                   style="width: 100%;"></el-input-number>
                </el-form-item>

                <el-button type="primary" @click="generateMultiplePhones"
                           class="generate-button">生成手机号</el-button>
              </el-form>
            </el-tab-pane>

            <!-- 身份证生成 -->
            <el-tab-pane v-if="showIDTab" label="身份证生成" name="id">
              <h2 class="section-title">生成设置</h2>
              <el-form :model="formID" label-width="100px">
                <el-form-item label="省">
                  <el-select v-model="formID.province" placeholder="请选择省" @change="updateCities"
                             :loading="loadingProvince">
                    <el-option v-for="(cities, province) in provinceMap" :key="province"
                               :label="province" :value="province"></el-option>
                    <el-option v-if="loadingProvince" :key="'loading-province'" :label="'数据正在加载...'"
                               :value="true" disabled></el-option>
                  </el-select>
                </el-form-item>

                <el-form-item label="市">
                  <el-select v-model="formID.city" placeholder="请选择市" @change="updateCounties">
                    <el-option v-for="(counties, city) in cityMap" :key="city" :label="city"
                               :value="city"></el-option>
                  </el-select>
                </el-form-item>

                <el-form-item label="县(区)">
                  <el-select v-model="formID.county" placeholder="请选择县(区)">
                    <el-option v-for="(code, county) in countyMap" :key="county" :label="county"
                               :value="county"></el-option>
                  </el-select>
                </el-form-item>

                <el-form-item label="性别">
                  <el-radio-group v-model="formID.gender">
                    <el-radio value="男">男</el-radio>
                    <el-radio value="女">女</el-radio>
                  </el-radio-group>
                </el-form-item>

                <el-form-item label="出生日期">
                  <div style="display:flex;gap:8px;align-items:center;">
                    <el-radio-group v-model="formID.birthdayMode" size="small" style="margin-right: 8px;">
                      <el-radio-button value="picker">选择</el-radio-button>
                      <el-radio-button value="manual">手动输入</el-radio-button>
                    </el-radio-group>

                    <div style="flex:1;">
                      <el-date-picker
                          v-if="formID.birthdayMode === 'picker'"
                          v-model="formID.birthday"
                          type="date"
                          placeholder="选择出生日期"
                          style="width: 100%;"
                          :editable="true"
                          @change="onBirthdayPickerChange"
                      ></el-date-picker>

                      <el-input
                          v-else
                          v-model="formID.birthdayInput"
                          placeholder="手动输入 YYYY-MM-DD"
                          style="width: 100%;"
                          @blur="validateBirthdayManual"
                      ></el-input>
                    </div>
                  </div>
                </el-form-item>

                <el-form-item label="生成数量">
                  <el-input-number v-model="formID.generateCount" :min="1" :max="100000"
                                   style="width: 100%;"></el-input-number>
                </el-form-item>

                <el-button type="primary" @click="generateMultipleIDs"
                           class="generate-button">生成身份证</el-button>
              </el-form>
            </el-tab-pane>

            <!-- 新增：社工字典 -->
            <el-tab-pane label="社工字典" name="socialDict">
              <h2 class="section-title">社工字典生成</h2>

              <!-- 模式切换 -->
              <div class="mode-switch">
                <el-radio-group v-model="socialDictMode" size="small">
                  <el-radio-button value="simple">简易字典</el-radio-button>
                  <el-radio-button value="complex">复杂字典</el-radio-button>
                </el-radio-group>
              </div>

              <!-- 简易字典模式 -->
              <div v-if="socialDictMode === 'simple'" class="simple-mode">
                <el-form :model="formSocialDictSimple" label-width="100px">
                  <el-form-item label="关键词">
                    <el-input
                        v-model="formSocialDictSimple.keywords"
                        placeholder="输入关键词，多个用逗号分隔"
                        type="textarea"
                        :rows="3"
                    ></el-input>
                  </el-form-item>

                  <el-form-item label="连接符">
                    <el-input
                        v-model="formSocialDictSimple.connectors"
                        placeholder="如↓=@58^~7%+=/!"
                    ></el-input>
                  </el-form-item>

                  <el-form-item label="拼接词组">
                    <el-input
                        v-model="formSocialDictSimple.wordGroups"
                        placeholder="输入拼接词组"
                        type="textarea"
                        :rows="4"
                    ></el-input>
                    <div class="hint-text">默认: 123,888,666,000,111,aaa,abc,qaz,qwe,asd,zxc,[@#,1234,1qaz,qwer,asdf,zxcv,[@#$,1357,2468,0123,6789,6666,8888,12345,123456</div>
                  </el-form-item>

                  <el-form-item label="最近年份">
                    <el-radio-group v-model="formSocialDictSimple.recentYears">
                      <el-radio :value="10">10</el-radio>
                      <el-radio value="all">全部</el-radio>
                    </el-radio-group>
                  </el-form-item>

                  <div class="action-buttons">
                    <el-button type="primary" @click="generateSocialDict" class="generate-button">生成密码</el-button>
                    <el-button @click="resetSocialDict" class="reset-button">重置</el-button>
                  </div>
                </el-form>
              </div>

              <!-- 复杂字典模式 -->
              <div v-else class="complex-mode social-dict-form">
                <el-form :model="formSocialDictComplex" label-width="100px">
                  <el-form-item label="用户名/姓名">
                    <el-input
                        v-model="formSocialDictComplex.username"
                        placeholder="拼音，如zhang san"
                    ></el-input>
                  </el-form-item>

                  <el-form-item label="生日">
                    <el-date-picker
                        v-model="formSocialDictComplex.birthday"
                        type="date"
                        placeholder="请选择日期"
                        style="width: 100%;"
                    ></el-date-picker>
                  </el-form-item>

                  <el-form-item>
                    <template #label>
                      <span>邮箱</span>
                      <el-checkbox v-model="formSocialDictComplex.includeEmail" style="margin-left: 8px;"></el-checkbox>
                    </template>
                    <el-input
                        v-model="formSocialDictComplex.email"
                        placeholder="输入邮箱"
                    ></el-input>
                  </el-form-item>

                  <el-form-item>
                    <template #label>
                      <span>电话</span>
                      <el-checkbox v-model="formSocialDictComplex.includePhone" style="margin-left: 8px;"></el-checkbox>
                    </template>
                    <el-input
                        v-model="formSocialDictComplex.phone"
                        placeholder="输入电话"
                    ></el-input>
                  </el-form-item>

                  <el-form-item>
                    <template #label>
                      <span>身份证</span>
                      <el-checkbox v-model="formSocialDictComplex.includeID" style="margin-left: 8px;"></el-checkbox>
                    </template>
                    <el-input
                        v-model="formSocialDictComplex.idCard"
                        placeholder="输入身份证"
                    ></el-input>
                  </el-form-item>

                  <el-form-item label="组织名称">
                    <el-input
                        v-model="formSocialDictComplex.organization"
                        placeholder="组织/公司名称，拼音"
                    ></el-input>
                  </el-form-item>

                  <el-form-item>
                    <template #label>
                      <span>工号</span>
                      <el-checkbox v-model="formSocialDictComplex.includeWorkId" style="margin-left: 8px;"></el-checkbox>
                    </template>
                    <el-input
                        v-model="formSocialDictComplex.workId"
                        placeholder="输入工号"
                    ></el-input>
                  </el-form-item>

                  <el-form-item label="连接符">
                    <el-input
                        v-model="formSocialDictComplex.connectors"
                        placeholder="如↓=@58^~7%+=/!"
                    ></el-input>
                  </el-form-item>

                  <el-form-item label="拼接词组">
                    <el-input
                        v-model="formSocialDictComplex.wordGroups"
                        placeholder="输入拼接词组"
                        type="textarea"
                        :rows="3"
                    ></el-input>
                    <div class="hint-text">默认: 123,888,666,000,111,aaaabc,qaz,qwe,asd,zxc,l@#;1234,1qaz,qwer,asd(zxcv,l@#5,1357,2468,0123,6789,6666,8888,12345,123456</div>
                  </el-form-item>

                  <el-form-item label="最近年份">
                    <el-radio-group v-model="formSocialDictComplex.recentYears">
                      <el-radio :value="10">10</el-radio>
                      <el-radio value="all">全部</el-radio>
                      <el-radio value="close">关闭</el-radio>
                    </el-radio-group>
                  </el-form-item>

                  <div class="action-buttons">
                    <el-button type="primary" @click="generateSocialDict" class="generate-button">生成密码</el-button>
                    <el-button @click="resetSocialDict" class="reset-button">重置</el-button>
                  </div>
                </el-form>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </section>

      <!-- 右侧结果 -->
      <section class="result-section">
        <el-card shadow="always" class="result-card">
          <div v-if="generatedIDs.length > 0 || generatedPassword.length > 0 || generatedPhone.length > 0 || generatedSocialDict.length > 0"
               class="result-container">
            <div v-if="generatedIDs.length > 0" class="result-box">
              <h3>身份证生成结果</h3>
              <pre class="result-text">{{ generatedIDs.join('\n') }}</pre>
            </div>

            <div v-if="generatedPassword.length > 0" class="result-box">
              <h3>密码生成结果</h3>
              <pre class="result-text">{{ generatedPassword.join('\n') }}</pre>
            </div>

            <div v-if="generatedPhone.length > 0" class="result-box">
              <h3>手机号生成结果</h3>
              <pre class="result-text">{{ generatedPhone.join('\n') }}</pre>
            </div>

            <div v-if="generatedSocialDict.length > 0" class="result-box">
              <h3>社工字典生成结果</h3>
              <pre class="result-text">{{ generatedSocialDict.join('\n') }}</pre>
            </div>

            <el-button type="success" @click="copyToClipboard" class="copy-button">一键复制</el-button>
          </div>
          <p v-else class="placeholder-text">请在左侧填写信息后点击生成按钮。</p>
        </el-card>
      </section>
    </main>
  </div>
</template>

<script>
import { getToken } from '@/utils/token';

export default {
  name: "RandomInfoView",
  data() {
    return {
      activeTab: 'password',
      showIDTab: true,

      // 社工字典相关数据
      socialDictMode: 'simple', // 'simple' | 'complex'
      formSocialDictSimple: {
        keywords: '',
        connectors: '',
        wordGroups: '123,888,666,000,111,aaa,abc,qaz,qwe,asd,zxc,[@#,1234,1qaz,qwer,asdf,zxcv,[@#$,1357,2468,0123,6789,6666,8888,12345,123456',
        recentYears: 10
      },
      formSocialDictComplex: {
        username: '',
        birthday: null,
        email: '',
        phone: '',
        idCard: '',
        organization: '',
        workId: '',
        connectors: '',
        wordGroups: '123,888,666,000,111,aaaabc,qaz,qwe,asd,zxc,l@#;1234,1qaz,qwer,asd(zxcv,l@#5,1357,2468,0123,6789,6666,8888,12345,123456',
        recentYears: 10,
        includeEmail: false,
        includePhone: false,
        includeID: false,
        includeWorkId: false
      },
      generatedSocialDict: [],

      // 其他现有数据...
      formID: {
        province: '',
        city: '',
        county: '',
        gender: '男',
        birthday: null,
        birthdayInput: '',
        birthdayMode: 'picker',
        generateCount: 1,
      },
      formPassword: {
        length: 8,
        includeChars: ['数字 (0-9)', '小写字母 (a-z)', '大写字母 (A-Z)', '特殊字符 (!@#$%^&*()...)'],
        excludeChars: '',
        generateCount: 1,
      },
      formPhone: {
        province: '',
        city: '',
        county: '',
        generateCount: 1,
      },
      generatedIDs: [],
      generatedPassword: [],
      generatedPhone: [],

      cityMap: {},
      countyMap: {},
      phoneCityMap: {},
      phoneCountyMap: {},
      loadingProvince: false,
      loadingKey: 'loading',
      loadingPhoneProvince: false,

      phoneProvinceMap: this.cachedPhoneProvinceMap || {},
      provinceMap: this.cachedProvinceMap || {},
    };
  },
  async created() {
    const token = getToken();
    if (token === "muhan"){
      this.showIDTab = true
    }

    this.phoneProvinceMap = await this.loadPhoneProvinceMap();
    this.provinceMap = await this.loadProvinceMap();
  },
  methods: {
    // 社工字典相关方法
    generateSocialDict() {
      if (this.socialDictMode === 'simple') {
        this.generateSimpleSocialDict();
      } else {
        this.generateComplexSocialDict();
      }
    },

    generateSimpleSocialDict() {
      const { keywords, connectors, wordGroups, recentYears } = this.formSocialDictSimple;

      if (!keywords && !wordGroups) {
        this.$message.error('请至少输入关键词或使用拼接词组');
        return;
      }

      const keywordList = keywords.split(',').map(k => k.trim()).filter(k => k);
      const wordList = wordGroups.split(',').map(w => w.trim()).filter(w => w);
      const connector = connectors || '';

      // 生成年份列表
      const yearList = this.getYearList(recentYears);

      const results = [];

      // 组合生成密码
      if (keywordList.length > 0 && wordList.length > 0) {
        // 关键词 + 连接符 + 词组
        for (const keyword of keywordList) {
          for (const word of wordList) {
            results.push(`${keyword}${connector}${word}`);

            // 添加年份变体
            if (yearList.length > 0) {
              for (const year of yearList) {
                results.push(`${keyword}${connector}${word}${connector}${year}`);
                results.push(`${keyword}${connector}${year}${connector}${word}`);
              }
            }
          }
        }

        // 仅关键词 + 年份
        for (const keyword of keywordList) {
          for (const year of yearList) {
            results.push(`${keyword}${connector}${year}`);
          }
        }
      } else if (keywordList.length > 0) {
        // 仅关键词相关组合
        for (const keyword of keywordList) {
          for (const year of yearList) {
            results.push(`${keyword}${connector}${year}`);
          }
        }
      } else if (wordList.length > 0) {
        // 仅词组相关组合
        for (const word of wordList) {
          for (const year of yearList) {
            results.push(`${word}${connector}${year}`);
          }
        }
      }

      this.generatedSocialDict = [...new Set(results)]; // 去重
      this.$message.success(`生成 ${this.generatedSocialDict.length} 个密码组合`);
    },

    generateComplexSocialDict() {
      const {
        username, birthday, email, phone, idCard, organization, workId,
        connectors, wordGroups, recentYears,
        includeEmail, includePhone, includeID, includeWorkId
      } = this.formSocialDictComplex;

      const baseItems = [];

      // 收集基础信息
      if (username) baseItems.push(username);
      if (birthday) {
        const birthDate = new Date(birthday);
        const formats = [
          `${birthDate.getFullYear()}${String(birthDate.getMonth() + 1).padStart(2, '0')}${String(birthDate.getDate()).padStart(2, '0')}`,
          `${String(birthDate.getMonth() + 1).padStart(2, '0')}${String(birthDate.getDate()).padStart(2, '0')}`,
          `${birthDate.getFullYear()}`
        ];
        baseItems.push(...formats);
      }
      if (includeEmail && email) baseItems.push(email.split('@')[0]);
      if (includePhone && phone) baseItems.push(phone);
      if (includeID && idCard) baseItems.push(idCard.slice(-4), idCard.slice(-6));
      if (organization) baseItems.push(organization);
      if (includeWorkId && workId) baseItems.push(workId);

      if (baseItems.length === 0) {
        this.$message.error('请至少输入一项基本信息');
        return;
      }

      const wordList = wordGroups.split(',').map(w => w.trim()).filter(w => w);
      const connector = connectors || '';
      const yearList = recentYears !== 'close' ? this.getYearList(recentYears) : [];

      const results = new Set();

      // 生成各种组合
      for (const baseItem of baseItems) {
        // 基础项本身
        results.add(baseItem);

        // 基础项 + 词组
        for (const word of wordList) {
          results.add(`${baseItem}${connector}${word}`);
          results.add(`${word}${connector}${baseItem}`);
        }

        // 基础项 + 年份
        for (const year of yearList) {
          results.add(`${baseItem}${connector}${year}`);
          results.add(`${year}${connector}${baseItem}`);
        }

        // 基础项 + 词组 + 年份
        for (const word of wordList) {
          for (const year of yearList) {
            results.add(`${baseItem}${connector}${word}${connector}${year}`);
            results.add(`${baseItem}${connector}${year}${connector}${word}`);
            results.add(`${word}${connector}${baseItem}${connector}${year}`);
          }
        }
      }

      // 仅词组 + 年份
      for (const word of wordList) {
        for (const year of yearList) {
          results.add(`${word}${connector}${year}`);
        }
      }

      this.generatedSocialDict = Array.from(results);
      this.$message.success(`生成 ${this.generatedSocialDict.length} 个密码组合`);
    },

    getYearList(recentYears) {
      const currentYear = new Date().getFullYear();
      if (recentYears === 'all') {
        // 返回1970到当前年份
        const years = [];
        for (let year = 1970; year <= currentYear; year++) {
          years.push(year.toString());
        }
        return years;
      } else {
        // 返回最近N年
        const years = [];
        const n = parseInt(recentYears);
        for (let i = 0; i < n; i++) {
          years.push((currentYear - i).toString());
        }
        return years;
      }
    },

    resetSocialDict() {
      if (this.socialDictMode === 'simple') {
        this.formSocialDictSimple = {
          keywords: '',
          connectors: '',
          wordGroups: '123,888,666,000,111,aaa,abc,qaz,qwe,asd,zxc,[@#,1234,1qaz,qwer,asdf,zxcv,[@#$,1357,2468,0123,6789,6666,8888,12345,123456',
          recentYears: 10
        };
      } else {
        this.formSocialDictComplex = {
          username: '',
          birthday: null,
          email: '',
          phone: '',
          idCard: '',
          organization: '',
          workId: '',
          connectors: '',
          wordGroups: '123,888,666,000,111,aaaabc,qaz,qwe,asd,zxc,l@#;1234,1qaz,qwer,asd(zxcv,l@#5,1357,2468,0123,6789,6666,8888,12345,123456',
          recentYears: 10,
          includeEmail: false,
          includePhone: false,
          includeID: false,
          includeWorkId: false
        };
      }
      this.generatedSocialDict = [];
      this.$message.success('已重置表单');
    },

    // 其他现有方法保持不变...
    async loadProvinceMap() {
      this.loadingProvince = true;
      const { provinceMap } = await import('@/api/province');
      this.loadingProvince = false;
      return provinceMap;
    },

    async loadPhoneProvinceMap() {
      this.loadingProvince = true;
      const { phoneProvinceMap } = await import('@/api/phoneProvince');
      this.loadingProvince = false;
      return phoneProvinceMap;
    },

    async updateCities() {
      if (!this.formID.province) return;

      if (!this.cityMap[this.formID.province]) {
        const provinceData = await this.loadProvinceMap();
        this.cityMap[this.formID.province] = provinceData[this.formID.province] || {};
      }

      this.cityMap = this.cityMap[this.formID.province];
      this.formID.city = '';
      this.formID.county = '';
      this.countyMap = {};
    },

    updateCounties() {
      if (this.formID.city && this.cityMap[this.formID.city]) {
        this.countyMap = this.cityMap[this.formID.city];
      } else {
        this.countyMap = {};
      }
    },

    onBirthdayPickerChange(val) {
      if (val) {
        this.formID.birthdayInput = '';
      }
    },

    validateBirthdayManual() {
      const val = (this.formID.birthdayInput || '').trim();
      if (!val) return;

      const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(val);
      if (!match) {
        this.$message.error('手动输入格式必须为 YYYY-MM-DD');
        return false;
      }
      const y = Number(match[1]), m = Number(match[2]), d = Number(match[3]);
      if (m < 1 || m > 12 || d < 1 || d > 31) {
        this.$message.error('日期不合法，请检查月份或日期范围');
        return false;
      }
      const testDate = new Date(`${y}-${String(m).padStart(2,'0')}-${String(d).padStart(2,'0')}`);
      if (isNaN(testDate.getTime()) || testDate.getFullYear() !== y || (testDate.getMonth()+1) !== m || testDate.getDate() !== d) {
        this.$message.error('日期不合法，请检查输入');
        return false;
      }
      return true;
    },

    generateMultipleIDs() {
      if (!this.formID.province || !this.formID.city || !this.formID.county || !this.formID.gender) {
        this.$message.error('请完整填写省市县及性别');
        return;
      }

      let birthdayRaw = null;
      if (this.formID.birthdayMode === 'picker') {
        birthdayRaw = this.formID.birthday;
        if (!birthdayRaw) {
          this.$message.error('请选择出生日期');
          return;
        }
      } else {
        if (!this.formID.birthdayInput) {
          this.$message.error('请手动输入出生日期');
          return;
        }
        if (!this.validateBirthdayManual()) return;
        birthdayRaw = this.formID.birthdayInput;
      }

      const birthdayStr = this.formatDate(birthdayRaw);
      if (!birthdayStr) {
        this.$message.error('出生日期格式错误');
        return;
      }

      const count = this.formID.generateCount;
      this.generatedIDs = [];
      for (let i = 0; i < count; i++) {
        const areaCode = this.countyMap[this.formID.county];
        if (!areaCode) {
          this.$message.error('找不到该县(区)的地区编码');
          return;
        }

        const id17 = `${String(areaCode)}${birthdayStr}${this.generateRandomCode()}`;
        const checkCode = this.calculateCheckCode(id17);
        this.generatedIDs.push(id17 + checkCode);
      }
    },

    formatDate(date) {
      if (!date) return '';
      if (Object.prototype.toString.call(date) === '[object Date]') {
        const d = date;
        const year = d.getFullYear();
        const month = String(d.getMonth() + 1).padStart(2, '0');
        const day = String(d.getDate()).padStart(2, '0');
        return `${year}${month}${day}`;
      }
      const s = String(date).trim();
      const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
      if (m) {
        const year = m[1];
        const month = m[2];
        const day = m[3];
        return `${year}${month}${day}`;
      }
      const d2 = new Date(s);
      if (!isNaN(d2.getTime())) {
        const y = d2.getFullYear();
        const mo = String(d2.getMonth() + 1).padStart(2, '0');
        const da = String(d2.getDate()).padStart(2, '0');
        return `${y}${mo}${da}`;
      }
      return '';
    },

    generateRandomCode() {
      const genderParity = this.formID.gender === '男' ? 1 : 0;
      const random = Math.floor(Math.random() * 100) * 10 + genderParity;
      return String(random).padStart(3, '0');
    },

    calculateCheckCode(id17) {
      const weights = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2];
      const checkCodes = ['1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'];

      let sum = 0;
      for (let i = 0; i < 17; i++) {
        sum += parseInt(id17.charAt(i)) * weights[i];
      }

      const remainder = sum % 11;
      return checkCodes[remainder];
    },

    async updatePhoneCities() {
      if (!this.formPhone.province) return;

      if (!this.phoneProvinceMap) {
        this.phoneProvinceMap = await this.loadPhoneProvinceMap();
      }

      if (this.formPhone.province && this.phoneProvinceMap[this.formPhone.province]) {
        this.phoneCityMap = this.phoneProvinceMap[this.formPhone.province];
      } else {
        this.phoneCityMap = {};
      }

      this.formPhone.city = '';
      this.formPhone.county = '';
      this.phoneCountyMap = {};
    },

    updatePhoneCounties() {
      if (this.formPhone.city && this.phoneProvinceMap[this.formPhone.province]) {
        this.phoneCountyMap = this.phoneProvinceMap[this.formPhone.province][this.formPhone.city] || {};
      } else {
        this.phoneCountyMap = {};
      }
      this.formPhone.county = '';
    },

    generateMultiplePhones() {
      if (!this.formPhone.province || !this.formPhone.city || !this.formPhone.county) {
        this.$message.error('请选择完整的省、市和县');
        return;
      }
      const count = this.formPhone.generateCount;
      this.generatedPhone = [];
      for (let i = 0; i < count; i++) {
        const countyData = this.phoneProvinceMap[this.formPhone.province]?.[this.formPhone.city]?.[this.formPhone.county];

        if (!countyData) {
          this.$message.error('找不到该地区的手机号数据');
          return;
        }

        const prefixArray = Array.isArray(countyData) ? countyData : Object.values(countyData);
        const prefix = prefixArray[Math.floor(Math.random() * prefixArray.length)];

        const randomSuffix = Math.floor(Math.random() * 10000).toString().padStart(4, '0');
        const phone = `${prefix}${randomSuffix}`;

        if (phone.length === 11) {
          this.generatedPhone.push(phone);
        }
      }
    },

    generateMultiplePasswords() {
      if (this.formPassword.includeChars.length === 0) {
        this.$message.error('请至少选择一种字符类型');
        return;
      }
      let characters = '';
      if (this.formPassword.includeChars.includes('数字 (0-9)')) characters += '0123456789';
      if (this.formPassword.includeChars.includes('小写字母 (a-z)')) characters += 'abcdefghijklmnopqrstuvwxyz';
      if (this.formPassword.includeChars.includes('大写字母 (A-Z)')) characters += 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
      if (this.formPassword.includeChars.includes('特殊字符 (!@#$%^&*()...)')) characters += '!@#$%^&*()_+[]{}|;:,.<>?';

      let count = this.formPassword.generateCount;
      this.generatedPassword = [];
      for (let j = 0; j < count; j++) {
        let password = '';
        const excludeChars = this.formPassword.excludeChars.split('').join('');
        for (let i = 0; i < this.formPassword.length; i++) {
          let char;
          do {
            char = characters.charAt(Math.floor(Math.random() * characters.length));
          } while (excludeChars.includes(char));
          password += char;
        }
        this.generatedPassword.push(password);
      }
    },

    copyToClipboard() {
      let text = '';
      const separator = '\n###########\n';
      const parts = [];

      if (this.generatedIDs.length > 0) {
        parts.push(this.generatedIDs.join('\n'));
      }
      if (this.generatedPassword.length > 0) {
        parts.push(this.generatedPassword.join('\n'));
      }
      if (this.generatedPhone.length > 0) {
        parts.push(this.generatedPhone.join('\n'));
      }
      if (this.generatedSocialDict.length > 0) {
        parts.push(this.generatedSocialDict.join('\n'));
      }

      if (parts.length > 0) {
        text = parts.join(separator) + '\n';
      }

      navigator.clipboard.writeText(text).then(() => {
        this.$message.success('已复制到剪贴板');
      }).catch(() => {
        this.$message.error('复制失败');
      });
    }
  }
};
</script>

<style lang="scss" scoped>
.id-generator {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: #f5f7fa;
}

.header {
  text-align: center;
  padding: 10px;
  margin-top: 10px;
  background: #449a8a;
  color: white;
  border-radius: 10px;
}

.main-content {
  display: flex;
  flex: 1;
  padding: 20px;
  gap: 20px;
  background-color: #e8efed;
  min-height: 0;
}

:deep(h3) {
  display: block;
  font-size: 1.17em;
  margin-block-start: 1em;
  margin-block-end: 1em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

.result-section {
  flex: 1;
  min-width: 0;
}

.form-section {
  flex: 0 0 55%; /* 占据45%宽度 */
  max-width: 600px; /* 最大宽度限制 */
  min-width: 400px; /* 最小宽度保证可用性 */
  overflow: hidden;
}

.form-card,
.result-card {
  padding: 10px;
  border-radius: 10px;
}

.form-card {
  padding: 10px;
  border-radius: 10px;
  height: 100%;
  overflow-y: auto;
  max-height: calc(100vh - 100px);
}

.section-title {
  margin-bottom: 20px;
  color: #333;
}

// 社工字典特定样式
.mode-switch {
  margin-bottom: 20px;
  text-align: center;
}

.simple-mode,
.complex-mode {
  margin-top: 15px;
}

.social-dict-form {
  width: 100%;
  min-width: 0; /* 确保表单可以适应容器 */
}

:deep(.social-dict-form .el-form) {
  width: 100%;
}

.hint-text {
  font-size: 12px;
  color: #888;
  margin-top: 5px;
}

.action-buttons {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 20px;
}

.generate-button {
  flex: 1;
}

.reset-button {
  flex: 1;
}

// 复杂模式表单标签样式
:deep(.el-form-item__label) {
  display: flex;
  align-items: center;
}

.result-box {
  padding: 15px;
  background: #f3f7ff;
  border-radius: 5px;
  max-height: 400px;
  overflow-y: auto;
}

.result-text {
  font-family: monospace;
  color: #d62323;
  font-size: 1.2em;
}

.copy-button {
  margin-top: 15px;
  width: 100%;
}

.placeholder-text {
  color: #888;
  text-align: center;
  margin-top: 20px;
}
</style>