package main

import (
	"net"
	"time"
	"sync/atomic"
	"fmt"
	"github.com/valyala/fasthttp/reuseport"
	"github.com/valyala/fasthttp"
	"os"
	"os/signal"
	"log"
	"sync"
	"strconv"
)

type GracefulListener struct {
	// inner listener
	ln net.Listener

	// maximum wait time for graceful shutdown
	maxWaitTime time.Duration

	// this channel is closed during graceful shutdown on zero open connections.
	done chan struct{}

	// the number of open connections
	connsCount uint64

	// becomes non-zero when graceful shutdown starts
	shutdown uint64
}

// NewGracefulListener wraps the given listener into 'graceful shutdown' listener.
func newGracefulListener(ln net.Listener, maxWaitTime time.Duration) net.Listener {
	return &GracefulListener{
		ln:          ln,
		maxWaitTime: maxWaitTime,
		done:        make(chan struct{}),
	}
}

func (ln *GracefulListener) Accept() (net.Conn, error) {
	c, err := ln.ln.Accept()

	if err != nil {
		return nil, err
	}

	atomic.AddUint64(&ln.connsCount, 1)

	return &gracefulConn{
		Conn: c,
		ln:   ln,
	}, nil
}

func (ln *GracefulListener) Addr() net.Addr {
	return ln.ln.Addr()
}

// Close closes the inner listener and waits until all the pending open connections
// are closed before returning.
func (ln *GracefulListener) Close() error {
	err := ln.ln.Close()

	if err != nil {
		return nil
	}

	return ln.waitForZeroConns()
}

func (ln *GracefulListener) waitForZeroConns() error {
	atomic.AddUint64(&ln.shutdown, 1)

	if atomic.LoadUint64(&ln.connsCount) == 0 {
		close(ln.done)
		return nil
	}

	select {
	case <-ln.done:
		return nil
	case <-time.After(ln.maxWaitTime):
		return fmt.Errorf("cannot complete graceful shutdown in %s", ln.maxWaitTime)
	}

	return nil
}

func (ln *GracefulListener) closeConn() {
	connsCount := atomic.AddUint64(&ln.connsCount, ^uint64(0))

	if atomic.LoadUint64(&ln.shutdown) != 0 && connsCount == 0 {
		close(ln.done)
	}
}

type gracefulConn struct {
	net.Conn
	ln *GracefulListener
}

func (c *gracefulConn) Close() error {
	err := c.Conn.Close()

	if err != nil {
		return err
	}

	c.ln.closeConn()

	return nil
}

type WechatCtx struct {
	Ctx     *fasthttp.RequestCtx
	Account map[string]string
}

func (wcctx *WechatCtx) GetFormValue(key string) string {
	mf, err := (*wcctx).Ctx.MultipartForm()
	if err == nil && mf.Value != nil {
		vv := mf.Value[key]
		if len(vv) > 0 {
			return vv[0]
		} else {
			return ""
		}
	}
	return ""
}

func (wcctx *WechatCtx) Json(statusCode int, msg string, data string) {
	(*wcctx).Ctx.SetStatusCode(statusCode)
	(*wcctx).Ctx.SetContentType("application/json")
	if data == "" {
		(*wcctx).Ctx.WriteString(`{"code":` + strconv.Itoa(statusCode) + `, "msg":"` + msg + `"}`)
	} else {
		(*wcctx).Ctx.WriteString(`{"code":` + strconv.Itoa(statusCode) + `, "msg":"` + msg + `", "data":` + data + `}`)
	}
}

var WechatCtxPool sync.Pool

func InitServer(port string) {

	ln, err := reuseport.Listen("tcp4", ":"+port)
	if err != nil {
		log.Fatalf("error in reuseport listener: %s", err)
	}

	duration := 30 * time.Second
	graceful := newGracefulListener(ln, duration)

	go func() {
		fasthttp.Serve(graceful, func(ctx *fasthttp.RequestCtx) {
			path := string(ctx.Path())

			var (
				wcctx *WechatCtx
				ok    bool
			)

			if wcctx, ok = WechatCtxPool.Get().(*WechatCtx); ok {
				wcctx.Ctx = ctx
				wcctx.Account = map[string]string{}
			} else {
				wcctx = &WechatCtx{
					ctx,
					map[string]string{},
				}
			}

			switch path {
			case "/call":
				CallMethod(wcctx)
			default:
				defer handle(wcctx)
				wcctx.Json(fasthttp.StatusNotFound, "错误的路径", "")
			}
		})
	}()

	osSignals := make(chan os.Signal)
	signal.Notify(osSignals, os.Interrupt)

	<-osSignals

	log.Printf("graceful shutdown signal received.\n")

	if err := graceful.Close(); err != nil {
		log.Fatalf("error with graceful close: %s", err)
	}
}
