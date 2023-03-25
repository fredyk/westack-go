package memorykv

import (
	"fmt"
	"sync"
	"time"
)

type Options struct {
	Name string
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvDb interface {
	GetBucket(name string) MemoryKvBucket
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvBucket interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	SetEx(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Expire(key string, ttl time.Duration) error
}

type kvPair struct {
	value     []byte
	expiresAt int64 // unix timestamp in seconds
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvBucketImpl struct {
	name string
	data map[string]kvPair
}

var dataLock sync.RWMutex

func (kvBucket *MemoryKvBucketImpl) Get(key string) ([]byte, error) {
	dataLock.RLock()
	pair, ok := kvBucket.data[key]
	dataLock.RUnlock()
	if ok {
		return pair.value, nil
	} else {
		return nil, nil
	}
}

func (kvBucket *MemoryKvBucketImpl) Set(key string, value []byte) error {
	dataLock.RLock()
	pair, ok := kvBucket.data[key]
	dataLock.RUnlock()
	if ok {
		dataLock.Lock()
		pair.value = value
		kvBucket.data[key] = pair
		dataLock.Unlock()
		return nil
	} else {
		dataLock.Lock()
		kvBucket.data[key] = kvPair{
			value: value,
		}
		dataLock.Unlock()
		return nil
	}
}

func (kvBucket *MemoryKvBucketImpl) SetEx(key string, value []byte, ttl time.Duration) error {
	err := kvBucket.Set(key, value)
	if err != nil {
		return err
	}
	err = kvBucket.Expire(key, ttl)
	if err != nil {
		return err
	}
	return nil
}

func waitAndDoExpire(ttl time.Duration, kvBucket *MemoryKvBucketImpl, key string) {
	time.Sleep(ttl)
	err := kvBucket.Delete(key)
	if err != nil {
		fmt.Printf("[memorykv] Error deleting key: %v\n", err)
	}
}

func (kvBucket *MemoryKvBucketImpl) Expire(key string, ttl time.Duration) error {
	dataLock.RLock()
	pair, ok := kvBucket.data[key]
	dataLock.RUnlock()
	if ok {
		dataLock.Lock()
		pair.expiresAt = time.Now().Add(ttl).Unix()
		kvBucket.data[key] = pair
		dataLock.Unlock()
		go waitAndDoExpire(ttl, kvBucket, key)
		return nil
	} else {
		return fmt.Errorf("key not found")
	}
}

func (kvBucket *MemoryKvBucketImpl) Delete(key string) error {
	dataLock.Lock()
	delete(kvBucket.data, key)
	dataLock.Unlock()
	return nil
}

func createBucket() MemoryKvBucket {
	return &MemoryKvBucketImpl{
		data: make(map[string]kvPair),
	}
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvDbImpl struct {
	name    string
	buckets map[string]MemoryKvBucket
}

var bucketsLock sync.RWMutex

func (kvDb *MemoryKvDbImpl) GetBucket(name string) MemoryKvBucket {
	bucketsLock.RLock()
	bucket, ok := kvDb.buckets[name]
	bucketsLock.RUnlock()
	if ok {
		return bucket
	}
	bucketsLock.Lock()
	defer bucketsLock.Unlock()
	bucket, ok = kvDb.buckets[name]
	if ok {
		return bucket
	}
	bucket = createBucket()
	kvDb.buckets[name] = bucket
	return bucket
}

func NewMemoryKvDb(options Options) MemoryKvDb {
	return &MemoryKvDbImpl{
		name:    options.Name,
		buckets: make(map[string]MemoryKvBucket),
	}
}
