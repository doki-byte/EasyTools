package restmate

import (
	"os"
	"strings"

	"github.com/goccy/go-json"
	"github.com/matoous/go-nanoid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *RestMate) ImportCollection() (resp JSResp) {
	s, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择文件",
	})
	if err != nil || s == "" {
		resp.Msg = "错误！无法选择文件"
		return
	}
	j := strings.HasSuffix(s, ".json")
	if !j {
		resp.Msg = "错误！所选文件不是json"
		return
	}
	f, err := os.ReadFile(s)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	type checkinfo struct {
		Info struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		} `json:"info"`
	}
	var checkin checkinfo
	err = json.Unmarshal(f, &checkin)
	if err != nil {
		resp.Msg = "错误！集合文件无效"
		return
	}
	var pmSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	var rmSchema = "https://schema.restmate.com/json/collection/collection.json"

	restFile, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	var c []Collection
	newcols := false
	err = json.Unmarshal(restFile, &c)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	if checkin.Info.Schema == pmSchema {
		var pm PMCollection
		err = json.Unmarshal(f, &pm)
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		if pm.Info.Name == "" {
			resp.Msg = "错误！集合文件无效"
			return
		}
		var col Collection
		col.Name = pm.Info.Name
		col_id, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		col.ID = col_id
		PMRecursion(&pm.Item, &col.Requests, col_id)
		c = append(c, col)
		newcols = true
	} else if checkin.Info.Schema == rmSchema {
		var expCol ExportCollection
		err = json.Unmarshal(f, &expCol)
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		if expCol.Collection.ID == "" || expCol.Collection.Name == "" {
			resp.Msg = "错误！读取文件失败"
			return
		}
		col_id, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		expCol.Collection.ID = col_id
		//new IDs for requests
		for i := range expCol.Collection.Requests {
			if expCol.Collection.Requests[i].ID == "" || expCol.Collection.Requests[i].Name == "" {
				continue
			}
			req_id, err := gonanoid.New()
			if err != nil {
				continue
			}
			expCol.Collection.Requests[i].ID = req_id
			expCol.Collection.Requests[i].Method = validateMethod(expCol.Collection.Requests[i].Method)
			expCol.Collection.Requests[i].CollId = col_id
		}
		c = append(c, expCol.Collection)
		newcols = true
	} else {
		resp.Msg = "错误！集合文件无效"
		return
	}
	if newcols {
		b, err := json.Marshal(c)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		err = os.WriteFile(a.db, b, 0644)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		collRspSlice := makeCollRsp(&c)
		resp.Success = true
		resp.Msg = "集合导入成功"
		resp.Data = collRspSlice
		return
	}
	resp.Msg = "错误！无法导入集合"
	return
}
func PMRecursion(items *[]Item, reqs *[]Request, coll_id string) {
	for i := range *items {
		itm := (*items)[i]
		if len(itm.Item) > 0 {
			PMRecursion(&itm.Item, reqs, coll_id)
			continue
		}
		if itm.Request.Method == "" {
			continue
		}
		var rq Request
		req_id, err := gonanoid.New()
		if err != nil {
			continue
		}
		rq.ID = req_id
		rq.Name = itm.Name
		rq.Method = validateMethod(itm.Request.Method)
		rq.Url = itm.Request.URL.Raw
		rq.CollId = coll_id
		for h := range itm.Request.Header {
			hl := itm.Request.Header[h]
			if hl.Key == "" {
				continue
			}
			hid, err := gonanoid.New()
			if err != nil {
				continue
			}
			var myhead KeyValue
			myhead.ID = hid
			myhead.Active = true
			myhead.Key = hl.Key
			myhead.Value = hl.Value
			rq.Headers = append(rq.Headers, myhead)
		}
		if itm.Request.Body.Mode == "formdata" {
			rq.Body.BodyType = "formdata"
			for f := range itm.Request.Body.FormData {
				fdl := itm.Request.Body.FormData[f]
				if fdl.Type != "text" {
					continue
				}
				var formDataSet FormData
				nid, err := gonanoid.New()
				if err != nil {
					continue
				}
				formDataSet.ID = nid
				formDataSet.Key = fdl.Key
				formDataSet.Type = "text"
				formDataSet.Value = fdl.Value
				formDataSet.Active = true
				rq.Body.FormData = append(rq.Body.FormData, formDataSet)
			}
		} else if itm.Request.Body.Mode == "raw" {
			rq.Body.BodyType = "json"
		} else {
			rq.Body.BodyType = "none"
		}
		rq.Body.BodyRaw = itm.Request.Body.Raw
		*reqs = append(*reqs, rq)
	}
}
func validateMethod(str string) string {
	str = strings.ToLower(str)
	switch str {
	case "get", "post", "put", "delete":
		return str
	default:
		return "get"
	}
}
