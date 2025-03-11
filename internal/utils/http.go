package utils

import (
	"net/http"
	"time"
)

// HTTPClient 是一个全局的 HTTP 客户端实例
var HTTPClient = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
} 