package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// MaskRule 描述单个脱敏类型的完整规则。
type MaskRule struct {
	MaskChar   string `json:"mask_char"`
	KeepPrefix int    `json:"keep_prefix"`
	KeepSuffix int    `json:"keep_suffix"`
}

// MaskRuleUpdate 用于部分更新，字段为指针以便区分“未提供”与“显式置零”。
type MaskRuleUpdate struct {
	MaskChar   *string `json:"mask_char"`
	KeepPrefix *int    `json:"keep_prefix"`
	KeepSuffix *int    `json:"keep_suffix"`
}

// Config 保存所有脱敏规则，并发安全。
type Config struct {
	mu    sync.RWMutex
	rules map[string]MaskRule
}

// DefaultRules 返回内置默认规则。
func DefaultRules() map[string]MaskRule {
	return map[string]MaskRule{
		"phone":    {MaskChar: "*", KeepPrefix: 3, KeepSuffix: 4},
		"idcard":   {MaskChar: "*", KeepPrefix: 6, KeepSuffix: 4},
		"email":    {MaskChar: "*", KeepPrefix: 1, KeepSuffix: 0},
		"bankcard": {MaskChar: "*", KeepPrefix: 4, KeepSuffix: 4},
		"custom":   {MaskChar: "*", KeepPrefix: 0, KeepSuffix: 0},
	}
}

// New 创建一个使用默认规则的配置。
func New() *Config {
	return &Config{rules: DefaultRules()}
}

// NewFrom 使用给定规则创建配置（深拷贝）。
func NewFrom(rules map[string]MaskRule) *Config {
	c := &Config{rules: make(map[string]MaskRule, len(rules))}
	for k, v := range rules {
		c.rules[k] = v
	}
	return c
}

// Get 返回指定类型的规则及其是否存在。
func (c *Config) Get(typ string) (MaskRule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rule, ok := c.rules[typ]
	return rule, ok
}

// All 返回所有规则的快照。
func (c *Config) All() map[string]MaskRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]MaskRule, len(c.rules))
	for k, v := range c.rules {
		out[k] = v
	}
	return out
}

// Update 部分更新规则，仅覆盖提供的字段。
func (c *Config) Update(partial map[string]MaskRuleUpdate) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for typ, incoming := range partial {
		typ = strings.ToLower(strings.TrimSpace(typ))
		existing, ok := c.rules[typ]
		if !ok {
			existing = MaskRule{MaskChar: "*", KeepPrefix: 0, KeepSuffix: 0}
		}
		if incoming.MaskChar != nil {
			existing.MaskChar = *incoming.MaskChar
		}
		if incoming.KeepPrefix != nil {
			existing.KeepPrefix = *incoming.KeepPrefix
		}
		if incoming.KeepSuffix != nil {
			existing.KeepSuffix = *incoming.KeepSuffix
		}
		if err := validateRule(existing); err != nil {
			return fmt.Errorf("type %q: %w", typ, err)
		}
		c.rules[typ] = existing
	}
	return nil
}

// SetFull 完整替换某个类型的规则。
func (c *Config) SetFull(typ string, rule MaskRule) error {
	if err := validateRule(rule); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules[typ] = rule
	return nil
}

// Remove 删除某个类型的规则。
func (c *Config) Remove(typ string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rules, typ)
}

// Types 返回排序后的类型列表。
func (c *Config) Types() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	types := make([]string, 0, len(c.rules))
	for k := range c.rules {
		types = append(types, k)
	}
	sort.Strings(types)
	return types
}

// SnapshotJSON 返回规则的 JSON 序列化字节。
func (c *Config) SnapshotJSON() ([]byte, error) {
	return json.MarshalIndent(c.All(), "", "  ")
}

// Load 从 JSON 字节加载配置。
func (c *Config) Load(data []byte) error {
	var rules map[string]MaskRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	for typ, rule := range rules {
		if err := validateRule(rule); err != nil {
			return fmt.Errorf("type %q: %w", typ, err)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = rules
	return nil
}

// LoadFromFile 从文件加载配置。
func (c *Config) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	return c.Load(data)
}

// SaveToFile 将当前配置保存到文件。
func (c *Config) SaveToFile(path string) error {
	data, err := c.SnapshotJSON()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// validateRule 校验规则合法性。
func validateRule(rule MaskRule) error {
	if rule.MaskChar == "" {
		return errors.New("mask_char must not be empty")
	}
	if len([]rune(rule.MaskChar)) != 1 {
		return errors.New("mask_char must be a single character")
	}
	if rule.KeepPrefix < 0 {
		return errors.New("keep_prefix must be non-negative")
	}
	if rule.KeepSuffix < 0 {
		return errors.New("keep_suffix must be non-negative")
	}
	return nil
}
