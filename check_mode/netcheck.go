package check_mode

import (
	"crypto/tls"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 格式化时间输出
var currentTime = time.Now()
var formattedTime = currentTime.Format("2006-01-02-15-04-05.000")
var succ = "suc-" + formattedTime + ".log"
var erro = "err-" + formattedTime + ".log"

// 根据运行时间创建两个日志文件
var suclog, _ = os.Create(succ)
var errlog, _ = os.Create(erro)

// 创建两个日志记录器,开头设置prefix
var successlog = log.New(suclog, "SUCESS: ", log.LstdFlags)
var errorlog = log.New(errlog, "ERROR: ", log.LstdFlags)

// 定义waitgroup
var wg sync.WaitGroup
var wg2 sync.WaitGroup

// 把yaml分割成多个结构体数据
type PortCheckYaml struct {
	Name  string   `mapstructure:"name"`
	IPs   []string `mapstructure:"ip"`
	Ports []int    `mapstructure:"port"`
	Url   []string `mapstructure:"url"`
}

// 用于viper解析结构体数据,viper最外层为map,遍历拿到yaml的每个最外层key,根据key名构建切片,然后根据切片内结构构建上面结构体,最后解析到结构体.
type config struct {
	//Viper使用mapstructure包来完成这个映射，所以你需要在结构体字段上添加mapstructure标签来指导映射过程。
	//mapstructure:"servers" 是结构体标签，用于指定配置文件中对应的键。在Viper中，通过这个标签可以告诉它在解析JSON或YAML等格式的配置文件时，将值与字段进行映射。
	Servers       []PortCheckYaml `mapstructure:"servers"`
	Https         []PortCheckYaml `mapstructure:"https"`
	Http          []PortCheckYaml `mapstructure:"http"`
	O_F_O_servers []PortCheckYaml `mapstructure:"o-f-o-servers"`
}

// 加载参数
func Loadyaml(suctun chan string, errtun chan string) string {
	wg2.Add(1)
	go logger(errtun, suctun)
	defer suclog.Close()
	defer errlog.Close()

	configdir := "./otherconfig"
	dir, err := os.ReadDir(configdir)
	if err != nil {
		log.Println("目录不存在")
		return ""
	}
	//dns检查单独进行
	DnsCheck("ddm.sinotruk.com", suctun, errtun)

	for _, file := range dir {
		//是否以yaml结尾
		if !strings.HasSuffix(file.Name(), "yaml") {
			return ""
		}
		//用filepath.Join连接起来，你会得到一个指向otherconfig目录下file.Name()所指定文件的完整路径
		pathjoin := filepath.Join(configdir, file.Name())
		//遍历所有yaml判断
		viper.SetConfigFile(pathjoin)
		if err := viper.ReadInConfig(); err != nil {
			log.Println("yaml文件格式错误")
		}
		var config = config{}
		err := viper.Unmarshal(&config)
		if err != nil {
			log.Println("viper解析失败")
			return ""
		}
		keys := viper.AllSettings()
		for key, _ := range keys {
			Check(key, config, suctun, errtun)
		}
	}
	wg.Wait()
	wg2.Add(1)
	go closetun(suctun, errtun)
	wg2.Wait()
	return succ
}

func Check(key string, config config, suctun chan string, errtun chan string) {
	switch key {
	case "o-f-o-servers":
		OfoPortCheck(config, suctun, errtun)
	case "servers":
		PortCheck(config, suctun, errtun)
	case "http":
		HttpCheck(config, suctun, errtun)
	case "https":
		HttpsCheck(config, suctun, errtun)
	}
}

func closetun(suctun chan string, errtun chan string) {
	for {
		//判断队列是否为空
		if len(suctun) == 0 && len(errtun) == 0 {
			close(suctun)
			close(errtun)
			wg2.Done()
			return
		} else {
			fmt.Printf("suctun中剩余数据量为: %v,errtun中剩余数据量为: %v", len(suctun), len(errtun))
		}
	}
}

func logger(errtun chan string, suctun chan string) {
	checkslice := make([]string, 60)
	for {
		if checkslice[0] == "suctun已关闭" && checkslice[1] == "errtun已关闭" {
			fmt.Println("通道关闭")
			wg2.Done()
			return
		}
		select {
		case s, ok := <-suctun:
			if !ok {
				checkslice[0] = "suctun已关闭"
				continue
			} else {
				successlog.Println(s)
			}
		case s2, ok2 := <-errtun:
			if !ok2 {
				checkslice[1] = "errtun已关闭"
				continue
			} else {
				errorlog.Println(s2)
			}
		}
	}

}

// dns探测
func DnsCheck(www string, suctun chan string, errtun chan string) bool {
	start := time.Now()
	_, err := net.LookupHost(www)
	if err != nil {
		n := fmt.Sprintf("DNS探测失败")
		errtun <- n
		return false
	}
	duration := time.Since(start)
	y := fmt.Sprintf("DNS探测成功,花费时间为: %v", duration)
	suctun <- y
	return true
}

// port探测
func OfoPortCheck(config config, suctun chan string, errtun chan string) {
	//异常捕获处理
	defer func() {
		if a := recover(); a != nil {
			log.Println("异常退出")
		}
	}()
	for _, ips := range config.O_F_O_servers {
		for _, ip := range ips.IPs {
			ip := ip
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				con, err := net.DialTimeout("tcp", ip, time.Duration(1)*time.Second)
				if err != nil {
					n := fmt.Sprintf("端口未开放或无法访问-%s,%s", ip, ips.Name)
					errtun <- n
				} else {
					y := fmt.Sprintf("端口开放-%s,%s", ip, ips.Name)
					suctun <- y
					//最后关掉端口
					con.Close()
				}
			}(ip)
		}
	}
}

func PortCheck(config config, suctun chan string, errtun chan string) {
	for _, server := range config.Servers {
		ips := server.IPs
		ports := server.Ports
		for _, ip := range ips {
			for _, port := range ports {
				wg.Add(1)
				go func(port int, ip string) {
					defer wg.Done()
					con, err := net.DialTimeout("tcp", ip+":"+strconv.Itoa(port), time.Duration(1)*time.Second)
					if err != nil {
						n := fmt.Sprintf("端口未开放或无法访问-%s,%s", ip+":"+strconv.Itoa(port), server.Name)
						errtun <- n
					} else {
						y := fmt.Sprintf("端口开放-%s,%s", ip+":"+strconv.Itoa(port), server.Name)
						suctun <- y
						//最后关掉端口
						con.Close()
					}
				}(port, ip)
			}
		}
	}
}

// http探测
func HttpCheck(config config, suctun chan string, errtun chan string) {
	//异常捕获处理
	defer func() {
		if a := recover(); a != nil {
			log.Println("异常退出")
		}
	}()

	for _, netserver := range config.Http {
		for _, url := range netserver.Url {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				if resp, err := http.Get(url); err != nil {
					n := fmt.Sprintf("url为 %v, err为 %s, 模块为 %s", url, err, netserver.Name)
					errtun <- n
				} else {
					defer resp.Body.Close()
					if body, err := io.ReadAll(resp.Body); err != nil {
						n := fmt.Sprintf("url为 %v, err为 %s,模块为 %s, 读取失败", url, err, netserver.Name)
						errtun <- n
					} else {
						if resp.StatusCode == 200 {
							y := fmt.Sprintf("url为 %v, 响应码为 %v, 模块为 %v", url, resp.StatusCode, netserver.Name)
							suctun <- y
						} else if resp.StatusCode != 200 {
							n := fmt.Sprintf("url为 %v, 响应码为 %v, 包体为 %v,模块为 %v", url, resp.StatusCode, string(body), netserver.Name)
							errtun <- n
						}
					}
				}
			}(url)
		}
	}
}

func HttpsCheck(config config, suctun chan string, errtun chan string) {
	//异常捕获处理
	defer func() {
		if a := recover(); a != nil {
			log.Println("异常退出")
		}
	}()
	// 创建一个 http.Client，其 Transport 配置为跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	for _, httpsserver := range config.Https {
		for _, url := range httpsserver.Url {
			defer wg.Add(1)
			go func(url string) {
				defer wg.Done()
				// 发起请求
				resp, err := client.Get(url)
				if err != nil {
					n := fmt.Sprintf("https请求失败，url为 %v, err为 %s", url, err)
					errtun <- n
				} else {
					defer resp.Body.Close()
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						n := fmt.Sprintf("https包体读取失败", err, url)
						errtun <- n
					}
					if resp.StatusCode == 200 {
						y := fmt.Sprintf("url为 %v, 响应码为 %v,模块为 %v", url, resp.StatusCode, httpsserver.Name)
						suctun <- y
					} else if resp.StatusCode != 200 {
						n := fmt.Sprintf("url为 %v, 响应码为 %v, 包体为 %v, 模块为 %v", url, resp.StatusCode, string(body), httpsserver.Name)
						errtun <- n
					}
				}
			}(url)
		}
	}
}
