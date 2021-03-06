package euphonyqr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// 返回给DEMO的结果
type DemoDetectResult struct {
	Message  string   `json:"msg"`
	Tags     []string `json:"tags"`
	RecordID string   `json:"record_id"`
}

type EuphonyQRDetectRequestParam struct {
	Version   int32  `json:"version"`
	RequestID string `json:"request_id,omitempty"`
	Appkey    string `json:"appkey"`
	Seckey    string `json:"seckey"`
	Test      bool   `json:"test,omitempty"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Language  string `json:"language,omitempty"`
	App       struct {
		AppName     string `json:"app_name"`
		PackageName string `json:"package_name"`
		Platform    string `json:"platform"`
	} `json:"App"`
	Device struct {
		DeviceID   string `json:"device_id"`
		OS         string `json:"os,omitempty"`
		OSVersion  string `json:"osv,omitempty"`
		DeviceType string `json:"device_type,omitempty"`
		Brand      string `json:"brand,omitempty"`
		Operator   string `json:"operator,omitempty"`
		Network    string `json:"network,omitempty"`
		Longtitude string `json:"lon,omitempty"`
		Latitude   string `json:"lat,omitempty"`
		MAC        string `json:"mac,omitempty"`
	} `json:"Device"`
	User struct {
		UserID    string `json:"user_id,omitempty"`
		WXOpenID  string `json:"wx_open_id,omitempty"`
		WXUnionID string `json:"wx_union_id,omitempty"`
	} `json:"User"`
}

type EuphonyQRDetectResult struct {
	Message          string `json:"msg"`
	ResponseID       string `json:"response_id"`
	ValidResultCount int    `json:"count"`
	Result           []struct {
		Tags     []string `json:"tags"`
		Power    float32  `json:"power"`
		Channel  int      `json:"channel"`
		Distance float32  `json:"distance"`
		Range    float32  `json:"range"`
	} `json:"result"`
}

//FetchHandler 处理sdk中的fetchinfo请求
func FetchHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			http.Error(rw, err.Error(), 500)
		}
	}()

	if req.Method == "GET" {
		//URL由客户端给出，再加上必须的参数后POST得到结果
		refer := req.Header.Get("Referer")      //微信小程序的referer
		appkey := req.URL.Query().Get("appkey") //应该与main中注册的appkey一致
		url := req.URL.Query().Get("url")       //请求URL
		platform := req.URL.Query().Get("platform")
		deviceID := req.URL.Query().Get("device_id")
		IP := req.Header.Get("X-real-ip")
		if IP == "" {
			IP = strings.Split(req.RemoteAddr, ":")[0]
		}
		UA := req.Header.Get("User-Agent")
		wxresult := DemoDetectResult{}
		wxresult.Tags = make([]string, 0)

		tokenInterface, ok := tokens.Load(appkey)
		if !ok {
			wxresult.Message = "Error1"
		} else {
			token := (tokenInterface.(*EuphonyQRToken))
			if !checkWxAPPID(token, refer) {
				//检查微信小程序APPID
				wxresult.Message = "Error2"
			} else {
				result, err := FetchDetectInfo(url, appkey, token.Seckey, token.UseSandbox, platform, deviceID, IP, UA)
				if err != nil {
					// println(err.Error())
					wxresult.Message = "Error3"
				} else {
					wxresult.Message = "OK"
					wxresult.RecordID = result.ResponseID
					if result.ValidResultCount > 0 && result.Result != nil {
						for _, result := range result.Result {
							wxresult.Tags = append(wxresult.Tags, result.Tags...)
						}
					}
				}
			}
		}

		resp, err := json.Marshal(wxresult)
		if err != nil {
			return
		}
		rw.Header().Set("Content-Type", "application/json;charset=utf-8")
		rw.Write(resp)
	} else {
		rw.WriteHeader(http.StatusForbidden)
	}

}

//FetchInfo 获取动听服务器的解析结果
//参数appkey,seckey从动听后台中获得
//useSandbox说明APPKEY是否是sandbox.euphonyqr.com
func FetchDetectInfo(fetchurl string, appkey string, seckey string, useSandbox bool, platform string, deviceID string, IP string, UA string) (detectResult *EuphonyQRDetectResult, err error) {
	requireParams := EuphonyQRDetectRequestParam{}
	requireParams.Version = 1
	requireParams.Appkey = appkey
	requireParams.Seckey = seckey
	requireParams.Test = useSandbox
	requireParams.RequestID = "please set a unique id" //此ID可用于查询日志，请自行设置，确保唯一，如果不用请置空
	requireParams.IP = IP
	requireParams.UserAgent = UA

	requireParams.App.Platform = platform // ios或android或wx_app
	requireParams.App.AppName = "demo_app"
	requireParams.Device.DeviceID = deviceID
	if platform == "ios" {
		requireParams.App.PackageName = "package name of your app"
	} else if platform == "android" {
		requireParams.App.PackageName = "bundle id of your app"
	} else if platform == "wx_app" {
		requireParams.App.PackageName = "wx70bcdd12873c3cb1"
	}

	requireParams.User.UserID = "please give a custom user id"
	requireParams.User.WXOpenID = "please give a custom user id"
	requireParams.User.WXUnionID = "please give a custom user id"

	jsonStr, _ := json.Marshal(&requireParams)
	// println("fetch url: " + fetchurl)
	// println("params: " + string(jsonStr))
	req, err := http.NewRequest("POST", fetchurl, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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
	err = json.Unmarshal(body, &detectResult)
	if err != nil {
		return nil, err
	}
	if detectResult.Message != "OK" {
		err = fmt.Errorf("Server return error: %s", detectResult.Message)
	}
	// println(string(body))
	return
}
