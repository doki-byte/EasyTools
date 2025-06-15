package model

import (
	"log"

	"gorm.io/gorm"
)

const TableNameSites = "sites"

type Sites struct {
	ID       int    `json:"id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Remark   string `json:"remark"`
	URL      string `json:"url"`
	Icon     string `json:"icon,omitempty"` // 可选字段
	CateSort int    `json:"catesort"`       // 新增分类排序字段
	SiteSort int    `json:"sitesort"`       // 新增分类排序字段
}

func (*Sites) TableName() string {
	return TableNameSites
}

func (s *Sites) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&Sites{}).Count(&count).Error; err != nil {
		log.Fatalf("Error counting records: %v", err)
	}

	if count == 0 {
		defaultSites := []Sites{

			{Category: "空间测绘", Icon: "/assets/site/default.png", Title: "Censys Search", Remark: "Censys helps organizations, individuals, and researchers find and monitor every server on the Internet to reduce exposure and improve security.", URL: "https://search.censys.io/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/99992fed6211d16147dcbb145b02890c.png", Title: "FoFa网络空间测绘", Remark: "FOFA 是白帽汇推出的一款网络空间搜索引擎，它通过进行网络空间测绘，能够帮助研究人员或者企业迅速进行网络资产匹配，例如进行漏洞影响范围分析、应用分布统计、应用流行度排名统计等。", URL: "https://fofa.info/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/551def794158fba06668a0bb4fa29528.png", Title: "ZoomEye", Remark: "钟馗之眼", URL: "https://www.zoomeye.org/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/5599504c66c02fd456f15100e1e6ca53.png", Title: "Shodan Search Engine", Remark: "Search Engine for the Internet of Things", URL: "https://www.shodan.io/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/40b4edd8cbe6e6e43f2162e565ce3be6.png", Title: "360网络空间测绘", Remark: "360网络空间测绘 — 因为看见，所以安全", URL: "https://quake.360.net/quake/#/index", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/82167f602fbf4f302a7b13fbc9214b20.png", Title: "鹰图平台", Remark: "鹰图平台（HUNTER）是奇安信全球鹰推出的一款全球互联网资产搜集平台，助力HW、未知资产发现、暴露面梳理、定位热点应用等场景。自主研发扫描引擎，全端口扫描，覆盖全球ipv4，7天更新国内资产，30天更新海外资产。类似产品：国内FOFA、QUAKE、ZOOMEYE；海外SHODAN、CENSYS、SPYSE", URL: "https://hunter.qianxin.com/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/d18bb3ad705fd0b49f1c3fa88b83c732.png", Title: "谛听", Remark: "谛听 - 专注工控安全的搜索引擎", URL: "https://www.ditecting.com/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/69273f6e8d9f8fd7c66b60293c47cd93.png", Title: "零零信安", Remark: "00SEC-ASM™ 零零信安攻击面管理平台，专注于企业外部攻击面管理（信息系统、小程序、公众号、APK、邮箱、代码/文档、域名等）及暗网情报监控。", URL: "https://0.zone/", CateSort: 0, SiteSort: 0},
			{Category: "空间测绘", Icon: "/assets/site/default.png", Title: "virustotal", Remark: "C段 域名/IP地址信息", URL: "https://www.virustotal.com/gui/home/upload", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/default.png", Title: "牧云 webshell 查杀", Remark: "CT Stack 安全社区致力于守护安全工具成长，秉持探索与共享的理念，收录更多优质工具。规范安全工具标准，与之共生共", URL: "https://stack.chaitin.com/security-challenge/webshell/index", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/299f8180d3f230f0135b0492fb5e6f7a.png", Title: "大圣云沙箱检测系统", Remark: "大圣云沙箱检测系统", URL: "https://sandbox.vulbox.com/detect", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/7221e75130843f81e1edb5a75c62769a.svg", Title: "VirusTotal", Remark: "VirusTotal", URL: "https://www.virustotal.com/gui/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/default.png", Title: "WEBDIR", Remark: "WEBDIR+ - WebShell 扫描服务 - OpenRASP 团队", URL: "https://scanner.baidu.com/#/pages/intro", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/511814d1fe08b3a1c9b6fb495f87b072.png", Title: "河马在线查杀", Remark: "shellpub.com专注webshell查杀,免费查杀软件,web查杀,webshell在线查杀,在线检测网站后门，速度快，误报低，准确度高", URL: "https://n.shellpub.com/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/bfa2f176bfc2290c374bf2d547e8abfe.png", Title: "D盾网站查杀", Remark: "D盾,D盾_防火墙,D盾_IIS防火墙,D盾_web查杀,IIS防火墙,webshell查杀", URL: "https://www.d99net.net/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/5f40ad8b41bcaadb293fb5a0d297fe7a.png", Title: "在线webshell查杀", Remark: "webshell,一句话后门,php变异一句话清除,这些难题交给我,让查杀更简单,你只管喝茶，剩下的交给我!", URL: "http://tools.bugscaner.com/killwebshell/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/f49203308261c95add08e2fd788cb5ea.png", Title: "腾讯哈勃文件分析", Remark: "腾讯哈勃文件分析", URL: "https://habo.qq.com/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/0f9cea77a78b812aa5167393664b9221.png", Title: "VirScan", Remark: "VirScan - 多引擎文件在线检测平台", URL: "https://www.virscan.org/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/0b60588630c1fc70e8de38572bf02c50.png", Title: "在线查毒-安全实验室-腾讯手机管家官方网站", Remark: "腾讯手机管家", URL: "https://m.qq.com/security_lab/scans_online.jsp", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/default.png", Title: "360手机应用检测", Remark: "360手机应用检测", URL: "http://scan.shouji.360.cn/", CateSort: 0, SiteSort: 0},
			{Category: "在线查杀", Icon: "/assets/site/c8f0aa92f0d039c8a6dcee37e5f97845.png", Title: "阿里云恶意文件检测平台", Remark: "Webshell检测平台", URL: "https://ti.aliyun.com/#/webshell", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/a194cb3217db6a3465f74abada6a3fdf.png", Title: "Tool.lu", Remark: "在线工具,开发人员工具,代码格式化、压缩、加密、解密", URL: "https://tool.lu/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/0dd368af9563bb1ad75a07fe58059c9e.png", Title: "CMD5破解", Remark: "全球最大md5解密", URL: "https://www.cmd5.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/f4a2a05d1a3854100729cc3fad7a6758.png", Title: "SOMD5", Remark: "输入让你无语的MD5", URL: "https://www.somd5.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/dc34c2e39c2301515521181ab4b5078f.png", Title: "PMD5", Remark: "MD5在线解密", URL: "https://pmd5.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/195d3e7aea8a7bfd68b4c8877f56c62c.png", Title: "超强js加密", Remark: "Online Javascript Obfuscator", URL: "https://gin-gonic.com/zh-cn/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/b389447b6b46571c38949fd5bcbaa254.png", Title: "16进制转换", Remark: "16进制转换，16进制转换文本字符串，在线16进制转换 | 在线工具", URL: "https://www.sojson.com/hexadecimal.html", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/7db6a19faa29297515ccac4c22a00f77.png", Title: "Base64 在线编码解码", Remark: "Base64 在线编码解码 | Base64 加密解密 - Base64.us", URL: "https://base64.us/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/default.png", Title: "在线加密解密", Remark: "文字在线加密解密、散列/哈希、BASE64、SHA1、SHA224、SHA256、SHA384、SHA512、MD5、HmacSHA1、HmacSHA224、HmacSHA256、HmacSHA384、HmacSHA512、HmacMD5、urlencode、urldecode", URL: "http://encode.chahuo.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/default.png", Title: "RSA密码分解", Remark: "RSA密码分解", URL: "http://www.factordb.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/default.png", Title: "CyberChef", Remark: "The Cyber Swiss Army Knife", URL: "https://www.chinabaiker.com/cyberchef.htm", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/ed8b4329c1652f686100040e04f27910.png", Title: "CTF在线工具", Remark: "CTF在线工具-CTF工具|CTF编码|CTF密码学|CTF加解密|程序员工具|在线编解码", URL: "http://www.hiencode.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/8d11328900e2aa7e601dbd5ffd5a0b08.png", Title: "中文电码查询", Remark: " Chinese Commercial Code 中文电码查询工具，提供标准中文电码在线查询服务，如美国签证表格填写的中文电码一项可在这里免费查询，电码查询结果只供参考，权威电报码查询可到邮政局。", URL: "http://code.mcdvisa.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/679078c973145a0d90951e34c882402e.png", Title: "quipqiup", Remark: "quipqiup是Edwin Olson的快速自动密码求解器", URL: "https://www.quipqiup.com/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/5bdc85f4dded9097229b18ea7c7ed1a2.png", Title: "php免费在线解密", Remark: "php免费在线解密-PHP在线解密", URL: "http://dezend.qiling.org/free.html", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/924fdd477a70d9fce5feac2902fe9602.png", Title: "python反编译", Remark: "pyc反编译,py反编译,python反编译,python字节码反编译,支持所有python版本", URL: "https://tool.lu/pyc/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/cfa1ae9d74bc4fd8dd4dfff49335ac97.png", Title: "在线pyc,pyo,python,py文件反编译，目前支持python1.5到3.6版本的反编译-在线工具", Remark: "验证码：0168本工具目前支持反编译Python的版本有Python 1.5 ,Python 2.1 ,Python 2.2 ,Python 2.3 ,Python 2.4 ,Python 2.5 ,Python 2.6 ,Python 2.7 ,Python 3.0,Python 3.1,Python 3.2,Python 3.3 ,Python 3.4 ,Python 3.5,Python 3.6", URL: "http://tools.bugscaner.com/decompyle/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/74f20036ce6150ad440394fc802d0ae2.png", Title: " jwt.io", Remark: "JSON Web Token (JWT) is a compact url-safe means of representing claims to be transferred between two parties. The claims in a JWT are encoded as a JSON object that is digitally signed using JSON Web Signature (JWS).", URL: "https://jwt.io/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/4911522719f691bb4d2a25154c27c5e8.png", Title: "HTML 编码/解码 | 菜鸟工具", Remark: "菜鸟工具-HTML 编码/解码", URL: "https://c.runoob.com/front-end/691/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/9330d18550556e863bb07562f9e6c47b.png", Title: "16进制到文本字符串的转换", Remark: "16进制到文本字符串的转换，在线实时转换", URL: "https://www.bejson.com/convert/ox2str/", CateSort: 0, SiteSort: 0},
			{Category: "加密解密", Icon: "/assets/site/default.png", Title: "在线3DES加密解密", Remark: "在线3DES加密解密、3DES在线加密解密、3DES encryption and decryption", URL: "http://tool.chacuo.net/crypt3des", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/3eb1b53177e40306bb9a3f766b944f22.png", Title: "火线安全平台-火线Zone安全社区", Remark: "", URL: "https://www.huoxian.cn/community#project", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "华为安全奖励计划", Remark: "华为SRC,华为安全奖励计划, 华为终端安全漏洞奖励计划, 华为终端云服务安全奖励计划, 华为云漏洞奖励计划, 华为终端IoT产品安全漏洞奖励计划,huawei bug bounty program", URL: "https://bugbounty.huawei.com/hbp/#/home", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "字节跳动安全中心", Remark: "", URL: "https://security.bytedance.com/src", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/68f4b01d0596f7d8759c4f70eef4d8cf.png", Title: "雷神众测", Remark: "雷神众测,安全众测,红队服务,网络安全,Bug Bounty,Red Team| Bounty Team，加入雷神众测 网聚优秀的安全力量 连接数万白帽,快速发现企业安全问题", URL: "https://www.bountyteam.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "CNVD官方漏洞", Remark: "CNVD官方漏洞库", URL: "https://www.cnvd.org.cn/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "补天 - 企业和白帽子共赢的漏洞响应平台，帮助企业建立SRC", Remark: "补天漏洞响应平台旨在建立企业与白帽子之间的桥梁，帮助企业建立SRC(安全应急响应中心)，让企业更安全，让白帽子获益。", URL: "https://www.butian.net/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "漏洞盒子", Remark: "", URL: "https://www.vulbox.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/9f716d456e1989a1e88a69507230f476.png", Title: "教育漏洞报告平台(EDUSRC)    ", Remark: "教育漏洞报告平台是一个高校相关漏洞报告平台。", URL: "https://src.sjtu.edu.cn/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/7eae0b80a3a6eba1b0ed77ab2ef90205.png", Title: "360众测平台", Remark: "", URL: "https://zhongce.360.net/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/6b12662247cfbbb4c2652b5bc3023354.png", Title: "BUGBANK 官方网站 | 领先的网络安全漏洞发现品牌 | 开放安全的提出者与倡导者 | 创新的漏洞发现平台", Remark: "BUGBANK是国内首家互联网安全服务SAAS平台，也是安全众测平台，在保护企业隐私的前提下，帮助企业建立漏洞应急响应中心，并在第一时间发现最具威胁的零日漏洞", URL: "https://www.bugbank.cn/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/761361777a0f164ba0c97182fe98ffdd.png", Title: "蚂蚁集团安全应急响应中心官网 - AntSRC - 蚂蚁集团安全响应中心,支付宝安全响应中心,阿里巴巴漏洞反馈,支付宝漏洞反馈,蚂蚁集团漏洞反馈,网商银行漏洞反馈,芝麻信用漏洞反馈,口碑漏洞反馈,花呗漏洞反馈,蚂蚁财富漏洞反馈,支付宝情报反馈,阿里巴巴情报反馈,网商银行漏洞,芝麻信用漏洞,口碑漏洞,花呗漏洞,蚂蚁财富漏洞,阿里巴巴漏洞,支付宝漏洞,蚂蚁集团漏洞,支付宝情报,阿里巴巴情报", Remark: "支付宝安全应急响应中心,AntSRC,蚂蚁集团安全响应中心,支付宝安全响应中心,阿里巴巴漏洞反馈,支付宝漏洞反馈,蚂蚁集团漏洞反馈,网商银行漏洞反馈,芝麻信用漏洞反馈,口碑漏洞反馈,花呗漏洞反馈,蚂蚁财富漏洞反馈,支付宝情报反馈,阿里巴巴情报反馈,网商银行漏洞,芝麻信用漏洞,口碑漏洞,花呗漏洞,蚂蚁财富漏洞,阿里巴巴漏洞,支付宝漏洞,蚂蚁集团漏洞,支付宝情报,阿里巴巴情报", URL: "https://security.alipay.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/be1bc194fb6f0a87318f1f8f674acafb.png", Title: "美团安全应急响应中心", Remark: "", URL: "https://security.meituan.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/d43eb9219575b1d0b33a493f6b9600fa.png", Title: "滴滴出行安全应急响应中心", Remark: "", URL: "https://sec.didichuxing.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/9cd72a3a9d59136d659182ddfeae84c7.png", Title: "腾讯安全应急响应中心        ", Remark: "腾讯安全应急响应中心,Tencent Security Response Center,TSRC", URL: "https://security.tencent.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/87a2e1166641368e5962fc4c869d75a4.png", Title: "爱奇艺安全应急响应中心", Remark: "爱奇艺,iQIYI,应急响应,爱奇艺安全应急响应中心,71SRC,iQIYI Security Response Center", URL: "https://security.iqiyi.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/c0c8a1121d1ebff13c6688ccc6d2723f.png", Title: "安全狗-领先云安全服务与解决方案提供商|云原生安全|服务器安全|网站安全|态势感知", Remark: "安全狗,领先云安全服务与解决方案提供商,融合大数据分析、可视化、态势感知、威胁情报分析技术，为客户提供一站式云安全产品、服务和解决方案,实现服务器、网站及业务的安全稳定运行。", URL: "https://www.safedog.cn/index.html", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/74d96d2eb873d36533dcd58c0efe38e1.png", Title: "vivo 安全应急响应平台", Remark: "", URL: "https://security.vivo.com.cn/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "小米安全中心", Remark: "小米安全中心", URL: "https://sec.xiaomi.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "微博安全应急响应中心", Remark: "微博安全应急响应中心", URL: "https://wsrc.weibo.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/8c28c6d97603c671e6f5d870583868b1.svg", Title: "顺丰安全应急响应中心", Remark: "", URL: "https://sfsrc.sf-express.com/home", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/05446dd1c4ecbba2cca76c073eed53cd.png", Title: "京东安全应急响应中心", Remark: "", URL: "https://security.jd.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/5f6b396f3322720db2404c9bdf04c4af.png", Title: "阿里安全中心", Remark: "", URL: "https://security.alibaba.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "平安安全应急响应中心", Remark: "", URL: "https://security.pingan.com/homePage/index", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "苏宁安全应急响应中心", Remark: "", URL: "https://security.suning.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/49caa57c6498e1d0d204a46c8e51f6a3.png", Title: "OPPO 安全中心", Remark: "", URL: "https://security.oppo.com/cn/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/cc1ee31d9abe66c85a6526824c8182cc.png", Title: "携程安全应急响应中心", Remark: "携程安全中心主要为广大白帽黑客提供一个向携程提交漏洞，交流安全信息的平台，提交携程安全信息根据您的贡献值可以获得丰厚的礼品和现金奖励。携程还为您提供特价酒店查询、机票预订、飞机票查询、时刻表、票价查询、航班查询、度假预订、商旅管理、为您的出行提供全方位旅行服务。", URL: "https://sec.ctrip.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/a3a86542d88bcb15f957685540b9543e.png", Title: "哔哩哔哩安全应急响应中心", Remark: "哔哩哔哩安全应急响应中心", URL: "https://security.bilibili.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/a1b909096cda3a04cd91da5b947fab95.png", Title: "360安全应急响应中心", Remark: "360安全应急响应中心,QIHOO 360 Security Response Center,360SRC", URL: "https://security.360.cn/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/f9e925ffb33a69ec3e1147c2a0492789.png", Title: "百度安全应急响应中心", Remark: "百度安全响应中心联合广大的白帽子，以建设安全的互联网为己任!", URL: "https://bsrc.baidu.com/v2/#/home", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/823fcee5d7e3a31c8f543c1aecbb704c.png", Title: "美丽联合集团安全应急响应中心", Remark: "美丽联合集团安全应急响应中心", URL: "https://security.mogu.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/e96ce976bfd76f15d2df739dd5a86306.png", Title: "贝壳安全应急响应中心", Remark: "", URL: "https://security.ke.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "", Title: "唯品会应急响应中心", Remark: "", URL: "https://sec.vip.com/", CateSort: 0, SiteSort: 0},
			{Category: "SRC众测", Icon: "/assets/site/00207577baa62af00aff36ba8f9ad223.png", Title: "魅族安全中心", Remark: "", URL: "https://sec.meizu.com/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/741a5b3b981aac391db067dad2012fb3.png", Title: "微步在线X情报社区", Remark: "微步在线X情报社区是国内首个综合性威胁分析平台和威胁情报共享的开放社区，同时提供威胁情报查询、域名反查、IP反查，行业情报等服务，辅助个人及企业快速定位及排除安全隐患", URL: "https://x.threatbook.com/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/a32721256b850d01ac05250530c4f570.png", Title: "360威胁情报中心", Remark: "", URL: "https://ti.360.net/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "", Title: "RedQueen威胁情报中心", Remark: "", URL: "https://redqueen.tj-un.com/IntelHome.html", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "", Title: "奇安信威胁情报中心", Remark: "奇安信ALPHA威胁分析平台是面向安全分析师、事件响应人员的综合性威胁情报分析平台，以海量多维度网络空间安全数据为基础，实现报警研判、攻击定性等功能，威胁研判分析平台是构建新型安全架构的核心组件之一。", URL: "https://ti.qianxin.com/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/f9b44814856d988c6e670fa9a6e1de89.png", Title: "VenusEye威胁情报中心", Remark: "启明星辰威胁情报中心VenusEye提供威胁情报实时在线查询服务，与各类网络安全设备和软件系统协同工作，为用户现场威胁分析和防护决策提供数据支撑。", URL: "https://www.venuseye.com.cn/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/986352bef0aefa02ce60dbbd925acb4c.png", Title: "绿盟 威胁情报中心", Remark: "NTI - 提供专业的情报在线查询服务，帮助客户及时洞悉公网资产面临的安全问题，为客户提供最新的威胁动态，结合安全数据深度分析，提升未知攻击检测和主动的威胁防御响应", URL: "https://ti.nsfocus.com/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/90a255404fed6ed4e1d154e4155d1323.png", Title: "安恒星图平台", Remark: "安全星图平台拥有一支专注于未知威胁分析挖掘的顶尖团队，基于大数据架构对全网数据、情报进行收集积累，具备10多年恶意代码研究经验的研究团队打造了一套AI智能的威胁情报挖掘生产机制。通过持续提供威胁情报数据与服务，可为用户提升区域安全态势感知能力，检测未知威胁，分析溯源威胁行为，提高主动防御能力等。", URL: "https://ti.dbappsecurity.com.cn/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/b650cf37f3363100cee433d93529247b.png", Title: "安天威胁情报中心", Remark: "安全", URL: "https://www.antiycloud.com/#/antiy/index", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/82d63abd804039291bb4e43a3be4c59a.png", Title: "IBM X-Force Exchange", Remark: "IBM X-Force Exchange is a threat intelligence sharing platform enabling research on security threats, aggregation of intelligence, and collaboration with peers", URL: "https://exchange.xforce.ibmcloud.com/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "", Title: "华为安全中心平台", Remark: "", URL: "https://isecurity.huawei.com/sec/web/intelligencePortal.do", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/f9c671e8d7a5408da611fa1370885e73.png", Title: "绿盟威胁情报中心（NTI）", Remark: "绿盟威胁情报依托于绿盟科技二十年安全攻防能力的沉淀，致力于为全球企业客户提供最快速、最准确、最可信的威胁情报数据。秉承公司&quot;专攻术业，成就所托&quot;的宗旨，成为企业客户最放心的威胁预警和响应处置专家。绿盟科技作为入选Gartner《全球威胁情报指南》的国内知名厂商，将为客户的每一分安全，供献自己的全部力量。", URL: "https://nti.nsfocus.com/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/59724546c7e42032b6d82901558cbb56.png", Title: "山石云瞻 威胁情报中心", Remark: "", URL: "https://ti.hillstonenet.com.cn/", CateSort: 0, SiteSort: 0},
			{Category: "威胁情报", Icon: "/assets/site/8625415dc145f7f77da8c487926bad28.png", Title: "深信服威胁情报中心", Remark: "", URL: "https://ti.sangfor.com.cn/analysis-platform", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/5748d00351f99ab4afad90961ff2edd8.png", Title: "掌控安全导航", Remark: "掌控安全导航 - 安全人必备的导航，带你探索安全、白帽黑客暗网世界", URL: "https://i.zkaq.cn/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "代码审计", Remark: "Accelerate Clean Code for developers and teams to enable clear, readable, understandable, maintainable, portable, reliable and secure code standards across your organization.  ", URL: "https://www.sonarsource.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "显示查询自己的IP地址", Remark: "从中国和美国显示查询自己的IP地址", URL: "http://ip111.cn/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/a0d2e0e8ad95434ee262bd09e90ffd9a.png", Title: "DeepL翻译", Remark: "Translate texts &amp;amp;amp; full document files instantly. Accurate translations for individuals and Teams. Millions translate with DeepL every day.", URL: "https://www.deepl.com/translator", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/8afcd9e56294fd0cf2d1157020d7074b.png", Title: "爱资料工具-好用的在线工具箱", Remark: "爱资料工具(toolnb.com),为开发运维提供全面的在线工具箱,已开发工具400款,包含开发工具,运维工具,常用工具,SEO站长工具等,是好用,方便的在线工具网站.", URL: "https://www.toolnb.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/50bd831e2aaf58fdc31038fb5db32113.png", Title: "tools.fun", Remark: "开发人员工具箱，时间戳转换，json格式化，正则表达式，URLEncode，加密解密，Crontab，websocket，md5编码，base64编码，颜色转换，JSON/YAML/XML转换，AES、DES、RSA，数字进制转换，图片压缩，二维码生成，JavaScript格式化压缩，代码差异对比", URL: "https://tools.fun/index.html", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/3ed841d6567acb3abbb5883afb639b11.png", Title: "免费在线思维导图生成", Remark: "ProcessOn是一款专业在线作图工具和知识分享社区，提供AI生成思维导图流程图。支持思维导图、流程图、组织结构图、网络拓扑图、鱼骨图、UML图等多种图形，同时可实现人与人之间的实时协作和共享，提升团队工作效率。", URL: "https://www.processon.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "Cubox 个人碎片知识库", Remark: "Cubox 知识库 - 轻松收集、自动整理、深度阅读、灵活管理。帮助你从容面对海量信息，善用网络碎片获得提升。", URL: "https://cubox.pro/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/8ba03034292d191efab6297699ca597e.svg", Title: "正则 Learn - 逐步从零基础到高阶。", Remark: "交互式学习正则表达式，在您所处的阶段练习、测试和分享您自己的正则表达式。", URL: "https://regexlearn.com/zh-cn", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "vscode 网页版", Remark: "", URL: "https://vscode.dev/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "PHP代码语法检查", Remark: "", URL: "https://cn.piliapp.com/php-syntax-check/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/380c4a365b7f470e564713e83f9f04e5.png", Title: "在线思维导图", Remark: "xmind, naotu, 脑图, nt, swdt, siweidaotu, 思维导图", URL: "https://tools.fun/xmind.html", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/ab77157a7a20e33cea94f7e5768b1918.png", Title: "菜鸟工具", Remark: "菜鸟工具，为开发设计人员提供在线工具，网址导航，提供在线PHP、Python、 CSS、JS 调试，中文简繁体转换，进制转换等工具。致力于打造国内专业WEB开发工具，集成开发环境，WEB开发教程。..", URL: "https://c.runoob.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/d0c1da2a2e30370398c048a0cd11e9fe.png", Title: "YoviSun工具集", Remark: "YoviSun个人应用系列，科研生活，无处不在。", URL: "https://tool.yovisun.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "谷歌学术镜像_Google镜像站", Remark: "主要提供谷歌学术搜索Google Scholar镜像和谷歌网页搜索镜像的导航站,实时更新最新镜像网站", URL: "http://scholar.scqylaw.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "谷歌镜像站 -- 密码：心灵之约、水朝夕、csxy@123", Remark: "心灵之约、水朝夕、csxy@123", URL: "https://g.luciaz.me/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/0198978e4926064f8db32f3a717e0b2f.png", Title: "MSDN, 我告诉你 - 做一个安静的工具站", Remark: "MSDN, 我告诉你", URL: "https://msdn.itellyou.cn/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/f74d02eb84ea5eeafa2f0c9506c24016.png", Title: "GitHub 文件加速", Remark: "", URL: "http://gh.301.ee/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/c820bce1f1de6ba7e59673760eb7681f.png", Title: "正则匹配", Remark: "Regular expression tester with syntax highlighting, PHP / PCRE &amp; JS Support, contextual help, cheat sheet, reference, and searchable community patterns.", URL: "https://regexr.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "json2yaml", Remark: "", URL: "http://www.json2yaml.com/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/f37f11d8580a2ca5eb26157a6fa69d6a.png", Title: "在线JS代码格式化、JS代码美化工具", Remark: "在线JS代码格式化、JS代码美化工具", URL: "https://www.qianbo.com.cn/Tool/Beautify/Js-Formatter.html", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "de4js | JavaScript Deobfuscator and Unpacker", Remark: "JavaScript Deobfuscator and Unpacker", URL: "https://lelinhtinh.github.io/de4js/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/d3fc993f811a2b643cb67ef763c99ead.png", Title: "摸鱼大闯关", Remark: "", URL: "https://p.hancel.org/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/3e2e35bf08d5226e68eb561340afff59.png", Title: "在线生成透明ICO图标——ICO图标制作", Remark: "提供ico图标在线制作、快速ico图标制作、icon图标制作、favicon、可以将png转ico、favicon在线制作、所有图片转ico，透明ico图标制作、动态ico图标制作方法及将所制作的ico图标下载下来，作为favicon.png文件。", URL: "https://www.png51.cn/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/cf2b5c9b887f5dc0b6e03907baa173bd.png", Title: "批量备案查询", Remark: "狗狗查询备案查询提供专业的备案信息查询、ICP备案查询、网站备案查询等查询功能。", URL: "https://www.ggcx.com/main/batch/record", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "/assets/site/55a9c4da9f1010dab32ff07a4e60ea17.png", Title: "代码在线执行", Remark: "在线运行php,c,c++,go,python,nodejs,java,groovy代码，测试代码", URL: "https://tool.lu/coderunner/", CateSort: 0, SiteSort: 0},
			{Category: "常用工具", Icon: "", Title: "AbeimAPI", Remark: "AbeimAPI是免费为用户提供网络数据接口调用的服务平台，我们致力于为用户提供稳定、快速的免费API数据接口服务。", URL: "https://res.abeim.cn/api", CateSort: 0, SiteSort: 0},
		}
		if err := db.Create(&defaultSites).Error; err != nil {
			log.Fatalf("Error inserting default data: %v", err)
		}
	}
}
