package config

import (
	"encoding/json"
	"github.com/timshannon/bolthold"
)

var GobEncode = bolthold.DefaultEncode
var GobDecode = bolthold.DefaultDecode

func JsonEncode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func JsonDecode(data []byte, value interface{}) error {
	return json.Unmarshal(data, value)
}
