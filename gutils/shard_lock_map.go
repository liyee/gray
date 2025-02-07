package gutils

import (
	"encoding/json"
	"sync"
)

var ShardCount = 32

type ShardLockMaps struct {
	shards []*SingleShardMap
	hash   IHash
}

type SingleShardMap struct {
	items map[string]interface{}
	sync.RWMutex
}

func createShardLOckMpas(hash IHash) ShardLockMaps {
	slm := ShardLockMaps{
		shards: make([]*SingleShardMap, ShardCount),
		hash:   hash,
	}

	for i := 0; i < ShardCount; i++ {
		slm.shards[i] = &SingleShardMap{items: make(map[string]interface{})}
	}

	return slm
}

func NewShardLockMaps() ShardLockMaps {
	return createShardLOckMpas(DefaultHash())
}

func NewWithCustomHash(hash IHash) ShardLockMaps {
	return createShardLOckMpas(hash)
}

func (slm ShardLockMaps) GetShard(key string) *SingleShardMap {
	return slm.shards[slm.hash.Sum(key)%uint32(ShardCount)]
}

func (slm ShardLockMaps) Count() int {
	count := 0
	for i := 0; i < ShardCount; i++ {
		shard := slm.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

func (slm ShardLockMaps) Get(key string) (interface{}, bool) {
	shard := slm.GetShard(key)
	shard.RLock()
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

func (slm ShardLockMaps) Set(key string, value interface{}) {
	shard := slm.GetShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

func (slm ShardLockMaps) SetNX(key string, value interface{}) bool {
	shard := slm.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok
}

func (slm ShardLockMaps) MSet(data map[string]interface{}) {
	for key, value := range data {
		shard := slm.GetShard(key)
		shard.Lock()
		shard.items[key] = value
		shard.Unlock()
	}
}

func (slm ShardLockMaps) Has(key string) bool {
	shard := slm.GetShard(key)
	shard.RLock()
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

func (slm ShardLockMaps) Remove(key string) {
	shard := slm.GetShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

type RemoveCb func(key string, v interface{}, exists bool) bool

func (slm ShardLockMaps) RemoveCb(key string, cb RemoveCb) bool {
	shard := slm.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	remove := cb(key, v, ok)
	if remove && ok {
		delete(shard.items, key)
	}
	shard.Unlock()
	return remove
}

func (slm ShardLockMaps) Pop(key string) (v interface{}, exists bool) {
	shard := slm.GetShard(key)
	shard.Lock()
	v, exists = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return v, exists
}

func (slm ShardLockMaps) IsEmpty() bool {
	return slm.Count() == 0
}

type Tuple struct {
	Key string
	Val interface{}
}

func snapshot(slm ShardLockMaps) (chanList []chan Tuple) {
	chanList = make([]chan Tuple, ShardCount)
	wg := sync.WaitGroup{}
	wg.Add(ShardCount)
	for index, shard := range slm.shards {
		go func(index int, shard *SingleShardMap) {
			shard.RLock()
			chanList[index] = make(chan Tuple, len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chanList[index] <- Tuple{key, val}
			}
			shard.RUnlock()
			close(chanList[index])
		}(index, shard)
	}
	wg.Wait()
	return chanList
}

func fanIn(chanList []chan Tuple, out chan Tuple) {
	wg := sync.WaitGroup{}
	wg.Add(len(chanList))
	for _, ch := range chanList {
		go func(ch chan Tuple) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

func (slm ShardLockMaps) IterBuffered() <-chan Tuple {
	chanList := snapshot(slm)
	total := 0
	for _, c := range chanList {
		total += cap(c)
	}
	ch := make(chan Tuple, total)
	go fanIn(chanList, ch)
	return ch
}

func (slm ShardLockMaps) Clear() {
	for item := range slm.IterBuffered() {
		slm.Remove(item.Key)
	}
}

func (slm ShardLockMaps) Items() map[string]interface{} {
	tmp := make(map[string]interface{})

	for item := range slm.IterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
}

func (slm ShardLockMaps) Keys() []string {
	count := slm.Count()
	ch := make(chan string, count)
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(ShardCount)
		for _, shard := range slm.shards {
			go func(shard *SingleShardMap) {
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	keys := make([]string, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

type IterCb func(key string, v interface{})

func (slm ShardLockMaps) IterCb(fn IterCb) {
	for idx := range slm.shards {
		shard := (slm.shards)[idx]
		shard.RLock()
		for key, value := range shard.items {
			fn(key, value)
		}
		shard.RUnlock()
	}
}

func (slm ShardLockMaps) MarshalJSON() ([]byte, error) {
	tmp := make(map[string]interface{})

	for item := range slm.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

func (slm ShardLockMaps) UnmarshalJSON(b []byte) (err error) {
	tmp := make(map[string]interface{})

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	for key, val := range tmp {
		slm.Set(key, val)
	}
	return nil
}
