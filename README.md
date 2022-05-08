# MYDB
It is a storage package containing both in-memory and file-type databases that you can use to hold simple data.

Use:
- bbolt: go.etcd.io/bbolt
- sniper: github.com/recoilme/sniper

NOTE:
> If use only boltdb, all key-value data are at in-memory and saves all data to snapshot file for recovery

> If use only sniperdb, all index data are at in-memory and save all key-value data to file (multiple files)
> sniperdb have to use bboltdb index for list, prevlist, exist methods

> You can use both db together without any problems.

## Examples

Example use go to mydb-server repository -> (https://github.com/uretgec/mydb-server)

## Methods
```
CloseStore()
SyncStore() // For only bbolt
HasBucket(bucketName []byte) bool

Set(bucketName []byte, k []byte, data []byte) ([]byte, error)
Get(bucketName []byte, k []byte) ([]byte, error)
MGet(bucketName []byte, keys ...[]byte) (interface{}, error)
List(bucketName []byte, cursor []byte, perpage int) ([]string, error)
PrevList(bucketName []byte, cursor []byte, perpage int) ([]string, error)

Exist(bucketName []byte, k []byte) (bool, error)
ValueExist(bucketName []byte, v []byte) (bool, error) // For only bbolt/bbolt-sniper index

Del(bucketName []byte, k []byte) error
BStats(bucketName []byte) int
Backup(path, filename string) error
Restore(path, filename string) error // For only sniper - bbolt no need
```

## Install

```
go get  github.com/uretgec/mydb
```

## TODO
- Add test files
- Add new examples

## Links

Bbolt (go.etcd.io/bbolt)

Sniper (github.com/recoilme/sniper)