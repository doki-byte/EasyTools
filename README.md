## 开发说明

在日常学习过程中，面对各种各样的测试任务，需要使用到多种工具助力，可是工具一多管理起来就不是很方便。

本着不重复造轮子的原则，本工具目前糅合了**工具导航**、**网址导航、简练助手、Redis连接、SSH连接、FTP连接、信息查询、信息处理、编码解码**、**随机生成**、**免杀生成**、**OSS资源桶信息提取**、**小程序反编译**、**JWT秘钥爆破、地图接口测试**等功能

针对能直接使用的功能，通过嵌入直接调用，在此感谢各位师傅的辛苦开源。

## 更新说明

### v1.8.1

感谢师傅们提供的修改建议，目前已更新一下功能：

1.  新增Note备忘录功能;
2.  优化工具、网址搜索功能;
3.  新增jwt爆破功能;
4.  新增地图接口调用功能;
5.  优化redis连接及显示功能;

## 工具介绍

默认密码：**EasyTools/EasyTools**，如需修改直接点击状态栏左下角"修改密码"即可。（如果师傅不需要密码，可以自行打包删除即可）

tips：点一下输入框，但是不要输入就会提示密码，然后复制即可

**注：如果以前有使用EasyTools工具箱，需要注意一下内容：**

**1. 删除antivirus_list表，重新打开以更新最新版杀软列表**

![image-20250706195046767](images/image-20250706195046767.png)

工具使用主打一个简洁，双击即可。

![image-20250706195924284](images/image-20250706195924284.png)

### 工具仓库

通过右键支持新增、修改、删除、**打开文件夹位置**。

优点：

+ 支撑自定义工具路径、不管你是C、D、E盘还是啥、都可以快速定位，无需将文件进行移动
+ 区分GUI程序与终端程序，优化打开体验，避免全屏cmd
+ 如果程序有图标的话，烦请放在`EasyToolsFiles\icon`路径下，程序会自动导入，当然直接使用图片url也是可以的
+ tips: 例如java程序，如果需要使用多个版本，可以直接通过绝对路径进行启动哦、`C:\Java\jdk1.8\bin\java.exe -jar xxxx.jar`

![image-20250706200000413](images/image-20250706200000413.png)

![image-20250706200016560](images/image-20250706200016560.png)

### 网址导航

依旧是右键新增、修改、删除。

当然查询也是可以的

如果程序有图标的话，烦请放在`EasyToolsFiles\icon`路径下，程序会自动导入，当然直接使用图片url也是可以的

![image-20250706200032136](images/image-20250706200032136.png)

### 信息查询

信息查询包括：Google语法、默认密码查询、反弹shell、杀软进程查询、地图测试

![image-20250706200052129](images/image-20250706200052129.png)

![image-20250706200102965](images/image-20250706200102965.png)

![image-20250706200113759](images/image-20250706200113759.png)

![image-20250706200302040](images/image-20250706200302040.png)

![image-20250706200333026](images/image-20250706200333026.png)

### 信息处理

信息处理包括：Fscan结果解析、蓝队大批量封禁IP处置、OSS资源桶遍、小程序反编译、jwt秘钥破解

![image-20250706200341212](images/image-20250706200341212.png)

![image-20250706200349106](images/image-20250706200349106.png)

![image-20250706200358660](images/image-20250706200358660.png)

![image-20250706200406727](images/image-20250706200406727.png)

![image-20250706200419009](images/image-20250706200419009.png)

### 简连助手

新增简单的SSH连接功能、FTP连接功能、Redis连接功能，便于在某些特殊情况下应急使用。

#### SSH

![image-20250706200458613](images/image-20250706200458613.png)

![image-20250706200555907](images/image-20250706200555907.png)![image-20250706200605504](images/image-20250706200605504.png)

#### FTP

![image-20250706200811565](images/image-20250706200811565.png)

#### Redis

![image-20250706200748776](images/image-20250706200748776.png)

### 编码解码

编码解码直接使用的CyberChef，避免重复造轮子

![image-20250706200853249](images/image-20250706200853249.png)

### 随机生成

随机生成这里提供两种，分别是密码生成，手机号生成（这个主要是在方便测试过程中限制归属地使用）

![image-20250706200914937](images/image-20250706200914937.png)

### 备忘笔记

备忘笔记主要就是简单实现了一下md的预览与编辑功能，方便咱们在测试的过程中快捷查询需要的命令

![image-20250706201044272](images/image-20250706201044272.png)

### 免杀生成（20250706版）

安装教程请见文档：

~~~
https://www.yuque.com/yuqueyonghuoxdahr/aae1ol/tdqgk1gwxns8g6ts?singleDoc# 《EasyTools免杀模块安装教程》
~~~

tips：请详细查看安装文档，可以避免诸多使用bug~~~

推荐编译方式使用garble，有点慢，请耐心等待

![image-20250706201113320](images/image-20250706201113320.png)

选择需要处理的bin文件，拖拽进行处理

选择加载模式、运行模式、加密方式、需要规避的杀软类型、编译方式

目前提供3种运行模式、3种加密方式、2种编译方式

选择完成之后，点击处理bin文件、然后编译生成

针对360Qvm报毒，可以点击bypassQvm一键对生成的文件进行处理

处理完的文件在EasyToolsFiles/file目录下，可以点击下方按钮一键打开

### 免杀监测——20250706测试

更新已查杀的加载器（某数字不行，只能过某绒、某df）

![image-20250706201311796](images/image-20250706201311796.png)

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