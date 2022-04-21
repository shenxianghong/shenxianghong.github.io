package handlers

import "fmt"

type Goodbye interface {
	Goodbye() string
}

type GoodbyeHandler struct {
	User string
}

func (h GoodbyeHandler) Goodbye() string {
	return fmt.Sprintf("Bye %s, see you.", h.User)
}

func NewGoodbyeHandler() Goodbye {
	var handler GoodbyeHandler
	handler.User = "Arthur"
	return handler
}
