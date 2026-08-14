package masker

import (
	"strconv"
	"strings"

	"data-masker/internal/config"
)

// Masker 是数据脱敏的核心处理器。
type Masker struct {
	config *config.Config
}

// New 创建一个新的 Masker，若配置为 nil 则使用默认配置。
func New(cfg *config.Config) *Masker {
	if cfg == nil {
		cfg = config.New()
	}
	return &Masker{config: cfg}
}

// Config 返回底层配置。
func (m *Masker) Config() *config.Config {
	return m.config
}

// MaskField 对单个字段执行脱敏。
func (m *Masker) MaskField(item *MaskItem) (*MaskResult, error) {
	if item == nil {
		return nil, ErrNilItem
	}

	if err := item.Validate(); err != nil {
		return nil, err
	}

	v := item.Value
	value := v.(string) // BUG mask-other-001: 未使用 comma-ok 模式进行类型断言

	if strings.TrimSpace(value) == "" {
		return nil, ErrEmptyValue
	}

	typ := NormalizeType(item.Type)
	rule := m.ruleFor(typ, item.KeepPrefix, item.KeepSuffix)

	masked := m.applyForType(value, typ, rule)

	return &MaskResult{
		Masked:   masked,
		Original: value,
		Type:     typ,
	}, nil
}

// MaskValue 对单个字符串值脱敏，供对象递归与批量使用。
func (m *Masker) MaskValue(value, typ string) string {
	typ = NormalizeType(typ)
	if typ == "" {
		typ = string(TypePhone)
	}
	rule := m.ruleFor(typ, 0, 0)
	return m.applyForType(value, typ, rule)
}

// MaskString 对给定字符串按类型脱敏并返回结果。
func (m *Masker) MaskString(value, typ string) (string, error) {
	item := &MaskItem{Value: value, Type: typ}
	res, err := m.MaskField(item)
	if err != nil {
		return "", err
	}
	return res.Masked, nil
}

// MaskItemJSON 从 JSON 字节解码单条字段并脱敏。
func (m *Masker) MaskItemJSON(data []byte) (*MaskResult, error) {
	item := &MaskItem{}
	if err := FromJSON(data, item); err != nil {
		return nil, err
	}
	return m.MaskField(item)
}

// ruleFor 解析某个类型的有效规则。
func (m *Masker) ruleFor(typ string, keepPrefix, keepSuffix int) config.MaskRule {
	if rule, ok := m.config.Get(typ); ok {
		if typ == string(TypeCustom) {
			rule.KeepPrefix = keepPrefix
			rule.KeepSuffix = keepSuffix
		}
		return rule
	}
	return config.MaskRule{MaskChar: "*", KeepPrefix: 3, KeepSuffix: 4}
}

// applyForType 根据类型选择掩码算法。
func (m *Masker) applyForType(value, typ string, rule config.MaskRule) string {
	maskChar := ensureMaskChar(rule.MaskChar)
	if typ == string(TypeEmail) {
		return applyEmailMask(value, maskChar)
	}
	return applyGenericMask(value, maskChar, rule.KeepPrefix, rule.KeepSuffix)
}

// maskCharFor 返回某类型的掩码字符。
func (m *Masker) maskCharFor(typ string) string {
	if rule, ok := m.config.Get(typ); ok {
		return ensureMaskChar(rule.MaskChar)
	}
	return "*"
}

// RuleSummary 返回某类型规则的人类可读摘要。
func (m *Masker) RuleSummary(typ string) string {
	rule, ok := m.config.Get(typ)
	if !ok {
		return "unknown"
	}
	return strings.Join([]string{
		"mask_char=" + ensureMaskChar(rule.MaskChar),
		"keep_prefix=" + strconv.Itoa(rule.KeepPrefix),
		"keep_suffix=" + strconv.Itoa(rule.KeepSuffix),
	}, ", ")
}

// ResolveType 将类型标准化，空类型回退为 phone。
func (m *Masker) ResolveType(typ string) string {
	typ = NormalizeType(typ)
	if typ == "" {
		return string(TypePhone)
	}
	return typ
}

// MaskCustom 按自定义前缀/后缀长度掩码。
func (m *Masker) MaskCustom(value string, keepPrefix, keepSuffix int) string {
	rule := m.ruleFor(string(TypeCustom), keepPrefix, keepSuffix)
	return m.applyForType(value, string(TypeCustom), rule)
}

// SupportedTypes 返回支持的脱敏类型列表。
func (m *Masker) SupportedTypes() []string {
	out := make([]string, 0, len(SupportedTypes))
	for _, t := range SupportedTypes {
		out = append(out, string(t))
	}
	return out
}
