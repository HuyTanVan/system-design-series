package store

type Node struct {
	key   string // 8 bytes
	value string // 8 bytes
	next  *Node  // 8 bytes
	prev  *Node  // 8 bytes
}

func (n *Node) size() int {
	return len(n.key) + len(n.value) + 16 // 16 bytes for next and prev pointers
}

type LRUCache struct {
	capacity int // cap(MB) * 1024 * 1024 = bytes
	size     int // current size of the cache
	cache    map[string]*Node
	head     *Node
	tail     *Node
}

func NewLRUCache(capacity int) *LRUCache {
	dummyHead, dummyTail := &Node{}, &Node{}
	dummyHead.next = dummyTail
	dummyTail.prev = dummyHead
	return &LRUCache{
		capacity: capacity,
		size:     0,
		cache:    make(map[string]*Node),
		head:     dummyHead,
		tail:     dummyTail,
	}
}
func (l *LRUCache) insertToHead(node *Node) {
	cur := l.head.next
	cur.prev = node
	l.head.next = node
	node.prev = l.head
	node.next = cur
}
func (l *LRUCache) moveToHead(node *Node) {
	l.removeNode(node)
	l.insertToHead(node)
}
func (l *LRUCache) removeNode(node *Node) {
	prev := node.prev
	next := node.next
	prev.next = next
	next.prev = prev
}

func (l *LRUCache) Get(key string) (string, bool) {
	node, ok := l.cache[key]
	if !ok {
		return "", false
	}
	l.moveToHead(node)
	return node.value, true
}

func (l *LRUCache) Set(key, value string) {
	if node, ok := l.cache[key]; ok {
		l.size -= node.size() // subtract current size
		node.value = value    // update new value
		l.size += node.size() // add new size
		l.moveToHead(node)    // move to head
		return
	}

	node := &Node{key: key, value: value}
	l.cache[key] = node
	l.insertToHead(node)
	l.size += node.size()
	for l.size > l.capacity {
		tail := l.tail.prev
		l.size -= tail.size()
		l.removeNode(tail)
		delete(l.cache, tail.key)
	}
}

func (l *LRUCache) Del(key string) int {
	if node, ok := l.cache[key]; ok {
		l.size -= node.size() // subtract size
		l.removeNode(node)    // rm node
		delete(l.cache, key)  // rm from map
		return 1
	}
	return 0
}
