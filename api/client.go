package api

import (
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11435"
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}