package main

import (
	pb "dailzCache/dailzCachepb"
	"dailzCache/singleFlight"
	"math/rand"
	"sync"
)

// A Getter loads data for a key.
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
// a group of 1 or more machines.
type Group struct {
	name       string
	getter     Getter
	peersOnce  sync.Once
	cacheBytes int64 // sum of mainCache and hotCache size
	mainCache  cache
	hotCache   cache
	peers      PeerPicker
	loadGroup  *singleFlight.Group
	Stats      Stats
}

type Stats struct {
	Gets           AtomicInt // any Get request, including from peers
	CacheHits      AtomicInt // either cache was good
	PeerLoads      AtomicInt // either remote load or remote cache hit (not an error)
	PeerErrors     AtomicInt
	Loads          AtomicInt // (gets - cacheHits)
	LoadsDeduped   AtomicInt // after singleflight
	LocalLoads     AtomicInt // total good local loads
	LocalLoadErrs  AtomicInt // total bad local loads
	ServerRequests AtomicInt // gets that came over the network from peers
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	return newGroup(name, cacheBytes, getter, nil)
}

// NewGroup creates a new group, the name must be unique for each getter.
func newGroup(name string, cacheBytes int64, getter Getter, peers PeerPicker) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	if _, ok := groups[name]; ok {
		panic("duplicate registration of group " + name)
	}

	g := &Group{
		name:       name,
		getter:     getter,
		cacheBytes: cacheBytes,
		peers:      peers,
		loadGroup:  &singleFlight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g

}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) initPeers() {
	if g.peers == nil {
		g.peers = GetPeers(g.name)
	}
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	g.peersOnce.Do(g.initPeers)
	g.Stats.Gets.Add(1)

	value, cacheHit := g.lookupCache(key)
	if cacheHit {
		return value, nil
	}

	value, err := g.load(key)
	if err != nil {
		return ByteView{}, err
	}
	return value, err
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// load loads key either by invoking the getter locally or by sending it to another machine.
func (g *Group) load(key string) (ByteView, error) {
	g.Stats.Loads.Add(1)
	view, err := g.loadGroup.Do(key, func() (interface{}, error) {
		if value, cacheHit := g.lookupCache(key); cacheHit {
			g.Stats.CacheHits.Add(1)
			return value, nil
		}
		g.Stats.LoadsDeduped.Add(1)
		// 1.从一致性哈希中获取到存有 key 的 peer
		if peer, ok := g.peers.PickPeer(key); ok {
			// 2.使用 http 从刚刚获取到的 peer 中获取 key 对应的 value
			value, err := g.getFromPeer(key, peer)
			if err == nil {
				g.Stats.PeerLoads.Add(1)
				return value, nil
			}
			g.Stats.PeerErrors.Add(1)
		}
		value, err := g.getLocally(key)
		if err != nil {
			g.Stats.LocalLoadErrs.Add(1)
			return ByteView{}, err
		}
		g.Stats.LocalLoads.Add(1)
		g.populateCache(key, value, &g.mainCache)
		return value, nil
	})
	if err == nil {
		return view.(ByteView), nil
	}
	return ByteView{}, err
}

func (g *Group) lookupCache(key string) (value ByteView, ok bool) {
	if g.cacheBytes <= 0 {
		return
	}
	value, ok = g.mainCache.get(key)
	if ok {
		return
	}
	value, ok = g.hotCache.get(key)
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		//fmt.Println(err)
		return ByteView{}, err
	}
	value := ByteView{data: cloneBytes(bytes)}
	//g.populateCache(key, value)
	return value, nil
}

func (g *Group) getFromPeer(key string, peer ProtoGetter) (ByteView, error) {
	req := &pb.GetRequest{
		Group: g.name,
		Key:   key,
	}
	res := &pb.GetResponse{}

	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{data: res.Value}
	//在本地 peer 中备份热点数据，备份依据是根据随机数进行判断，有十分之一的概率进行备份
	// 这样，如果每个 peer 存储 1G 数据，那么是个 peer 就能存储 10 G 数据，然后有 1G 的热点数据，因此就能存储 9G 数据
	if rand.Intn(10) == 0 {
		g.populateCache(key, value, &g.hotCache)
	}
	return value, nil
}
func (g *Group) populateCache(key string, value ByteView, cache *cache) {
	if g.cacheBytes <= 0 {
		return
	}

	cache.add(key, value)

	// 判断是否超出 g.cacheByte，如果是，那就需要进行删除
	for {
		mainBytes := g.mainCache.bytes()
		hotBytes := g.hotCache.bytes()
		if mainBytes+hotBytes <= g.cacheBytes {
			return
		}
		victim := &g.mainCache
		if hotBytes > mainBytes/8 {
			victim = &g.hotCache
		}
		victim.removeOldest()
	}
}
