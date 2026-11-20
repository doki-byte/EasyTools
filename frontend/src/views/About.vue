<template>
  <div id="wails-app" class="sponsor-page">
    <header class="header glass-effect">
      <div class="header-left">
        <div class="header-text">
          <h1>
            赞助与支持
            <span class="chip chip-primary">最新版本 {{ latestVersion }}</span>
          </h1>
          <p>感谢您对项目的关注与支持,一起推动创新。</p>
        </div>
        <div class="cta-group">
          <button class="btn ghost" @click="openUrl('https://qm.qq.com/q/VEzWDiDMgm')">加入QQ交流群</button>
          <button class="btn ghost" @click="openUrl('https://github.com/doki-byte/EasyTools')">查看 GitHub</button>
          <button class="btn ghost"
            @click="openUrl('https://h5.fenchuan8.com/#/index?forum=102706&yqm=CM4RP')">纷传(叮叮当当)</button>
        </div>
      </div>
      <div class="header-right row-inline">
        <div class="qr-card">
          <img :src="qrImage" alt="公众号二维码" class="qr-image" />
          <div class="qr-wrapper">
            <h3>关注公众号</h3>
            <p>扫码关注,获取更多资讯</p>
          </div>
        </div>
      </div>
    </header>

    <div class="content-area">
      <div class="main-column glass-effect card update-card">
        <div class="card-header">
          <div>
            <h2>更新说明</h2>
            <p class="card-subtitle">最新版本动态与历史记录</p>
          </div>
        </div>
        <ul class="list scrollable tall-list">
          <li v-for="(u, i) in updates" :key="i" class="update-item">
            <div class="timeline-dot" :class="{ active: u.isNew }"></div>
            <span class="date">{{ u.date }}</span>
            <span class="text large-text">{{ u.content }}</span>
            <span v-if="u.isNew" class="new-badge">NEW</span>
          </li>
        </ul>
      </div>

      <div class="side-column">
        <div class="card glass-effect compact-card">
          <div class="card-header">
            <h2>特别感谢<span class="subtitle">(时间顺序)</span></h2>
          </div>
          <ul class="thanks-grid scrollable small-max">
            <li v-for="(p, i) in specialThanks" :key="i" class="thanks-item" :class="{ clickable: !!p.url }"
              @click="handlePersonClick(p)">
              <div class="avatar-container">
                <img :src="p.src" alt="avatar" class="avatar-large" />
              </div>
              <div class="name">{{ p.name }}</div>
            </li>
          </ul>
        </div>

        <div class="card glass-effect compact-card">
          <div class="card-header">
            <h2>友情赞助<span class="subtitle">(时间顺序)</span></h2>
          </div>
          <div class="sponsor-grid scrollable small-max">
            <div v-for="(p, i) in friendlySponsorship" :key="`sponsor-${i}`" class="sponsor-card"
              :class="{ clickable: !!p.url }" @click="handlePersonClick(p)">
              <div class="sponsor-img-wrap">
                <img :src="p.src" alt="sponsor" class="sponsor-img" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { BrowserOpenURL } from "../../wailsjs/runtime";
import { GetAboutInfo } from "../../wailsjs/go/system/SoftwareCenter";

export default {
  name: 'SponsorPage',
  data() {
    // 本地内置数据作为 fallback
    const defaultUpdates = [
      {date: 'v2.3.1', content: '修复免杀生成400报错', isNew: true },
      {date: 'v2.3.0', content: '更新免杀模块，新增20种loader加载器，已测试卡巴、火绒、360等'},
      {date: 'v2.2.9', content: '1. 重新设计便携发包的布局; 2. 修复index目录遍历失败的BUG; '},
      {date: 'v2.2.8', content: '1. 更新微信小程序反编译目录排列顺序; 2. 修复index目录遍历失败的BUG; 3. 修复旧版本drios数据库无法连接的BUG'},
      {date: 'v2.2.7', content: '1. 新增微信小程序官方查询接口; 2. 修复密码爆破模块telnet执行命令失败的bug; 3. jsfinder中新增VueCrack提取功能'},
      {date: 'v2.2.6', content: '1. 修复主机扫描闪退的bug; 2. 工具仓库新增右键打开终端目录; 3. 渗透测试结果新增搜索功能; 4. 修复小程序反编译名称查询不到的bug'},
      {date: 'v2.2.5', content: '1. 优化备忘笔记功能,支持显示非md文件,支持拖拽文件夹进行展示; 2. 优化端口扫描逻辑 3. 密码爆破模块新增mqtt协议，并支持对应命令执行'},
      { date: 'v2.2.4', content: '1. 修复批量存储桶扫描任务无法暂停的bug; 2. 优化备忘笔记功能，支持显示非md文件; 3. 全局敏感操作增加二次确认，避免出现误操作的情况'},
      { date: 'v2.2.3', content: '1. 修复零零信安接口无法使用的bug; 2. 修复渗透测试、信息收集模块右键菜单被遮挡的bug'},
      { date: 'v2.2.2', content: '1. 新增云上安全，支持阿里云、腾讯云、百度云、华为云等aksk的连接与账号接管; 2. 优化数据库管理，修复数据无法保存的bug; 3. 空间测绘新增零零信安接口; 4. 修复登录状态一直失效的bug'},  
      { date: 'v2.2.1', content: '1. 修复随机生成密码缺少类型的bug; 2. 优化ssh连接文件传输，修复文件路径支持输入路径并进去; 3. 删除渗透测试各模块空白页面; 4. 删除iflow配置，现在直接使用系统管理中的ai大模型配置; 5. 地图测试新增google地图api'},
      { date: 'v2.2.0', content: '1. 修复数据库管理中数据展示，以及侧边栏搜索展开样式; 2. 优化主机扫描，目录扫描，空间测绘支持联动jsfinder; 3. 优化代理池功能; 4. 优化小程序反编译功能，简化修复逻辑' },
      { date: 'v2.1.9', content: '1. 修复mac下ssh文件管理无法打开的bug; 2. 修复数据库管理第一条信息无法删除的bug' },
      { date: 'v2.1.8', content: '1. 优化端口扫描规则，提高扫描效率; 2. 重新设计ssh、ftp界面，优化上传下载逻辑; 3. 优化简练助手中各数据库数据展示，支持双击查看数据详情并进行编辑'},
      { date: 'v2.1.7', content: '1. 修复空间测绘查询bug' },
      { date: 'v2.1.6', content: '1. 渗透测试模块新增druid利用功能; 2. 优化restmate界面显示逻辑; 3. 修复密码爆破模块部分协议无法直接执行命令的bug' },
      { date: 'v2.1.5', content: '1. 优化资源桶遍历功能，优化kkfileview预览逻辑; 2. 修复一个bug' },
      { date: 'v2.1.4', content: '1. 优化资源桶遍历功能，支持kkfileview预览; 2. 重构渗透测试模块，调用httpx提高性能与准确度; 3. 新增AI底座; 4. 新增空间测绘; 5. 新增AI版域名收集功能; 6. 优化微信小程序反编译目录获取; 7. 优化密码爆破功能' },
      { date: 'v2.1.3', content: '1. 修复一个bug' },
      { date: 'v2.1.2', content: '1. 优化密码爆破功能; 2. 优化 CLI 工具与 MCP 集成; 3. 优化便携代理功能; 4. 修复mac部分界面显示bug; 5. 新增浏览器js提取并发包功能; 6. ip归属地查询新增高精接口' },
      { date: 'v2.1.1', content: '1. 优化目录扫描功能; 2. 优化免杀生成，支持自行选择32位与64位架构进行构建' },
      { date: 'v2.1.0', content: '1. 数据库管理新增语句管理功能; 2. 新增系统配置备份功能' },
      { date: 'v2.0.9', content: '1. 简练助手新增mysql、redis、mariadb、postgres、sqlserver、sqlite、mongodb等诸多数据库的连接功能，支持自动获取已保存的ssh信息进行隧道连接; 2. 优化主机扫描的口令爆破功能，支持扫描结果联动发送简练助手进行一键连接' },
      { date: 'v2.0.8', content: '1. 优化渗透测试模块各功能联动; 2. 修复mac部分状态白屏bug; 3. 修复便携发包逻辑; 4. 优化程序代理功能，支持系统代理设置(win专属); 5. 新增dll代理转发功能' },
      { date: 'v2.0.7', content: '1. 重新设计redis界面; 2. 优化ssh界面终端显示不全的bug; 3. 修复地图测试bug; 4. 优化google界面语法管理显示bug; 5. 修复指纹识别、目录扫描、端口扫描、备忘笔记右键菜单偏移的bug; 6. 优化漏洞扫描界面，指纹识别扫描; 7. 修复便携代理逻辑bug' },
      { date: 'v2.0.6', content: '1. 修复自定义主题频闪的bug; 2. 修复Mac下ssh连接失败的bug; 3. 新增mac类型的顶部栏' },
      { date: 'v2.0.5', content: '1. 新增自定义主题功能; 2. 优化nuclei漏洞扫描功能' },
      { date: 'v2.0.4', content: '1. 优化漏洞扫描模块，支持暂停任务与断点恢复功能，支持多任务同时扫描' },
      { date: 'v2.0.3', content: '1. 免杀生成新增白加黑挖掘功能,并且支持联动生成可利用白加黑Dll;  2. 新增在线软件中心功能;  3. 新增便携发包数据包文件上传一键bypass功能; 4. 优化便携代理功能; 5. 优化Ip查询功能;6. 修复最新版本Mac无法运行的bug;' },
      { date: 'v2.0.2', content: '1. 新增漏洞扫描与poc管理功能; 2. 优化指纹识别、端口扫描功能;  3. 修复cli命令参数过多导致页面不显示的bug @Tjsdrj; 4. 修复辅助功能-地图功能报错; 5. 增加pcap、cli文件拖拽功能; 6. 优化全局数据包高亮编辑与显示功能;7. 优化简练助手ssh功能,支持工具箱直接连接与打开浏览器进行连接管理' },
      { date: 'v2.0.1', content: '1. 优化指纹识别、端口扫描、目录扫描功能; 2. 新增便携发包自动获取swagger接口、批量发包功能; 3. 优化存在的其他bug;' },
      { date: 'v2.0.0', content: '1. 优化指纹识别功能; 2. 修复ssh界面显示不全bug; 3. 便携发包新增网页预览功能;' },
      { date: 'v1.9.9', content: '1. 优化工具添加功能,支持拖拽添加; 2. 优化jwt解析,新增jwt漏洞利用功能; 3. 新增目录扫描、端口扫描、指纹识别功能; 4. 优化Cli命令执行逻辑' },
      { date: 'v1.9.8', content: '1. 新增目录遍历提取功能(git泄露、svn泄露、index泄露、Ds_store泄露); 2. 优化左侧菜单分类,移动随机生成至辅助功能; 3. 优化google hackle功能、密码查询功能' },
      { date: 'v1.9.7', content: '1. 修复便携代理模块fofa语法出错#geelph; 2. 新增cli命令执行功能,支持任意脚本运行,支持定时脚本执行' },
      { date: 'v1.9.6', content: '1. 新增Ctf wireshark流量包自动分析功能; 2. 优化系统架构' },
      { date: 'v1.9.5', content: '1. 新增小程序针对游戏的反编译功能; 2. 新增批量查询IP归属地功能;3. 优化便携代理功能; 4. 优化编码解码、简连助手功能,添加状态栏浏览器打开功能,去除弹框功能; 5. 修复便携发包响应包一直显示loading的bug。' },
      { date: 'v1.9.4', content: '1. 修复mac小程序反编译一直报错bug; 2. 修复mac运行无法直接显示主界面的bug;3. 修复构建的mac程序无法直接运行的bug;4. 优化OSS资源桶功能; 5. 新增全局代理功能; 6. 新增自动下载替换更新包的功能。' },
      { date: 'v1.9.3', content: '1. 优化redis界面,修复连接过多导致的页面无限下拉的bug; 2. 新增显示更新详情功能; 3. 参考fine修改小程序反编译界面; 4. tools新建工具的时间添加打开工具文件位置的功能; 5. 新增二级菜单的显示/隐藏,位置移动; 6. 修改ftp下载方式为浏览器下载,避免EasyTools卡死; 7. 优化JwtCreck功能;' },
      { date: 'v1.9.2', content: '1. 新增redis命令执行功能;' },
      { date: 'v1.9.1', content: '1. 修复note笔记显示逻辑;' },
      { date: 'v1.9.0', content: '1. 优化note笔记模块图片显示规则,现在支持直接粘贴图片并保存预览; 2. 优化部分功能的界面显示; 3. 修复点击便携发包之后,左边菜单栏大小变化的bug; 4. 修复windows下,未选择工具目录时,直接运行工具报错的bug;' },
      { date: 'v1.8.9', content: '1. 新增程序自启动功能; 2.优化部分功能的bug; 3. 免杀模块新增白加黑、优化沙箱逻辑、修改免杀加载器等多个模块功能' },
      { date: 'v1.8.8', content: '1. 修复便携发包点击send之后白屏或者一直loading的bug' },
      { date: 'v1.8.7', content: '1. 新增便携发包功能,内置微信公众号/小程序、钉钉、企业微信调用接口; 2. 优化系统目录; ' },
      { date: 'v1.8.6', content: '1. 简化Tools导航添加工具逻辑; 2. 新增site导航自动识别获取站点信息功能; 3. 优化模拟生成身份证时时间选择功能; 4. 新增系统托盘; 5. 新增全局快捷键控制程序的显示与隐藏' },
      { date: 'v1.8.5', content: '1. 修改Note笔记界面为奶白色; 2. 新增菜单顺序调整功能' },
      { date: 'v1.8.4', content: '1. 新增fuzz调用界面' },
      { date: 'v1.8.3', content: '1. 新增代理池功能;2. 优化CyberChef、ssh连接功能' },
      { date: 'v1.8.2', content: '优化工具、网址搜索功能;' },
      { date: 'v1.8.1', content: '1.新增Note备忘录功能;2.优化工具、网址搜索功能;3.新增jwt爆破功能;4.新增地图接口调用功能;5.优化redis连接及显示功能;6.修改CyberChef为中文版本的' },
      { date: 'v1.8.0', content: '1.新增工具箱打开路径;2.新增图标选择功能;3.优化ftp、ssh连接弹窗提示;4.小程序反编译模块新增node环境监测功能;5.新增杀软检测目录进程清单;6.修改CyberChef为中文版本的' },
      { date: 'v1.7.5', content: '1.新增小程序反编译功能(需要自行先安装node环境,版本14以上都行,或者直接安装最新版即可);2.修改oss资源桶遍历逻辑;3.修改免杀加载器模版' },
      { date: 'v1.7.4', content: '1.修改文件捆绑失败的bug;2.更新ByPassQvm图标' },
      { date: 'v1.7.3', content: '1.更新杀软识别数据库' },
      { date: 'v1.7.2', content: '1.新增FTP连接功能(匿名登录时,用户名为anonymous ,密码为空)' },
      { date: 'v1.7.1', content: '修复一个小bug' },
      { date: 'v1.7.0', content: '新增工具仓库、网址站点的任意拖拽功能,修改工具摆放顺序' },
      { date: 'v1.6', content: '新增登录页面默认密码修改功能;更新webssh功能,新增批量ssh命令执行功能' },
      { date: 'v1.5', content: '免杀生成:新增分离、远程两种加载模式' },
      { date: 'v1.4', content: '新增一键去除upx特征的功能(测试,只是规避明显特征,避免直接upx -d脱壳);新增一键添加数字签名的小功能' },
      { date: 'v1.3', content: '集成ShellCode在线免杀处理平台;新增文件体积修改功能;新增免杀:5种运行模式、2种加密方式、2种编译方式' },
      { date: 'v1.2', content: '新增信息处理模块;迁移蓝队批量封禁IP模块至信息处理  ' },
      { date: 'v1.1', content: '目前集合功能:网址导航、工具导航、信息查询、编码解码%' }
    ];
    const defaultSpecialThanks = [
      { name: 'joker', src: '../assets/about/joker.jpg' },
      { name: 'ryuuz4k1', src: '../assets/about/ryuuz4k1.jpg' },
      { name: 'ToPaint', src: '../assets/about/ToPaint.jpg' },
      { name: '吃人尘', src: '../assets/about/吃人尘.jpg' },
      { name: '夜晓喽', src: '../assets/about/夜晓喽.jpg' },
      { name: '李白鸽', src: '../assets/about/libaige.jpg' },
      { name: 'LiuXiaHuu', src: '../assets/about/LiuXiaHuu.jpg' },
      { name: 'b4l', src: '../assets/about/b4l.jpg' },
      { name: 'smi', src: '../assets/about/smi.jpg' },
      { name: '重剑无锋', src: '../assets/about/重剑无锋.png' },
    ];
    const defaultFriendlySponsorship = [
      { name: '杨cc', src: '../assets/about/yangcc.png', url: "https://space.bilibili.com/500076166" },
      { name: '岱宗安全研究实验室', src: '../assets/about/daizong.png', url: "https://monttaisec.github.io/" },
      { name: '架构安全观', src: '../assets/about/架构安全观.png', url: "" },
      { name: 'Rot5pider安全团队', src: '../assets/about/Rot5pider安全团队.png', url: "" }, 
    ];

    return {
      updates: defaultUpdates,
      specialThanks: defaultSpecialThanks,
      friendlySponsorship: defaultFriendlySponsorship,
      // 保存默认数据用于 fallback
      defaultUpdates,
      defaultSpecialThanks,
      defaultFriendlySponsorship,
      relatedLinks: [
        { text: 'GitHub', url: 'https://github.com/doki-byte/EasyTools' },
        { text: '纷传(叮叮当当)', url: 'https://h5.fenchuan8.com/#/index?forum=102706&yqm=CM4RP' },
        { text: 'QQ交流群(702376033)', url: 'https://qm.qq.com/q/VEzWDiDMgm' },
      ],
      qrImage: '../assets/about/二维码.jpg',
    };
  },
  computed: {
    latestVersion() {
      return this.updates[0]?.date || 'v2.x';
    },
    updateCount() {
      return this.updates.length;
    },
    thanksCount() {
      return this.specialThanks.length;
    },
    linkCount() {
      return this.relatedLinks.length;
    }
  },
  methods: {
    openUrl(url) {
      BrowserOpenURL(url);
    },
    handlePersonClick(person) {
      if (person?.url) {
        this.openUrl(person.url);
      }
    },
    async loadAboutInfo() {
      try {
        // 尝试从远程获取数据
        const aboutInfo = await GetAboutInfo();

        if (aboutInfo) {
          let hasUpdate = false;
          // 使用远程数据
          if (aboutInfo.updates && aboutInfo.updates.length > 0) {
            this.updates = aboutInfo.updates;
            hasUpdate = true;
          }
          if (aboutInfo.specialThanks && aboutInfo.specialThanks.length > 0) {
            this.specialThanks = aboutInfo.specialThanks;
            hasUpdate = true;
          }
          if (aboutInfo.friendlySponsorship && aboutInfo.friendlySponsorship.length > 0) {
            this.friendlySponsorship = aboutInfo.friendlySponsorship;
            hasUpdate = true;
          }

          if (hasUpdate) {
            console.log('✅ 成功从远程仓库加载 about.json 数据');
          } else {
            console.warn('⚠️ 远程数据为空，使用本地内置数据');
          }
        } else {
          console.warn('⚠️ 远程返回数据为空，使用本地内置数据');
        }
      } catch (error) {
        // 确保使用默认数据
        this.updates = this.defaultUpdates;
        this.specialThanks = this.defaultSpecialThanks;
        this.friendlySponsorship = this.defaultFriendlySponsorship;
      }
    },
  },
  async mounted() {
    // 组件挂载时加载远程数据
    await this.loadAboutInfo();
  }
};
</script>

<style scoped>
/* 基础样式重置 */
:deep(*) {
  box-sizing: border-box;
}

:deep(h1) {
  display: block;
  font-size: 2em;
  margin-block-start: 0.67em;
  margin-block-end: 0.67em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(h2) {
  display: block;
  font-size: 1.5em;
  margin-block-start: 0.83em;
  margin-block-end: 0.83em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(p) {
  display: block;
  margin-block-start: 1em;
  margin-block-end: 1em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  unicode-bidi: isolate;
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

:deep(li) {
  display: list-item;
  text-align: -webkit-match-parent;
  unicode-bidi: isolate;
}

/* 整体页面样式 - 白色主题 */
.sponsor-page {
  background: linear-gradient(135deg, #f6f0f0 0%, #f8f9fa 100%);
  min-height: 93vh;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  color: #333;
  position: relative;
  overflow-x: hidden;
}

.sponsor-page::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  opacity: 0.3;
  z-index: -1;
}

.sponsor-page::after {
  content: '';
  position: absolute;
  width: 420px;
  height: 420px;
  background: radial-gradient(circle at 30% 30%, rgba(66, 153, 225, 0.12), transparent 55%);
  filter: blur(12px);
  top: -60px;
  right: -120px;
  z-index: -2;
}

/* 毛玻璃效果通用类 - 白色主题 */
.glass-effect {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.5);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.08);
  border-radius: 16px;
}

/* 头部样式 */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  margin-bottom: 0.2rem;
  gap: 1rem;
  height: auto;
  position: relative;
  overflow: hidden;
}

.header-left {
  display: flex;
  align-items: flex-start;
  gap: 0.8rem;
  width: auto;
  flex-wrap: wrap;
}

.header-text h1 {
  margin: 0;
  font-size: 2.2rem;
  font-weight: 600;
  color: #2d3748;
  text-shadow: none;
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.header-text p {
  margin: 0.5rem 0 0;
  color: #718096;
  font-size: 1.1rem;
}

.cta-group {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.btn {
  border: none;
  padding: 0.65rem 1.1rem;
  border-radius: 10px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s ease;
  font-size: 0.95rem;
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.06);
  display: inline-block !important;
  visibility: visible !important;
}

.btn.primary {
  background: linear-gradient(135deg, #4299e1, #667eea);
  color: #fff;
}

.btn.primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 24px rgba(102, 126, 234, 0.35);
}

.btn.ghost {
  background: rgba(66, 153, 225, 0.08);
  color: #2d3748;
  border: 1px solid rgba(66, 153, 225, 0.2);
}

.btn.ghost:hover {
  background: rgba(66, 153, 225, 0.12);
  transform: translateY(-1px);
}

.chip {
  padding: 0.2rem 0.7rem;
  border-radius: 999px;
  background: rgba(66, 153, 225, 0.12);
  color: #2b6cb0;
  font-weight: 600;
  font-size: 0.9rem;
  border: 1px solid rgba(66, 153, 225, 0.2);
}

.chip-primary {
  box-shadow: 0 10px 30px rgba(66, 153, 225, 0.12);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.qr-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(5px);
  border: 1px solid rgba(0, 0, 0, 0.05);
}

.qr-image {
  width: 100px;
  height: 100px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.qr-wrapper h3 {
  margin-bottom: 0.5rem;
  font-size: 1.2rem;
  color: #2d3748;
}

.qr-wrapper p {
  margin: 0;
  color: #718096;
  font-size: 0.9rem;
}

/* 内容区域 */
.content-area {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 0.6rem;
  padding: 0 8px;
  align-items: stretch;
}

.highlight-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.6rem;
  padding: 0 8px;
  margin: 0.2rem 0 0.5rem;
}

.mini-card {
  padding: 1rem 1.2rem;
  border-radius: 14px;
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.mini-card::after {
  content: '';
  position: absolute;
  width: 120px;
  height: 120px;
  background: radial-gradient(circle, rgba(102, 126, 234, 0.18), transparent 60%);
  top: -40px;
  right: -40px;
}

.mini-title {
  font-size: 0.95rem;
  color: #718096;
  margin-bottom: 0.35rem;
}

.mini-value {
  font-size: 1.8rem;
  font-weight: 700;
  color: #2d3748;
}

.mini-desc {
  margin: 0.15rem 0 0;
  color: #4a5568;
  font-size: 0.9rem;
}

.main-column {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.side-column {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  height: 100%;
}

.compact-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* 卡片样式 */
.card {
  padding: 1.5rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.card:hover {
  transform: translateY(-5px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.12);
}

.link-card {
  padding: 0.5rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.link-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.12);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  margin-bottom: 0.2rem;
}

.card-icon {
  font-size: 1.8rem;
  color: #4a5568;
  background: rgba(255, 255, 255, 0.9);
  width: 50px;
  height: 50px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(0, 0, 0, 0.05);
}

.card h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #2d3748;
}

.card-subtitle {
  margin: 0.15rem 0 0;
  color: #718096;
  font-size: 0.95rem;
}

.pill {
  padding: 0.25rem 0.7rem;
  border-radius: 999px;
  font-weight: 600;
  font-size: 0.8rem;
  border: 1px solid transparent;
}

.pill-blue {
  background: rgba(66, 153, 225, 0.12);
  color: #2b6cb0;
  border-color: rgba(66, 153, 225, 0.2);
  box-shadow: 0 6px 18px rgba(66, 153, 225, 0.12);
}

.subtitle {
  font-size: 0.9rem;
  margin-left: 0.5rem;
  opacity: 0.7;
  font-weight: normal;
  color: #718096;
}

/* 滚动区域 */
.tall-list {
  max-height: 68vh;
  overflow-y: auto;
}

.small-max {
  max-height: 260px;
  overflow-y: auto;
}

.scrollable::-webkit-scrollbar {
  width: 6px;
}

.scrollable::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.scrollable::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 3px;
}

/* 更新列表 */
.list {
  list-style: none;
  padding: 0;
  margin: 0;
  position: relative;
}

.list::before {
  content: '';
  position: absolute;
  left: 70px;
  top: 0.5rem;
  bottom: 0.5rem;
  width: 2px;
  background: linear-gradient(180deg, rgba(66, 153, 225, 0.12), rgba(102, 126, 234, 0.18));
}

.update-item {
  display: flex;
  align-items: flex-start;
  padding: 0.8rem 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  position: relative;
  gap: 0.6rem;
  padding-left: 1.2rem;
}

.update-item:last-child {
  border-bottom: none;
}

.date {
  width: 70px;
  color: #4299e1;
  font-size: 0.85rem;
  font-weight: 600;
  flex-shrink: 0;
}

.timeline-dot {
  position: absolute;
  left: 64px;
  top: 1.1rem;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #138dda;
  border: 3px solid #edb3b3;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.timeline-dot.active {
  background: linear-gradient(135deg, #ff3c00, #ff0000);
  box-shadow: 0 6px 16px rgba(102, 126, 234, 0.35);
}

.text {
  flex: 1;
  font-size: 0.95rem;
  color: #4a5568;
  line-height: 1.5;
}

.large-text {
  font-size: 1.05rem;
}

.new-badge {
  color: #ff0000;
  padding: 0.2rem 0.5rem;
  border-radius: 12px;
  font-size: 0.7rem;
  font-weight: bold;
  margin-left: 0.5rem;
  flex-shrink: 0;
}

/* 特别感谢网格 */
.thanks-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 1rem;
  list-style: none;
  padding: 0;
  margin: 0;
}

.thanks-item {
  text-align: center;
  transition: transform 0.3s ease;
}

.thanks-item.clickable {
  cursor: pointer;
}

.thanks-item:hover {
  transform: translateY(-5px);
}

.sponsor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(100px, 1fr));
  gap: 1rem;
  padding: 0;
  margin: 0;
}

@media (min-width: 900px) {
  .sponsor-grid {
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  }
}

.sponsor-card {
  display: flex;
  align-items: stretch;
  justify-content: stretch;
  padding: 0;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(0, 0, 0, 0.04);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.06);
  transition: all 0.2s ease;
}

.sponsor-card.clickable {
  cursor: pointer;
}

.sponsor-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.08);
}

.sponsor-img-wrap {
  width: 100%;
  height: 100%;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.08);
}

.sponsor-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
  background: #fff;
}

.avatar-container {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  overflow: hidden;
  margin: 0 auto 0.5rem;
  border: 2px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.avatar-large {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.name {
  font-size: 0.9rem;
  color: #4a5568;
  font-weight: 500;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .content-area {
    grid-template-columns: 1.6fr 1fr;
  }

  .thanks-grid {
    grid-template-columns: repeat(4, 1fr);
  }

}

@media (max-width: 992px) {
  .highlight-row {
    grid-template-columns: repeat(2, 1fr);
  }

  .header {
    flex-direction: column;
    text-align: center;
    gap: 1.5rem;
  }

  .header-left {
    flex-direction: column;
    gap: 1rem;
  }

  .content-area {
    grid-template-columns: 1fr;
  }

  .side-column {
    flex-direction: row;
    flex-wrap: wrap;
  }
}

@media (max-width: 768px) {
  .sponsor-page {
    padding: 1rem;
  }

  .highlight-row {
    grid-template-columns: 1fr;
  }

  .side-column {
    flex-direction: column;
  }

  .thanks-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .header-text h1 {
    font-size: 1.8rem;
  }

  .header-text p {
    font-size: 1rem;
  }
}

@media (max-width: 576px) {
  .thanks-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .card-header {
    flex-direction: column;
    text-align: center;
    gap: 0.5rem;
  }

  .update-item {
    flex-direction: column;
    gap: 0.3rem;
  }

  .date {
    width: auto;
  }
}
</style>