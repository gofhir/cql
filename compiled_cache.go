package cql

import (
	"container/list"
	"hash/fnv"
	"sync"

	"github.com/gofhir/cql/ast"
	"github.com/gofhir/cql/sema"
)

// sourceKey is the cache key for a CQL source.
func sourceKey(cqlSource string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(cqlSource))
	return h.Sum64()
}

// DefaultCompiledCacheSize is how many parsed libraries an engine keeps when no
// size is chosen.
//
// The cache used to be unbounded, which is fine for a process that evaluates
// its own fixed set of libraries and is not fine for a server that evaluates
// CQL arriving over HTTP: the key is a hash of the source, so a caller who can
// vary one space can add an entry per request and nothing ever removes one.
// That is memory exhaustion reachable by anyone who can reach the endpoint.
//
// A few hundred entries covers the libraries a real deployment evaluates while
// putting a ceiling on the ones it will never see again.
const DefaultCompiledCacheSize = 256

// cachedLibrary is a compiled library together with the source it came from.
//
// The source is kept so a hit can be confirmed. Indexing by hash alone means a
// collision hands back a different library's AST — quietly, and with no way for
// the caller to tell. fnv64a over arbitrary CQL is not a cryptographic digest;
// two sources colliding is unlikely, not impossible, and the failure is silent
// and total.
type cachedLibrary struct {
	source string
	lib    *ast.Library
	plan   *sema.Result
}

// compiledCache holds parsed libraries, evicting the least recently used once
// it is full.
//
// A mutex replaces the sync.Map this used to be. Eviction needs the recency
// order updated on every hit, which sync.Map cannot express, and the trade is
// cheap: a miss costs a full ANTLR parse — tens of milliseconds for a
// measure-sized library — so the lock is not what anyone waits on.
type compiledCache struct {
	mu    sync.Mutex
	limit int // <0 unbounded, 0 disabled, >0 entry limit
	byKey map[uint64]*list.Element
	order *list.List // front is most recently used
}

type cacheEntry struct {
	key   uint64
	value cachedLibrary
}

func newCompiledCache(limit int) *compiledCache {
	return &compiledCache{
		limit: limit,
		byKey: make(map[uint64]*list.Element),
		order: list.New(),
	}
}

// load returns the entry for a key, but only when it came from this exact
// source: a hash collision is a miss, not a wrong answer.
func (c *compiledCache) load(key uint64, source string) (cachedLibrary, bool) {
	if c == nil || c.limit == 0 {
		return cachedLibrary{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.byKey[key]
	if !ok {
		return cachedLibrary{}, false
	}
	entry, _ := el.Value.(*cacheEntry)
	if entry.value.source != source {
		// A collision. Leave the entry alone — including its place in the
		// recency order, which this lookup did not use.
		return cachedLibrary{}, false
	}
	c.order.MoveToFront(el)
	return entry.value, true
}

// store records a parse, evicting the least recently used entry if that puts
// the cache over its limit.
func (c *compiledCache) store(key uint64, value cachedLibrary) {
	if c == nil || c.limit == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.byKey[key]; ok {
		entry, _ := el.Value.(*cacheEntry)
		if entry.value.source == value.source {
			c.order.MoveToFront(el)
			return
		}
		// Whichever source reached the slot first keeps it, as before; the
		// other pays the parse each time rather than evicting a good entry
		// over a collision.
		return
	}
	c.byKey[key] = c.order.PushFront(&cacheEntry{key: key, value: value})
	if c.limit > 0 {
		for c.order.Len() > c.limit {
			oldest := c.order.Back()
			if oldest == nil {
				break
			}
			c.order.Remove(oldest)
			if entry, ok := oldest.Value.(*cacheEntry); ok {
				delete(c.byKey, entry.key)
			}
		}
	}
}

// len reports how many parses are held. It exists for tests.
func (c *compiledCache) len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
