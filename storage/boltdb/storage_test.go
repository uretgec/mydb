package boltdbstorage

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd(t *testing.T) {
	store, err := OpenStore()
	assert.NoError(t, err)

	_, err = store.Set([]byte("posts"), []byte("test_1"), []byte("number one"))
	assert.NoError(t, err)

	res, err := store.Get([]byte("posts"), []byte("test_1"))
	assert.NoError(t, err)

	assert.Equal(t, true, bytes.Equal(res, []byte("number one")))

	err = store.CloseStore()
	assert.NoError(t, err)

	err = DeleteStore()
	assert.NoError(t, err)
}

func OpenStore() (*Store, error) {
	return NewStore([]string{"options", "posts", "pages"}, []string{"posts", "pages"}, "./", "storage_test", false)
}

func DeleteStore() error {
	return os.RemoveAll("./storage_test.db")
}
