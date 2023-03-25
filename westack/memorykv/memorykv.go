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
type MemoryKvStats struct {
	Entries                int     `json:"entries"`
	AvgExpirationTime      float64 `json:"avgExpirationTime"`
	EarliestExpirationTime int64   `json:"earliestExpirationTime"`
	LatestExpirationTime   int64   `json:"latestExpirationTime"`
	ExpirationQueueSize    int64   `json:"expirationQueueSize"`
	TotalSize              int64   `json:"totalSize"`
	AvgObjSize             float64 `json:"avgObjSize"`
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvDb interface {
	GetBucket(name string) MemoryKvBucket
	Stats() map[string]MemoryKvStats
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvBucket interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	SetEx(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Expire(key string, ttl time.Duration) error
	Stats() MemoryKvStats
}

type kvPair struct {
	key       string
	value     []byte
	expiresAt int64 // unix timestamp in seconds
}

type expirationQueue struct {
	expirationQueue []kvPair
}

var expirationLock sync.RWMutex

func (queue *expirationQueue) Add(key string, expiresAt int64) {
	expirationLock.Lock()
	inserted := false
	for i, pair := range queue.expirationQueue {
		if pair.expiresAt > expiresAt {
			queue.expirationQueue = append(queue.expirationQueue[:i], append([]kvPair{{key: key, value: nil, expiresAt: expiresAt}}, queue.expirationQueue[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		queue.expirationQueue = append(queue.expirationQueue, kvPair{key: key, value: nil, expiresAt: expiresAt})
	}
	expirationLock.Unlock()
}

func (queue *expirationQueue) Pop() (string, bool) {
	expirationLock.Lock()
	if len(queue.expirationQueue) == 0 {
		expirationLock.Unlock()
		return "", false
	}
	pair := queue.expirationQueue[0]
	queue.expirationQueue = queue.expirationQueue[1:]
	expirationLock.Unlock()
	return pair.key, true
}

func (queue *expirationQueue) Peek() (string, bool) {
	expirationLock.RLock()
	if len(queue.expirationQueue) == 0 {
		expirationLock.RUnlock()
		return "", false
	}
	pair := queue.expirationQueue[0]
	expirationLock.RUnlock()
	return pair.key, true
}

func (queue *expirationQueue) Remove(key string) {
	expirationLock.Lock()
	for i, pair := range queue.expirationQueue {
		if pair.key == key {
			queue.expirationQueue = append(queue.expirationQueue[:i], queue.expirationQueue[i+1:]...)
			break
		}
	}
	expirationLock.Unlock()
}

func (queue *expirationQueue) Len() int64 {
	expirationLock.RLock()
	defer expirationLock.RUnlock()
	return int64(len(queue.expirationQueue))
}

func (queue *expirationQueue) Update(key string, expiresAt int64) {
	expirationLock.Lock()
	for i, pair := range queue.expirationQueue {
		if pair.key == key {
			queue.expirationQueue = append(queue.expirationQueue[:i], queue.expirationQueue[i+1:]...)
			break
		}
	}
	expirationLock.Unlock()
	queue.Add(key, expiresAt)
}

func (queue *expirationQueue) Cursor() chan kvPair {
	expirationLock.RLock()
	defer expirationLock.RUnlock()
	ch := make(chan kvPair)
	go func() {
		for _, pair := range queue.expirationQueue {
			ch <- pair
		}
		close(ch)
	}()
	return ch
}

func newExpirationQueue() *expirationQueue {
	return &expirationQueue{
		expirationQueue: make([]kvPair, 0),
	}
}

//goland:noinspection GoNameStartsWithPackageName
type MemoryKvBucketImpl struct {
	name            string
	data            map[string]kvPair
	expirationQueue *expirationQueue
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
	} else {
		dataLock.Lock()
		kvBucket.data[key] = kvPair{
			key:   key,
			value: value,
		}
		dataLock.Unlock()
	}
	kvBucket.expirationQueue.Add(key, time.Now().Unix()+86400*365)
	return nil
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

func (kvBucket *MemoryKvBucketImpl) Expire(key string, ttl time.Duration) error {
	dataLock.RLock()
	pair, ok := kvBucket.data[key]
	dataLock.RUnlock()
	if ok {
		dataLock.Lock()
		pair.expiresAt = time.Now().Add(ttl).Unix()
		kvBucket.data[key] = pair
		dataLock.Unlock()
		kvBucket.expirationQueue.Update(key, pair.expiresAt)
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

func (kvBucket *MemoryKvBucketImpl) Stats() MemoryKvStats {
	dataLock.RLock()
	defer dataLock.RUnlock()
	var avgExpirationTime float64
	var _avgExpirationCount float64
	var _avgExpirationSum float64
	var earliestExpirationTime int64
	var latestExpirationTime int64
	var totalSize int64
	var avgObjSize float64
	var _avgObjSizeCount float64
	var _avgObjSizeSum float64
	for pair := range kvBucket.expirationQueue.Cursor() {
		if earliestExpirationTime == 0 || earliestExpirationTime > pair.expiresAt {
			earliestExpirationTime = pair.expiresAt
		}
		if latestExpirationTime == 0 || latestExpirationTime < pair.expiresAt {
			latestExpirationTime = pair.expiresAt
		}
		_avgExpirationSum += float64(pair.expiresAt)
		_avgExpirationCount += 1

		realPair, ok := kvBucket.data[pair.key]
		if ok {
			bytelen := len(realPair.value)
			totalSize += int64(bytelen)
			_avgObjSizeSum += float64(bytelen)
			_avgObjSizeCount += 1
		}
	}
	if _avgExpirationCount > 0 {
		avgExpirationTime = _avgExpirationSum / _avgExpirationCount
	}
	if _avgObjSizeCount > 0 {
		avgObjSize = _avgObjSizeSum / _avgObjSizeCount
	}
	return MemoryKvStats{
		Entries:                len(kvBucket.data),
		AvgExpirationTime:      avgExpirationTime,
		EarliestExpirationTime: earliestExpirationTime,
		LatestExpirationTime:   latestExpirationTime,
		ExpirationQueueSize:    kvBucket.expirationQueue.Len(),
		TotalSize:              totalSize,
		AvgObjSize:             avgObjSize,
	}
}

func createBucket() MemoryKvBucket {
	kvBucket := &MemoryKvBucketImpl{
		data:            make(map[string]kvPair),
		expirationQueue: newExpirationQueue(),
	}
	go performExpirations(kvBucket)
	return kvBucket
}

func performExpirations(kvBucket *MemoryKvBucketImpl) {
	for {
		key, ok := kvBucket.expirationQueue.Peek()
		if ok {
			dataLock.RLock()
			pair, ok := kvBucket.data[key]
			dataLock.RUnlock()
			if ok {
				if pair.expiresAt > time.Now().Unix() {
					toWait := time.Duration(pair.expiresAt-time.Now().Unix()) * time.Second
					time.Sleep(toWait)
				}
				dataLock.Lock()
				delete(kvBucket.data, key)
				dataLock.Unlock()
				kvBucket.expirationQueue.Remove(key)
			} else {
				kvBucket.expirationQueue.Remove(key)
			}
		} else {
			time.Sleep(1 * time.Second)
		}
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

func (kvDb *MemoryKvDbImpl) Stats() map[string]MemoryKvStats {
	stats := make(map[string]MemoryKvStats)
	bucketsLock.RLock()
	for name, bucket := range kvDb.buckets {
		stats[name] = bucket.Stats()
	}
	bucketsLock.RUnlock()
	return stats
}

func NewMemoryKvDb(options Options) MemoryKvDb {
	return &MemoryKvDbImpl{
		name:    options.Name,
		buckets: make(map[string]MemoryKvBucket),
	}
}
