package web

import (
	"elegant-coding/handlers"

	"errors"
	"net/http"
)

type Config struct {
	WelcomeHandler handlers.Welcome
	GoodbyeHandler handlers.Goodbye
}

func (c *Config) defaults() error {
	if c.WelcomeHandler == nil {
		return errors.New("welcome handler is missing")
	}
	if c.GoodbyeHandler == nil {
		return errors.New("goodbye handler is missing")
	}
	return nil
}

type handler struct {
	welcome handlers.Welcome
	goodbye handlers.Goodbye

	handler http.Handler
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

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
