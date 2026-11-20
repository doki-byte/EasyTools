import { defineStore } from "pinia";
import { GetProfile, SaveConfig, StopListening } from "../../../../wailsjs/go/proxy/Proxy";

export interface Config {
    Maxpage: string;

    // 搜索语句配置
    FofaSocks5Query: string;
    FofaXuiQuery: string;
    HunterSocks5Query: string;
    HunterXuiQuery: string;
    QuakeSocks5Query: string;
    QuakeXuiQuery: string;

    CoroutineCount: number;
    LiveProxies: number;
    AllProxies: number;
    LiveProxyLists: string[];
    Timeout: string;
    SocksAddress: string;
    FilePath: string;

    Status: number;

    Code: number;
    Error: string;
    Xui: string;
}

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
        CheckTimeout: 0,
        Maxpage: 0,
        LiveProxyLists: [] as any[],
        Xui: "0",
        CurrentIP: "N/A",

        // 搜索语句配置
        FofaSocks5Query: "",
        FofaXuiQuery: "",
        HunterSocks5Query: "",
        HunterXuiQuery: "",
        QuakeSocks5Query: "",
        QuakeXuiQuery: ""
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
                this.Maxpage = Number(profile.Maxpage);
                this.Xui = profile.Xui;

                // 搜索语句配置
                this.FofaSocks5Query = profile.FofaSocks5Query;
                this.FofaXuiQuery = profile.FofaXuiQuery;
                this.HunterSocks5Query = profile.HunterSocks5Query;
                this.HunterXuiQuery = profile.HunterXuiQuery;
                this.QuakeSocks5Query = profile.QuakeSocks5Query;
                this.QuakeXuiQuery = profile.QuakeXuiQuery;
            } catch (err) {
                console.error("获取配置失败:", err);
            }
        },

        // 保存配置
        async saveConfig(configData: Config): Promise<void>  {
            try {
                await SaveConfig(configData);
                console.log("配置已保存！");
            } catch (err) {
                console.error("保存失败:", err);
            }
        },

        // 停止任务方法
        async stopTask() {
            try {
                this.setStatus(0);
                await StopListening();
                this.setStatus(3);
            } catch (err) {
                console.error("停止任务失败:", err);
                this.setStatus(1);
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
        setLiveProxies(count: number) {
            this.LiveProxies = count;
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
        getXui(){
            return this.Xui;
        },
        setXui(xui: string) {
            this.Xui = xui;
        },
        // 搜索语句的getter和setter
        getFofaSocks5Query() {
            return this.FofaSocks5Query;
        },
        setFofaSocks5Query(query: string) {
            this.FofaSocks5Query = query;
        },
        getFofaXuiQuery() {
            return this.FofaXuiQuery;
        },
        setFofaXuiQuery(query: string) {
            this.FofaXuiQuery = query;
        },
        getHunterSocks5Query() {
            return this.HunterSocks5Query;
        },
        setHunterSocks5Query(query: string) {
            this.HunterSocks5Query = query;
        },
        getHunterXuiQuery() {
            return this.HunterXuiQuery;
        },
        setHunterXuiQuery(query: string) {
            this.HunterXuiQuery = query;
        },
        getQuakeSocks5Query() {
            return this.QuakeSocks5Query;
        },
        setQuakeSocks5Query(query: string) {
            this.QuakeSocks5Query = query;
        },
        getQuakeXuiQuery() {
            return this.QuakeXuiQuery;
        },
        setQuakeXuiQuery(query: string) {
            this.QuakeXuiQuery = query;
        },
        // 添加当前IP的getter和setter
        getCurrentIP() {
            return this.CurrentIP;
        },
        setCurrentIP(ip: string) {
            this.CurrentIP = ip;
        }
    }
});