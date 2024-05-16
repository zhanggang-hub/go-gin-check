package check_mode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type RebootID struct {
	MediaID string `json:"media_id"`
}

var mediaids RebootID
var data string

func ErrLog() (string, error) {
	errlog, err2 := os.ReadFile(erro)
	if err2 != nil {
		return "err.log文件不存在", err2
	}
	if len(errlog) == 0 {
		return "err.log文件内容为空", nil
	}
	//输出文本的引号替换
	string := strings.Replace(string(errlog), "\"", " ", -1)
	//string = strings.Replace(string, "\n", "||", -1)
	return string, nil
}

func SendMessage() (string, error) {
	errlog, err2 := os.ReadFile(erro)
	if err2 != nil {
		return "err.log文件不存在", err2
	}
	fmt.Println(errlog)
	if len(errlog) == 0 {
		return "err.log文件内容为空", nil
	}
	//输出文本的引号替换
	string := strings.Replace(string(errlog), "\"", " ", -1)
	//string = strings.Replace(string, "\n", "||", -1)
	//大于2048字节用文本输出
	if len(string) > 2048 {
		Postfile(erro)
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "mentioned_list":["@all"]
    }
}
`)
	} else if (strings.Contains(string, "yundao") || strings.Contains(string, "yunzhi")) && (strings.Contains(string, "yunyan") || strings.Contains(string, "yunyi")) && (strings.Contains(string, "yunqiao") || strings.Contains(string, "menhu")) {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["@all"]
    }
}
`, string)
	} else if (strings.Contains(string, "yundao") || strings.Contains(string, "yunzhi")) && (strings.Contains(string, "yunqiao") || strings.Contains(string, "menhu")) {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["L","Alone"]
    }
}
`, string)
	} else if (strings.Contains(string, "yundao") || strings.Contains(string, "yunzhi")) && (strings.Contains(string, "yunyi") || strings.Contains(string, "yunyan")) {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["L","yuanhen","cc6c6102174b3050bc3397c724f00f63"]
    }
}
`, string)
	} else if (strings.Contains(string, "yunyi") || strings.Contains(string, "yunyan")) && (strings.Contains(string, "yunqiao") || strings.Contains(string, "menhu")) {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["yuanhen","cc6c6102174b3050bc3397c724f00f63","Alone"]
    }
}
`, string)
	} else if strings.Contains(string, "yundao") || strings.Contains(string, "yunzhi") {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["L"]
    }
}
`, string)
	} else if strings.Contains(string, "yunyi") || strings.Contains(string, "yunyan") {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["yuanhen","cc6c6102174b3050bc3397c724f00f63"]
    }
}
`, string)
	} else if strings.Contains(string, "yunqiao") || strings.Contains(string, "menhu") {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["Alone"]
    }
}
`, string)
	} else if strings.Contains(string, "DNS近五次探测内有失败") {
		data = fmt.Sprintf(`{
    "msgtype": "text",
    "text": {
        "content": "%s",
        "mentioned_list":["yuanhen"]
    }
}
`, string)
	}

	client := &http.Client{}

	var data = strings.NewReader(data)
	req, err := http.NewRequest("POST", "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=bd0e673a-f064-428b-be51-9c883b35706b", data)
	if err != nil {
		return "构造POST请求失败,请检查网络是否有问题", err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "POST请求失败,请检查网络是否有问题", err
	}
	defer resp.Body.Close()
	msg := fmt.Sprintf("发送消息的响应code为 %v", resp.StatusCode)
	return msg, nil
}

func Postfile(path string) (string, error) {
	//请求微信接口，上传文件
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	fw, err := writer.CreateFormFile("filename", filepath.Base(path))
	if err != nil {
		return "表单数据标题创建失败", err
	}
	fd, err := os.Open(path)
	if err != nil {
		return "打开文件失败,请检查路径是否正确", err
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		return "拷贝文件失败", err
	}

	writer.Close()

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=bd0e673a-f064-428b-be51-9c883b35706b&type=file", form)
	if err != nil {
		return "上传文件-构造POST请求失败,请检查网络是否有问题", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return "上传文件-POST请求失败,请检查网络是否有问题", err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return "上传文件-读取body失败", err
	}
	err = json.Unmarshal(bodyText, &mediaids)
	if err != nil {
		return "上传文件-json解析失败", err
	}
	//拿结构体切片
	id := mediaids.MediaID

	//调微信接口发送
	client2 := &http.Client{}
	//转下字符串，把id传进去
	jsonstr := fmt.Sprintf(`{
    "msgtype": "file",
    "file": {
         "media_id": "%s"
    }
}`, id)
	//构建header
	var data = strings.NewReader(jsonstr)
	req2, err := http.NewRequest("POST", "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=bd0e673a-f064-428b-be51-9c883b35706b", data)
	if err != nil {
		return "发送文件-构造POST请求失败,请检查网络是否有问题", err
	}
	req2.Header.Set("content-type", "application/json")
	resp2, err := client2.Do(req2)
	if err != nil {
		return "发送文件-POST请求失败,请检查网络是否有问题", err
	}
	defer resp2.Body.Close()
	msg := fmt.Sprintf("发送文件的返回值为 %v", resp2.StatusCode)
	return msg, nil
}
