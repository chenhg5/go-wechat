package main

import (
	"github.com/valyala/fasthttp"
	"strconv"
	"fmt"
	"runtime/debug"
	"log"
	"github.com/go-sql-driver/mysql"
	"github.com/mgutz/ansi"
	"os"
	"time"
	"io"
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

	if EnvConfig["LOG_IN_FILE"].(bool) {

		f, _ := os.Create(EnvConfig["ACCESS_LOG_PATH"].(string))
		defer f.Close()

		str := "[GoWechat] " + time.Now().Format("2006-01-02 15:04:05") + " | " + strconv.Itoa(wcctx.Ctx.Response.StatusCode()) + " | " + string(wcctx.Ctx.Method()[:]) + " | " + string(wcctx.Ctx.Path())
		f.Write([]byte(str))
	}

	if err := recover(); err != nil {
		if EnvConfig["DEBUG"].(bool) {
			fmt.Println(err)
			fmt.Println(string(debug.Stack()[:]))
		}

		var (
			errMsg string
			mysqlError *mysql.MySQLError
			ok bool
		)
		if errMsg, ok = err.(string); ok {
		} else if mysqlError, ok = err.(*mysql.MySQLError); ok {
			errMsg = mysqlError.Error()
		} else {
			errMsg = "系统错误"
		}

		wcctx.Json(fasthttp.StatusInternalServerError, errMsg, "")

		if EnvConfig["LOG_IN_FILE"].(bool) {
			f, _ := os.Create(EnvConfig["ERROR_LOG_PATH"].(string))
			defer f.Close()

			defaultWriter := io.MultiWriter(f)

			fmt.Fprintf(defaultWriter, "%s", "\n")
			fmt.Fprintf(defaultWriter, "%s", "["+time.Now().Format("2006-01-02 15:04:05")+"] app.ERROR: ")
			fmt.Fprintf(defaultWriter, "%s", err)
			fmt.Fprintf(defaultWriter, "%s", "\nStack trace:\n")
			fmt.Fprintf(defaultWriter, "%s", debug.Stack())
			fmt.Fprintf(defaultWriter, "%s", "\n")
		}

		WechatCtxPool.Put(wcctx)
		return
	}

	WechatCtxPool.Put(wcctx)
}
