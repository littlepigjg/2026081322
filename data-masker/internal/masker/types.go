package masker

import (
	"encoding/json"
	"errors"
	"strings"
)

// Type 表示脱敏类型。
type Type string

const (
	TypePhone    Type = "phone"
	TypeIDCard   Type = "idcard"
	TypeEmail    Type = "email"
	TypeBankCard Type = "bankcard"
	TypeCustom   Type = "custom"
)

// SupportedTypes 列出内置支持的脱敏类型。
var SupportedTypes = []Type{
	TypePhone,
	TypeIDCard,
	TypeEmail,
	TypeBankCard,
	TypeCustom,
}

// 错误定义
var (
	ErrNilItem          = errors.New("mask item is nil")
	ErrEmptyValue       = errors.New("value is required")
	ErrEmptyType        = errors.New("type is required")
	ErrUnknownType      = errors.New("unknown mask type")
	ErrCustomNeedParams = errors.New("custom type requires keep_prefix and keep_suffix")
	ErrInvalidKeep      = errors.New("keep_prefix and keep_suffix must be non-negative")
	ErrValueNotString   = errors.New("value must be a string")
	ErrBatchEmpty       = errors.New("items must not be empty")
	ErrBatchTooLarge    = errors.New("too many items in batch")
)

// MaxBatchSize 限制批量脱敏的最大条目数。
const MaxBatchSize = 1000

// MaskItem 表示一条待脱敏字段。
type MaskItem struct {
	Value      interface{} `json:"value"`
	Type       string      `json:"type"`
	KeepPrefix int         `json:"keep_prefix,omitempty"`
	KeepSuffix int         `json:"keep_suffix,omitempty"`
}

// MaskResult 表示单条脱敏结果。
type MaskResult struct {
	Masked   string `json:"masked"`
	Original string `json:"original"`
	Type     string `json:"type"`
}

// BatchRequest 是批量脱敏请求体。
type BatchRequest struct {
	Items []MaskItem `json:"items"`
}

// BatchResponse 是批量脱敏响应体。
type BatchResponse struct {
	Results []BatchResultItem `json:"results"`
	Total   int               `json:"total"`
}

// BatchResultItem 是批量结果中的单条记录。
type BatchResultItem struct {
	Masked   string `json:"masked"`
	Original string `json:"original,omitempty"`
	Type     string `json:"type,omitempty"`
	Error    string `json:"error,omitempty"`
}

// NormalizeType 将类型标准化为小写。
func NormalizeType(typ string) string {
	return strings.ToLower(strings.TrimSpace(typ))
}

// IsKnownType 判断是否为内置类型。
func IsKnownType(typ string) bool {
	switch NormalizeType(typ) {
	case string(TypePhone), string(TypeIDCard), string(TypeEmail),
		string(TypeBankCard), string(TypeCustom):
		return true
	default:
		return false
	}
}

// Validate 校验单条脱敏字段的基础参数。
func (item *MaskItem) Validate() error {
	if item == nil {
		return ErrNilItem
	}
	typ := NormalizeType(item.Type)
	if typ == "" {
		return ErrEmptyType
	}
	if !IsKnownType(typ) {
		return ErrUnknownType
	}
	if typ == string(TypeCustom) {
		if item.KeepPrefix < 0 || item.KeepSuffix < 0 {
			return ErrInvalidKeep
		}
	}
	return nil
}

// Clone 深拷贝一条脱敏字段。
func (item *MaskItem) Clone() *MaskItem {
	if item == nil {
		return nil
	}
	return &MaskItem{
		Value:      item.Value,
		Type:       item.Type,
		KeepPrefix: item.KeepPrefix,
		KeepSuffix: item.KeepSuffix,
	}
}

// ValidateBatch 校验批量请求。
func ValidateBatch(req *BatchRequest) error {
	if req == nil || len(req.Items) == 0 {
		return ErrBatchEmpty
	}
	if len(req.Items) > MaxBatchSize {
		return ErrBatchTooLarge
	}
	return nil
}

// ToJSON 将任意值序列化为 JSON 字节。
func ToJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// FromJSON 将 JSON 字节反序列化为任意值。
func FromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
