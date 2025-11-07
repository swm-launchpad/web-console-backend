package model

import (
	"time"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
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
	PasswordHash      string
	PasswordUpdatedAt *time.Time
	Nickname          string
	Email             string
	Phone             *string
	Organization      *string
	Status            UserStatus
	IsDeleted         bool
	DeletedAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         *time.Time
}

func NewUser(email, nickname string) (*User, error) {
	if email == "" {
		return nil, usererrors.ErrEmailRequired
	}
	if nickname == "" {
		return nil, usererrors.ErrNicknameRequired
	}
	if len(nickname) < 2 {
		return nil, usererrors.ErrNicknameTooShort
	}

	now := time.Now()
	return &User{
		Email:     email,
		Nickname:  nickname,
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
		return usererrors.ErrCannotActivateDeletedUser
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
