package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// routes 注册所有路由。
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /mask", s.handleMask)
	mux.HandleFunc("POST /mask/batch", s.handleMaskBatch)
	mux.HandleFunc("POST /mask/object", s.handleMaskObject)
	mux.HandleFunc("GET /config", s.handleConfig)
	mux.HandleFunc("POST /config/update", s.requireAdmin(s.handleConfigUpdate))
	mux.HandleFunc("GET /stats", s.handleStats)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /", s.handleIndex)
	return s.requestLogger(mux)
}

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError 写入结构化错误响应。
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// readJSON 读取并解析 JSON 请求体到 dst，并拒绝多余内容。
func readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected trailing data in json body")
	}
	return nil
}

// decodeObject 读取任意 JSON 值（数字保留为 json.Number）。
func decodeObject(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	var obj interface{}
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, errors.New("unexpected trailing data in json body")
	}
	return obj, nil
}

// contentTypeIsJSON 判断请求 Content-Type 是否为 JSON。
func contentTypeIsJSON(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return true
	}
	return strings.Contains(ct, "application/json")
}

// queryType 读取 query 参数 type。
func queryType(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("type"))
}

// queryInt 读取 query 中的整型参数，默认返回 fallback。
func queryInt(r *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

// methodNotAllowed 返回 405 响应。
func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// respondNoContent 返回 204 响应。
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// applyCommonHeaders 设置通用响应头。
func applyCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// writeInternalError 返回 500 响应。
func writeInternalError(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusInternalServerError, msg)
}

// truncate 截断字符串到指定长度。
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// extractClientIP 从请求头或 RemoteAddr 中提取客户端 IP。
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	return r.RemoteAddr
}

// sanitizeType 规范化类型字符串。
func sanitizeType(typ string) string {
	return strings.ToLower(strings.TrimSpace(typ))
}

// isEmptyBody 判断请求体是否为空。
func isEmptyBody(r *http.Request) bool {
	return r.Body == nil || r.ContentLength == 0
}
