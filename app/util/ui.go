package util

import (
	"hash/fnv"
	"math/rand"
	"strings"
)

var letters = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}

// https://github.com/n1try/wakapi/blob/7a950c9ff36a398b0df7507e948b98a396836292/static/index.html#L255
func RandomColor(seedKey string) string {
	var sb strings.Builder

	h := fnv.New64a()
	h.Write([]byte(seedKey))
	r := rand.New(rand.NewSource(int64(h.Sum64())))

	perm := r.Perm(len(letters))

	sb.WriteString("#")
	for i := 0; i < 6; i++ {
		sb.WriteString(letters[perm[i]])
	}

	return sb.String()
}
