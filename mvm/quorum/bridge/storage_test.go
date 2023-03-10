package main

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	assert := assert.New(t)

	path := "/tmp/bridge"
	store, err := OpenStorage(path)

	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)
	defer store.Close()

	ip := "127.0.0.1"
	keys := store.limiterAvailable(ip)
	size := len(keys)
	log.Println("keys size", size)

	for i := 0; i < 10; i++ {
		err = store.writeLimiter(ip)
		assert.Nil(err)
	}
	keys = store.limiterAvailable(ip)
	assert.Equal(size+10, len(keys))
}
