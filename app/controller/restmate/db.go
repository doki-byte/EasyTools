package restmate

import (
	"EasyTools/app/controller/system"
	"path/filepath"
)

func (a *RestMate) InitFile(fileName, t string) {
	folderName := "restmate"
	folderPath := filepath.Join(system.GetAppBaseDir(), "config", folderName)
	filePath := filepath.Join(folderPath, fileName)
	if t == "db" {
		a.db = filePath
	}
	if t == "env" {
		a.env = filePath
	}
	if t == "jar" {
		a.jarFile = filePath
	}
}
