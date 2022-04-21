package web

import "net/http"

func (h handler) router(router *http.ServeMux) {
	router.Handle("/welcome", h.welcomeHandler())
	router.Handle("/goodbye", h.goodbyeHandler())
	return
}
