package stats

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Metric 表示某个类型的累计统计。
type Metric struct {
	Count      int64 `json:"count"`
	TotalNanos int64 `json:"total_nanos"`
	MinNanos   int64 `json:"min_nanos"`
	MaxNanos   int64 `json:"max_nanos"`
}

// TypeReport 表示单个类型的统计报告。
type TypeReport struct {
	Type      string  `json:"type"`
	Count     int64   `json:"count"`
	AvgMillis float64 `json:"avg_millis"`
	MinMillis float64 `json:"min_millis"`
	MaxMillis float64 `json:"max_millis"`
}

// Collector 线程安全地收集脱敏调用统计。
type Collector struct {
	mu         sync.RWMutex
	metrics    map[string]*Metric
	errors     map[string]int64
	lastReport time.Time
}

// New 创建统计收集器。
func New() *Collector {
	return &Collector{
		metrics:    make(map[string]*Metric),
		errors:     make(map[string]int64),
		lastReport: time.Now(),
	}
}

// Record 记录一次脱敏调用及耗时。
func (c *Collector) Record(typ string, duration time.Duration) {
	if typ == "" {
		typ = "unknown"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.metrics[typ]
	if !ok {
		m = &Metric{MinNanos: duration.Nanoseconds()}
		c.metrics[typ] = m
	}
	m.Count++
	m.TotalNanos += duration.Nanoseconds()
	if duration.Nanoseconds() < m.MinNanos {
		m.MinNanos = duration.Nanoseconds()
	}
	if duration.Nanoseconds() > m.MaxNanos {
		m.MaxNanos = duration.Nanoseconds()
	}
}

// RecordError 记录一次错误分类。
func (c *Collector) RecordError(kind string) {
	if kind == "" {
		kind = "unknown"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors[kind]++
}

// Snapshot 返回统计快照。
func (c *Collector) Snapshot() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	reports := c.buildReportsLocked()
	errors := make(map[string]int64, len(c.errors))
	for k, v := range c.errors {
		errors[k] = v
	}
	return map[string]interface{}{
		"reports":      reports,
		"errors":       errors,
		"last_report":  c.lastReport.Format(time.RFC3339),
		"generated_at": time.Now().Format(time.RFC3339),
	}
}

// buildReportsLocked 在持有读锁的情况下生成报告。
func (c *Collector) buildReportsLocked() []TypeReport {
	types := make([]string, 0, len(c.metrics))
	for k := range c.metrics {
		types = append(types, k)
	}
	sort.Strings(types)
	reports := make([]TypeReport, 0, len(types))
	for _, typ := range types {
		m := c.metrics[typ]
		reports = append(reports, TypeReport{
			Type:      typ,
			Count:     m.Count,
			AvgMillis: avgMillis(m),
			MinMillis: float64(m.MinNanos) / 1e6,
			MaxMillis: float64(m.MaxNanos) / 1e6,
		})
	}
	return reports
}

// Report 打印当前统计报告并返回摘要文本。
func (c *Collector) Report() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastReport = time.Now()
	var b strings.Builder
	b.WriteString("== mask stats ==\n")
	types := make([]string, 0, len(c.metrics))
	for k := range c.metrics {
		types = append(types, k)
	}
	sort.Strings(types)
	for _, typ := range types {
		m := c.metrics[typ]
		fmt.Fprintf(&b, "%s: count=%d avg=%.3fms min=%.3fms max=%.3fms\n",
			typ, m.Count, avgMillis(m), float64(m.MinNanos)/1e6, float64(m.MaxNanos)/1e6)
	}
	for kind, n := range c.errors {
		fmt.Fprintf(&b, "error[%s]: %d\n", kind, n)
	}
	return b.String()
}

// avgMillis 计算平均耗时（毫秒）。
func avgMillis(m *Metric) float64 {
	if m == nil || m.Count == 0 {
		return 0
	}
	return float64(m.TotalNanos) / float64(m.Count) / 1e6
}
