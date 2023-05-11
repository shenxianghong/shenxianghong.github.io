---
title: "「 Golang 」Web Server 代码结构"
excerpt: "记一次在 Kubewebhook 项目中学到的 web 代码结构"
cover: https://picsum.photos/0?sig=20211201
thumbnail: https://go.dev/blog/go-brand/Go-Logo/SVG/Go-Logo_Blue.svg
date: 2021-12-01
toc: true
categories:
- Programming
tag:
- Golang
---

<div align=center><img width="100" style="border: 0px" src="https://go.dev/images/go-logo-blue.svg"></div>

------

# Open Source

https://github.com/slok/kubewebhook（Go framework to create Kubernetes mutating and validating webhooks.）。是一个用于创建 Kubernetes mutating 和 validating webhook 的 Golang 框架，其中提供了用于生产环境的[示例模板](https://github.com/slok/k8s-webhook-example)。

# Sample

https://github.com/shenxianghong/shenxianghong.github.io/tree/main/elegant-code/web-structure

# Structure

## handlers

handlers 中聚焦实际的业务处理逻辑

### welcome.go

请求到来时，组装返回的消息内容

```go
package handlers

import "fmt"

// Welcome 抽象了一系列的业务逻辑
// 例如 Hello 用于组装返回的消息内容
type Welcome interface {
	Hello() string
}

// 接口实现
type WelcomeHandler struct {
	User string
}

// 业务逻辑
func (h WelcomeHandler) Hello() string {
	return fmt.Sprintf("hello %s, welcome.", h.User)
}

// 工厂函数
func NewWelcomeHandler() Welcome {
	var handler WelcomeHandler
	handler.User = "Arthur"
	return handler
}
```

### goodbye.go

原理类似 welcome.go

## web

web 框架的基础结构

### handlers.go

调用业务逻辑，构建 http.Handler 类型，作为请求的 handler 函数

```go
package web

import "net/http"

// 逻辑路由的工厂函数
func (h handler) welcomeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(h.welcome.Hello()))
		w.WriteHeader(http.StatusOK)
	})
}

func (h handler) goodbyeHandler() http.Handler  {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(h.goodbye.Goodbye()))
		w.WriteHeader(http.StatusOK)
	})
}
```

### router.go

用于注册业务逻辑部分的路由，也就是子路由

```go
package web

import "net/http"

func (h handler) router(router *http.ServeMux) {
    // 逻辑路由的工厂函数返回 http.Handler 类型,用于注册子路由
	router.Handle("/welcome", h.welcomeHandler())
	router.Handle("/goodbye", h.goodbyeHandler())
	return
}
```

### web.go

web 框架的上层对象，包括配置和根路由 handler 的生成等

```go
package web

import (
	"elegant-coding/handlers"

	"errors"
	"net/http"
)

// 配置信息
// 上层会传入业务逻辑的工厂函数来初始化这个结构体，同样的，也可以传入比如全局的 logger 等信息
// 初始化下面的 handler 时会根据这个配置信息
type Config struct {
	WelcomeHandler handlers.Welcome
	GoodbyeHandler handlers.Goodbye
}

// 用于做一些校验设置默认信息等
func (c *Config) defaults() error {
	if c.WelcomeHandler == nil {
		return errors.New("welcome handler is missing")
	}
	if c.GoodbyeHandler == nil {
		return errors.New("goodbye handler is missing")
	}
	return nil
}

// 封装了 http.Handler，同时也包含了一系列的业务逻辑的接口实现
type handler struct {
    // 业务逻辑的 interface
	welcome handlers.Welcome
	goodbye handlers.Goodbye
	// 原生 http handler 功能
	handler http.Handler
}

// 调用原生的 http.Handler.ServeHTTP，启动服务
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

// 初始化上面的 handler
// 返回的是一个 http.Handler 类型，是为了在主函数的时候作为根路由的 handler
func New(config Config) http.Handler {
	if err := config.defaults(); err != nil {
		panic("handler configuration is not valid")
	}

	mux := http.NewServeMux()
	h := handler{
		welcome: config.WelcomeHandler,
		goodbye: config.GoodbyeHandler,
		handler: mux,
	}
	h.router(mux)

	return h
}
```

### main.go

```go
package main

import (
	"fmt"
	"net/http"

	"elegant-coding/handlers"
	"elegant-coding/web"
)


func main() {
    // 根路由的 handler 函数
	handler := web.New(web.Config{
        // 通过工厂函数生成 handler
		WelcomeHandler: handlers.NewWelcomeHandler(),
		GoodbyeHandler: handlers.NewGoodbyeHandler(),
	})
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := http.Server{
		Addr:    ":8081",
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
```

