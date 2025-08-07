import { defineStore } from "pinia";
import { GetProfile, SaveConfig, StopListening } from "../../../../wailsjs/go/proxy/Proxy";

export interface Config {
    Email: string;
    FofaKey: string;
    HunterKey: string;
    QuakeKey: string;
    Country: string;
    Maxpage: string; // 保持与 Go 中一致，字符串类型

    CoroutineCount: number;
    LiveProxies: number;
    AllProxies: number;
    LiveProxyLists: string[]; // 字符串数组类型
    Timeout: string; // 保持与 Go 中一致，字符串类型
    SocksAddress: string;
    FilePath: string;

    Status: number;

    Code: number;
    Error: string;
    GlobalProxy: string;
}



// 定义 Pinia store
export const useConfigStore = defineStore('config', {
    state: () => ({
        Code: 0,
        Error: '',
        Status: 0,
        FilePath: '路径为空',
        CoroutineCount: 0,
        LiveProxies: 0,
        AllProxies: 0,
        Timeout: 0,
        SocksAddress: 'NULL',
        Email: "",
        FofaKey: "",
        HunterKey: "",
        QuakeKey: "",
        CheckTimeout: 0,
        Maxpage: 0,
        LiveProxyLists: [] as any[], // 可以根据实际数据改成具体类型
        Country: "0",
        GlobalProxy:"0"
    }),

    actions: {
        // 获取配置文件
        async getProfile() {
            try {
                const profile = await GetProfile();
                this.FilePath = profile.FilePath;
                this.CoroutineCount = profile.CoroutineCount;
                this.LiveProxies = profile.LiveProxies;
                this.AllProxies = profile.AllProxies;
                this.Timeout = Number(profile.Timeout);
                this.SocksAddress = profile.SocksAddress;
                this.Email = profile.Email;
                this.FofaKey = profile.FofaKey;
                this.HunterKey = profile.HunterKey;
                this.QuakeKey = profile.QuakeKey;
                this.Maxpage = Number(profile.Maxpage);
                this.Country = profile.Country;
                this.GlobalProxy=profile.GlobalProxy;
            } catch (err) {
                console.error("获取配置失败:", err);
            }
        },

        // 保存配置，传入参数类型明确
        async saveConfig(configData: Config): Promise<void>  {
            try {
                await SaveConfig(configData); // SaveConfig 应该接受 Config 类型
                console.log("配置已保存！");
            } catch (err) {
                console.error("保存失败:", err);
            }
        },

        // 新增的停止任务方法
        async stopTask() {
            try {
                this.setStatus(0);  // 更新状态为停止中
                await StopListening();  // 调用后端的 stopListening 方法
                this.setStatus(3);  // 更新任务状态为已取消
            } catch (err) {
                console.error("停止任务失败:", err);
                this.setStatus(1);  // 设置为失败状态
            }
        },

        // 获取和设置方法
        getSocksAddress() {
            return this.SocksAddress;
        },
        setSocksAddress(socksAddress: string) {
            this.SocksAddress = socksAddress;
        },
        getTimeout() {
            return this.Timeout;
        },
        setTimeout(timeout: number) {
            this.Timeout = timeout;
        },
        getLiveProxies() {
            return this.LiveProxies;
        },
        getAllProxies() {
            return this.AllProxies;
        },
        getCoroutineCount() {
            return this.CoroutineCount;
        },
        setCoroutineCount(count: number) {
            this.CoroutineCount = count;
        },
        getFilePath() {
            return this.FilePath;
        },
        setFilePath(path: string) {
            this.FilePath = path;
        },
        getStatus() {
            return this.Status;
        },
        setStatus(status: number) {
            this.Status = status;
        },
        getEmail() {
            return this.Email;
        },
        setEmail(email: string) {
            this.Email = email;
        },
        getFofaKey() {
            return this.FofaKey;
        },
        setFofaKey(FofaKey: string) {
            this.FofaKey = FofaKey;
        },
        getHunterKey() {
            return this.HunterKey;
        },
        setHunterKey(HunterKey: string) {
            this.HunterKey = HunterKey;
        },
        getQuakeKey() {
            return this.QuakeKey;
        },
        setQuakeKey(QuakeKey: string) {
            this.QuakeKey = QuakeKey;
        },
        getCheckTimeout() {
            return this.CheckTimeout;
        },
        setCheckTimeout(timeout: number) {
            this.CheckTimeout = timeout;
        },
        getMaxpage() {
            return this.Maxpage;
        },
        setMaxpage(maxpage: number) {
            this.Maxpage = maxpage;
        },
        getCountry(){
            return this.Country;
        },
        setCountry(country: string) {
            this.Country = country;
        },
        getGlobalProxy(){
            return this.GlobalProxy;
        },
        setGlobalProxy(proxy: string) {
            this.GlobalProxy = proxy;
        }
    }
});
