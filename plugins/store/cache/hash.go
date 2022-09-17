package mem

import "hash/fnv"

const (
	hash_len = 256
)

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32() % hash_len
}
