/*
此DEMO程序的功能为
1. 返回一个缓存的动听token
	此请求是一个GET，参数为:
	appkey
	refresh
	请参考 function TokenHandler

2. 动听token是从动听服务器获取的，请求地址为：
	GET https://api.euphonyqr.com/api/token/v1
参数为:
   appkey
   seckey
   test:是否是正式服(www.euphonyqr.com)或测试服(sandbox.euphonyqr.com)
   请参考 function RequestBuyfullToken
*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

//HTTPGetMaxSize 最大HTTP BODY
const HTTPGetMaxSize = 10 * 1024

//WXResult 返回给微信buyfullsdk的结果
type WXResult struct {
	Code         int    `json:"code"`
	BuyfullToken string `json:"token"`
}

//BuyfullTokenResult 动听服务器返回的结果
type BuyfullTokenResult struct {
	Message string `json:"msg"`
	Token   string `json:"token"`
}

//BuyfullToken 参考main函数中的解释
type BuyfullToken struct {
	Seckey        string
	Token         string //保存从动听服务器返回的Token值
	WxAppID       string
	CheckWxPrefix string
	CheckWxAPPID  bool
	UseSandbox    bool
}

var (
	buyfullTokens     sync.Map   //保存APPKEY=>TOKEN信息的MAP
	buyfullTokenMutex sync.Mutex //确保同一时间只有一个请求发给动听服务器
	client            http.Client
)

func init() {
	client = http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(5 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*10)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		Timeout: time.Duration(15 * time.Second),
	}
}

//BuyfullTokenHandler 处理buyfullsdk中的buyfullTokenUrl请求
func BuyfullTokenHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			http.Error(rw, err.Error(), 500)
		}
	}()

	if req.Method == "GET" {
		//1 检查是否有注册的APPKEY，否则返回status code 404
		//2 检查微信Referer是否正确，否则返回status code 404
		//3 如果没有APPKEY对应的TOKEN或者TOKEN有可能已经失效则向动听服务器申请一个BuyfullToken
		//4 如果动听服务器没有返回TOKEN，返回status code 404，否则是status code 200,加上token
		//返回格式是JSON
		//{"code":200, "token": "xxxxxxxxxxxxxxxxxx"}

		refer := req.Header.Get("Referer")             //微信小程序的referer
		appkey := req.URL.Query().Get("appkey")        //应该与main中注册的appkey一致
		refreshToken := req.URL.Query().Get("refresh") //是否强制刷新Token
		wxresult := WXResult{}
		tokenInterface, ok := buyfullTokens.Load(appkey)
		if !ok {
			wxresult.Code = 401
		} else {
			buyfullToken := (tokenInterface.(*BuyfullToken))
			if !checkWxAPPID(buyfullToken, refer) {
				//检查微信小程序APPID
				wxresult.Code = 402
			} else {
				if buyfullToken.Token == "" || refreshToken == "true" {
					//向动听服务器请求
					RequestBuyfullToken(appkey, buyfullToken.Seckey, buyfullToken.Token, buyfullToken.UseSandbox)
				}
				if buyfullToken.Token == "" || buyfullToken.Token == "no" {
					//请求返回错误
					wxresult.Code = 404
				} else {
					//请求返回正确
					wxresult.Code = 200
					wxresult.BuyfullToken = buyfullToken.Token
				}
			}

			resp, err := json.Marshal(wxresult)
			if err != nil {
				return
			}
			rw.Header().Set("Content-Type", "application/json;charset=utf-8")
			rw.Write(resp)
		}
	} else {
		rw.WriteHeader(http.StatusForbidden)
	}
}

//RequestBuyfullToken 向动听服务器请求Token
//GET请求, url 是 https://api.euphonyqr.com/api/token/v1
//参数appkey,seckey从动听后台中获得
//useSandbox说明APPKEY是否是sandbox.euphonyqr.com
func RequestBuyfullToken(appkey string, seckey string, oldtoken string, useSandbox bool) (tokenResult *BuyfullTokenResult, err error) {
	buyfullTokenMutex.Lock() //多线程锁，只有一个线程会向动听请求，其它线程会等待请求返回
	defer buyfullTokenMutex.Unlock()

	tokenInterface, ok := buyfullTokens.Load(appkey)
	if !ok {
		return nil, fmt.Errorf("Invalid appkey")
	}
	buyfullToken := (tokenInterface.(*BuyfullToken))
	if buyfullToken.Token != oldtoken {
		return nil, fmt.Errorf("Token changed") //防止多线程下重复请求
	}
	req, err := http.NewRequest("GET", "https://api.euphonyqr.com/api/token/v1", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("nocache", strconv.Itoa(rand.Int())) //防止get方法cache
	q.Add("appkey", appkey)                    //main中设的APPKEY
	q.Add("seckey", seckey)                    //main中设的SECKEY
	if useSandbox {
		q.Add("test", "true") //APPKEY是否是sandbox.euphonyqr.com的
	} else {
		q.Add("test", "false")
	}

	req.URL.RawQuery = q.Encode()

	var resp *http.Response

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	status := resp.StatusCode
	if status != 200 {
		return nil, fmt.Errorf("http network error")
	}
	body, err := ioutil.ReadAll(http.MaxBytesReader(nil, resp.Body, HTTPGetMaxSize))
	if err != nil {
		return nil, err
	}
	var buyfullResult BuyfullTokenResult
	err = json.Unmarshal(body, &buyfullResult)
	if err != nil {
		return nil, err
	}
	//动听服务器返回结果
	switch buyfullResult.Message {
	case "OK": //OK
		{
			buyfullToken.Token = buyfullResult.Token
		}
	default:
		{
			buyfullToken.Token = "no"
		}
	}
	buyfullTokens.Store(appkey, buyfullToken) //保存APPKEY=>TOKEN信息
	return &buyfullResult, nil
}

//检查请求是否来自特定的微信小程序
func checkWxAPPID(buytoken *BuyfullToken, referer string) (valid bool) {
	if !buytoken.CheckWxAPPID {
		return true
	}

	if strings.HasPrefix(referer, buytoken.CheckWxPrefix) {
		return true
	}
	return false
}

func main() {
	/*
		建议为每一个微信小程序单独布署一个微型服务来处理buyfullToken的请求，请确保SECKEY的安全
		以下是环境变量
		PORT_HTTP			监听端口
		APPKEY				在动听后台创建APP后获取
		SECKEY				在动听后台创建APP后获取
		SANDBOX				如果是在www.euphonyqr.com创建的APP，设为false。
							如果是在sandbox.euphonyqr.com创建的APP，设为true。
		WXURL				监听的路径

		WXAPPID				微信小程序APPID，请检查在动听后台创建APP时填写的小程序APPID是否相同
		CHECKWXAPPID		如果为true，会检测referrer确定是从指定的小程序中发出的请求
	*/
	port := os.Getenv("PORT_HTTP")
	if port == "" {
		port = "8080"
	}

	appKey := os.Getenv("APPKEY")
	if appKey == "" {
		appKey = "121e87d73077403eadd9ab4fec2d9973"
	}

	secKey := os.Getenv("SECKEY")
	if secKey == "" {
		secKey = "95319779d4af58272619f612f7fa1156"
	}

	wxAppID := os.Getenv("WXAPPID")
	if wxAppID == "" {
		wxAppID = "wx787a08bcd2904b06"
	}

	useSandbox := true
	useSandboxStr := os.Getenv("SANDBOX")
	if useSandboxStr == "false" {
		useSandbox = false
	}

	wxHandleURL := os.Getenv("WXURL")
	if wxHandleURL == "" {
		wxHandleURL = "/wx/buyfulltoken"
	}

	checkWxAPPID := false
	checkWxAPPIDStr := os.Getenv("CHECKWXAPPID")
	if checkWxAPPIDStr == "true" {
		checkWxAPPID = true
	}

	buyfullDemoToken := BuyfullToken{Seckey: secKey, UseSandbox: useSandbox,
		WxAppID: wxAppID, CheckWxPrefix: "https://servicewechat.com/" + wxAppID + "/",
		CheckWxAPPID: checkWxAPPID /*if you want to check wx appid , set true*/}
	buyfullTokens.Store(appKey, &buyfullDemoToken)

	runtime.GOMAXPROCS(runtime.NumCPU())
	http.HandleFunc(wxHandleURL, BuyfullTokenHandler)
	log.Fatalln(http.ListenAndServe("0.0.0.0:"+port, nil))
}
