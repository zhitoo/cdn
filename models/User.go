package models

import "time"

type User struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	FirstName        *string    `json:"first_name"`
	LastName         *string    `json:"last_name"`
	UserName         string     `json:"user_name" gorm:"unique"`
	Mobile           *string    `json:"mobile" gorm:"unique"`
	Email            *string    `json:"email" gorm:"unique"`
	Balance          int64      `json:"balance"`
	Admin            *bool      `json:"admin"`
	Active           *bool      `json:"active"`
	Password         string     `json:"password"`
	MobileVerifiedAt *time.Time `json:"mobile_verified_at"`
	EmailVerifiedAt  *time.Time `json:"email_verified_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `gorm:"deleted_at"`
}
