package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//1 建立errgroup
	g, ctx := errgroup.WithContext(context.Background())

	httpServerChan := make(chan struct{})
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, "HTTP, Hello")
	})

	http.HandleFunc("/out", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, "HTTP, Out")
		httpServerChan <- struct{}{}
	})

	server := http.Server{
		Addr: ":8080",
	}
	// 2 建立http server 并放到errgroup
	g.Go(func() error {
		return server.ListenAndServe()
	})

	//3  接收输入信号
	g.Go(func() error {
		quit := make(chan os.Signal, 0)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		// http 发生 error 触发
		case <-ctx.Done():
			log.Println("http out")
			signal.Stop(quit) // 停止系统信号的监听。
			close(quit)
			return ctx.Err()
		//	退出信号
		case sig := <-quit:
			return errors.Errorf("get os signal: %v", sig)
		}
	})

	//4 监听 http 主动退出
	g.Go(func() error {
		select {
		// 触发信号退出
		case <-ctx.Done():
			log.Println("signal errgroup exit...")
		case <-httpServerChan:
			log.Println("server will out...")
		}

		log.Println("shutting down server...")
		return server.Shutdown(ctx)
	})

	fmt.Printf("errgroup exiting: %+v\n", g.Wait())
}
