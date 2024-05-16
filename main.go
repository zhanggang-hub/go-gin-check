package main

import (
	"github.com/gin-gonic/gin"
	"go-gin/check_mode"
	"sync"
)

//0、补充接口
//1、get返回一个json，解析json内容
//2、向企业微信发送错误信息，超过2048字节后发送错误日志
//3、上传文件到企业微信

// 定义waitgroup
var wg sync.WaitGroup

type Response struct {
	Code  int    `json:"code"`
	Data  string `json:"data"`
	Error error  `json:"error"`
	Msg   string `json:"msg"`
}

var suctun = make(chan string, 60)
var errtun = make(chan string, 60)

// 只返回巡检结果
func _geterr(c *gin.Context) {
	//巡检程序
	check_mode.Loadyaml(suctun, errtun)
	if errlog, err := check_mode.ErrLog(); err != nil {
		c.JSON(500, Response{1, errlog, err, "执行错误"})
	} else {
		c.JSON(200, Response{0, errlog, nil, "执行成功"})
	}

}

// 发送信息接口message | err.log
func _sendmessage(c *gin.Context) {
	check_mode.Loadyaml(suctun, errtun)
	//发送错误信息，超过2048字节后发送错误日志
	if msg, err := check_mode.SendMessage(); err != nil {
		c.JSON(500, Response{1, msg, err, "请检查网络和url是否正确"})
	} else {
		c.JSON(200, Response{0, msg, nil, "请检查企业微信信息发送情况"})
	}
}

// 发送文件接口suc.log
func _sendfile(c *gin.Context) {
	check_mode.Loadyaml(suctun, errtun)
	succ := check_mode.Successlog()
	//发送错误信息，超过2048字节后发送错误日志
	if msg, err := check_mode.Postfile(succ); err != nil {
		c.JSON(500, Response{1, msg, err, "请检查网络和url是否正确"})
	} else {
		c.JSON(200, Response{0, msg, nil, "请检查企业微信信息发送情况"})
	}

}

func main() {
	router := gin.Default()
	router.GET("/getall", _geterr)
	router.GET("/sendmsg", _sendmessage)
	router.GET("/sendfile", _sendfile)
	router.Run(":8088")
}
