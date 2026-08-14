package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"data-masker/internal/config"
	"data-masker/internal/graceful"
	"data-masker/internal/masker"
	"data-masker/internal/server"
	"data-masker/internal/stats"
)

const (
	defaultAddr       = ":8080"
	defaultConfigPath = "mask_config.json"
	statsReportEvery  = 60 * time.Second
	version           = "1.0.0"
)

func main() {
	addr := flag.String("addr", defaultAddr, "listen address")
	configPath := flag.String("config", "", "path to config file (optional)")
	saveConfig := flag.Bool("save-config", false, "save config snapshot on shutdown")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	listen := resolveAddr(*addr)
	cfg := loadConfig(*configPath)
	m := masker.New(cfg)
	sc := stats.New()
	srv := server.New(listen, cfg, m, sc)

	go runStatsReporter(sc, statsReportEvery)

	serveErr := make(chan error, 1)
	go func() {
		printBanner(listen)
		serveErr <- srv.Start()
	}()

	opts := graceful.Options{
		Timeout:    5 * time.Second,
		ConfigPath: configSnapshotPath(*configPath, *saveConfig),
		SaveConfig: *saveConfig,
		Config:     cfg,
	}

	if err := graceful.Wait(srv.Shutdown, opts); err != nil {
		os.Exit(1)
	}

	if err := <-serveErr; err != nil {
		log.Printf("serve error: %v", err)
		os.Exit(1)
	}
}

// loadConfig 加载配置，失败时回退到默认配置。
func loadConfig(path string) *config.Config {
	cfg := config.New()
	if path == "" {
		return cfg
	}
	if err := cfg.LoadFromFile(path); err != nil {
		log.Printf("warn: cannot load config from %s: %v (using defaults)", path, err)
		return config.New()
	}
	log.Printf("config loaded from %s", path)
	return cfg
}

// resolveAddr 若未通过 flag 指定地址，则读取环境变量 MASK_ADDR。
func resolveAddr(flagAddr string) string {
	if flagAddr != "" && flagAddr != defaultAddr {
		return flagAddr
	}
	if env := os.Getenv("MASK_ADDR"); env != "" {
		return env
	}
	return defaultAddr
}

// configSnapshotPath 返回配置快照保存路径。
func configSnapshotPath(path string, save bool) string {
	if !save {
		return ""
	}
	if path == "" {
		return defaultConfigPath
	}
	return path
}

// runStatsReporter 每 60 秒打印一次统计报告。
func runStatsReporter(sc *stats.Collector, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for range ticker.C {
		log.Println(sc.Report())
	}
}

// usage 打印使用说明。
func usage() {
	fmt.Fprintf(os.Stderr, "data-masker - lightweight data masking service\n")
	fmt.Fprintf(os.Stderr, "usage: data-masker [-addr :8080] [-config path] [-save-config]\n")
}

// versionInfo 返回完整版本信息。
func versionInfo() string {
	return fmt.Sprintf("data-masker v%s (report every %s)", version, statsReportEvery)
}

// printBanner 打印启动横幅。
func printBanner(addr string) {
	log.Printf("%s, listening on %s", versionInfo(), addr)
}

// defaultReportInterval 返回默认统计上报间隔。
func defaultReportInterval() time.Duration {
	return statsReportEvery
}

// resolveConfigFile 返回配置文件路径，空则返回默认。
func resolveConfigFile(path string) string {
	if path == "" {
		return defaultConfigPath
	}
	return path
}
