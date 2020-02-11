package errors

type KitHubError error

type NotFound struct{}

type Internal struct{}

func (n NotFound) Error() string {
	return "Resource was not found"
}

func (i Internal) Error() string {
	return "Something went wrong"
}
