## 开发说明

在日常学习过程中，面对各种各样的测试任务，需要使用到多种工具助力，可是工具一多管理起来就不是很方便。

本着不重复造轮子的原则，本工具目前糅合了**工具导航**、**网址导航、简练助手、Redis连接、SSH连接、FTP连接、信息查询、信息处理、编码解码**、**随机生成**、**免杀生成**、**OSS资源桶信息提取**、**小程序反编译**等功能

针对能直接使用的功能，通过嵌入直接调用，在此感谢各位师傅的辛苦开源。

## 更新说明

### v1.8.0

1.  新增工具箱打开路径
2.  新增图标选择功能
3.  优化ftp、ssh连接弹窗提示
4.  需要删除已经存在的tools/webssh配置文件（否则提示连接失败）
5.  小程序反编译模块新增node环境监测功能
6.  修改site、tools 默认item显示宽度
7.  新增site、tools目录树
8.  修改CyberChef为中文版本的
9.  新增杀软检测目录进程清单

## 工具介绍

默认密码：**EasyTools/EasyTools**，如需修改直接点击状态栏左下角"修改密码"即可。（如果师傅不需要密码，可以自行打包删除即可）

tips：点一下输入框，但是不要输入就会提示密码，然后复制即可

**注：如果以前有使用EasyTools工具箱，需要注意一下内容：**

**1. 删除antivirus_list表，重新打开以更新最新版杀软列表**

![image-20250615232036645](images/image-20250615232036645.png)

工具使用主打一个简洁，双击即可。

![image-20250615230923844](images/image-20250615230923844.png)

### 工具仓库

通过右键支持新增、修改、删除、**打开文件夹位置**（新增）。

优点：

+ 支撑自定义工具路径、不管你是C、D、E盘还是啥、都可以快速定位，无需将文件进行移动
+ 区分GUI程序与终端程序，优化打开体验，避免全屏cmd
+ 如果程序有图标的话，烦请放在`EasyToolsFiles\icon`路径下，程序会自动导入，当然直接使用图片url也是可以的
+ tips: 例如java程序，如果需要使用多个版本，可以直接通过绝对路径进行启动哦、`C:\Java\jdk1.8\bin\java.exe -jar xxxx.jar`

![image-20250615231039735](images/image-20250615231039735.png)

![image-20250615231245199](images/image-20250615231245199.png)

### 网址导航

依旧是右键新增、修改、删除。

当然查询也是可以的

如果程序有图标的话，烦请放在`EasyToolsFiles\icon`路径下，程序会自动导入，当然直接使用图片url也是可以的

![image-20250615231314067](images/image-20250615231314067.png)

### 信息查询

信息查询包括：Google语法、默认密码查询、反弹shell、杀软进程查询

![image-20250615231341493](images/image-20250615231341493.png)

![image-20250615231353619](images/image-20250615231353619.png)

![image-20250615231413334](images/image-20250615231413334.png)

![image-20250615231624337](images/image-20250615231624337.png)

### 信息处理

信息处理包括：Fscan结果解析、蓝队大批量封禁IP处置、OSS资源桶遍、小程序反编译

![image-20250615231738319](images/image-20250615231738319.png)

![image-20250615231750623](images/image-20250615231750623.png)

![image-20250615231801286](images/image-20250615231801286.png)

![image-20250615231814876](images/image-20250615231814876.png)

![image-20250607232038762](images/image-20250607232038762.png)

### 简连助手

新增简单的SSH连接功能、FTP连接功能、Redis连接功能，便于在某些特殊情况下应急使用。

#### SSH

![image-20250508220726029](images/image-20250508220726029.png)

![image-20250508220743169](images/image-20250508220743169.png)

![image-20250508220754790](images/image-20250508220754790.png)

#### FTP

![image-20250508220833535](images/image-20250508220833535.png)

#### Redis

![image-20250508221012303](images/image-20250508221012303.png)

![image-20250508221133274](images/image-20250508221133274.png)

### 编码解码

编码解码直接使用的CyberChef，避免重复造轮子

![image-20250615231922470](images/image-20250615231922470.png)

### 随机生成

随机生成这里提供两种，分别是密码生成，手机号生成（这个主要是在方便测试过程中限制归属地使用）

![image-20250508221244309](images/image-20250508221244309.png)

### 免杀生成（20250515版）

安装教程请见文档：

~~~
https://www.yuque.com/yuqueyonghuoxdahr/aae1ol/tdqgk1gwxns8g6ts?singleDoc# 《EasyTools免杀模块安装教程》
~~~

tips：请详细查看安装文档，可以避免诸多使用bug~~~

推荐编译方式使用garble，有点慢，请耐心等待

![image-20250515235751223](images/image-20250515235751223.png)

选择需要处理的bin文件，拖拽进行处理

选择加载模式、运行模式、加密方式、需要规避的杀软类型、编译方式

目前提供5种运行模式、3种加密方式、2种编译方式

选择完成之后，点击处理bin文件、然后编译生成

针对360Qvm报毒，可以点击bypassQvm一键对生成的文件进行处理

处理完的文件在EasyToolsFiles/file目录下，可以点击下方按钮一键打开

通过填充随机字符等方式实现增加文件提交，避免传输过程直接报毒

![image-20250508221924598](images/image-20250508221924598.png)

新增文件捆绑功能

![image-20250508221934654](images/image-20250508221934654.png)

### 免杀监测——20250515测试

更新已查杀的加载器

![image-20250515235459842](images/image-20250515235459842.png)

## 程序编译

+ 下载程序

~~~
git clone https://github.com/doki-byte/EasyTools.git
~~~

+ 安装前端依赖

~~~
cd frontend
npm install
~~~

+ 运行程序

~~~
wails dev
~~~

+ 编译程序

~~~
wails build --trimpath -ldflags="-w -s"
~~~

## 参考

本工具参考一下开源项目，感谢师傅的热心开源，谢谢。

后续将逐步更新功能，有好的建议也欢迎师傅提出，感激。

~~~html
https://github.com/xbuntu/godesk
https://github.com/0dayCTF/reverse-shell-generator
https://github.com/gchq/CyberChef
https://github.com/ZororoZ/fscanOutput
https://github.com/o8oo8o/WebSSH
https://github.com/broken5/unveilr
~~~