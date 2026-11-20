package assistive

import (
	"EasyTools/app/controller/system"
)

// Antivirus 控制器
type Assistive struct {
	system.Base
}

// NewAntivirus 创建新的 Antivirus 控制器
func NewAssistive() *Assistive {
	return &Assistive{}
}
