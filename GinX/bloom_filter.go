package ginx

import (
	"context"
	"math"

	"github.com/redis/go-redis/v9"
	"github.com/spaolacci/murmur3"
)

// BloomFilter 基于Redis的布隆过滤器实现
type BloomFilter struct {
	cmd   redis.Cmdable // Redis客户端
	key   string        // Redis存储键名
	m     uint          // 位数组大小
	k     uint          // 哈希函数数量
	seeds []uint        // 哈希种子
}

// NewBloomFilter 创建布隆过滤器
// n: 预期元素数量
// p: 期望的误判率（0 < p < 1）
func NewBloomFilter(cmd redis.Cmdable, key string, n uint, p float64) *BloomFilter {
	m := calculateM(n, p)
	k := calculateK(m, n)
	return &BloomFilter{
		cmd:   cmd,
		key:   key,
		m:     m,
		k:     k,
		seeds: generateSeeds(k),
	}
}

// Add 添加元素到布隆过滤器
func (bf *BloomFilter) Add(ctx context.Context, data []byte) error {
	for _, seed := range bf.seeds {
		pos := bf.hash(data, seed)
		_, err := bf.cmd.SetBit(ctx, bf.key, int64(pos), 1).Result()
		if err != nil {
			return err
		}
	}
	return nil
}

// Contains 检查元素是否可能存在
func (bf *BloomFilter) Contains(ctx context.Context, data []byte) (bool, error) {
	for _, seed := range bf.seeds {
		pos := bf.hash(data, seed)
		bit, err := bf.cmd.GetBit(ctx, bf.key, int64(pos)).Result()
		if err != nil {
			return false, err
		}
		if bit == 0 {
			return false, nil
		}
	}
	return true, nil
}

// hash 计算元素的哈希位置
func (bf *BloomFilter) hash(data []byte, seed uint) uint {
	h := murmur3.New32WithSeed(uint32(seed))
	h.Write(data)
	return uint(h.Sum32()) % bf.m
}

// calculateM 计算位数组大小
func calculateM(n uint, p float64) uint {
	return uint(math.Ceil(-float64(n) * math.Log(p) / (math.Pow(math.Log(2), 2))))
}

// calculateK 计算哈希函数数量
func calculateK(m, n uint) uint {
	return uint(math.Ceil(float64(m) / float64(n) * math.Log(2)))
}

// generateSeeds 生成哈希种子
func generateSeeds(k uint) []uint {
	seeds := make([]uint, k)
	for i := uint(0); i < k; i++ {
		seeds[i] = uint(i) * 0x12345678 // 简单生成不同的种子
	}
	return seeds
}
