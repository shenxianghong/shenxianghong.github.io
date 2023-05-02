package web

import "net/http"

func (h handler) welcomeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(h.welcome.Hello()))
		w.WriteHeader(http.StatusOK)
	})
}

func (h handler) goodbyeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(h.goodbye.Goodbye()))
		w.WriteHeader(http.StatusOK)
	})
}
