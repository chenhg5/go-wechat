package main

import (
	"github.com/valyala/fasthttp"
	"strconv"
	"fmt"
	"runtime/debug"
	"log"
	"github.com/go-sql-driver/mysql"
	"github.com/mgutz/ansi"
	"time"
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
	"GetNewAccessToken":             GetNewAccessToken,
	"GetWebOauthAccessToken":        GetWebOauthAccessToken,
	"RefreshWebOauthAccessToken":    RefreshWebOauthAccessToken,
	"GetWebOauthUserinfo":           GetWebOauthUserinfo,
	"CheckWebOauthAccessTokenValid": CheckWebOauthAccessTokenValid,
	"SendTemplateMessage":           SendTemplateMessage,
	"WxappOauth":                    WxappOauth,
	"GetWxappCode":                  GetWxappCode,
	"GetWxappCodeUnlimit":           GetWxappCodeUnlimit,
	"GetWxappCodeQrcode":            GetWxappCodeQrcode,
	"SendWxappTemplateMessage":      SendWxappTemplateMessage,
	"PayUnifiedOrder":               PayUnifiedOrder,
	"DecodeWxappData":               DecodeWxappData,
}

func handle(wcctx *WechatCtx) {

	if EnvConfig["DEBUG"].(bool) {
		log.Println("[GoWechat]",
			ansi.Color(" "+strconv.Itoa(wcctx.Ctx.Response.StatusCode())+" ", "white:blue"),
			ansi.Color(" "+string(wcctx.Ctx.Method()[:])+"   ", "white:blue+h"),
			string(wcctx.Ctx.Path()))
	}

	if EnvConfig["LOG_IN_FILE"].(bool) {

		AccessLogger.log("[GoWechat] ")
		AccessLogger.log(time.Now().Format("2006-01-02 15:04:05") + " | ")
		AccessLogger.log(strconv.Itoa(wcctx.Ctx.Response.StatusCode()) + " | ")
		AccessLogger.log(string(wcctx.Ctx.Method()[:]) + " | ")
		AccessLogger.log(string(wcctx.Ctx.Path()))
	}

	if err := recover(); err != nil {
		if EnvConfig["DEBUG"].(bool) {
			fmt.Println(err)
			fmt.Println(string(debug.Stack()[:]))
		}

		var (
			errMsg     string
			mysqlError *mysql.MySQLError
			ok         bool
		)

		if errMsg, ok = err.(string); !ok {
			if mysqlError, ok = err.(*mysql.MySQLError); ok {
				errMsg = mysqlError.Error()
			} else {
				errMsg = "系统错误"
			}
		}

		wcctx.Json(fasthttp.StatusInternalServerError, errMsg, "")

		if EnvConfig["LOG_IN_FILE"].(bool) {

			ErrorLogger.log( "\n")
			ErrorLogger.log( "["+time.Now().Format("2006-01-02 15:04:05")+"] app.ERROR: ")
			ErrorLogger.log( err)
			ErrorLogger.log( "\nStack trace:\n")
			ErrorLogger.log( debug.Stack())
			ErrorLogger.log( "\n")
		}

		WechatCtxPool.Put(wcctx)
		return
	}

	WechatCtxPool.Put(wcctx)
}
