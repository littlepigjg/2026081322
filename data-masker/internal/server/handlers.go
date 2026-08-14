package server

import (
	"net/http"
	"time"

	"data-masker/internal/config"
	"data-masker/internal/masker"
)

// handleMask 处理 POST /mask。
func (s *Server) handleMask(w http.ResponseWriter, r *http.Request) {
	if !contentTypeIsJSON(r) {
		writeError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	item := &masker.MaskItem{}
	if err := readJSON(w, r, item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	start := time.Now()
	res, err := s.maskField(item)
	dur := time.Since(start)
	if err != nil {
		s.stats.RecordError("mask:" + err.Error())
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.stats.Record(masker.NormalizeType(item.Type), dur)
	writeJSON(w, http.StatusOK, res)
}

// handleMaskBatch 处理 POST /mask/batch。
func (s *Server) handleMaskBatch(w http.ResponseWriter, r *http.Request) {
	req := &masker.BatchRequest{}
	if err := readJSON(w, r, req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	start := time.Now()
	resp, err := s.maskBatch(req)
	dur := time.Since(start)
	if err != nil {
		s.stats.RecordError("batch:" + err.Error())
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.stats.Record("batch", dur)
	writeJSON(w, http.StatusOK, resp)
}

// handleMaskObject 处理 POST /mask/object。
func (s *Server) handleMaskObject(w http.ResponseWriter, r *http.Request) {
	obj, err := decodeObject(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	typ := queryType(r)

	start := time.Now()
	masked := s.masker.MaskObject(obj, typ)
	dur := time.Since(start)
	s.stats.Record("object:"+masker.NormalizeType(typ), dur)
	writeJSON(w, http.StatusOK, masked)
}

// handleConfig 处理 GET /config。
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.config.All())
}

// handleConfigUpdate 处理 POST /config/update（由 requireAdmin 保护）。
func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	var partial map[string]config.MaskRuleUpdate
	if err := readJSON(w, r, &partial); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := s.config.Update(partial); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"updated": true,
		"config":  s.config.All(),
	})
}

// handleStats 处理 GET /stats。
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.stats.Snapshot())
}

// maskField 在隔离 goroutine 中执行单字段脱敏，使未恢复的 panic 直接终止进程。
func (s *Server) maskField(item *masker.MaskItem) (*masker.MaskResult, error) {
	var res *masker.MaskResult
	var err error
	runIsolated(func() {
		res, err = s.masker.MaskField(item)
	})
	return res, err
}

// maskBatch 在隔离 goroutine 中执行批量脱敏，使未恢复的 panic 直接终止进程。
func (s *Server) maskBatch(req *masker.BatchRequest) (*masker.BatchResponse, error) {
	var resp *masker.BatchResponse
	var err error
	runIsolated(func() {
		resp, err = s.masker.MaskBatch(req)
	})
	return resp, err
}

// handleHealth 处理 GET /health。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleIndex 处理 GET /，返回可用端点列表。
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service": "data-masker",
		"endpoints": []string{
			"POST /mask",
			"POST /mask/batch",
			"POST /mask/object",
			"GET /config",
			"POST /config/update",
			"GET /stats",
			"GET /health",
		},
	})
}
