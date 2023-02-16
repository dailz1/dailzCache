package consistentHash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 定义函数类型 Hash，采取依赖注入的方式，允许替换成自定义的 Hash 函数，默认采用 crc32.ChecksumIEEE 算法
type Hash func(data []byte) uint32

// Map 是一致性哈希算法的主数据结构
type Map struct {
	hash     Hash           // Hash 函数
	replicas int            // 虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string //虚拟节点与真实节点的映射表
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// IsEmpty returns true if there are no items available.
func (m *Map) IsEmpty() bool {
	return len(m.keys) == 0
}

// Add adds some keys to the hash.
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 对每一个真实节点 key，对应地创建 m.replicas 个虚拟节点
		// 虚拟节点的名称为 strconv.Itoa(i) + key
		// 使用 m.hash() 计算虚拟节点的哈希值，然后添加到环上
		// 最后在 hashMap 中添加虚拟节点和真实节点的映射关系
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	// 将环上的哈希值排序
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if m.IsEmpty() {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	// Binary search for appropriate replica.
	//顺时针找到第一个匹配的虚拟节点的下标 index
	index := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// m.keys 是一个环状结构，因此如果 index == len(m.keys)，则需将 index 置为 0
	if index == len(m.keys) {
		index = 0
	}
	// 从 m.keys 中获取到对应的哈希值，然后通过 m.hashMap 映射得到真实的节点
	return m.hashMap[m.keys[index]]
}
