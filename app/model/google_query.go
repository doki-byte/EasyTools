package model

import (
	"log"

	"gorm.io/gorm"
)

const TableNameGoogle = "google_query"

type Google_query struct {
	ID          int    `json:"id"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Command     string `json:"command"`
}

func (*Google_query) TableName() string {
	return TableNameGoogle
}

func (s *Google_query) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&Google_query{}).Count(&count).Error; err != nil {
		log.Fatalf("Error counting records: %v", err)
	}

	if count == 0 {
		defaultSites := []Google_query{
			{Category: "获取二级域名", Description: "Google Command 1", Command: "site:${googleDomain}"},
			{Category: "获取管理入口地址", Description: "关键词匹配", Command: "site:${googleDomain} intext:管理|后台|登陆|用户名|密码|验证码|系统|帐号|admin|login|sys|managetem|password|username|manage|manager|admin_login|system"},
			{Category: "获取管理入口地址", Description: "URL匹配", Command: "site:${googleDomain} inurl:\"login\"|inurl:\"logon\"|inurl:\"admin\"|inurl:\"manage\"|inurl:\"manager\"|inurl:\"member\"|inurl:\"admin_login\"|inurl:\"ad_login\"|inurl:\"ad_manage\"|inurl:\"houtai\"|inurl:\"guanli\"|inurl:\"htdl\"|inurl:\"htgl\"|inurl:\"members\"|inurl:\"system\"|inurl:\"login_admin\"|inurl:\"system\"|inurl:\"user\"|inurl:\"main\"|inurl:\"cms\""},
			{Category: "学校专项", Description: "Student Command 1", Command: "site:${googleDomain} filetype:docx | xls 学号"},
			{Category: "学校专项", Description: "Student Command 2", Command: "site:${googleDomain} intitle:账号|密码|工号|学号|身份证|忘记密码|优秀员工|三好学生|手机号|四六级|贫困申请|成绩单|优秀学员|评奖评优"},
			{Category: "学校专项", Description: "Student Command 3", Command: "site:${googleDomain} intext:*@${googleDomain}"},
			{Category: "学校专项", Description: "Student Command 4", Command: "site:${googleDomain} filetype:xls QQ site:cn"},
			{Category: "上传漏洞寻找", Description: "Upload Command 1", Command: "site:${googleDomain} inurl:file | upload上传"},
			{Category: "注入页面寻找", Description: "Sql Command 1", Command: "site:${googleDomain} inurl:php?id="},
			{Category: "注入页面寻找", Description: "Sql Command 2", Command: "site:${googleDomain} inurl:file | load | editor | Files"},
			{Category: "注入页面寻找", Description: "Sql Command 3", Command: "site:${googleDomain} inurl:aspx | jsp | php | asp"},
			{Category: "存在的数据库", Description: "Db Command 1", Command: "site:${googleDomain} filetype: mdb | asp | #"},
			{Category: "敏感信息", Description: "办公文件", Command: "site:${googleDomain} filetype:pdf | filetype:doc | filetype:xls | filetype:docx | filetype:xlsx | filetype:ppt | filetype:pptx | filetype:csv | filetype:odt | filetype:rtf | filetype:sxw"},
			{Category: "敏感信息", Description: "配置文件", Command: "site:${googleDomain} ext: .xml | .conf | .cnf | .reg | .inf | .rdp | .cfg | .txt | .ora | .ini"},
			{Category: "敏感信息", Description: "数据库文件", Command: "site:${googleDomain} ext:.sql | .dbf | .mdb | .db | .log"},
			{Category: "敏感信息", Description: "备份文件", Command: "site:${googleDomain} ext: .bkf | .bkp | .old | .backup | .bak | .swp | .rar | .txt | .zip | .7z | .sql | .tar.gz | .tgz | .tar"},
			{Category: "敏感信息", Description: "手册文件", Command: "site:${googleDomain} intext:\"手册\""},
			{Category: "目录遍历", Description: "Menu Command 1", Command: "site:${googleDomain} index of /admin"},
			{Category: "目录遍历", Description: "Menu Command 2", Command: "site:${googleDomain} index of /upfiles"},
			{Category: "目录遍历", Description: "Menu Command 3", Command: "site:${googleDomain} index of /fckeditor/editor/"},
			{Category: "目录遍历", Description: "Menu Command 4", Command: "site:${googleDomain} index of /admin/uploadfile"},
			{Category: "目录遍历", Description: "Menu Command 5", Command: "site:${googleDomain} index of /admin/file"},
			{Category: "目录遍历", Description: "Menu Command 6", Command: "site:${googleDomain} index of /system/file"},
			{Category: "目录遍历", Description: "Menu Command 7", Command: "site:${googleDomain} index of /phpmyadmin"},
			{Category: "目录遍历", Description: "Menu Command 8", Command: "site:${googleDomain} index of /web/backup/"},
			{Category: "目录遍历", Description: "Menu Command 9", Command: "site:${googleDomain} inurl:/phpmyadmin/index.php"},
			{Category: "目录遍历", Description: "Menu Command 10", Command: "site:${googleDomain} inurl:/phpinfo.php"},
			{Category: "目录遍历", Description: "Menu Command 11", Command: "site:${googleDomain} filetype:log \"PHP Parse error\"| \"PHP Warning\""},
			{Category: "目录遍历", Description: "Menu Command 12", Command: "site:${googleDomain} \"id=\" & intext:\"Warning: mysql_fetch_array()\""},
			{Category: "目录遍历", Description: "Menu Command 13", Command: "site:${googleDomain} \"id=\" & intext:\"Warning: array_merge()\""},
			{Category: "目录遍历", Description: "Menu Command 14", Command: "site:${googleDomain} \"id=\" & intext:\"Warning: getimagesize()\""},
			{Category: "目录遍历", Description: "Menu Command 15", Command: "site:${googleDomain} \"id=\" & intext:\"Warning: mysql_fetch_assoc()\""},
			{Category: "目录遍历", Description: "Menu Command 16", Command: "site:${googleDomain} \"id=\" & intext:\"Warning: mysql_result()\""},
			{Category: "目录遍历", Description: "Menu Command 17", Command: "site:${googleDomain} \"Index of /\""},
			{Category: "目录遍历", Description: "Menu Command 18", Command: "site:${googleDomain} \"Index of /\" +passwd"},
			{Category: "目录遍历", Description: "Menu Command 19", Command: "site:${googleDomain} \"Index of /\" +password.txt"},
			{Category: "目录遍历", Description: "Menu Command 20", Command: "site:${googleDomain} \"Index of /\" +.htaccess"},
			{Category: "目录遍历", Description: "Menu Command 21", Command: "site:${googleDomain} \"Index of /root\""},
			{Category: "目录遍历", Description: "Menu Command 22", Command: "site:${googleDomain} \"Index of /logs\""},
			{Category: "目录遍历", Description: "Menu Command 23", Command: "site:${googleDomain} intitle:\"index of\" etc"},
			{Category: "目录遍历", Description: "Menu Command 24", Command: "site:${googleDomain} intitle:\"index of\" admin"},
			{Category: "目录遍历", Description: "Menu Command 25", Command: "site:${googleDomain} intitle:\"Index of\" .bash_history"},
			{Category: "目录遍历", Description: "Menu Command 26", Command: "site:${googleDomain} intitle:\"index of\" pwd.db"},
			{Category: "目录遍历", Description: "Menu Command 27", Command: "site:${googleDomain} intitle:\"index of\" data"},
			{Category: "目录遍历", Description: "Menu Command 28", Command: "site:${googleDomain} intitle:index.of filetype:log"},
			{Category: "账号密码", Description: "Pass Command 1", Command: "site:${googleDomain} \"index of/\" \"ws_ftp.ini\" \"parent directory\""},
			{Category: "账号密码", Description: "Pass Command 2", Command: "site:${googleDomain} \"your password is\" filetype:log"},
			{Category: "账号密码", Description: "Pass Command 3", Command: "site:${googleDomain} filetype:ini inurl:\"serv-u.ini\""},
			{Category: "账号密码", Description: "Pass Command 4", Command: "site:${googleDomain} filetype:ini inurl:flashFXP.ini"},
			{Category: "账号密码", Description: "Pass Command 5", Command: "site:${googleDomain} filetype:ini ServUDaemon"},
			{Category: "账号密码", Description: "Pass Command 6", Command: "site:${googleDomain} filetype:ini wcx_ftp"},
			{Category: "账号密码", Description: "Pass Command 7", Command: "site:${googleDomain} filetype:ini ws_ftp pwd"},
			{Category: "账号密码", Description: "Pass Command 8", Command: "site:${googleDomain} filetype:sql inurl:backup inurl:wp-content"},
			{Category: "VPN", Description: "Vpn Command 1", Command: "site:${googleDomain} inurl:/sslvpn"},
		}
		if err := db.Create(&defaultSites).Error; err != nil {
			log.Fatalf("Error inserting default data: %v", err)
		}
	}
}
