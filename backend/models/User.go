package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type User struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Username         string     `json:"username" db:"username"`
	Email            string     `json:"email" db:"email"`
	PasswordHash     string     `json:"-" db:"password_hash"`
	FirstName        string     `json:"first_name" db:"first_name"`
	LastName         string     `json:"last_name" db:"last_name"`
	Role             string     `json:"role" db:"role"`
	BusinessName     string     `json:"business_name" db:"business_name"`
	Phone            string     `json:"phone" db:"phone"`
	Location         string     `json:"location" db:"location"`
	CountyID         *uuid.UUID `json:"county_id,omitempty" db:"county_id"`
	LightningAddress string     `json:"lightning_address" db:"lightning_address"`
	BalanceSats      int64      `json:"balance_sats" db:"balance_sats"`
	Rating           float64    `json:"rating" db:"rating"`
	CompletedPickups int        `json:"completed_pickups" db:"completed_pickups"`
	IsVerified       bool       `json:"is_verified" db:"is_verified"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

func (u User) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

type Users []User

func (u Users) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

func (u *User) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (u *User) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (u *User) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
