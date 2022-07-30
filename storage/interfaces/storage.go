package interfaces

type Storage interface {
	NewStore(bucketList, indexList []string, path string, dbName string, readOnly bool)
	CloseStore()
	SyncStore()

	Set(bucketName []byte, k []byte, data []byte) ([]byte, error)
	Get(bucketName []byte, k []byte) ([]byte, error)
	MGet(bucketName []byte, keys ...[]byte) (interface{}, error)
	List(bucketName []byte, cursor []byte, perpage int) ([]string, error)
	PrevList(bucketName []byte, cursor []byte, perpage int) ([]string, error)
	Delete(bucketName []byte, k []byte) error

	KeyExist(bucketName []byte, k []byte) (bool, error)
	ValueExist(bucketName []byte, v []byte) (bool, error)

	HasBucket(bucketName []byte) bool
	StatsBucket(bucketName []byte) int
	ListBucket(bucketName []byte) int
	DeleteBucket(bucketName []byte) int

	Backup(path, filename string) error
	Restore(path, filename string) error
}
