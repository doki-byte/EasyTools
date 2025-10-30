<template>
  <div id="wails-app" class="sponsor-page">
    <header class="header">
      <div class="header-left">
        <div class="header-text">
          <h1>赞助与支持</h1>
          <p>感谢您对项目的关注与支持，一起推动创新。</p>
        </div>
      </div>
      <div class="header-right row-inline">
        <div class="qr-card">
          <img :src="qrImage" alt="公众号二维码" class="qr-image" />
          <div class="qr-wrapper">
            <h3>关注公众号</h3>
            <p>扫码关注，获取更多资讯</p>
          </div>
        </div>
      </div>
    </header>

    <div class="content-area">
      <section class="left-panel">
        <div class="row">
          <div class="card glass-effect">
            <div class="card-header">
              <h2>更新说明</h2>
            </div>
            <ul class="list scrollable max-limited">
              <li v-for="(u,i) in updates" :key="i" class="update-item">
                <span class="date">{{ u.date }}</span>
                <span class="text">{{ u.content }}</span>
                <span v-if="u.isNew" class="new-badge">NEW</span>
              </li>
            </ul>
          </div>
          <div class="card glass-effect">
            <div class="card-header">
              <h2>特别感谢<span class="subtitle">(以下排名以时间顺序排列)</span></h2>
            </div>
            <ul class="thanks-grid scrollable max-limited">
              <li v-for="(p,i) in specialThanks" :key="i" class="thanks-item">
                <div class="avatar-container">
                  <img :src="p.src" alt="avatar" class="avatar-large" />
                </div>
                <div class="name">{{ p.name }}</div>
              </li>
            </ul>
          </div>
        </div>
        <div class="row">
          <div class="link-card glass-effect">
            <div class="card-header">
              <h2>相关链接</h2>
            </div>
            <ul class="links scrollable max-limited">
              <li v-for="(l,i) in relatedLinks" :key="i" class="link-item">
                <a href="#" class="link-button" @click.prevent="openUrl(l.url)">
                  <span>{{ l.text }}</span>
                </a>
              </li>
            </ul>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import {BrowserOpenURL} from "../../wailsjs/runtime";

export default {
  name: 'SponsorPage',
  data() {
    return {
      updates: [
        {date:'v1.9.3',content: '1. 优化redis界面，修复连接过多导致的页面无限下拉的bug; 2. 新增显示更新详情功能; 3. 参考fine修改小程序反编译界面; 4. tools新建工具的时间添加打开工具文件位置的功能; 5. 新增二级菜单的显示/隐藏,位置移动; 6. 修改ftp下载方式为浏览器下载，避免EasyTools卡死; 7. 优化JwtCreck功能;',isNew:true},
        {date:'v1.9.2',content: '1. 新增redis命令执行功能;'},
        {date:'v1.9.1',content: '1. 修复note笔记显示逻辑;'},
        {date:'v1.9.0',content: '1. 优化note笔记模块图片显示规则，现在支持直接粘贴图片并保存预览; 2. 优化部分功能的界面显示; 3. 修复点击便携发包之后，左边菜单栏大小变化的bug; 4. 修复windows下，未选择工具目录时，直接运行工具报错的bug;'},
        {date:'v1.8.9',content: '1. 新增程序自启动功能; 2.优化部分功能的bug; 3. 免杀模块新增白加黑、优化沙箱逻辑、修改免杀加载器等多个模块功能'},
        {date:'v1.8.8',content: '1. 修复便携发包点击send之后白屏或者一直loading的bug'},
        {date:'v1.8.7',content: '1. 新增便携发包功能，内置微信公众号/小程序、钉钉、企业微信调用接口; 2. 优化系统目录; '},
        {date:'v1.8.6',content: '1. 简化Tools导航添加工具逻辑; 2. 新增site导航自动识别获取站点信息功能; 3. 优化模拟生成身份证时时间选择功能; 4. 新增系统托盘; 5. 新增全局快捷键控制程序的显示与隐藏'},
        {date:'v1.8.5',content: '1. 修改Note笔记界面为奶白色; 2. 新增菜单顺序调整功能'},
        {date:'v1.8.4',content: '1. 新增fuzz调用界面'},
        {date:'v1.8.3',content: '1. 新增代理池功能；2. 优化CyberChef、ssh连接功能'},
        {date:'v1.8.2',content: '优化工具、网址搜索功能;'},
        { date: 'v1.8.1', content: '1.新增Note备忘录功能;2.优化工具、网址搜索功能;3.新增jwt爆破功能;4.新增地图接口调用功能;5.优化redis连接及显示功能;6.修改CyberChef为中文版本的'},
        { date: 'v1.8.0', content: '1.新增工具箱打开路径;2.新增图标选择功能;3.优化ftp、ssh连接弹窗提示;4.小程序反编译模块新增node环境监测功能;5.新增杀软检测目录进程清单;6.修改CyberChef为中文版本的'},
        { date: 'v1.7.5', content: '1.新增小程序反编译功能（需要自行先安装node环境，版本14以上都行，或者直接安装最新版即可）;2.修改oss资源桶遍历逻辑;3.修改免杀加载器模版' },
        { date: 'v1.7.4', content: '1.修改文件捆绑失败的bug;2.更新ByPassQvm图标' },
        { date: 'v1.7.3', content: '1.更新杀软识别数据库' },
        { date: 'v1.7.2', content: '1.新增FTP连接功能（匿名登录时，用户名为anonymous ，密码为空）' },
        { date: 'v1.7.1', content: '修复一个小bug' },
        { date: 'v1.7.0', content: '新增工具仓库、网址站点的任意拖拽功能，修改工具摆放顺序' },
        { date: 'v1.6', content: '新增登录页面默认密码修改功能;更新webssh功能，新增批量ssh命令执行功能' },
        { date: 'v1.5', content: '免杀生成：新增分离、远程两种加载模式' },
        { date: 'v1.4', content: '新增一键去除upx特征的功能（测试，只是规避明显特征，避免直接upx -d脱壳）;新增一键添加数字签名的小功能' },
        { date: 'v1.3', content: '集成ShellCode在线免杀处理平台;新增文件体积修改功能;新增免杀：5种运行模式、2种加密方式、2种编译方式' },
        { date: 'v1.2', content: '新增信息处理模块;迁移蓝队批量封禁IP模块至信息处理  ' },
        { date: 'v1.1', content: '目前集合功能：网址导航、工具导航、信息查询、编码解码%' }
      ],
      specialThanks: [
        { name: 'joker', src: '../assets/about/joker.jpg' },
        { name: 'ryuuz4k1', src: '../assets/about/ryuuz4k1.jpg' },
        { name: 'ToPaint', src: '../assets/about/ToPaint.jpg' },
        { name: '吃人尘', src: '../assets/about/吃人尘.jpg'},
        { name: '夜晓喽', src: '../assets/about/夜晓喽.jpg'},
      ],
      relatedLinks: [
        { text: 'GitHub', url: 'https://github.com/doki-byte/EasyTools' },
        { text: '纷传(叮叮当当)', url: 'https://h5.fenchuan8.com/#/index?forum=102706&yqm=CM4RP'},
        { text: 'QQ交流群(702376033)', url: 'https://qm.qq.com/q/VEzWDiDMgm'},
      ],
      qrImage: '../assets/about/二维码.jpg',
    };
  },
  methods: {
    openUrl(url) {
      BrowserOpenURL(url);
    },
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
  min-height: 100vh;
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
  padding: 0.5rem 2rem;
  margin-bottom: 0.2rem;
  gap: 1rem;
  height: auto;
  composes: glass-effect;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: auto;
}

.header-text h1 {
  margin: 0;
  font-size: 2.2rem;
  font-weight: 600;
  color: #2d3748;
  text-shadow: none;
}

.header-text p {
  margin: 0.5rem 0 0;
  color: #718096;
  font-size: 1.1rem;
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
  display: flex;
  gap: 0.5rem;
}

.left-panel {
  flex: 2;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.row {
  display: flex;
  gap: 0.5rem;
  padding: 0 8px;
}

/* 卡片样式 */
.card {
  composes: glass-effect;
  padding: 1.5rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 420px;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.card:hover {
  transform: translateY(-5px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.12);
}

.link-card {
  composes: glass-effect;
  padding: 0.5rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 200px;
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

.subtitle {
  font-size: 0.9rem;
  margin-left: 0.5rem;
  opacity: 0.7;
  font-weight: normal;
  color: #718096;
}

/* 滚动区域 */
.max-limited {
  max-height: 400px;
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
}

.update-item {
  display: flex;
  align-items: flex-start;
  padding: 0.8rem 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
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

.text {
  flex: 1;
  font-size: 0.95rem;
  color: #4a5568;
  line-height: 1.4;
}

.new-badge {
  background: #e53e3e;
  color: #fff;
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

.thanks-item:hover {
  transform: translateY(-5px);
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

/* 链接样式 */
.links {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  list-style: none;
  padding: 0;
  margin: 0;
}

.link-item {
  margin: 0;
}

.link-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  color: #4a5568;
  text-decoration: none;
  padding: 0.8rem 1rem;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(5px);
  transition: all 0.3s ease;
  font-weight: 500;
  border: 1px solid rgba(0, 0, 0, 0.05);
}

.link-button:hover {
  background: rgba(255, 255, 255, 1);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  color: #2d3748;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .thanks-grid {
    grid-template-columns: repeat(4, 1fr);
  }

  .links {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 992px) {
  .row {
    flex-direction: column;
  }

  .card {
    height: auto;
    min-height: 300px;
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
    flex-direction: column;
  }
}

@media (max-width: 768px) {
  .sponsor-page {
    padding: 1rem;
  }

  .thanks-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .links {
    grid-template-columns: 1fr;
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