package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var (
	once  sync.Once
	group *Group

	//stringc = make(chan string)

	// cacheFills is the number of times stringGroup or
	// protoGroup's Getter have been called. Read using the
	// cacheFills function.
	//cacheFills AtomicInt
)

const (
	stringGroupName = "string-group"
	//protoGroupName  = "proto-group"
	//testMessageType = "google3/net/groupcache/go/test_proto.TestMessage"
	//fromChan        = "from-chan"
	cacheSize = 1 << 20
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createDB() {
	for i := 0; i < 1000; i++ {
		db[strconv.Itoa(i)] = strconv.Itoa(i) + strconv.Itoa(i)
	}
}

func createGroup() {
	group = NewGroup(stringGroupName, cacheSize, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if value, ok := db[key]; ok {
				return []byte(value), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

}

func startCacheServer(addr string, addrs []string, group *Group) {
	/*opts := &HTTPPoolOptions{
		BasePath: stringGroupName,
		Replicas: 0,
		HashFn:   nil,
	}*/
	peers := NewHTTPPool(addr)
	peers.Set(addrs...)
	group.RegisterPeers(peers)
	log.Println("dailzCache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, group *Group) {
	http.Handle("/api", http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			//log.Println(request.URL)
			key := request.URL.Query().Get("key")
			//log.Println(key)
			view, err := group.Get(key)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Content-Type", "application/octet-stream")
			writer.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	createDB()
	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	createGroup()

	//go startAPIServer(apiAddr, group)
	if api {
		go startAPIServer(apiAddr, group)
	}
	startCacheServer(addrMap[port], []string(addrs), group)
	/*for _, value := range addrMap {
		startCacheServer(value, addrs, group)
	}*/
}
