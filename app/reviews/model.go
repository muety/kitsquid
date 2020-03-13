package reviews

const KeyMainRating = "overall"

// TODO: View models!
type Review struct {
	Id      string           `json:"" boltholdKey:"Id"`
	EventId string           `json:"event_id" boltholdIndex:"EventId"`
	UserId  string           `json:"" boltholdIndex:"UserId"`
	Ratings map[string]uint8 `json:"ratings"`
}

type ReviewQuery struct {
	EventIdEq string
	UserIdEq  string
}
