package storage

import (
	"context"
	"hash/fnv"
	"math"

	"github.com/go-redis/redis/v8"
)

type BloomFilter struct {
	Key         string
	Size        uint
	HashFuncNum uint
}

// NewBloomFilter 初始化一个布隆过滤器
// n 预期的数据量
// p 误判率
func NewBloomFilter(key string, n uint, p float64) *BloomFilter {
	sizeF := -float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))
	hashNumF := sizeF / float64(n) * math.Log(2)

	size := uint(math.Ceil(sizeF))
	if size == 0 {
		size = 1
	}
	hashNum := uint(math.Ceil(hashNumF))
	if hashNum == 0 {
		hashNum = 1
	}

	return &BloomFilter{
		Key:         key,
		Size:        size,
		HashFuncNum: hashNum,
	}
}

// Add 向布隆过滤器添加数据
func (bf *BloomFilter) Add(data string) error {
	locations := bf.getLocations(data)
	ctx := context.Background()

	pipe := Rdb.Pipeline()
	for _, loc := range locations {
		pipe.SetBit(ctx, bf.Key, int64(loc), 1)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Exists 检查数据是否存在
func (bf *BloomFilter) Exists(data string) (bool, error) {
	locations := bf.getLocations(data)
	ctx := context.Background()

	pipe := Rdb.Pipeline()
	for _, loc := range locations {
		pipe.GetBit(ctx, bf.Key, int64(loc))
	}
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 仅所有位置都是1此数据才可能存在
	for _, cmd := range cmds {
		// 解析返回值
		val, err := cmd.(*redis.IntCmd).Result()
		if err != nil {
			return false, err
		}
		if val == 0 {
			return false, nil
		}
	}

	return true, nil
}

// getLocations 计算数据映射到的bitmap位置
// FNV哈希算法，双重哈希模拟k个哈希函数
func (bf *BloomFilter) getLocations(data string) []uint {
	bytes := []byte(data)
	locations := make([]uint, int(bf.HashFuncNum))

	// 计算两个基础哈希值
	h1 := fnvHash(bytes)
	h2 := fnvHash(append(bytes, byte(1))) // 简单的变种

	for i := uint(0); i < bf.HashFuncNum; i++ {
		// 公式：hash_i = (h1 + i * h2) % size
		// 这是一个生成多个哈希值的标准技巧
		loc := (h1 + uint32(i)*h2) % uint32(bf.Size)
		locations[i] = uint(loc)
	}
	return locations
}

func fnvHash(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32()
}
