package users

import (
	"time"
)

type User struct {
	Id        string    `form:"user" binding:"required"`
	Password  string    `form:"password" binding:"required"`
	Active    bool      `form:""`
	Gender    string    `form:"gender" binding:"required"`
	Major     string    `form:"major" binding:"required"`
	Degree    string    `form:"degree" binding:"required"`
	CreatedAt time.Time `form:""`
}

type UserValidator func(s *User) bool

func (s *User) IsValid(validator UserValidator) bool {
	return validator(s)
}
