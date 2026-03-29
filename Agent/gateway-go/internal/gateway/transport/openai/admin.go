package openai

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/admin"
)

func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	resp, err := s.admin.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminServices(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	resp, err := s.admin.Services(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": resp})
}

func (s *Server) handleAdminModels(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	resp, err := s.admin.Models(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminModes(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	resp, err := s.admin.Modes(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminSessionLogs(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	limit := parseLimit(r, 10)
	resp, err := s.admin.SessionLogs(limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

func (s *Server) handleAdminToolLogs(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	limit := parseLimit(r, 10)
	resp, err := s.admin.ToolLogs(limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

func (s *Server) handleAdminServiceStart(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req admin.ServiceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	if err := s.admin.StartService(r.Context(), req.Name); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": req.Name})
}

func (s *Server) handleAdminServiceStop(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req admin.ServiceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	if err := s.admin.StopService(r.Context(), req.Name); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": req.Name})
}

func (s *Server) handleAdminModelSwitch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		ProfileID string `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.ProfileID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}
	if err := s.admin.SwitchModel(r.Context(), req.ProfileID); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "profile_id": req.ProfileID})
}

func (s *Server) handleAdminModelUpdate(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req admin.UpdateModelProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Profile.ID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile.id is required")
		return
	}
	if err := s.admin.UpdateModelProfile(r.Context(), req.Profile, req.ApplyNow); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"profile":   req.Profile,
		"apply_now": req.ApplyNow,
	})
}

func (s *Server) handleAdminLlamaStart(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := s.admin.StartModel(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminLlamaStop(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := s.admin.StopModel(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminLlamaRestart(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := s.admin.RestartModel(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminHostIPs(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	resp, err := s.admin.HostIPs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "admin_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminUI(w http.ResponseWriter, r *http.Request) {
	distDir := strings.TrimSpace(s.cfg.Admin.UIDistDir)
	if distDir == "" {
		writeError(w, http.StatusServiceUnavailable, "admin_ui_unavailable", "admin ui dist directory is not configured")
		return
	}

	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusServiceUnavailable, "admin_ui_unavailable", "admin ui has not been built yet")
			return
		}
		writeError(w, http.StatusInternalServerError, "admin_ui_error", err.Error())
		return
	}

	requestPath := strings.TrimPrefix(r.URL.Path, "/admin")
	requestPath = strings.TrimPrefix(requestPath, "/")
	target := indexPath
	if requestPath != "" {
		candidate := filepath.Join(distDir, filepath.FromSlash(requestPath))
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			target = candidate
		}
	}
	http.ServeFile(w, r, target)
}

func parseLimit(r *http.Request, defaultValue int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}
	if value > 100 {
		return 100
	}
	return value
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}

	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	return false
}
