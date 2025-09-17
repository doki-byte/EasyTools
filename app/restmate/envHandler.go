package restmate

import (
	"os"

	"github.com/goccy/go-json"
	"github.com/matoous/go-nanoid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *RestMate) GetEnvs() (resp JSResp) {
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！无法获取Envs"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！无法获取Envs"
		return
	}
	resp.Success = true
	resp.Msg = "Envs 获取成功"
	resp.Data = e
	return
}

func (a *RestMate) DuplicateEnv(id string) (resp JSResp) {
	if id == "" {
		resp.Msg = "错误！无法复制env"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！无法复制env"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！无法复制env"
		return
	}
	var d Env
	for i := range e {
		if e[i].ID == id {
			d = e[i]
			break
		}
	}
	if d.ID != "" {
		nid, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！无法复制env"
			return
		}
		d.ID = nid
		d.Selected = false
		d.Name += " Copy"
		e = append(e, d)
	}
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！无法复制env"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法复制env"
		return
	}
	resp.Success = true
	resp.Msg = "Env 复制成功"
	resp.Data = e
	return
}
func (a *RestMate) SelectEnv(id string) (resp JSResp) {
	if id == "" {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	for i := range e {
		if e[i].ID == id {
			e[i].Selected = true
			continue
		}
		e[i].Selected = false
	}
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	resp.Success = true
	resp.Msg = "Env 重命名成功"
	resp.Data = e
	return
}

func (a *RestMate) RenameEnv(id, name string) (resp JSResp) {
	if name == "" || id == "" {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	for i := range e {
		if e[i].ID == id {
			e[i].Name = name
			break
		}
	}
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！Env 重命名失败"
		return
	}
	resp.Success = true
	resp.Msg = "Env 重命名成功"
	resp.Data = e
	return
}

func (a *RestMate) AddEnv(name string) (resp JSResp) {
	if name == "" {
		resp.Msg = "错误！无法添加env"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！无法添加env"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！无法添加env"
		return
	}
	nid, err := gonanoid.New()
	if err != nil {
		resp.Msg = "错误！无法添加env"
		return
	}
	newEnv := Env{
		ID:       nid,
		Name:     name,
		Variable: map[string]string{},
	}
	e = append(e, newEnv)
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！无法添加env"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法添加env"
		return
	}
	resp.Success = true
	resp.Msg = "Env 保存成功"
	resp.Data = e
	return
}

func (a *RestMate) DeleteEnv(id string) (resp JSResp) {
	if id == "" {
		resp.Msg = "错误！无法删除env"
		return
	}
	str, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "删除集合",
		Message:       "您确定要删除环境吗？此环境中的所有变量都将被永久删除。",
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "No",
	})
	if err != nil {
		resp.Msg = "错误！无法删除env"
		return
	}
	if str != "Yes" && str != "Ok" {
		resp.Msg = "错误！无法删除env"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！无法删除env"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！无法删除env"
		return
	}
	for i := range e {
		if e[i].ID == id {
			e = append(e[:i], e[i+1:]...)
			break
		}
	}
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！无法删除env"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法删除env"
		return
	}
	resp.Success = true
	resp.Msg = "Env 删除成功"
	resp.Data = e
	return
}

func (a *RestMate) AddVar(id, key, value string) (resp JSResp) {
	if id == "" || key == "" || value == "" {
		resp.Msg = "错误！无法添加变量"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！无法添加变量"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！无法添加变量"
		return
	}
	for i := range e {
		if e[i].ID == id {
			if e[i].Variable == nil {
				e[i].Variable = make(map[string]string)
			}
			e[i].Variable[key] = value
			break
		}
	}
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！无法添加变量"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法添加变量"
		return
	}
	resp.Success = true
	resp.Msg = "变量添加成功"
	resp.Data = e
	return
}

func (a *RestMate) DeleteVar(id, key string) (resp JSResp) {
	if id == "" || key == "" {
		resp.Msg = "错误！无法删除变量"
		return
	}
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "错误！无法删除变量"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "错误！无法删除变量"
		return
	}
	for i := range e {
		if e[i].ID == id {
			delete(e[i].Variable, key)
			break
		}
	}
	b, err := json.Marshal(e)
	if err != nil {
		resp.Msg = "错误！无法删除变量"
		return
	}
	err = os.WriteFile(a.env, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法删除变量"
		return
	}
	resp.Success = true
	resp.Msg = "变量删除成功"
	resp.Data = e
	return
}
