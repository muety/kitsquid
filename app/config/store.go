package config

import (
	"encoding/json"
	"github.com/timshannon/bolthold"
)

var (
	GobEncode  = bolthold.DefaultEncode
	GobDecode  = bolthold.DefaultDecode
	JsonEncode = json.Marshal
	JsonDecode = json.Unmarshal
)
