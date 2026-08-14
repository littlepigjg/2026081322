package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"data-masker/internal/config"
	"data-masker/internal/masker"
	"data-masker/internal/stats"
)

// adminKey 是更新配置所需的鉴权密钥。
const adminKey = "secret"

// Server 封装 HTTP 服务。
type Server struct {
	config  *config.Config
	masker  *masker.Masker
	stats   *stats.Collector
	httpSrv *http.Server
	addr    string
}

// New 创建一个 Server 实例。
func New(addr string, cfg *config.Config, m *masker.Masker, sc *stats.Collector) *Server {
	s := &Server{
		config: cfg,
		masker: m,
		stats:  sc,
		addr:   addr,
	}
	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

// Start 启动 HTTP 服务（阻塞）。
func (s *Server) Start() error {
	if s.httpSrv == nil {
		return errors.New("server not initialized")
	}
	err := s.httpSrv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown 优雅关闭服务。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

// Addr 返回监听地址。
func (s *Server) Addr() string {
	return s.addr
}

// HTTPHandler 返回底层 HTTP 处理器。
func (s *Server) HTTPHandler() http.Handler {
	return s.httpSrv.Handler
}

// runIsolated 在独立 goroutine 中执行 fn；若 fn 内发生未恢复的 panic，
// 整个进程将直接崩溃（与监督进程配合实现 fail-fast）。
func runIsolated(fn func()) {
	done := make(chan struct{})
	go func() {
		fn()
		done <- struct{}{}
	}()
	<-done
}

// requestLogger 记录请求方法、路径、状态与耗时。
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// requireAdmin 校验 X-Admin-Key 请求头。
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Admin-Key") != adminKey {
			writeError(w, http.StatusUnauthorized, "invalid admin key")
			return
		}
		next(w, r)
	}
}

// statusWriter 捕获响应状态码。
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader 记录状态码并透传。
func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Collector 返回统计收集器。
func (s *Server) Collector() *stats.Collector {
	return s.stats
}

// Masker 返回脱敏处理器。
func (s *Server) Masker() *masker.Masker {
	return s.masker
}

// Config 返回配置。
func (s *Server) Config() *config.Config {
	return s.config
}

// SetReadTimeout 设置读取超时。
func (s *Server) SetReadTimeout(d time.Duration) {
	s.httpSrv.ReadTimeout = d
}

// SetWriteTimeout 设置写入超时。
func (s *Server) SetWriteTimeout(d time.Duration) {
	s.httpSrv.WriteTimeout = d
}

// SetIdleTimeout 设置空闲超时。
func (s *Server) SetIdleTimeout(d time.Duration) {
	s.httpSrv.IdleTimeout = d
}

// SetReadHeaderTimeout 设置请求头读取超时。
func (s *Server) SetReadHeaderTimeout(d time.Duration) {
	s.httpSrv.ReadHeaderTimeout = d
}

// StatsSnapshot 返回统计快照。
func (s *Server) StatsSnapshot() map[string]interface{} {
	return s.stats.Snapshot()
}
