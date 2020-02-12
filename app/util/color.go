package util

import (
	"github.com/patrickmn/go-cache"
	"hash/fnv"
	"math/rand"
	"strings"
	"time"
)

// https://github.com/n1try/wakapi/blob/7a950c9ff36a398b0df7507e948b98a396836292/static/index.html#L255
var letters = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}
var colorCache = cache.New(60*time.Minute, 60*2*time.Minute)

func RandomColor(seedKey string) string {
	if c, ok := colorCache.Get(seedKey); ok {
		return c.(string)
	}

	var sb strings.Builder

	h := fnv.New64a()
	h.Write([]byte(seedKey))
	r := rand.New(rand.NewSource(int64(h.Sum64())))

	perm := r.Perm(len(letters))

	sb.WriteString("#")
	for i := 0; i < 6; i++ {
		sb.WriteString(letters[perm[i]])
	}

	c := sb.String()
	colorCache.SetDefault(seedKey, c)
	return c
}
