package errors

type KitSquidError error

type NotFound struct{}

type BadRequest struct{}

type Conflict struct{}

type Internal struct{}

type Unauthorized struct{}

func (n NotFound) Error() string {
	return "Resource was not found"
}

func (i Internal) Error() string {
	return "Something went wrong"
}

func (b BadRequest) Error() string {
	return "Invalid input"
}

func (c Conflict) Error() string {
	return "Resource already exists"
}

func (u Unauthorized) Error() string {
	return "Not authorized"
}
