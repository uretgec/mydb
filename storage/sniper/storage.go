package sniperstore

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/uretgec/mydb/storage"

	"github.com/recoilme/sniper"
	bolt "go.etcd.io/bbolt"
)

// Index: boltdb
// Database: sniper - because of sniper memory index not working true
type Store struct {
	db         *sniper.Store
	dbIndex    *bolt.DB
	bucketList []string
	readOnly   bool
	indexList  []string
	allBuckets []string
}

func NewStore(bucketList, indexList []string, path string, dbName string, readOnly bool) (*Store, error) {
	s := &Store{}
	s.bucketList = bucketList
	s.readOnly = readOnly
	s.indexList = indexList
	s.allBuckets = append(bucketList, indexList...)

	// Open DB
	db, err := sniper.Open(sniper.Dir(fmt.Sprintf("%s%s", path, dbName)))
	if err != nil {
		return s, err
	}

	s.db = db

	// IndexDB
	// Create dir if necessary
	_ = storage.CreateDir(path)

	// Open DB
	dbIndex, err := bolt.Open(fmt.Sprintf("%s%s.db", path, "indexstore"), 0600, &bolt.Options{ReadOnly: readOnly})
	if err != nil {
		return s, err
	}

	if !readOnly {
		err = dbIndex.Update(func(t *bolt.Tx) error {
			// Create Bucket
			// Not necessary create bucket for sniper database

			// Create Index
			for _, indexName := range s.indexList {
				_, err := t.CreateBucketIfNotExists([]byte(indexName))
				if err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return s, err
		}
	}

	s.dbIndex = dbIndex
	return s, nil
}

func (s *Store) Set(bucketName []byte, k []byte, v []byte) ([]byte, error) {
	if s.readOnly {
		return nil, errors.New("readonly mod active")
	}

	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	if len(k) == 0 || len(v) == 0 {
		return nil, errors.New("key or value not found")
	}

	key := string(k)
	if len(bucketName) > 0 {
		key = string(bucketName) + key
	}

	err := s.db.Set([]byte(key), v, 0)
	if err == nil {
		err = s.dbIndex.Update(func(t *bolt.Tx) error {
			b := t.Bucket(bucketName)

			return b.Put(k, []byte(fmt.Sprint(0)))
		})
	}

	return k, err
}

func (s *Store) KeyExist(bucketName []byte, k []byte) (bool, error) {
	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return false, errors.New("unknown bucket name")
	}

	key := string(k)
	if len(bucketName) > 0 {
		key = string(bucketName) + key
	}

	v, err := s.db.Get([]byte(key))
	if err == sniper.ErrNotFound {
		return false, nil
	}

	return (len(v) > 0), err
}

func (s *Store) ValueExist(bucketName []byte, v []byte) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *Store) Get(bucketName []byte, k []byte) ([]byte, error) {
	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	key := string(k)
	if len(bucketName) > 0 {
		key = string(bucketName) + key
	}

	var item []byte
	v, err := s.db.Get([]byte(key))
	if err == sniper.ErrNotFound {
		return item, nil
	}

	return v, err
}

func (s *Store) MGet(bucketName []byte, keys ...[]byte) (list map[string]interface{}, err error) {
	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	items := make(map[string]interface{})

	for _, k := range keys {
		key := string(k)
		if len(bucketName) > 0 {
			key = string(bucketName) + key
		}

		v, err := s.db.Get([]byte(key))
		if err == sniper.ErrNotFound {
			continue
		} else if err != nil {
			continue
		}

		items[key] = string(v)
	}

	return items, nil
}

func (s *Store) Delete(bucketName []byte, k []byte) error {
	if s.readOnly {
		return errors.New("readonly mod active")
	}

	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return errors.New("unknown bucket name")
	}

	if len(k) == 0 {
		return errors.New("key not found")
	}

	key := string(k)
	if len(bucketName) > 0 {
		key = string(bucketName) + key
	}

	_, err := s.db.Delete([]byte(key))
	if err == nil && storage.Contains(s.indexList, bucketName) {
		err = s.dbIndex.Update(func(t *bolt.Tx) error {
			b := t.Bucket(bucketName)
			return b.Delete(k)
		})
	}

	return err
}

/*
First()  Move to the first key.
Last()   Move to the last key.
Seek()   Move to a specific key.
Next()   Move to the next key.
Prev()   Move to the previous key.
*/
func (s *Store) List(bucketName []byte, k []byte, perpage int) (list []string, err error) {
	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	counter := 1

	items := []string{}

	err = s.dbIndex.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		c := b.Cursor()

		if len(k) > 0 {
			for key, _ := c.Seek(k); key != nil; key, _ = c.Next() {
				if bytes.Equal(key, k) {
					continue
				}

				v, err := s.Get(bucketName, key)
				if err == sniper.ErrNotFound {
					continue
				} else if err != nil {
					continue
				}

				items = append(items, string(v))

				if counter >= perpage {
					break
				}

				counter++
			}
		} else {
			for key, _ := c.First(); key != nil; key, _ = c.Next() {

				v, err := s.Get(bucketName, key)
				if err == sniper.ErrNotFound {
					continue
				} else if err != nil {
					continue
				}

				items = append(items, string(v))

				if counter >= perpage {
					break
				}

				counter++
			}
		}

		return nil
	})

	if len(items) == 0 {
		return nil, err
	}

	return items, err
}

// Not stable
func (s *Store) PrevList(bucketName []byte, k []byte, perpage int) (list []string, err error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	counter := 1

	items := []string{}

	err = s.dbIndex.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		c := b.Cursor()

		if len(k) > 0 {
			for key, _ := c.Seek(k); key != nil; key, _ = c.Prev() {
				if bytes.Equal(key, k) {
					continue
				}

				v, err := s.Get(bucketName, key)
				if err == sniper.ErrNotFound {
					continue
				} else if err != nil {
					continue
				}

				items = append(items, string(v))

				if counter >= perpage {
					break
				}

				counter++
			}
		} else {
			for key, _ := c.Last(); key != nil; key, _ = c.Prev() {

				v, err := s.Get(bucketName, key)
				if err == sniper.ErrNotFound {
					continue
				} else if err != nil {
					continue
				}

				items = append(items, string(v))

				if counter >= perpage {
					break
				}

				counter++
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) Close() error {
	err := s.db.Close()
	if err == nil {
		err = s.dbIndex.Close()
	}

	return err
}

func (s *Store) Sync() {
	s.dbIndex.Sync()
}

func (s *Store) BucketList() (buckets []string, err error) {
	val, err := s.db.Get([]byte("[buckets]"))
	if err != nil {
		return nil, err
	}

	return strings.Split(string(val), ","), nil
}

func (s *Store) DeleteBucket(bucketName []byte) error {
	if s.readOnly {
		return errors.New("readonly mod active")
	}

	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return errors.New("unknown bucket name")
	}

	if len(bucketName) > 0 {
		err := s.dbIndex.View(func(t *bolt.Tx) error {
			b := t.Bucket(bucketName)
			c := b.Cursor()

			for key, _ := c.First(); key != nil; key, _ = c.Next() {
				err := s.Delete(bucketName, key)
				if err != nil {
					return err
				}
			}

			return nil
		})

		return err
	}

	return errors.New("not implemented")
}

func (s *Store) HasBucket(bucketName []byte) bool {
	return len(bucketName) > 0 && storage.Contains(s.allBuckets, bucketName)
}

func (s *Store) BucketStats(bucketName []byte) int {
	if len(bucketName) > 0 && !storage.Contains(s.allBuckets, bucketName) {
		return 0
	}

	var stats int
	err := s.dbIndex.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)

		stats = b.Stats().KeyN // total count key/value

		return nil
	})

	if err != nil {
		return 0
	}

	return stats
}

func (s *Store) Backup(path, filename string) error {
	// Create dir if necessary
	_ = storage.CreateDir(path)

	err := s.db.Backup(fmt.Sprintf("%s%s", path, filename))
	if err == nil {
		err = s.dbIndex.View(func(tx *bolt.Tx) error {

			path = strings.TrimSuffix(path, "/") + "/index-" + filename + ".backup"
			return tx.CopyFile(path, 0600)
		})
	}
	return err
}

func (s *Store) Restore(path, filename string) error {
	return s.db.Restore(fmt.Sprintf("%s%s", path, filename))
}
