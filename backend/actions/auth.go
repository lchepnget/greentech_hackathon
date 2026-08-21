package actions

import (
	"net/http"
	"strings"

	"backend/models"
	"backend/services"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type RegisterRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func RegisterHandler(c buffalo.Context) error {
	var req RegisterRequest

	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "invalid request body",
		}))
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "username, email and password are required",
		}))
	}

	if len(req.Password) < 8 {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "password must be at least 8 characters",
		}))
	}

	passwordHash, err := services.HashPassword(req.Password)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{
			"error": "failed to process password",
		}))
	}

	user := &models.User{
		ID:           uuid.Must(uuid.NewV4()),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
	}

	tx := c.Value("tx").(*pop.Connection)

	if err := tx.Create(user); err != nil {
		c.Logger().Errorf("failed to create user: %v", err)

		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return c.Render(http.StatusConflict, r.JSON(map[string]string{
				"error": "username or email already exists",
			}))
		}

		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{
			"error": "failed to create user",
		}))
	}

	return c.Render(http.StatusCreated, r.JSON(map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}))
}
