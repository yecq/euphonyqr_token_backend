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
   请参考 function RequestEuphonyqrToken

3. 获取动听服务器的解析结果，请求url来自客户端的返回，POST，BODY内容为JSON，具体数据格式见
function FetchHandler
*/
package main

import (
	"log"
	"net/http"
	"os"
	"runtime"

	"./euphonyqr"
)

func main() {
	/*
		建议为每一个微信小程序单独布署一个微服务来处理请求，请确保SECKEY的安全
		以下是环境变量
		PORT_HTTP			监听端口
		APPKEY				在动听后台创建APP后获取
		SECKEY				在动听后台创建APP后获取
		SANDBOX				如果是在www.euphonyqr.com创建的APP，设为false。
							如果是在sandbox.euphonyqr.com创建的APP，设为true。
		WXURL				TOKEN监听的路径
		WX_FETCH_URL		FETCH监听的路径
		WXAPPID				微信小程序APPID，请检查在动听后台创建APP时填写的小程序APPID是否相同
		CHECKWXAPPID		如果为true，会检测referrer确定是从指定的小程序中发出的请求
	*/
	port := os.Getenv("PORT_HTTP")
	if port == "" {
		port = "8080"
	}

	appKey := os.Getenv("APPKEY")
	if appKey == "" {
		appKey = "75ba120532f44aa7a8cd431a2c2a50ef"
	}

	secKey := os.Getenv("SECKEY")
	if secKey == "" {
		secKey = "64ebce4ce9b540bf85b5f09093a8eeff"
	}

	wxAppID := os.Getenv("WXAPPID")
	if wxAppID == "" {
		wxAppID = ""
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

	wxFetchHandleURL := os.Getenv("WX_FETCH_URL")
	if wxFetchHandleURL == "" {
		wxFetchHandleURL = "/wx/fetchinfo"
	}

	checkWxAPPID := false
	checkWxAPPIDStr := os.Getenv("CHECKWXAPPID")
	if checkWxAPPIDStr == "true" {
		checkWxAPPID = true
	}

	buyfullDemoToken := euphonyqr.EuphonyQRToken{Seckey: secKey, UseSandbox: useSandbox,
		WxAppID: wxAppID, CheckWxPrefix: "https://servicewechat.com/" + wxAppID + "/",
		CheckWxAPPID: checkWxAPPID /*if you want to check wx appid , set true*/}

	euphonyqr.StoreToken(appKey, &buyfullDemoToken)

	runtime.GOMAXPROCS(runtime.NumCPU())
	http.HandleFunc(wxHandleURL, euphonyqr.TokenHandler)
	http.HandleFunc(wxFetchHandleURL, euphonyqr.FetchHandler)
	log.Fatalln(http.ListenAndServe("0.0.0.0:"+port, nil))
}
