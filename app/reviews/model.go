package reviews

type Review struct {
	Id      string
	EventId string
	Comment string
	Ratings map[string]uint8
}
