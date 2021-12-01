package main

import (
	"fmt"
	"net/http"
	"webstructure/handlers"
	"webstructure/web"
)

func main() {
	handler := web.New(web.Config{
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
