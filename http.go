package main

import (
	"bytes"
	"dailzCache/consistentHash"
	pb "dailzCache/dailzCachepb"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_daiCache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// Context optionally specifies a context for the server to use when it
	// receives a request.
	// If nil, the server uses the request's context
	//Context func(r *Http.Request) context.Context

	// Transport optionally specifies an Http.RoundTripper for the client
	// to use when it makes a request.
	// If nil, the client uses Http.DefaultTransport.
	//Transport func(context.Context) Http.RoundTripper

	// this peer's base URL, e.g. "https://example.net:8000"
	self string

	// opts specifies the options.
	opts HTTPPoolOptions

	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistentHash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

// HTTPPoolOptions are the configurations of a HTTPPool.
type HTTPPoolOptions struct {
	// BasePath specifies the HTTP path that will serve daiCache requests.
	// If blank, it defaults to "/_daiCache/".
	BasePath string

	// Replicas specifies the number of key replicas on the consistent hash.
	// If blank, it defaults to 50.
	Replicas int

	// HashFn specifies the hash function of the consistent hash.
	// If blank, it defaults to crc32.ChecksumIEEE.
	HashFn consistentHash.Hash
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// NewHTTPPool initializes an HTTP pool of peers, and registers itself as a PeerPicker.
// For convenience, it also registers itself as an Http.Handler with Http.DefaultServeMux.
// The self argument should be a valid base URL that points to the current server,
// for example "http://example.net:8000".
func NewHTTPPool(self string) *HTTPPool {
	p := NewHTTPPoolOpts(self, nil)
	http.Handle(p.opts.BasePath, p)
	return p
}

var httpPoolMade bool

// NewHTTPPoolOpts initializes an HTTP pool of peers with the given options.
// Unlike NewHTTPPool, this function does not register the created pool as an HTTP handler.
// The returned *HTTPPool implements Http.Handler and must be registered using Http.Handle.
func NewHTTPPoolOpts(self string, opts *HTTPPoolOptions) *HTTPPool {
	if httpPoolMade {
		panic("daiCache: NewHTTPPool must be called only once")
	}

	httpPoolMade = true

	p := &HTTPPool{
		self:        self,
		httpGetters: make(map[string]*httpGetter),
	}

	if opts != nil {
		p.opts = *opts
	}
	if p.opts.BasePath == "" {
		p.opts.BasePath = defaultBasePath
	}
	if p.opts.Replicas == 0 {
		p.opts.Replicas = defaultReplicas
	}
	p.peers = consistentHash.New(p.opts.Replicas, p.opts.HashFn)

	RegisterPeerPicker(func() PeerPicker { return p })
	return p
}

// Set updates the pool's list of peers.
// Each peer value should be a valid base URL,
// for example "http://example.net:8000".
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//p.peers = consistentHash.New(p.opts.Replicas, p.opts.HashFn)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			//transport: p.Transport,
			baseURL: peer + p.opts.BasePath,
		}
	}
}

func (p *HTTPPool) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// 判断访问路径的前缀是否是 basePath
	//p.Log("%v", request.URL)
	if !strings.HasPrefix(request.URL.Path, p.opts.BasePath) {
		panic("HTTPPool serving unexpected path: " + request.URL.Path)
	}

	p.Log("%s %s", request.Method, request.URL.Path)

	// 约定访问路径的格式为 /basePath/groupName/key
	parts := strings.SplitN(request.URL.Path[len(p.opts.BasePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(writer, "bad request", http.StatusBadRequest)
	}

	groupName := parts[0]
	key := parts[1]

	//p.Log("%v %v", groupName, key)
	group := GetGroup(groupName)
	if group == nil {
		http.Error(writer, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 获取缓存数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// 使用 proto.Marshal() 编码 HTTP 响应
	body, err := proto.Marshal(&pb.GetResponse{Value: view.ByteSlice()})
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Write(body) // 将缓存值作为 httpResponse 的 body 返回
}

type httpGetter struct {
	baseURL string
}

// sync.Pool 使用对象重用机制，sync.Pool 用于存储那些被分配了但是没有被使用的，
// 未来可能会使用的值，这样不用再次经过内存分配，可以直接复用已有对象
// sync.Pool 大小可伸缩，高负载时会动态扩容，存放在池中的对象如果不活跃，会被自动清理
var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// 查询 key 对应的 value 时，从 in.Group 所在的 peer 中获取
func (h *httpGetter) Get(in *pb.GetRequest, out *pb.GetResponse) error {
	u := fmt.Sprintf("%v%v/%v",
		h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	//log.Println(u)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	// Get() 用于从对象池中获取对象，返回值是 interface{} 因此需要类型转换
	b := bufferPool.Get().(*bytes.Buffer)
	b.Reset()
	defer bufferPool.Put(b) // Put() 是在对象使用完毕后，返回对象池

	_, err = io.Copy(b, res.Body)
	//log.Println("[httpGetter]--Get:", b)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	// 使用 proto.Unmarshal() 解码 HTTP 响应
	err = proto.Unmarshal(b.Bytes(), out)
	if err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

func (p *HTTPPool) PickPeer(key string) (ProtoGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.peers.IsEmpty() {
		return nil, false
	}
	if peer := p.peers.Get(key); peer != p.self {
		p.Log("Pick Peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}
