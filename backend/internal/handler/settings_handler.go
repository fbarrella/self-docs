package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

// SettingsHandler serves the settings API (docs/api.md section 10).
type SettingsHandler struct {
	settings *repository.SettingsRepository
	sessions *private.SessionStore
	version  string
	redis    bool
}

// NewSettingsHandler constructs a SettingsHandler.
func NewSettingsHandler(
	settings *repository.SettingsRepository,
	sessions *private.SessionStore,
	version string,
	redisEnabled bool,
) *SettingsHandler {
	return &SettingsHandler{settings: settings, sessions: sessions, version: version, redis: redisEnabled}
}

// Get handles GET /api/settings.
func (h *SettingsHandler) Get(c *gin.Context) {
	profile, err := h.settings.GetProfile(c.Request.Context())
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"version":              h.version,
			"redis_enabled":        h.redis,
			"master_password_set":  h.masterPasswordSet(c),
			"private_session_open": h.privateSessionOpen(c),
			"profile": gin.H{
				"first_name": profile.FirstName,
				"last_name":  profile.LastName,
			},
		},
	})
}

// profileRequest is the PUT /api/settings/profile body.
type profileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UpdateProfile handles PUT /api/settings/profile. Blank names reset the
// profile to the default John Doe.
func (h *SettingsHandler) UpdateProfile(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	if len(firstName) > 100 || len(lastName) > 100 {
		respondValidation(c, "name is too long",
			fieldError{Field: "first_name", Issue: "must be at most 100 characters"})
		return
	}

	profile := repository.Profile{FirstName: firstName, LastName: lastName}
	if err := h.settings.SetProfile(c.Request.Context(), profile); err != nil {
		handleRepoError(c, err)
		return
	}

	saved, err := h.settings.GetProfile(c.Request.Context())
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"first_name": saved.FirstName,
			"last_name":  saved.LastName,
		},
	})
}

// changePasswordRequest is the PUT /api/settings/master-password body.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeMasterPassword handles PUT /api/settings/master-password. It requires
// either a valid private session or the correct current password.
func (h *SettingsHandler) ChangeMasterPassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	if len(req.NewPassword) < 8 {
		respondValidation(c, "new password is too short",
			fieldError{Field: "new_password", Issue: "must be at least 8 characters"})
		return
	}

	authorized, err := h.authorized(c, req.CurrentPassword)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	if !authorized {
		respondError(c, http.StatusUnauthorized, CodeUnauthorized,
			"current master password is incorrect")
		return
	}

	hash, err := private.HashPassword(req.NewPassword)
	if err != nil {
		if errors.Is(err, private.ErrPasswordTooLong) {
			respondValidation(c, "new password is too long",
				fieldError{Field: "new_password", Issue: "must be at most 72 bytes"})
			return
		}
		respondInternal(c, err)
		return
	}

	if err := h.settings.SetMasterPasswordHash(c.Request.Context(), hash); err != nil {
		respondInternal(c, err)
		return
	}
	// Invalidate every session so old unlocks cannot continue.
	h.sessions.Clear()
	c.Status(http.StatusNoContent)
}

// authorized reports whether the caller may change the master password.
func (h *SettingsHandler) authorized(c *gin.Context, currentPassword string) (bool, error) {
	if valid, _ := h.sessions.Touch(tokenFromRequest(c)); valid {
		return true, nil
	}

	hash, err := h.settings.GetMasterPasswordHash(c.Request.Context())
	if err != nil {
		if errors.Is(err, repository.ErrMasterPasswordUnset) {
			// Nothing set yet: allow the first configuration.
			return true, nil
		}
		return false, err
	}
	if currentPassword == "" {
		return false, nil
	}
	return private.VerifyPassword(hash, currentPassword), nil
}

func (h *SettingsHandler) masterPasswordSet(c *gin.Context) bool {
	_, err := h.settings.GetMasterPasswordHash(c.Request.Context())
	return err == nil
}

func (h *SettingsHandler) privateSessionOpen(c *gin.Context) bool {
	valid, _ := h.sessions.Touch(tokenFromRequest(c))
	return valid
}
