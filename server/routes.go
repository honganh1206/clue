package server

import (
	"net"
	"net/http"
)

func Serve(ln net.Listener) error {
	mux := http.NewServeMux()

	// mux.HandleFunc("/conversations", middleware(conversationsHandler))
	// mux.HandleFunc("/conversations/", middleware(conversationHandler))
	server := &http.Server{Handler: mux, Addr: ":11435"}
	return server.Serve(ln)
}
