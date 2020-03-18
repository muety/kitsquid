package admin

type registeredEntity struct {
	Name      string
	Instance  interface{}
	Resolvers crudResolvers
}

type crudResolvers struct {
	List    func() (interface{}, error)
	Get     func(string) (interface{}, error)
	Put     func(string, string) error
	Delete  func(string) error
	Flush   func()
	Reindex func()
}

type adminQuery struct {
	Action string `json:"action" binding:"required"`
	Entity string `json:"entity" binding:"required"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}
