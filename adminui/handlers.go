package adminui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/adminui/admintypes"
	"github.com/grishy/any-sync-bundle/adminui/views/pages"
)

// Handlers manages HTTP request handling.
type Handlers struct {
	service *Service
}

// newHandlers creates new handlers with the service.
func newHandlers(service *Service) *Handlers {
	return &Handlers{
		service: service,
	}
}

// handleIndex handles the dashboard/home page.
func (h *Handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetSystemStats(r.Context())
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	h.render(w, r, pages.IndexPage(stats, "Admin Dashboard", ""))
}

// handleSearch handles user search.
func (h *Handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		identity := strings.TrimSpace(r.FormValue("identity"))
		if identity == "" {
			h.renderError(w, r, "Identity is required")
			return
		}

		// Redirect to user detail page
		http.Redirect(w, r, "/admin/user/"+identity, http.StatusSeeOther)
		return
	}

	// GET - show search form
	h.render(w, r, pages.SearchPage("User Search"))
}

// handleUserDetail handles user detail page.
func (h *Handlers) handleUserDetail(w http.ResponseWriter, r *http.Request) {
	identity := r.PathValue("identity")
	if identity == "" {
		h.renderError(w, r, "Identity not specified")
		return
	}

	userInfo, err := h.service.GetUserInfo(r.Context(), identity)
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	// Check for success message
	message := ""
	if r.URL.Query().Get("updated") == "true" {
		message = "Quota updated successfully!"
	}

	// THIS IS THE FIX: Pass UserInfo directly, not wrapped in interface{}
	h.render(w, r, pages.UserPage(userInfo, "User: "+identity, message))
}

// handleQuotaEdit handles quota editing.
func (h *Handlers) handleQuotaEdit(w http.ResponseWriter, r *http.Request) {
	identity := r.PathValue("identity")
	if identity == "" {
		h.renderError(w, r, "Identity not specified")
		return
	}

	if r.Method == http.MethodPost {
		// Parse form values
		storageGB, err := strconv.ParseFloat(r.FormValue("storage_gb"), 64)
		if err != nil {
			h.renderError(w, r, "Invalid storage value")
			return
		}

		membersRead, _ := strconv.ParseUint(r.FormValue("members_read"), 10, 32)
		membersWrite, _ := strconv.ParseUint(r.FormValue("members_write"), 10, 32)
		sharedSpaces, _ := strconv.ParseUint(r.FormValue("shared_spaces"), 10, 32)
		reason := strings.TrimSpace(r.FormValue("reason"))

		// Validate
		if reason == "" {
			h.renderError(w, r, "Reason is required for quota changes")
			return
		}

		// Update quota
		req := admintypes.QuotaUpdateRequest{
			Identity:     identity,
			StorageBytes: uint64(storageGB * 1024 * 1024 * 1024),
			MembersRead:  uint32(membersRead),
			MembersWrite: uint32(membersWrite),
			SharedSpaces: uint32(sharedSpaces),
			Reason:       reason,
		}

		if quotaErr := h.service.SetUserQuota(r.Context(), req); quotaErr != nil {
			h.renderError(w, r, quotaErr)
			return
		}

		// Redirect back to user page with success message
		http.Redirect(w, r, "/admin/user/"+identity+"?updated=true", http.StatusSeeOther)
		return
	}

	// GET - show edit form
	userInfo, err := h.service.GetUserInfo(r.Context(), identity)
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	h.render(w, r, pages.QuotaPage(userInfo, "Edit Quota: "+identity))
}

// handleSpacesList handles spaces list page.
func (h *Handlers) handleSpacesList(w http.ResponseWriter, r *http.Request) {
	identity := r.URL.Query().Get("identity")
	if identity == "" {
		h.renderError(w, r, "Identity not specified")
		return
	}

	spaces, err := h.service.GetSpacesByIdentity(r.Context(), identity)
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	h.render(w, r, pages.SpacesPage(spaces, identity, "Spaces for: "+identity))
}

// handleDeletions handles deletion log page.
func (h *Handlers) handleDeletions(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.GetDeletionLog(r.Context())
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	h.render(w, r, pages.DeletionsPage(entries, "Deletion Log"))
}

// handleHealth handles health check endpoint.
func (h *Handlers) handleHealth(w http.ResponseWriter, _ *http.Request) {
	// Simple health check - could be extended to check service availability
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy","service":"adminui"}`))
}

// render renders a templ component.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		log.Error("template rendering failed", zap.Error(err))
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// handleAllUsers handles the all users list page.
func (h *Handlers) handleAllUsers(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	users, total, err := h.service.GetAllUsers(r.Context(), page)
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	h.render(w, r, pages.AllUsersPage(users, page, total))
}

// handleAllSpacesList handles the all spaces list page.
func (h *Handlers) handleAllSpacesList(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	filterType := -1
	if t := r.URL.Query().Get("type"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			filterType = parsed
		}
	}

	filterStatus := -1
	if s := r.URL.Query().Get("status"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil {
			filterStatus = parsed
		}
	}

	spaces, total, err := h.service.GetAllSpacesWithPagination(r.Context(), page, filterType, filterStatus)
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	h.render(w, r, pages.AllSpacesPage(spaces, page, total, filterType, filterStatus))
}

// renderError renders an error page.
func (h *Handlers) renderError(w http.ResponseWriter, r *http.Request, err interface{}) {
	log.Warn("rendering error", zap.String("path", r.URL.Path), zap.Any("error", err))

	w.WriteHeader(http.StatusInternalServerError)
	h.render(w, r, pages.ErrorPage(fmt.Sprintf("%v", err), "Error"))
}
