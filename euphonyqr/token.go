package euphonyqr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

//HTTPGetMaxSize 最大HTTP BODY
const HTTPGetMaxSize = 10 * 1024

var (
	tokens     sync.Map   //保存APPKEY=>TOKEN信息的MAP
	tokenMutex sync.Mutex //确保同一时间只有一个请求发给动听服务器
	client     http.Client
)

//DemoTokenResult 返回给DEMO的结果
type DemoTokenResult struct {
	Code  int    `json:"code"`
	Token string `json:"token"`
}

//EuphonyQRTokenResult 动听服务器返回的结果
type EuphonyQRTokenResult struct {
	Message string `json:"msg"`
	Token   string `json:"token"`
}

//EuphonyQRToken 参考main函数中的解释
type EuphonyQRToken struct {
	Seckey        string
	Token         string //保存从动听服务器返回的Token值
	WxAppID       string
	CheckWxPrefix string
	CheckWxAPPID  bool
	UseSandbox    bool
}

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

//TokenHandler 处理sdk中的TokenUrl请求
func TokenHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			http.Error(rw, err.Error(), 500)
		}
	}()

	if req.Method == "GET" {
		//1 检查是否有注册的APPKEY，否则返回status code 404
		//2 检查微信Referer是否正确，否则返回status code 404
		//3 如果没有APPKEY对应的TOKEN或者TOKEN有可能已经失效则向动听服务器申请一个EuphonyQRToken
		//4 如果动听服务器没有返回TOKEN，返回status code 404，否则是status code 200,加上token
		//返回格式是JSON
		//{"code":200, "token": "xxxxxxxxxxxxxxxxxx"}

		refer := req.Header.Get("Referer")             //微信小程序的referer
		appkey := req.URL.Query().Get("appkey")        //应该与main中注册的appkey一致
		refreshToken := req.URL.Query().Get("refresh") //是否强制刷新Token
		wxresult := DemoTokenResult{}
		tokenInterface, ok := tokens.Load(appkey)
		if !ok {
			wxresult.Code = 401
		} else {
			euphonyqrToken := (tokenInterface.(*EuphonyQRToken))
			if !checkWxAPPID(euphonyqrToken, refer) {
				//检查微信小程序APPID
				wxresult.Code = 402
			} else {
				if euphonyqrToken.Token == "" || refreshToken == "true" {
					//向动听服务器请求
					RequestEuphonyqrToken(appkey, euphonyqrToken.Seckey, euphonyqrToken.Token, euphonyqrToken.UseSandbox)
				}
				if euphonyqrToken.Token == "" || euphonyqrToken.Token == "no" {
					//请求返回错误
					wxresult.Code = 404
				} else {
					//请求返回正确
					wxresult.Code = 200
					wxresult.Token = euphonyqrToken.Token
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

//RequestEuphonyQRToken 向动听服务器请求Token
//GET请求, url 是 https://api.euphonyqr.com/api/token/v1
//参数appkey,seckey从动听后台中获得
//useSandbox说明APPKEY是否是sandbox.euphonyqr.com
func RequestEuphonyqrToken(appkey string, seckey string, oldtoken string, useSandbox bool) (tokenResult *EuphonyQRTokenResult, err error) {
	tokenMutex.Lock() //多线程锁，只有一个线程会向动听请求，其它线程会等待请求返回
	defer tokenMutex.Unlock()

	tokenInterface, ok := tokens.Load(appkey)
	if !ok {
		return nil, fmt.Errorf("Invalid appkey")
	}
	euphonyqrToken := (tokenInterface.(*EuphonyQRToken))
	if euphonyqrToken.Token != oldtoken {
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
	var euphonyqrResult EuphonyQRTokenResult
	err = json.Unmarshal(body, &euphonyqrResult)
	if err != nil {
		return nil, err
	}
	//动听服务器返回结果
	switch euphonyqrResult.Message {
	case "OK": //OK
		{
			euphonyqrToken.Token = euphonyqrResult.Token
		}
	default:
		{
			euphonyqrToken.Token = "no"
		}
	}
	tokens.Store(appkey, euphonyqrToken) //保存APPKEY=>TOKEN信息
	return &euphonyqrResult, nil
}

//检查请求是否来自特定的微信小程序
func checkWxAPPID(buytoken *EuphonyQRToken, referer string) (valid bool) {
	if !buytoken.CheckWxAPPID {
		return true
	}

	if strings.HasPrefix(referer, buytoken.CheckWxPrefix) {
		return true
	}
	return false
}

func StoreToken(appkey string, tokenInfo *EuphonyQRToken) {
	tokenMutex.Lock()
	tokens.Store(appkey, tokenInfo) //保存APPKEY=>TOKEN信息
	tokenMutex.Unlock()
}
