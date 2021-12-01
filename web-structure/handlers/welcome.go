package handlers

import "fmt"

type Welcome interface {
	Hello() string
}

type WelcomeHandler struct {
	User string
}

func (h WelcomeHandler) Hello() string {
	return fmt.Sprintf("hello %s, welcome.", h.User)
}

func NewWelcomeHandler() Welcome {
	var handler WelcomeHandler
	handler.User = "Arthur"
	return handler
}
