package model

import "time"

type User struct {
	Id        string
	Active    bool
	Gender    string
	CreatedAt time.Time
}
