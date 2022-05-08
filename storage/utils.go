package storage

import (
	"encoding/binary"
	"os"
)

// u64tob converts a uint64 into an 8-byte slice.
func U64tob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// btou64 converts an 8-byte slice into an uint64.
func Btou64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Usage: Bucket and Index list check
func Contains(s []string, str []byte) bool {
	for _, v := range s {
		if v == string(str) {
			return true
		}
	}

	return false
}

// Usage: for boltdb db storage folder
func CreateDir(path string) error {
	// Check if folder exists
	_, err := os.Stat(path)

	// Create directory if not exists
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModeDir|0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// Not use yet :)
func DeleteFileOrDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}

	return nil
}
