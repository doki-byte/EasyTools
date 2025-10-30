package model

import (
	"encoding/json"
	"log"

	"gorm.io/gorm"
)

const TableNameWechatConfig = "wechat_config"

type WechatConfig struct {
	ID            int    `gorm:"primarykey" json:"id"`
	AppletPath    string `json:"applet_path" gorm:"column:applet_path"`
	Rules         string `json:"rules" gorm:"column:rules"` // JSON格式存储规则
	AutoDecompile bool   `json:"auto_decompile" gorm:"column:auto_decompile"`
}

func (*WechatConfig) TableName() string {
	return TableNameWechatConfig
}

func (w *WechatConfig) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&WechatConfig{}).Count(&count).Error; err != nil {
		log.Printf("Error counting wechat config: %v", err)
		return
	}

	if count == 0 {
		defaultRules := []string{
			`(?i)[\"']?access[_-]?token[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[^(null)][\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?access[_-]?secret[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?access[_-]?key[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?access[_-]?key[_-]?id[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?access[_-]?key[_-]?secret[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?secret[_-]?access[_-]?key[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?secret[_-]?access[_-]?token[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?:[1-9]\\d{5}(?:(?:18|19|20)\\d{2})(?:(?:0[1-9]|1[0-2]))(?:(?:0[1-9]|[12]\\d|3[01]))\\d{3}[\\dXx]|[1-9]\\d{7}(?:(?:0[1-9]|1[0-2]))(?:(?:0[1-9]|[12]\\d|3[01]))\\d{3})`,
			`(?:\\+?86)?1[3-9]\\d{9}`,
			`(?i)(post|get|delete|put|url)[:\\(]+[^\\S\r\n]*[\"']\\/(?:[\\w-]+\\/)*(?:[\\w-]+)\\/*\\??(?:[\\w=&-]+)?[\"']\\)*`,
			`(?i)http(s)?://(?:[\\w-]+\\/)*(?:[\\w-]+)\\/*\\??(?:[\\w\\=&-]+)?[\"']\\)*`,
			`(?i)[\"']\\/(?:[\\w-]+\\/)*[\\w-]+(.cgi|.php|.action|.jsp|.jspx|.asp|.aspx|.py|.rb|.html|.htm|.tpl|.do|.jsf)?\\/*\\??(?:[\\w=&-]+)?\\)*`,
			`(?i)[\"']?app[_-]?(id|secret)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?.*corp[_-]?(Id|Secret)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?.*qq\\.im\\.(sdkappid|privateKey|identifier)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)(?:[\"']?user(?:name)?[\"']?[^\\S\r\n]*[=:])[^\\S\r\n]*[\"']?[一-龥\\w-]+[\"']?`,
			`(?i)[\"']?pass(?:word)?[\"']?[^\\S\n]*[=:][^\\S\r\n]*[\"']?[@#!$%^&*\\w-]+[\"']?`,
			`(?i)[\"']?(账户|账户名|用户名|账号|测试账户)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[一-龥\\w-]+[\"']?`,
			`(?i)[\"']?(默认口令|默认密码|口令|密码|测试密码)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[@#!$%^&*\\w-]+[\"']?`,
			`(?i)[\"']?jdbc\\.(driver|url|type)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?#jdbc\\.(driver|url|type)[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?session[_-]?key[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?api[_-]?key[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)[\"']?api[_-]?secret[\"']?[^\\S\r\n]*[=:][^\\S\r\n]*[\"']?[\\w-]+[\"']?`,
			`(?i)["'](/[\w-]+(?:/[\w-./?%&=]*?))(?:\.(?:exe|dll|so|dylib|sh|bat|cmd|ps1|py|js|php|asp|aspx|jsp|do|action|go|rb|pl|pm|tcl|swf|jar|war|ear|class|rpm|deb|msi|apk|ipa|app|bin|elf|com|sys|drv|vxd|ocx|cpl|scr|pif|vb|vbs|wsf|hta|msp|msc|msh|msh1|msh2|mshxml|msh1xml|msh2xml|psc1|psc2|ps1xml|ps2xml|psc1xml|psc2xml|scf|lnk|inf|reg|doc|docx|xls|xlsx|ppt|pptx|pdf|rtf|txt|csv|xml|json|html|htm|css|js|ts|jsx|tsx|vue|svelte))?["']`,
			`(?i)(?:http[s]?://)([\w-]+\.)+[\w-]+(/[\w-./?%&=]*?)(?:\.(?:exe|dll|so|dylib|sh|bat|cmd|ps1|py|js|php|asp|aspx|jsp|do|action|go|rb|pl|pm|tcl|swf|jar|war|ear|class|rpm|deb|msi|apk|ipa|app|bin|elf|com|sys|drv|vxd|ocx|cpl|scr|pif|vb|vbs|wsf|hta|msp|msc|msh|msh1|msh2|mshxml|msh1xml|msh2xml|psc1|psc2|ps1xml|ps2xml|psc1xml|psc2xml|scf|lnk|inf|reg|doc|docx|xls|xlsx|ppt|pptx|pdf|rtf|txt|csv|xml|json|html|htm|css|js|ts|jsx|tsx|vue|svelte))?`,
			`(?i)\b(?:post|get|delete|put|url)[:\(]+[^\S\r\n]*["']?\/((?:[\w-]+\/)*(?:[\w-]+))(?:\.(?:exe|dll|so|dylib|sh|bat|cmd|ps1|py|js|php|asp|aspx|jsp|do|action|go|rb|pl|pm|tcl|swf|jar|war|ear|class|rpm|deb|msi|apk|ipa|app|bin|elf|com|sys|drv|vxd|ocx|cpl|scr|pif|vb|vbs|wsf|hta|msp|msc|msh|msh1|msh2|mshxml|msh1xml|msh2xml|psc1|psc2|ps1xml|ps2xml|psc1xml|psc2xml|scf|lnk|inf|reg|doc|docx|xls|xlsx|ppt|pptx|pdf|rtf|txt|csv|xml|json|html|htm|css|js|ts|jsx|tsx|vue|svelte))?\/*\??((?:[\w=&-]+)?)["']?\)*`,
			`(?i)\b(?:post|get|delete|put)\(["']?(http[s]?://)?((?:[\w-]+\.)+[\w-]+)(/[\w-./?%&=]*?)(?:\.(?:exe|dll|so|dylib|sh|bat|cmd|ps1|py|js|php|asp|aspx|jsp|do|action|go|rb|pl|pm|tcl|swf|jar|war|ear|class|rpm|deb|msi|apk|ipa|app|bin|elf|com|sys|drv|vxd|ocx|cpl|scr|pif|vb|vbs|wsf|hta|msp|msc|msh|msh1|msh2|mshxml|msh1xml|msh2xml|psc1|psc2|ps1xml|ps2xml|psc1xml|psc2xml|scf|lnk|inf|reg|doc|docx|xls|xlsx|ppt|pptx|pdf|rtf|txt|csv|xml|json|html|htm|css|js|ts|jsx|tsx|vue|svelte))?["']?\)`,
			`(?i)\b(?:post|get|delete|put)\(["']?(/[\w-]+(?:/[\w-./?%&=]*?))(?:\.(?:exe|dll|so|dylib|sh|bat|cmd|ps1|py|js|php|asp|aspx|jsp|do|action|go|rb|pl|pm|tcl|swf|jar|war|ear|class|rpm|deb|msi|apk|ipa|app|bin|elf|com|sys|drv|vxd|ocx|cpl|scr|pif|vb|vbs|wsf|hta|msp|msc|msh|msh1|msh2|mshxml|msh1xml|msh2xml|psc1|psc2|ps1xml|ps2xml|psc1xml|psc2xml|scf|lnk|inf|reg|doc|docx|xls|xlsx|ppt|pptx|pdf|rtf|txt|csv|xml|json|html|htm|css|js|ts|jsx|tsx|vue|svelte))?["']?\)`,
			`(?i)(?:["']?[\w-]*user(?:name)?["']?[^\S\r\n]*[=:])[^\S\r\n]*('[^']+'|"[^"]+"|[\w-.]+\b)`,
			`(?i)["']?[\w-]*(pass(word)?|email)["']?[^\S\n]*[=:][^\S\r\n]*('[^']+'|"[^"]+"|[\w-.]+\b)`,
			`(?i)["']?(账户|账户名|用户名|账号|测试账户)["']?[^\S\r\n]*[=:][^\S\r\n]*('[^']+'|"[^"]+"|[\w-.]+\b)`,
			`(?i)["']?(默认口令|默认密码|口令|密码|测试密码)["']?[^\S\r\n]*[=:][^\S\r\n]*('[^']+'|"[^"]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*secret[_-]?access[-_]?key["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*api[_-]?(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*secret[_-]?(key|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*access[_-]?(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*session[_-]?(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*secret[_-]?(key|secret|token|0|1|2|3|4|5|6|7|8|9|10|11|12|13|14|15|16|17|18|19|20)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*access[_-]?(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*app[_-]?(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*(secret|api|app|access|session|corp|key|client|auth|bucket|account)[_-]?id["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*corp[_-]?secret["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*key[_-]?(secret|token|sid)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*account[_-]?(key|secret|token|sid)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*git(hub)?[\w-]*(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*client[_-]?(secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*auth[_-]?(key|secret|token)["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*bucket[-_]?(name)?["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']?[\w-.]*endpoint["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)[\["']X-TC-Key["'\]]?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`(?i)["']?.*?.*qq\.im\.(sdkappid|privateKey|identifier)["']?[^\S\r\n]*[=:][^\S\r\n]*('[^'\r\n]+'|"[^"\r\n]+"|[\w-.]+\b)`,
			`\b(?:\+?86)?1[3-9]\d{9}\b`,
			`\b[1-9]\d{5}(?:18|19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]\b`,
			`\b[1-9]\d{5}\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}\b`,
			`(?i)(?:src|href|data-[a-z-]+)\s*=\s*["']([^"'\s>]+\.(?:jpe?g|png|gif|webp|svgz?|mp4|webm|mp3|ogg|pdf)(?:\?[^"'\s>]*)?)["']`,
		}
		rulesJSON, _ := json.Marshal(defaultRules)

		defaultConfig := WechatConfig{
			ID:            1,
			AppletPath:    "",
			Rules:         string(rulesJSON),
			AutoDecompile: false,
		}

		if err := db.Create(&defaultConfig).Error; err != nil {
			log.Printf("Error creating default wechat config: %v", err)
		}
	}
}
