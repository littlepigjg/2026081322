package graceful

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"data-masker/internal/config"
)

// ShutdownFunc 是实际的关闭函数。
type ShutdownFunc func(context.Context) error

// Options 控制优雅关闭行为。
type Options struct {
	Timeout    time.Duration
	ConfigPath string
	SaveConfig bool
	Config     *config.Config
}

// DefaultOptions 返回默认关闭选项。
func DefaultOptions(cfg *config.Config) Options {
	return Options{
		Timeout:    5 * time.Second,
		ConfigPath: "mask_config.json",
		SaveConfig: true,
		Config:     cfg,
	}
}

// Wait 阻塞直到收到退出信号，并执行优雅关闭。
func Wait(shutdown ShutdownFunc, opts Options) error {
	ctx, stop := contextFromSignals()
	defer stop()

	<-ctx.Done()
	log.Printf("shutdown signal received")

	if opts.SaveConfig && opts.Config != nil {
		saveConfig(opts.Config, opts.ConfigPath)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	if err := shutdownWithTimeout(shutdown, timeout); err != nil {
		log.Printf("shutdown error: %v", err)
		return err
	}

	log.Println("server stopped gracefully")
	return nil
}

// saveConfig 保存配置快照。
func saveConfig(cfg *config.Config, path string) {
	if path == "" {
		path = "mask_config.json"
	}
	if err := cfg.SaveToFile(path); err != nil {
		log.Printf("failed to save config snapshot: %v", err)
		return
	}
	log.Printf("config snapshot saved to %s", path)
}

// signals 返回需要监听的信号。
func signals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}

// contextFromSignals 返回一个在收到信号时取消的上下文。
func contextFromSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), signals()...)
}

// shutdownWithTimeout 带超时执行关闭。
func shutdownWithTimeout(shutdown ShutdownFunc, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return shutdown(ctx)
}

// Manager 封装关闭函数与选项，便于复用。
type Manager struct {
	shutdown ShutdownFunc
	opts     Options
}

// NewManager 创建关闭管理器。
func NewManager(shutdown ShutdownFunc, opts Options) *Manager {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	return &Manager{shutdown: shutdown, opts: opts}
}

// Run 阻塞并执行优雅关闭。
func (m *Manager) Run() error {
	return Wait(m.shutdown, m.opts)
}

// Timeout 返回关闭超时时间。
func (m *Manager) Timeout() time.Duration {
	return m.opts.Timeout
}

// WithTimeout 返回设置新超时的副本。
func (m *Manager) WithTimeout(d time.Duration) *Manager {
	clone := *m
	clone.opts.Timeout = d
	return &clone
}

// Shutdown 返回底层关闭函数。
func (m *Manager) Shutdown() ShutdownFunc {
	return m.shutdown
}

// Options 返回当前选项副本。
func (m *Manager) Options() Options {
	return m.opts
}

// EnableConfigSave 开启配置快照保存并返回副本。
func (m *Manager) EnableConfigSave(path string) *Manager {
	clone := *m
	clone.opts.SaveConfig = true
	clone.opts.ConfigPath = path
	return &clone
}

// isTimeout 判断错误是否为上下文超时。
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	return err == context.DeadlineExceeded
}
