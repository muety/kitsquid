package comments

import "time"

type Comment struct {
	Id        string    `form:"" boltholdIndex:"Id"`
	Index     uint8     `form:"" boltholdIndex:"Index"`
	EventId   string    `form:"event_id" binding:"required" boltholdIndex:"EventId"`
	UserId    string    `form:"" boltholdIndex:"UserId"`
	Active    bool      `form:"" boltholdIndex:"Active"`
	Text      string    `form:"text" binding:"required"`
	CreatedAt time.Time `form:""`
}

type CommentQuery struct {
	EventIdEq string
	UserIdEq  string
	ActiveEq  bool
	Skip      int
	Limit     int
}
