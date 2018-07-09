package main

import (
	"github.com/valyala/fasthttp"
	"strconv"
	"fmt"
	"runtime/debug"
	"log"
	"github.com/go-sql-driver/mysql"
	"github.com/mgutz/ansi"
)

func CallMethod(wcctx *WechatCtx) {

	defer handle(wcctx)

	accountId, err := strconv.Atoi(wcctx.GetFormValue("accountId"))
	if err != nil {
		wcctx.Json(fasthttp.StatusBadRequest, "错误的参数", "")
		return
	}
	method := wcctx.GetFormValue("method")
	if method == "" {
		wcctx.Json(fasthttp.StatusBadRequest, "错误的参数", "")
		return
	}

	wcctx.Account = GetAccountInfo(accountId)

	res, callErr := GlobalFuncMap[method](wcctx)
	if callErr != nil {
		fmt.Println(callErr)
		wcctx.Json(fasthttp.StatusInternalServerError, "系统错误", "")
		return
	}
	dataStr := string(res[:])
	wcctx.Json(fasthttp.StatusOK, "ok", dataStr)
	return
}

type EndPoint func(*WechatCtx) ([]byte, error)

var GlobalFuncMap = map[string]EndPoint{
	"GetNewAccessToken": GetNewAccessToken,
	"GetWebOauthAccessToken": GetWebOauthAccessToken,
	"RefreshWebOauthAccessToken": RefreshWebOauthAccessToken,
	"GetWebOauthUserinfo": GetWebOauthUserinfo,
	"CheckWebOauthAccessTokenValid": CheckWebOauthAccessTokenValid,
	"SendTemplateMessage": SendTemplateMessage,
	"WxappOauth": WxappOauth,
	"GetWxappCode": GetWxappCode,
	"GetWxappCodeUnlimit": GetWxappCodeUnlimit,
	"GetWxappCodeQrcode": GetWxappCodeQrcode,
	"SendWxappTemplateMessage": SendWxappTemplateMessage,
	"PayUnifiedOrder": PayUnifiedOrder,
	"DecodeWxappData": DecodeWxappData,
}

func handle(wcctx *WechatCtx) {

	if EnvConfig["DEBUG"].(bool) {
		log.Println("[GoWechat]",
			ansi.Color(" "+strconv.Itoa(wcctx.Ctx.Response.StatusCode())+" ", "white:blue"),
			ansi.Color(" "+string(wcctx.Ctx.Method()[:])+"   ", "white:blue+h"),
			string(wcctx.Ctx.Path()))
	}

	if err := recover(); err != nil {
		fmt.Println(err)
		fmt.Println(string(debug.Stack()[:]))

		var (
			errMsg string
			mysqlError *mysql.MySQLError
			ok bool
		)
		if errMsg, ok = err.(string); ok {
			wcctx.Json(fasthttp.StatusInternalServerError, errMsg, "")
		} else if mysqlError, ok = err.(*mysql.MySQLError); ok {
			wcctx.Json(fasthttp.StatusInternalServerError, mysqlError.Error(), "")
		} else {
			wcctx.Json(fasthttp.StatusInternalServerError, "系统错误", "")

		}

		WechatCtxPool.Put(wcctx)
		return
	}

	WechatCtxPool.Put(wcctx)
}
