package users

import (
	"time"
)

/*
User represents a registered user in this application
*/
type User struct {
	Id        string    `form:"user" binding:"required" boltholdIndex:"Id"`
	Password  string    `form:"password" binding:"required"`
	Active    bool      `form:"" boltholdIndex:"Active"`
	Admin     bool      `form:""`
	Gender    string    `form:"gender" binding:"required"`
	Major     string    `form:"major" binding:"required"`
	Degree    string    `form:"degree" binding:"required"`
	CreatedAt time.Time `form:""`
}

/*
UserQuery is used to specify filter queries for users
*/
type UserQuery struct {
	ActiveEq bool
	GenderEq string
	MajorEq  string
	DegreeEq string
}

/*
UserSession represents a user's login session
*/
type UserSession struct {
	Token     string
	UserId    string
	CreatedAt time.Time
	LastSeen  time.Time
}

/*
Login represents the user's credentials sent during login
*/
type Login struct {
	UserId   string `form:"user" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type accountChange struct {
	OldPassword string `form:"old"`
	NewPassword string `form:"new"`
	Gender      string `form:"gender"`
	Major       string `form:"major"`
	Degree      string `form:"degree"`
}

type recaptchaClientRequest struct {
	GRecaptchaToken string `form:"grecaptcha-token" binding:"required"`
}

type recaptchaAPIResponse struct {
	Success bool `json:"success" binding:"required"`
}

type userValidator func(s *User) bool

type userResolver func(id string) (*User, error)

type userCredentialsValidator func(s *User) bool

type userSessionValidator func(s *UserSession) bool

/*
IsValid checks whether the user is valid according to a given validation function
*/
func (s *User) IsValid(validator userValidator) bool {
	return validator(s)
}

/*
HasValidCredentials checks whether the user has valid credentials according to a given validation function
*/
func (s *User) HasValidCredentials(validator userCredentialsValidator) bool {
	return validator(s)
}

/*
IsValid checks whether the session is valid according to a given validation function
*/
func (s *UserSession) IsValid(validator userSessionValidator) bool {
	return validator(s)
}
