package actions

import (
	"net/http"
	"strings"

	"backend/models"
	"backend/services"
	"backend/utils"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type RegisterRequest struct {
	Name             string `json:"name"`
	Role             string `json:"role"`
	Username         string `json:"username"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	BusinessName     string `json:"businessName"`
	Phone            string `json:"phone"`
	Location         string `json:"location"`
	LightningAddress string `json:"lightningAddress"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func RegisterHandler(c buffalo.Context) error {
	var req RegisterRequest

	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "invalid request body",
		}))
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		req.Username = strings.TrimSpace(req.Email)
	}
	if req.FirstName == "" {
		req.FirstName = strings.TrimSpace(req.Name)
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	req.LightningAddress = strings.ToLower(strings.TrimSpace(req.LightningAddress))
	if req.Role == "" {
		req.Role = "farmer"
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.LightningAddress == "" {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "username, email, password and Blink Lightning Address are required",
		}))
	}
	addressParts := strings.Split(req.LightningAddress, "@")
	if len(addressParts) != 2 || addressParts[0] == "" || !strings.Contains(addressParts[1], ".") {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": "invalid Blink Lightning Address"}))
	}

	if !utils.IsValidEmail(req.Email) {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "invalid email address",
		}))
	}

	if len(req.Password) < 8 {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "password must be at least 8 characters",
		}))
	}
	if req.Role != "producer" && req.Role != "farmer" {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "role must be producer or farmer",
		}))
	}

	passwordHash, err := services.HashPassword(req.Password)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{
			"error": "failed to process password",
		}))
	}

	user := &models.User{
		ID:               uuid.Must(uuid.NewV4()),
		Username:         req.Username,
		Email:            req.Email,
		PasswordHash:     passwordHash,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Role:             req.Role,
		BusinessName:     strings.TrimSpace(req.BusinessName),
		Phone:            strings.TrimSpace(req.Phone),
		Location:         strings.TrimSpace(req.Location),
		LightningAddress: strings.TrimSpace(req.LightningAddress),
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

	c.Session().Set("user_id", user.ID.String())

	return c.Render(http.StatusCreated, r.JSON(map[string]interface{}{"user": map[string]interface{}{
		"id":               user.ID,
		"name":             strings.TrimSpace(user.FirstName + " " + user.LastName),
		"email":            user.Email,
		"role":             user.Role,
		"businessName":     user.BusinessName,
		"phone":            user.Phone,
		"location":         user.Location,
		"lightningAddress": user.LightningAddress,
		"balanceSats":      user.BalanceSats,
	}}))
}

func LoginHandler(c buffalo.Context) error {
	var req LoginRequest

	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "invalid request body",
		}))
	}

	identifier := strings.TrimSpace(req.Username)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" || req.Password == "" {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{
			"error": "email and password are required",
		}))
	}

	tx := c.Value("tx").(*pop.Connection)

	user := &models.User{}
	if err := tx.Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)", identifier, identifier).First(user); err != nil {
		return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{
			"error": "invalid username or password",
		}))
	}

	if !services.CheckPassword(req.Password, user.PasswordHash) {
		return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{
			"error": "invalid username or password",
		}))
	}

	c.Session().Set("user_id", user.ID.String())

	return c.Render(http.StatusOK, r.JSON(map[string]interface{}{"user": map[string]interface{}{
		"id":               user.ID,
		"name":             strings.TrimSpace(user.FirstName + " " + user.LastName),
		"email":            user.Email,
		"role":             user.Role,
		"businessName":     user.BusinessName,
		"phone":            user.Phone,
		"location":         user.Location,
		"lightningAddress": user.LightningAddress,
		"balanceSats":      user.BalanceSats,
	}}))
}

func MeHandler(c buffalo.Context) error {
	userID := c.Value("user_id").(uuid.UUID)

	tx := c.Value("tx").(*pop.Connection)

	user := &models.User{}
	if err := tx.Find(user, userID); err != nil {
		return c.Render(http.StatusNotFound, r.JSON(map[string]string{
			"error": "user not found",
		}))
	}

	return c.Render(http.StatusOK, r.JSON(map[string]interface{}{
		"id":               user.ID,
		"name":             strings.TrimSpace(user.FirstName + " " + user.LastName),
		"email":            user.Email,
		"role":             user.Role,
		"businessName":     user.BusinessName,
		"phone":            user.Phone,
		"location":         user.Location,
		"lightningAddress": user.LightningAddress,
		"balanceSats":      user.BalanceSats,
		"rating":           user.Rating,
		"completedPickups": user.CompletedPickups,
	}))
}

func LogoutHandler(c buffalo.Context) error {
	c.Session().Delete("user_id")

	return c.Render(http.StatusOK, r.JSON(map[string]string{
		"message": "logged out successfully",
	}))
}
