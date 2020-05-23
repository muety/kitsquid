package errors

/*
KitSquidError represent a general error in this application
*/
type KitSquidError error

/*
NotFound represents an entity or resource not being found
*/
type NotFound struct{}

/*
BadRequest represents an invalid or unexpectedly formatted request
*/
type BadRequest struct{}

/*
Conflict represents a collision between a new and an already existing resource
*/
type Conflict struct{}

/*
Internal represents an internal server error
*/
type Internal struct{}

/*
Unauthorized represents a user not being authenticated or allowed to view a certain resource
*/
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
