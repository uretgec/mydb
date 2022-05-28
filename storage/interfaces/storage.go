package interfaces

type StorageClient interface {
	CloseStore()
	SyncStore()
	HasBucket(bucketName []byte) bool

	Set(bucketName []byte, k []byte, data []byte) ([]byte, error)
	Get(bucketName []byte, k []byte) ([]byte, error)
	MGet(bucketName []byte, keys ...[]byte) (interface{}, error)
	List(bucketName []byte, cursor []byte, perpage int) ([]string, error)
	PrevList(bucketName []byte, cursor []byte, perpage int) ([]string, error)

	Exist(bucketName []byte, k []byte) (bool, error)
	ValueExist(bucketName []byte, v []byte) (bool, error)

	Del(bucketName []byte, k []byte) error
	BStats(bucketName []byte) int
	Backup(path, filename string) error
	Restore(path, filename string) error
}
