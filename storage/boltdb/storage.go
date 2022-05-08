package boltdbstore

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/uretgec/mydb/storage"

	bolt "go.etcd.io/bbolt"
)

type Store struct {
	db         *bolt.DB
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

	// Create dir if necessary
	_ = storage.CreateDir(path)

	// Open DB
	db, err := bolt.Open(fmt.Sprintf("%s%s.db", path, dbName), 0600, &bolt.Options{ReadOnly: readOnly})
	if err != nil {
		return s, err
	}

	if !readOnly {
		err = db.Update(func(t *bolt.Tx) error {
			// Create Bucket
			for _, bucketName := range s.bucketList {
				_, err := t.CreateBucketIfNotExists([]byte(bucketName))
				if err != nil {
					return err
				}
			}

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

	s.db = db
	return s, nil
}

func (s *Store) Set(bucketName []byte, k []byte, v []byte) ([]byte, error) {
	if s.readOnly {
		return nil, errors.New("readonly mod active")
	}

	if !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	if len(v) == 0 {
		return nil, errors.New("value not found")
	}

	err := s.db.Update(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)

		if len(k) == 0 {
			id, _ := b.NextSequence()
			k = storage.U64tob(int(id))
		}

		return b.Put(k, v)
	})

	if len(k) == 0 {
		k = []byte(strconv.FormatUint(storage.Btou64(k), 10))
	}

	return k, err
}

func (s *Store) KeyExist(bucketName []byte, k []byte) (bool, error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return false, errors.New("unknown bucket name")
	}

	var exists bool
	err := s.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		rxData := b.Get(k)
		if rxData != nil {
			exists = true
		}

		return nil
	})

	return exists, err
}

func (s *Store) ValueExist(bucketName []byte, v []byte) (bool, error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return false, errors.New("unknown bucket name")
	}

	var exists bool
	err := s.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		c := b.Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			check := bytes.Compare(value, v)
			if check == 0 {
				exists = true
				break
			}
		}

		return nil
	})

	return exists, err
}

func (s *Store) Get(bucketName []byte, k []byte) ([]byte, error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	var item []byte
	err := s.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		rxData := b.Get(k)
		if rxData != nil {
			item = rxData
		}

		return nil
	})

	return item, err
}

func (s *Store) MGet(bucketName []byte, keys ...[]byte) (list map[string]interface{}, err error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	items := make(map[string]interface{})

	err = s.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)

		for index, key := range keys {
			rxData := b.Get(key)

			// NOTE: index eklenmesinin nedeni sort mekanizmasını bozuyordu
			index := strconv.Itoa(index)
			items[string(index)+":"+string(key)] = string(rxData)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) Delete(bucketName []byte, k []byte) error {
	if s.readOnly {
		return errors.New("readonly mod active")
	}

	if !storage.Contains(s.allBuckets, bucketName) {
		return errors.New("unknown bucket name")
	}

	if len(k) == 0 {
		return errors.New("key not found")
	}

	return s.db.Update(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		return b.Delete(k)
	})
}

/*
First()  Move to the first key.
Last()   Move to the last key.
Seek()   Move to a specific key.
Next()   Move to the next key.
Prev()   Move to the previous key.
*/
func (s *Store) List(bucketName []byte, k []byte, perpage int) (list []string, err error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	counter := 1

	items := []string{}

	err = s.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		c := b.Cursor()

		if len(k) > 0 {
			for key, value := c.Seek(k); key != nil; key, value = c.Next() {
				if bytes.Equal(key, k) {
					continue
				}

				var v []byte
				if storage.Contains(s.indexList, bucketName) {
					kv := storage.KV{
						Key:   string(key),
						Value: string(value),
					}
					v, _ = kv.MarshalBinary()
				} else {
					v = value
				}

				items = append(items, string(v))

				if counter >= perpage {
					break
				}

				counter++
			}
		} else {
			for key, value := c.First(); key != nil; key, value = c.Next() {

				var v []byte
				if storage.Contains(s.indexList, bucketName) {
					kv := storage.KV{
						Key:   string(key),
						Value: string(value),
					}
					v, _ = kv.MarshalBinary()
				} else {
					v = value
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

func (s *Store) PrevList(bucketName []byte, k []byte, perpage int) (list []string, err error) {
	if !storage.Contains(s.allBuckets, bucketName) {
		return nil, errors.New("unknown bucket name")
	}

	counter := 1

	items := []string{}

	err = s.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucketName)
		c := b.Cursor()

		if len(k) > 0 {
			for key, value := c.Seek(k); key != nil; key, value = c.Prev() {
				if bytes.Equal(key, k) {
					continue
				}

				var v []byte
				if storage.Contains(s.indexList, bucketName) {
					kv := storage.KV{
						Key:   string(key),
						Value: string(value),
					}
					v, _ = kv.MarshalBinary()
				} else {
					v = value
				}

				items = append(items, string(v))

				if counter >= perpage {
					break
				}

				counter++
			}
		} else {
			for key, value := c.Last(); key != nil; key, value = c.Prev() {

				var v []byte
				if storage.Contains(s.indexList, bucketName) {
					kv := storage.KV{
						Key:   string(key),
						Value: string(value),
					}
					v, _ = kv.MarshalBinary()
				} else {
					v = value
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
	return s.db.Close()
}

// Boltdb Stats
func (s *Store) Stats() bolt.Stats {
	return s.db.Stats()
}

func (s *Store) Sync() {
	s.db.Sync()
}

func (s *Store) BucketList() (buckets []string, err error) {
	bucketList := []string{}

	err = s.db.View(func(t *bolt.Tx) error {
		return t.ForEach(func(name []byte, b *bolt.Bucket) error {
			if storage.Contains(s.bucketList, name) {
				bucketList = append(bucketList, string(name))
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return bucketList, nil
}

func (s *Store) DeleteBucket(bucketName []byte) error {
	if s.readOnly {
		return errors.New("readonly mod active")
	}

	if !storage.Contains(s.allBuckets, bucketName) {
		return errors.New("unknown bucket name")
	}

	return s.db.Update(func(t *bolt.Tx) error {
		return t.DeleteBucket(bucketName)
	})
}

func (s *Store) HasBucket(bucketName []byte) bool {
	return storage.Contains(s.allBuckets, bucketName)
}

func (s *Store) BucketStats(bucketName []byte) int {
	if !storage.Contains(s.allBuckets, bucketName) {
		return 0
	}

	var stats int
	err := s.db.View(func(t *bolt.Tx) error {
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
	return s.db.View(func(tx *bolt.Tx) error {
		// Create dir if necessary
		_ = storage.CreateDir(path)

		path = strings.TrimSuffix(path, "/") + "/" + filename + ".backup"
		return tx.CopyFile(path, 0600)
	})
}

func (s *Store) Restore(path, filename string) error {
	return errors.New("not implemented")
}
