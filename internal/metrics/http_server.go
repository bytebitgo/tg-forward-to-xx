package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
)

// HTTPServer 指标 HTTP 服务
type HTTPServer struct {
	server     *http.Server
	tlsServer  *http.Server
	port       int
	tlsPort    int
	path       string
	stopChan   chan struct{}
	wg         sync.WaitGroup
	auth       bool
	apiKey     string
	headerName string
	tls        bool
	certFile   string
	keyFile    string
	forceHTTPS bool
}

// NewHTTPServer 创建新的 HTTP 服务
func NewHTTPServer(port int, path string) *HTTPServer {
	return &HTTPServer{
		port:       port,
		path:       path,
		stopChan:   make(chan struct{}),
		auth:       config.AppConfig.Metrics.HTTP.Auth,
		apiKey:     config.AppConfig.Metrics.HTTP.APIKey,
		headerName: config.AppConfig.Metrics.HTTP.HeaderName,
		tls:        config.AppConfig.Metrics.HTTP.TLS.Enabled,
		certFile:   config.AppConfig.Metrics.HTTP.TLS.CertFile,
		keyFile:    config.AppConfig.Metrics.HTTP.TLS.KeyFile,
		tlsPort:    config.AppConfig.Metrics.HTTP.TLS.Port,
		forceHTTPS: config.AppConfig.Metrics.HTTP.TLS.ForceHTTPS,
	}
}

// authMiddleware 认证中间件
func (s *HTTPServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 如果未启用认证，直接放行
		if !s.auth {
			next(w, r)
			return
		}

		// 检查 API Key
		apiKey := r.Header.Get(s.headerName)
		if apiKey == "" {
			http.Error(w, "未提供认证信息", http.StatusUnauthorized)
			logrus.Warnf("访问被拒绝：未提供认证信息，来自 %s", r.RemoteAddr)
			return
		}

		if apiKey != s.apiKey {
			http.Error(w, "认证失败", http.StatusUnauthorized)
			logrus.Warnf("访问被拒绝：认证失败，来自 %s", r.RemoteAddr)
			return
		}

		next(w, r)
	}
}

// redirectToHTTPS 重定向到 HTTPS
func (s *HTTPServer) redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	target := fmt.Sprintf("https://%s:%d%s", r.Host, s.tlsPort, r.URL.Path)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// Start 启动 HTTP 服务
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// 注册带认证的指标处理器
	mux.HandleFunc(s.path, s.authMiddleware(s.metricsHandler))

	// 注册带认证的健康检查处理器
	mux.HandleFunc("/health", s.authMiddleware(s.healthHandler))

	// 如果启用了 HTTPS
	if s.tls {
		// 检查证书文件
		if s.certFile == "" || s.keyFile == "" {
			return fmt.Errorf("启用 HTTPS 但未提供证书文件")
		}

		// 创建 HTTPS 服务器
		s.tlsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.tlsPort),
			Handler: mux,
		}

		// 启动 HTTPS 服务器
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			logrus.Infof("HTTPS 服务已启动，监听端口: %d，指标路径: %s，认证状态: %v",
				s.tlsPort, s.path, s.auth)
			if err := s.tlsServer.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && err != http.ErrServerClosed {
				logrus.Errorf("HTTPS 服务错误: %v", err)
			}
		}()

		// 如果不是强制 HTTPS，同时启动 HTTP 服务
		if !s.forceHTTPS {
			s.server = &http.Server{
				Addr:    fmt.Sprintf(":%d", s.port),
				Handler: mux,
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				logrus.Infof("HTTP 服务已启动，监听端口: %d，指标路径: %s，认证状态: %v",
					s.port, s.path, s.auth)
				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logrus.Errorf("HTTP 服务错误: %v", err)
				}
			}()
		} else {
			// 如果强制 HTTPS，HTTP 服务器只用于重定向
			redirectMux := http.NewServeMux()
			redirectMux.HandleFunc("/", s.redirectToHTTPS)
			s.server = &http.Server{
				Addr:    fmt.Sprintf(":%d", s.port),
				Handler: redirectMux,
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				logrus.Info("HTTP 服务已启动（仅用于重定向到 HTTPS）")
				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logrus.Errorf("HTTP 重定向服务错误: %v", err)
				}
			}()
		}
	} else {
		// 如果未启用 HTTPS，只启动 HTTP 服务
		s.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.port),
			Handler: mux,
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			logrus.Infof("HTTP 服务已启动，监听端口: %d，指标路径: %s，认证状态: %v",
				s.port, s.path, s.auth)
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.Errorf("HTTP 服务错误: %v", err)
			}
		}()
	}

	return nil
}

// Stop 停止 HTTP 服务
func (s *HTTPServer) Stop() error {
	close(s.stopChan)

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 停止 HTTP 服务
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			logrus.Errorf("关闭 HTTP 服务错误: %v", err)
		}
	}

	// 停止 HTTPS 服务
	if s.tlsServer != nil {
		if err := s.tlsServer.Shutdown(ctx); err != nil {
			logrus.Errorf("关闭 HTTPS 服务错误: %v", err)
		}
	}

	// 等待所有服务完全停止
	s.wg.Wait()
	logrus.Info("所有 HTTP/HTTPS 服务已停止")

	return nil
}

// 指标处理器
func (s *HTTPServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// 获取最新指标
	metrics := GetMetrics()

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 序列化指标数据
	data, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, fmt.Sprintf("序列化指标数据失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 写入响应
	w.Write(data)
}

// 健康检查处理器
func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
