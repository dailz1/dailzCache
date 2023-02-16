package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
)

func TestHTTPPool(t *testing.T) {
	NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:9999"
	peers := NewHTTPPool(addr)
	peers.Set(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}

/*var (
	peerAddrs = flag.String("test_peer_addrs", "", "Comma-separated list of peer addresses; used by TestHTTPPool")
	peerIndex = flag.Int("test_peer_index", -1, "Index of which peer this child is; used by TestHTTPPool")
	peerChild = flag.Bool("test_peer_child", false, "True if running as a child process; used by TestHTTPPool")
)*/

var peerChild = false

var peerAddrs = []string{"localhost:8001", "localhost:8002", "localhost:8003"}

func TestHTTPPool2(t *testing.T) {
	beChildForTestHTTPPool()
	log.Println("cache is running at", peerAddrs[0])
}

func beChildForTestHTTPPool() {
	addrs := peerAddrs
	p := NewHTTPPool("http://" + addrs[0])
	p.Set(addToURL(addrs)...)

	getter := GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	})
	NewGroup("test", 1<<20, getter)
	log.Fatal(http.ListenAndServe(addrs[0], p))
}

func addToURL(addr []string) []string {
	url := make([]string, len(addr))
	for i := range addr {
		url[i] = "http://" + addr[i]
	}
	return url
}
