package masker

import (
	"fmt"
	"strings"
)

// MaskBatch 批量脱敏多个字段（顺序处理，保证结果顺序）。
func (m *Masker) MaskBatch(req *BatchRequest) (*BatchResponse, error) {
	if err := ValidateBatch(req); err != nil {
		return nil, err
	}

	results := make([]BatchResultItem, 0, len(req.Items))
	for i := range req.Items {
		item := &req.Items[i]
		res, err := m.MaskField(item)
		if err != nil {
			results = append(results, BatchResultItem{Error: err.Error()})
			continue
		}
		results = append(results, toBatchResultItem(res))
	}
	return &BatchResponse{Results: results, Total: len(results)}, nil
}

// BatchRequestFromJSON 解析批量请求。
func BatchRequestFromJSON(data []byte) (*BatchRequest, error) {
	req := &BatchRequest{}
	if err := FromJSON(data, req); err != nil {
		return nil, fmt.Errorf("decode batch request: %w", err)
	}
	return req, nil
}

// BatchResponseToJSON 序列化批量响应。
func BatchResponseToJSON(resp *BatchResponse) ([]byte, error) {
	if resp == nil {
		return ToJSON(&BatchResponse{})
	}
	return ToJSON(resp)
}

// MaskBatchValues 对一批字符串按统一类型脱敏。
func (m *Masker) MaskBatchValues(values []string, typ string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, m.MaskValue(v, typ))
	}
	return out
}

// MaskBatchMap 对 map 中的一批字符串字段按统一类型脱敏。
func (m *Masker) MaskBatchMap(fields map[string]string, typ string) map[string]string {
	out := make(map[string]string, len(fields))
	for k, v := range fields {
		out[k] = m.MaskValue(v, typ)
	}
	return out
}

// toBatchResultItem 将单条结果转为批量条目。
func toBatchResultItem(res *MaskResult) BatchResultItem {
	if res == nil {
		return BatchResultItem{Error: ErrNilItem.Error()}
	}
	return BatchResultItem{
		Masked:   res.Masked,
		Original: res.Original,
		Type:     res.Type,
	}
}

// countErrors 统计批量结果中的错误数量。
func countErrors(results []BatchResultItem) int {
	n := 0
	for _, r := range results {
		if r.Error != "" {
			n++
		}
	}
	return n
}

// summarizeTypes 汇总批量结果中的类型分布。
func summarizeTypes(results []BatchResultItem) map[string]int {
	out := make(map[string]int)
	for _, r := range results {
		if r.Type == "" {
			continue
		}
		out[r.Type]++
	}
	return out
}

// joinErrors 拼接所有错误信息。
func joinErrors(results []BatchResultItem) string {
	var b strings.Builder
	for i, r := range results {
		if r.Error == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "item[%d]: %s", i, r.Error)
	}
	return b.String()
}

// NewBatchRequest 创建批量请求。
func NewBatchRequest(items []MaskItem) *BatchRequest {
	return &BatchRequest{Items: items}
}

// Len 返回条目数量。
func (r *BatchRequest) Len() int {
	if r == nil {
		return 0
	}
	return len(r.Items)
}

// Append 追加一条脱敏字段。
func (r *BatchRequest) Append(item MaskItem) {
	r.Items = append(r.Items, item)
}

// Succeeded 返回成功条数。
func (r *BatchResponse) Succeeded() int {
	if r == nil {
		return 0
	}
	return r.Total - r.Failed()
}

// Failed 返回失败条数。
func (r *BatchResponse) Failed() int {
	if r == nil {
		return 0
	}
	return countErrors(r.Results)
}

// BatchStats 表示批量处理的统计。
type BatchStats struct {
	Total     int
	Succeeded int
	Failed    int
}

// Stats 返回批量处理统计。
func (r *BatchResponse) Stats() BatchStats {
	return BatchStats{
		Total:     r.Total,
		Succeeded: r.Succeeded(),
		Failed:    r.Failed(),
	}
}
