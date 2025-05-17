<template>
    <div class="id-generator" @contextmenu.prevent>
        <header class="header">
            <h1>信息生成</h1>
        </header>

        <main class="main-content">
            <!-- 左侧表单 -->
            <section class="form-section">
                <el-card shadow="always" class="form-card">
                    <el-tabs v-model="activeTab" type="card">
                        <!-- 密码生成 -->
                        <el-tab-pane label="密码生成" name="password">
                            <h2 class="section-title">密码设置</h2>
                            <el-form :model="formPassword" label-width="100px" class="form-container">
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
                            <el-form :model="formPhone" label-width="100px" class="form-container">
                                <el-form-item label="省">
                                    <el-select v-model="formPhone.province" placeholder="请选择省"
                                        @change="updatePhoneCities" :loading="loadingPhoneProvince">
                                        <el-option v-for="(cities, province) in phoneProvinceMap" :key="province"
                                            :label="province" :value="province"></el-option>
                                        <!-- 显示正在加载的提示 -->
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

                        <el-tab-pane v-if="showIDTab" label="身份证生成" name="id">
                            <h2 class="section-title">生成设置</h2>
                            <el-form :model="formID" label-width="100px" class="form-container">
                                <el-form-item label="省">
                                    <el-select v-model="formID.province" placeholder="请选择省" @change="updateCities"
                                        :loading="loadingProvince">
                                        <el-option v-for="(cities, province) in provinceMap" :key="province"
                                            :label="province" :value="province"></el-option>
                                        <!-- 显示正在加载的提示 -->
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
                                    <el-date-picker v-model="formID.birthday" type="date" placeholder="选择出生日期"
                                        style="width: 100%;"></el-date-picker>
                                </el-form-item>

                                <el-form-item label="生成数量">
                                    <el-input-number v-model="formID.generateCount" :min="1" :max="100000"
                                        style="width: 100%;"></el-input-number>
                                </el-form-item>

                                <el-button type="primary" @click="generateMultipleIDs"
                                    class="generate-button">生成身份证</el-button>
                            </el-form>
                        </el-tab-pane>
                    </el-tabs>
                </el-card>
            </section>

            <!-- 右侧结果 -->
            <section class="result-section">
                <el-card shadow="always" class="result-card">
                    <!-- <h2 class="section-title">生成结果</h2> -->
                    <div v-if="generatedIDs.length > 0 || generatedPassword.length > 0 || generatedPhone.length > 0"
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
    data() {
        return {
            activeTab: 'password',  // 当前选中的tab
            showIDTab: true,  // 控制是否显示 "身份证生成" Tab
            formID: {
                province: '',
                city: '',
                county: '',
                gender: '',
                birthday: '',
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
            loadingProvince: false, // 省份加载状态
            loadingKey: 'loading', // 用来动态添加加载提示项
            loadingPhoneProvince: false,  // 或你需要的初始值

            phoneProvinceMap: this.cachedPhoneProvinceMap || {},
            provinceMap: this.cachedProvinceMap || {},
        };
    },
    async created() {
        // 在created生命周期中加载数据
        this.phoneProvinceMap = await this.loadPhoneProvinceMap();
        this.provinceMap = await this.loadProvinceMap();
    },
    methods: {
        // 动态加载省份信息
        async loadProvinceMap() {
            this.loadingProvince = true;  // 开始加载时设置为 true
            const { provinceMap } = await import('@/api/province');
            this.loadingProvince = false;  // 加载完成后设置为 false
            return provinceMap;
        },

        // 动态加载手机号省份信息
        async loadPhoneProvinceMap() {
            this.loadingProvince = true;  // 开始加载时设置为 true
            const { phoneProvinceMap } = await import('@/api/phoneProvince');
            this.loadingProvince = false;  // 加载完成后设置为 false
            return phoneProvinceMap;
        },
        // 更新身份证市区数据
        async updateCities() {
            if (!this.formID.province) return;

            // 如果当前省份的市区数据未加载，则加载
            if (!this.cityMap[this.formID.province]) {
                const provinceData = await this.loadProvinceMap();
                this.cityMap[this.formID.province] = provinceData[this.formID.province] || {}; // 保存当前省的数据
            }

            // 通过当前省份的市区数据更新 cityMap
            this.cityMap = this.cityMap[this.formID.province];

            // 重置市区和县区数据
            this.formID.city = '';
            this.formID.county = '';
            this.countyMap = {};
        }
        ,
        // 更新身份证县区数据
        updateCounties() {
            if (this.formID.city && this.cityMap[this.formID.city]) {
                this.countyMap = this.cityMap[this.formID.city];
            } else {
                this.countyMap = {};
            }
        },
        // 生成多个身份证
        generateMultipleIDs() {
            if (!this.formID.province || !this.formID.city || !this.formID.county || !this.formID.birthday || !this.formID.gender) {
                this.$message.error('请完整填写信息');
                return;
            }
            const count = this.formID.generateCount;
            this.generatedIDs = [];
            for (let i = 0; i < count; i++) {
                // 拼接身份证前17位
                const id = `${this.countyMap[this.formID.county]}${this.formatDate(this.formID.birthday)}${this.generateRandomCode()}`;

                // 计算校验码并拼接完整身份证号
                const checkCode = this.calculateCheckCode(id); // 将前17位传给校验码计算方法
                this.generatedIDs.push(id + checkCode); // 拼接完整身份证号
            }
        },
        // 格式化出生日期
        formatDate(date) {
            const d = new Date(date);
            const year = d.getFullYear();
            const month = String(d.getMonth() + 1).padStart(2, '0');
            const day = String(d.getDate()).padStart(2, '0');
            return `${year}${month}${day}`;
        },
        // 生成随机身份证编码
        generateRandomCode() {
            const genderCode = this.formID.gender === '男' ? 1 : 0;
            const random = Math.floor(Math.random() * 100) * 10 + genderCode;
            return String(random).padStart(3, '0');
        },
        // 计算身份证校验码
        calculateCheckCode(id17) {
            const weights = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2];
            const checkCodes = ['1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'];

            let sum = 0;
            // 前17位与加权系数相乘并求和
            for (let i = 0; i < 17; i++) {
                sum += parseInt(id17.charAt(i)) * weights[i];
            }

            // 求余数并获取校验码
            const remainder = sum % 11;
            return checkCodes[remainder];
        }
        ,
        // 更新手机号市区数据
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
        // 更新手机号县区数据
        updatePhoneCounties() {
            if (this.formPhone.city && this.phoneProvinceMap[this.formPhone.province]) {
                this.phoneCountyMap = this.phoneProvinceMap[this.formPhone.province][this.formPhone.city] || {};
            } else {
                this.phoneCountyMap = {};
            }
            // 重置县（区）
            this.formPhone.county = '';
        },
        // 生成多个手机号
        generateMultiplePhones() {
            if (!this.formPhone.province || !this.formPhone.city || !this.formPhone.county) {
                this.$message.error('请选择完整的省、市和县');
                return;
            }
            const count = this.formPhone.generateCount;
            this.generatedPhone = [];
            for (let i = 0; i < count; i++) {
                // 获取手机数据
                const countyData = this.phoneProvinceMap[this.formPhone.province]?.[this.formPhone.city]?.[this.formPhone.county];

                if (!countyData) {
                    this.$message.error('找不到该地区的手机号数据');
                    return;
                }

                // 随机选择一个前缀
                const prefixArray = Array.isArray(countyData) ? countyData : Object.values(countyData);
                const prefix = prefixArray[Math.floor(Math.random() * prefixArray.length)];

                // 生成 7 位随机数字，确保手机号为 11 位
                const randomSuffix = Math.floor(Math.random() * 10000).toString().padStart(4, '0');
                const phone = `${prefix}${randomSuffix}`;

                if (phone.length === 11) {
                    this.generatedPhone.push(phone);
                }
            }
        },

        // 生成多个密码
        generateMultiplePasswords() {
            // 判断是否至少选择了一个字符类型
            if (this.formPassword.includeChars.length === 0) {
                this.$message.error('请至少选择一种字符类型');
                return; // 结束函数执行
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
                    } while (excludeChars.includes(char)); // 排除字符
                    password += char;
                }
                this.generatedPassword.push(password);
            }
        },
        // 复制到剪贴板
        copyToClipboard() {
            let text = '';
            if (this.generatedIDs.length > 0) {
                text += this.generatedIDs.join('\n') + '\n';
            }
            if (this.generatedPassword.length > 0) {
                text += this.generatedPassword.join('\n') + '\n';
            }
            if (this.generatedPhone.length > 0) {
                text += this.generatedPhone.join('\n') + '\n';
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
}

.form-section,
.result-section {
    flex: 1;
}

.form-card,
.result-card {
    padding: 10px;
    border-radius: 10px;
}

.section-title {
    margin-bottom: 20px;
    color: #333;
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