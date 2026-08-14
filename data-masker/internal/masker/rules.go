package masker

import (
	"strings"
	"unicode/utf8"
)

// applyGenericMask 按前缀/后缀保留规则掩码字符串。
func applyGenericMask(value, maskChar string, keepPrefix, keepSuffix int) string {
	if value == "" {
		return value
	}
	maskChar = ensureMaskChar(maskChar)
	runes := []rune(value)
	n := len(runes)
	prefix := clampKeep(keepPrefix, n)
	suffix := clampKeep(keepSuffix, n)

	if prefix+suffix > n {
		suffix = n - prefix
		if suffix < 0 {
			suffix = 0
		}
	}

	maskLen := n - prefix - suffix
	if maskLen <= 0 {
		return value
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(string(runes[:prefix]))
	b.WriteString(strings.Repeat(maskChar, maskLen))
	b.WriteString(string(runes[n-suffix:]))
	return b.String()
}

// applyEmailMask 掩码邮箱本地部分（保留首字符）。
func applyEmailMask(value, maskChar string) string {
	if value == "" {
		return value
	}
	maskChar = ensureMaskChar(maskChar)
	at := strings.Index(value, "@")
	if at <= 0 {
		return applyGenericMask(value, maskChar, 1, 0)
	}
	local := value[:at]
	domain := value[at:]
	runes := []rune(local)
	if len(runes) <= 1 {
		return value
	}
	var b strings.Builder
	b.WriteString(string(runes[0]))
	b.WriteString(strings.Repeat(maskChar, maskEmailCount(len(runes))))
	b.WriteString(domain)
	return b.String()
}

// maskEmailCount 返回邮箱本地部分掩码字符数。
func maskEmailCount(localLen int) int {
	if localLen <= 1 {
		return 0
	}
	return 4
}

// ensureMaskChar 返回有效掩码字符。
func ensureMaskChar(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "*"
	}
	r, _ := utf8.DecodeRuneInString(c)
	if r == utf8.RuneError && c != string(utf8.RuneError) {
		return "*"
	}
	return string(r)
}

// clampKeep 将保留长度限制在 [0, max]。
func clampKeep(k, max int) int {
	if k < 0 {
		return 0
	}
	if k > max {
		return max
	}
	return k
}

// maskAll 将字符串全部掩码。
func maskAll(value, maskChar string) string {
	maskChar = ensureMaskChar(maskChar)
	return strings.Repeat(maskChar, utf8.RuneCountInString(value))
}

// maskMiddleKeepOne 仅保留首尾各一个字符。
func maskMiddleKeepOne(value, maskChar string) string {
	maskChar = ensureMaskChar(maskChar)
	runes := []rune(value)
	if len(runes) <= 2 {
		return value
	}
	return string(runes[0]) + strings.Repeat(maskChar, len(runes)-2) + string(runes[len(runes)-1])
}

// maskSuffixOnly 仅掩码后缀，保留前缀。
func maskSuffixOnly(value, maskChar string, keepPrefix int) string {
	maskChar = ensureMaskChar(maskChar)
	runes := []rune(value)
	n := len(runes)
	prefix := clampKeep(keepPrefix, n)
	if prefix >= n {
		return value
	}
	return string(runes[:prefix]) + strings.Repeat(maskChar, n-prefix)
}

// maskPrefixOnly 仅掩码前缀，保留后缀。
func maskPrefixOnly(value, maskChar string, keepSuffix int) string {
	maskChar = ensureMaskChar(maskChar)
	runes := []rune(value)
	n := len(runes)
	suffix := clampKeep(keepSuffix, n)
	if suffix >= n {
		return value
	}
	return strings.Repeat(maskChar, n-suffix) + string(runes[n-suffix:])
}

// countDigits 统计字符串中数字字符个数。
func countDigits(value string) int {
	n := 0
	for _, r := range value {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}
