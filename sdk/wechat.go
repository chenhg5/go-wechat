package wechat

import (
	"net/http"
	"io"
	"errors"
	"io/ioutil"
	"bytes"
	"compress/gzip"
	"github.com/json-iterator/go"
	"time"
	"github.com/xlstudio/wxbizdatacrypt"
)

// 内部api
// - 构建签名, 签名算法
// - post 请求
// - get 请求
// - 获取access_token

// 文档：

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1445241432  微信公众号
// https://developers.weixin.qq.com/miniprogram/dev/api/qrcode.html  微信小程序
// https://pay.weixin.qq.com/wiki/doc/api/index.html  微信支付

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// ---------------------------
// API常数
// ---------------------------

const (
	GET_ACCESS_TOKEN_API = "https://api.weixin.qq.com/cgi-bin/token" // 获取access_token

	// 网页授权

	GET_WEB_OAUTH_ACCESS_TOKEN         = "https://api.weixin.qq.com/sns/oauth2/access_token"                                         // 获取特殊的网页授权access_token
	REFRESH_WEB_OAUTH_ACCESS_TOKEN     = "https://api.weixin.qq.com/sns/oauth2/refresh_token"                                        // 刷新token
	GET_WEB_OAUTH_USERINFO             = "https://api.weixin.qq.com/sns/userinfo?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN" // 拉取用户信息(需scope为 snsapi_userinfo)
	CHECK_WEB_OAUTH_ACCESS_TOKEN_VALID = "https://api.weixin.qq.com/sns/auth?access_token=ACCESS_TOKEN&openid=OPENID"                // 检验token有效性

	// 模板消息

	SEND_TEMPLATE_MESSAGE = "https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=ACCESS_TOKEN" // 发送模板消息

	// 小程序登录

	WXAPP_OAUTH                 = "https://api.weixin.qq.com/sns/jscode2session"                                             // 小程序获取sessionkey
	GET_WXAPP_CODE              = "https://api.weixin.qq.com/wxa/getwxacode?access_token=ACCESS_TOKEN"                       // 获取小程序码
	GET_WXAPP_CODE_UNLIMIT      = "https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=ACCESS_TOKEN"                // 获取小程序码
	GET_WXAPP_CODE_QRCODE       = "https://api.weixin.qq.com/cgi-bin/wxaapp/createwxaqrcode?access_token=ACCESS_TOKEN"       // 获取小程序二维码
	SEND_WXAPP_TEMPLATE_MESSAGE = "https://api.weixin.qq.com/cgi-bin/message/wxopen/template/send?access_token=ACCESS_TOKEN" // 发送小程序服务通知

	// 微信支付

	PAY_UNIFIED_ORDER = "https://api.mch.weixin.qq.com/pay/unifiedorder" // 下订单
)

// GetNewAccessToken
//
// 参数：
// grant_type	获取access_token填写client_credential
// appid	    第三方用户唯一凭证
// secret       第三方用户唯一凭证密钥，即appsecret
//
// 返回：
// 成功返回 {"access_token":"ACCESS_TOKEN","expires_in":7200}
// 失败返回 {"errcode":40013,"errmsg":"invalid appid"}
func GetNewAccessToken(appId string, appSecret string) ([]byte, error) {
	resData, err := MakeGetReq(GET_ACCESS_TOKEN_API, map[string]string{
		"grant_type": "client_credential",
		"appid":      appId,
		"secret":     appSecret,
	})
	if err != nil {
		return []byte{}, err
	}

	var dataMap map[string]interface{}
	json.Unmarshal(resData, &dataMap)
	if dataMap["errcode"] == "" {
		RedisClient.Set("go-wechat:access_token", dataMap["access_token"].(string), time.Minute*110)
	}

	return resData, nil
}

// GetWebOauthAccessToken
//
// 参数：
// appid	   公众号的唯一标识
// secret	   公众号的appsecret
// code	       填写第一步获取的code参数
// grant_type  填写为authorization_code
//
// 返回：
// 成功返回 { "access_token":"ACCESS_TOKEN", "expires_in":7200, "refresh_token":"REFRESH_TOKEN", "openid":"OPENID", "scope":"SCOPE" }
// 失败返回 { "errcode":40029,"errmsg":"invalid code"}
func GetWebOauthAccessToken(appId string, appSecret string, code string) ([]byte, error) {
	resData, err := MakeGetReq(GET_WEB_OAUTH_ACCESS_TOKEN, map[string]string{
		"grant_type": "authorization_code",
		"appid":      appId,
		"secret":     appSecret,
		"code":       code,
	})
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// RefreshWebOauthAccessToken
//
// 参数：
// appid	    	公众号的唯一标识
// grant_type		填写为refresh_token
// refresh_token	填写通过access_token获取到的refresh_token参数
//
// 返回：
// 成功返回 { "access_token":"ACCESS_TOKEN", "expires_in":7200, "refresh_token":"REFRESH_TOKEN", "openid":"OPENID", "scope":"SCOPE" }
// 失败返回 { "errcode":40029,"errmsg":"invalid code"}
func RefreshWebOauthAccessToken(appId string, refreshToken string) ([]byte, error) {
	resData, err := MakeGetReq(REFRESH_WEB_OAUTH_ACCESS_TOKEN, map[string]string{
		"grant_type":    "refresh_token",
		"appid":         appId,
		"refresh_token": refreshToken,
	})
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// GetWebOauthUserinfo
//
// 参数：
// access_token	网页授权接口调用凭证,注意：此access_token与基础支持的access_token不同
// openid	    用户的唯一标识
// lang	        返回国家地区语言版本，zh_CN 简体，zh_TW 繁体，en 英语
//
// 返回：
// 成功返回 {
// 		"openid":" OPENID",
// 		"nickname": NICKNAME,
// 		"sex":"1",
// 		"province":"PROVINCE"
// 		"city":"CITY",
// 		"country":"COUNTRY",
// 		"headimgurl":    "http://thirdwx.qlogo.cn/mmopen/g3MonUZtNHkdmzicIlibx6iaFqAc56vxLSUfpb6n5WKSYVY0ChQKkiaJSgQ1dZuTOgvLLrhJbERQQ4eMsv84eavHiaiceqxibJxCfHe/46",
// 		"privilege":[ "PRIVILEGE1" "PRIVILEGE2"     ],
// 		"unionid": "o6_bmasdasdsad6_2sgVt7hMZOPfL"
// }
// 失败返回 { "errcode":40003,"errmsg":" invalid openid "}
func GetWebOauthUserinfo(openId string, lang string, accessToken string) ([]byte, error) {
	resData, err := MakeGetReq(GET_WEB_OAUTH_USERINFO, map[string]string{
		"lang":         lang,
		"openid":       openId,
		"access_token": accessToken,
	})
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// CheckWebOauthAccessTokenEffective
//
// 参数：
// access_token	网页授权接口调用凭证,注意：此access_token与基础支持的access_token不同
// openid	    用户的唯一标识
//
// 返回：
// 成功返回 { "errcode":0,"errmsg":"ok"}
// 失败返回 { "errcode":40003,"errmsg":"invalid openid"}
func CheckWebOauthAccessTokenValid(openId string, accessToken string) ([]byte, error) {
	resData, err := MakeGetReq(CHECK_WEB_OAUTH_ACCESS_TOKEN_VALID, map[string]string{
		"openid":       openId,
		"access_token": accessToken,
	})
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// SendTemplateMessage
//
// 参数：{
// 	  "touser":"OPENID",
// 	  "template_id":"ngqIpbwh8bUfcSsECmogfXcV14J0tQlEpBO27izEYtY",
// 	  "url":"http://weixin.qq.com/download",
// 	  "miniprogram":{
// 			"appid":"xiaochengxuappid12345",
//    		"pagepath":"index?foo=bar"
// 	  },
// 	  "data":{
// 			"first": {
// 				"value":"恭喜你购买成功！",
// 				"color":"#173177"
// 			},
// 			"keyword1":{
// 				"value":"巧克力",
// 				"color":"#173177"
// 			},
// 			"keyword2": {
// 				"value":"39.8元",
// 				"color":"#173177"
// 			},
// 			"keyword3": {
// 				"value":"2014年9月22日",
// 				"color":"#173177"
// 			},
// 			"remark":{
// 				"value":"欢迎再次购买！",
// 				"color":"#173177"
// 			}
// 		}
// }
//
// 返回：
// 成功返回 { "errcode":0, "errmsg":"ok", "msgid":200228332 }
// 失败返回 { "errcode":40003,"errmsg":"invalid openid"}
func SendTemplateMessage(accountid int, data map[string]string) ([]byte, error) {
	return []byte{}, nil
}

// WxappOauth
//
// 参数：
// appid		小程序唯一标识
// secret		小程序的 app secret
// js_code		登录时获取的 code
// grant_type	填写为 authorization_code
//
// 返回：
// 成功返回 { "openid": "OPENID", "session_key": "SESSIONKEY", "unionid": "UNIONID" }
// 失败返回 { "errcode":40029,"errmsg":"invalid openid"}
func WxappOauth(appId string, appSecret string, jsCode string) ([]byte, error) {
	resData, err := MakeGetReq(WXAPP_OAUTH, map[string]string{
		"appid":      appId,
		"secret":     appSecret,
		"js_code":    jsCode,
		"grant_type": "authorization_code",
	})
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

func DecodeWxappData(appId string, sessionKey string, iv string, encryptedData string) ([]byte, error) {
	pc := wxbizdatacrypt.WxBizDataCrypt{AppID: appId, SessionKey: sessionKey}
	result, err := pc.Decrypt(encryptedData, iv, false)
	if err != nil {
		return []byte{}, err
	}

	return json.Marshal(result)
}

// GetWxappCode
//
// 参数：
// path	        String								不能为空，最大长度 128 字节
// width	    Int	    430	                        二维码的宽度
// auto_color	Bool	false						自动配置线条颜色，如果颜色依然是黑色，则说明不建议配置主色调
// line_color	Object	{"r":"0","g":"0","b":"0"}	auth_color 为 false 时生效，使用 rgb 设置颜色 例如 {"r":"xxx","g":"xxx","b":"xxx"},十进制表示
// is_hyaline	Bool	false						是否需要透明底色， is_hyaline 为true时，生成透明底色的小程序码
//
// 返回：
// 成功返回 图片
// 失败返回 { "errcode":40029,"errmsg":"invalid openid"}
func GetWxappCode(data map[string]string) ([]byte, error) {
	resData, err := MakePostReq(GET_WXAPP_CODE, map[string]interface{}{
		"path":  data["page"],
		"width": data["width"],
	}, "application/json")
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// GetWxappCodeUnlimit
//
// 参数：
// scene		String								最大32个可见字符，只支持数字，大小写英文以及部分特殊字符：!#$&'()*+,/:;=?@-._~，其它字符请自行编码为合法字符（因不支持%，中文无法使用 urlencode 处理，请使用其他编码方式）
// page			String								必须是已经发布的小程序存在的页面（否则报错），例如 "pages/index/index" ,根路径前不要填加'/',不能携带参数（参数请放在scene字段里），如果不填写这个字段，默认跳主页面
// width		Int		430							二维码的宽度
// auto_color	Bool	false						自动配置线条颜色，如果颜色依然是黑色，则说明不建议配置主色调
// line_color	Object	{"r":"0","g":"0","b":"0"}	auto_color 为 false 时生效，使用 rgb 设置颜色 例如 {"r":"xxx","g":"xxx","b":"xxx"} 十进制表示
// is_hyaline	Bool	false						是否需要透明底色， is_hyaline 为true时，生成透明底色的小程序码
//
// 返回：
// 成功返回 图片
// 失败返回 { "errcode":40029,"errmsg":"invalid openid"}
func GetWxappCodeUnlimit(data map[string]string) ([]byte, error) {
	resData, err := MakePostReq(GET_WXAPP_CODE_UNLIMIT, map[string]interface{}{
		"path":  data["page"],
		"width": data["width"],
	}, "application/json")
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// GetWxappCodeQrcode
//
// 参数：
// path		String		不能为空，最大长度 128 字节
// width	Int		430	二维码的宽度
//
// 返回：
// 成功返回 图片
// 失败返回 { "errcode":40029,"errmsg":"invalid openid"}
func GetWxappCodeQrcode(data map[string]string) ([]byte, error) {
	resData, err := MakePostReq(GET_WXAPP_CODE_QRCODE, map[string]interface{}{
		"page":       data["page"],
		"width":      data["width"],
		"scene":      data["scene"],
		"auto_color": data["auto_color"],
	}, "application/json")
	if err != nil {
		return []byte{}, err
	}
	return resData, nil
}

// SendWxappTemplateMessage
//
// 参数：
// touser			是	接收者（用户）的 openid
// template_id		是	所需下发的模板消息的id
// page				否	点击模板卡片后的跳转页面，仅限本小程序内的页面。支持带参数,（示例index?foo=bar）。该字段不填则模板无跳转。
// form_id			是	表单提交场景下，为 submit 事件带上的 formId；支付场景下，为本次支付的 prepay_id
// data				是	模板内容，不填则下发空模板
// color			否	模板内容字体的颜色，不填默认黑色 【废弃】
// emphasis_keyword	否	模板需要放大的关键词，不填则默认无放大
//
// 返回：
// 成功返回 { "errcode":0, "errmsg":"ok" }
// 失败返回 { "errcode":40029,"errmsg":"invalid openid"}
func SendWxappTemplateMessage(accountid int, data map[string]string) ([]byte, error) {
	return []byte{}, nil
}

// PayUnifiedOrder
//
// 参数：
// <xml>
// 	<appid>wx2421b1c4370ec43b</appid>
// 	<attach>支付测试</attach>
// 	<body>JSAPI支付测试</body>
// 	<mch_id>10000100</mch_id>
// 	<detail><![CDATA[{ "goods_detail":[ { "goods_id":"iphone6s_16G", "wxpay_goods_id":"1001", "goods_name":"iPhone6s 16G", "quantity":1, "price":528800, "goods_category":"123456", "body":"苹果手机" }, { "goods_id":"iphone6s_32G", "wxpay_goods_id":"1002", "goods_name":"iPhone6s 32G", "quantity":1, "price":608800, "goods_category":"123789", "body":"苹果手机" } ] }]]></detail>
// 	<nonce_str>1add1a30ac87aa2db72f57a2375d8fec</nonce_str>
// 	<notify_url>http://wxpay.wxutil.com/pub_v2/pay/notify.v2.php</notify_url>
// 	<openid>oUpF8uMuAJO_M2pxb1Q9zNjWeS6o</openid>
// 	<out_trade_no>1415659990</out_trade_no>
// 	<spbill_create_ip>14.23.150.211</spbill_create_ip>
// 	<total_fee>1</total_fee>
// 	<trade_type>JSAPI</trade_type>
// 	<sign>0CB01533B8C1EF103065174F50BCA001</sign>
// </xml>
//
// 返回：
// <xml>
// 	<return_code><![CDATA[SUCCESS]]></return_code> 								通信标识
// 	<return_msg><![CDATA[OK]]></return_msg>
// 	<appid><![CDATA[wx2421b1c4370ec43b]]></appid>
// 	<mch_id><![CDATA[10000100]]></mch_id>
// 	<nonce_str><![CDATA[IITRi8Iabbblz1Jc]]></nonce_str>
// 	<openid><![CDATA[oUpF8uMuAJO_M2pxb1Q9zNjWeS6o]]></openid>
// 	<sign><![CDATA[7921E432F65EB8ED0CE9755F0E86D72F]]></sign>
// 	<result_code><![CDATA[SUCCESS]]></result_code> 								交易标识
// 	<prepay_id><![CDATA[wx201411101639507cbf6ffd8b0779950874]]></prepay_id>
// 	<trade_type><![CDATA[JSAPI]]></trade_type>
// </xml>
func PayUnifiedOrder(accountid int, data map[string]string) ([]byte, error) {
	return []byte{}, nil
}

// ---------------------------
// sdk内部Api
// ---------------------------

func GetToken() string {
	token, _ := RedisClient.Get("go-wechat:access_token")
	return token
}

func MakeGetReq(url string, data map[string]string) ([]byte, error) {

	var count = 0
	for k, v := range data {
		if count == 0 {
			url += "?" + k + "=" + v
		} else {
			url += "&" + k + "=" + v
		}
		count++
	}

	res, err := http.Get(url)

	if err != nil {
		return []byte{}, err
	}
	if res.StatusCode != 200 {
		return []byte{}, errors.New("网络错误")
	}

	var reader io.ReadCloser
	if res.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(res.Body)
		if err != nil {
			return []byte{}, err
		}
	} else {
		reader = res.Body
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return []byte{}, err
	}

	//var resJsonData map[string]string
	//err = json.Unmarshal(body, &resJsonData)
	//if err != nil {
	//	return map[string]string{}, err
	//}

	return body, nil
}

func MakePostReq(url string, postData map[string]interface{}, contentType string) ([]byte, error) {
	jsonData, jsonErr := json.Marshal(postData)
	if jsonErr != nil {
		return []byte{}, jsonErr
	}

	res, _ := http.Post(url, contentType, bytes.NewBuffer(jsonData))

	var (
		reader io.ReadCloser
		err    error
	)
	if res.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(res.Body)
		if err != nil {
			return []byte{}, err
		}
	} else {
		reader = res.Body
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return []byte{}, err
	}

	//var resJsonData map[string]string
	//err = json.Unmarshal(body, &resJsonData)
	//if err != nil {
	//	return map[string]string{}, err
	//}

	return body, nil
}
