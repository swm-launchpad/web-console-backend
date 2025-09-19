package model

import (
	"errors"
	"time"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusPending   UserStatus = "pending"
)

type User struct {
	UserID            uint
	Username          string
	PasswordHash      string
	PasswordUpdatedAt *time.Time
	Name              *string
	Email             string
	Phone             *string
	Organization      *string
	Status            UserStatus
	IsDeleted         bool
	DeletedAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         *time.Time
}

func NewUser(username, email string) (*User, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}

	now := time.Now()
	return &User{
		Username:  username,
		Email:     email,
		Status:    UserStatusPending,
		IsDeleted: false,
		CreatedAt: now,
		UpdatedAt: &now,
	}, nil
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && !u.IsDeleted
}

func (u *User) Activate() error {
	if u.IsDeleted {
		return errors.New("cannot activate deleted user")
	}
	u.Status = UserStatusActive
	now := time.Now()
	u.UpdatedAt = &now
	return nil
}

func (u *User) UpdatePassword(passwordHash string) {
	u.PasswordHash = passwordHash
	now := time.Now()
	u.PasswordUpdatedAt = &now
	u.UpdatedAt = &now
}
