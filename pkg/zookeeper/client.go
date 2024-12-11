package zookeeper

import (
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/pkg/errors"
)

var (
	zkClientPool   clientPool
	clientPoolOnce sync.Once

	// ErrNilZkClientConn no conn error
	ErrNilZkClientConn = errors.New("Zookeeper Client{conn} is nil")
	ErrStatIsNil       = errors.New("Stat of the node is nil")
)

// Client represents zookeeper Client Configuration
type Client struct {
	sync.RWMutex

	name              string
	zkAddrs           []string
	conn              *zk.Conn
	activeNumber      uint32
	timeout           time.Duration
	valid             uint32
	share             bool
	initialized       uint32
	reconnectCh       chan struct{}
	eventRegistry     map[string][]chan zk.Event
	eventRegistryLock sync.RWMutex
	eventHandler      EventHandler
	session           <-chan zk.Event
}

type clientPool struct {
	sync.Mutex
	zkClient map[string]*Client
}

// EventHandler interface
type EventHandler interface {
	HandleEvent(z *Client)
}

// DefaultHandler is default handler for zk event
type DefaultHandler struct{}

func initClientPool() {
	zkClientPool.zkClient = make(map[string]*Client)
}

// NewClient will create a Client
func NewClient(name string, zkAddrs []string, share bool, opts ...zkClientOption) (*Client, error) {
	if !share {
		return newClient(name, zkAddrs, share, opts...)
	}
	clientPoolOnce.Do(initClientPool)
	zkClientPool.Lock()
	defer zkClientPool.Unlock()
	if zkClient, ok := zkClientPool.zkClient[name]; ok {
		zkClient.activeNumber++
		return zkClient, nil
	}
	newZkClient, err := newClient(name, zkAddrs, share, opts...)
	if err != nil {
		return nil, err
	}
	zkClientPool.zkClient[name] = newZkClient
	return newZkClient, nil
}

func newClient(name string, zkAddrs []string, share bool, opts ...zkClientOption) (*Client, error) {
	newZkClient := &Client{
		name:          name,
		zkAddrs:       zkAddrs,
		activeNumber:  0,
		share:         share,
		reconnectCh:   make(chan struct{}),
		eventRegistry: make(map[string][]chan zk.Event),
		session:       make(<-chan zk.Event),
		eventHandler:  &DefaultHandler{},
	}
	for _, opt := range opts {
		opt(newZkClient)
	}
	if err := newZkClient.createConn(); err != nil {
		return nil, err
	}
	newZkClient.activeNumber++
	return newZkClient, nil
}

func (c *Client) createConn() error {
	var err error

	// connect to zookeeper
	c.conn, c.session, err = zk.Connect(c.zkAddrs, c.timeout, zk.WithLogInfo(false))
	if err != nil {
		return err
	}
	atomic.StoreUint32(&c.valid, 1)
	go c.eventHandler.HandleEvent(c)
	return nil
}

// HandleEvent handles zookeeper events
// nolint
func (d *DefaultHandler) HandleEvent(z *Client) {
	var (
		ok    bool
		state int
		event zk.Event
	)
	for {
		select {
		case event, ok = <-z.session:
			if !ok {
				// channel already closed
				return
			}
			switch event.State {
			case zk.StateDisconnected:
				atomic.StoreUint32(&z.valid, 0)
			case zk.StateConnected:
				z.eventRegistryLock.RLock()
				for path, a := range z.eventRegistry {
					if strings.HasPrefix(event.Path, path) {
						for _, e := range a {
							e <- event
						}
					}
				}
				z.eventRegistryLock.RUnlock()
			case zk.StateConnecting, zk.StateHasSession:
				if state == (int)(zk.StateHasSession) {
					continue
				}
				if event.State == zk.StateHasSession {
					atomic.StoreUint32(&z.valid, 1)
					//if this is the first connection, don't trigger reconnect event
					if !atomic.CompareAndSwapUint32(&z.initialized, 0, 1) {
						close(z.reconnectCh)
						z.reconnectCh = make(chan struct{})
					}
				}
				z.eventRegistryLock.RLock()
				if a, ok := z.eventRegistry[event.Path]; ok && 0 < len(a) {
					for _, e := range a {
						e <- event
					}
				}
				z.eventRegistryLock.RUnlock()
			}
			state = (int)(event.State)
		}
	}
}

// RegisterEvent registers zookeeper events
func (c *Client) RegisterEvent(zkPath string, event chan zk.Event) {
	if zkPath == "" {
		return
	}

	c.eventRegistryLock.Lock()
	defer c.eventRegistryLock.Unlock()
	a := c.eventRegistry[zkPath]
	a = append(a, event)
	c.eventRegistry[zkPath] = a
}

// UnregisterEvent unregisters zookeeper events
func (c *Client) UnregisterEvent(zkPath string, event chan zk.Event) {
	if zkPath == "" {
		return
	}

	c.eventRegistryLock.Lock()
	defer c.eventRegistryLock.Unlock()
	infoList, ok := c.eventRegistry[zkPath]
	if !ok {
		return
	}
	for i, e := range infoList {
		if e == event {
			infoList = append(infoList[:i], infoList[i+1:]...)
		}
	}
	if len(infoList) == 0 {
		delete(c.eventRegistry, zkPath)
	} else {
		c.eventRegistry[zkPath] = infoList
	}
}

// ConnValid validates zookeeper connection
func (c *Client) ConnValid() bool {
	return atomic.LoadUint32(&c.valid) == 1
}

// Create will create the node recursively, which means that if the parent node is absent,
// it will create parent node first.
// And the value for the basePath is ""
func (c *Client) Create(basePath string) error {
	return c.CreateWithValue(basePath, []byte{})
}

// CreateWithValue will create the node recursively, which means that if the parent node is absent,
// it will create parent node first.
// basePath should start with "/"
func (c *Client) CreateWithValue(basePath string, value []byte) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNilZkClientConn
	}

	if !strings.HasPrefix(basePath, string(os.PathSeparator)) {
		basePath = string(os.PathSeparator) + basePath
	}
	paths := strings.Split(basePath, string(os.PathSeparator))
	// Check the ancestor's path
	for idx := 2; idx < len(paths); idx++ {
		tmpPath := strings.Join(paths[:idx], string(os.PathSeparator))
		_, err := conn.Create(tmpPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil && !errors.Is(err, zk.ErrNodeExists) {
			return errors.WithMessagef(err, "Error while invoking zk.Create(path:%s), the reason maybe is: ", tmpPath)
		}
	}

	_, err := conn.Create(basePath, value, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		return err
	}
	return nil
}

// CreateTempWithValue will create the node recursively, which means that if the parent node is absent,
// it will create parent node first，and set value in last child path
// If the path exist, it will update data
func (c *Client) CreateTempWithValue(basePath string, value []byte) error {
	var (
		err     error
		tmpPath string
	)

	conn := c.getConn()
	if conn == nil {
		return ErrNilZkClientConn
	}

	if !strings.HasPrefix(basePath, string(os.PathSeparator)) {
		basePath = string(os.PathSeparator) + basePath
	}
	pathSlice := strings.Split(basePath, string(os.PathSeparator))[1:]
	length := len(pathSlice)
	for i, str := range pathSlice {
		tmpPath = path.Join(tmpPath, string(os.PathSeparator), str)
		// last child need be ephemeral
		if i == length-1 {
			_, err = conn.Create(tmpPath, value, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
			if err != nil {
				return errors.WithMessagef(err, "Error while invoking zk.Create(path:%s), the reason maybe is: ", tmpPath)
			}
			break
		}
		// we need ignore node exists error for those parent node
		_, err = conn.Create(tmpPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil && !errors.Is(err, zk.ErrNodeExists) {
			return errors.WithMessagef(err, "Error while invoking zk.Create(path:%s), the reason maybe is: ", tmpPath)
		}
	}

	return nil
}

// Delete will delete basePath
func (c *Client) Delete(basePath string) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNilZkClientConn
	}
	return errors.WithMessagef(conn.Delete(basePath, -1), "Delete(basePath:%s)", basePath)
}

// RegisterTemp registers temporary node by @basePath and @node
func (c *Client) RegisterTemp(basePath string, node string) (string, error) {
	zkPath := path.Join(basePath) + string(os.PathSeparator) + node
	conn := c.getConn()
	if conn == nil {
		return "", ErrNilZkClientConn
	}
	tmpPath, err := conn.Create(zkPath, []byte(""), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))

	if err != nil {
		return zkPath, errors.WithStack(err)
	}

	return tmpPath, nil
}

// RegisterTempSeq register temporary sequence node by @basePath and @data
func (c *Client) RegisterTempSeq(basePath string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	err = ErrNilZkClientConn
	conn := c.getConn()
	if conn != nil {
		tmpPath, err = conn.Create(
			path.Join(basePath)+string(os.PathSeparator),
			data,
			zk.FlagEphemeral|zk.FlagSequence,
			zk.WorldACL(zk.PermAll),
		)
	}
	if err != nil && !errors.Is(err, zk.ErrNodeExists) {
		return "", errors.WithStack(err)
	}
	return tmpPath, nil
}

// GetChildrenW gets children watch by @path
func (c *Client) GetChildrenW(path string) ([]string, <-chan zk.Event, error) {
	conn := c.getConn()
	if conn == nil {
		return nil, nil, ErrNilZkClientConn
	}
	children, stat, watcher, err := conn.ChildrenW(path)

	if err != nil {
		return nil, nil, errors.WithMessagef(err, "Error while invoking zk.ChildrenW(path:%s), the reason maybe is: ", path)
	}
	if stat == nil {
		return nil, nil, errors.WithMessagef(ErrStatIsNil, "Error while invokeing zk.ChildrenW(path:%s), the reason is: ", path)
	}

	return children, watcher.EvtCh, nil
}

// GetChildren gets children by @path
func (c *Client) GetChildren(path string) ([]string, error) {
	conn := c.getConn()
	if conn == nil {
		return nil, ErrNilZkClientConn
	}
	children, stat, err := conn.Children(path)

	if err != nil {
		return nil, errors.WithMessagef(err, "Error while invoking zk.Children(path:%s), the reason maybe is: ", path)
	}
	if stat == nil {
		return nil, errors.Errorf("Error while invokeing zk.Children(path:%s), the reason is that the stat is nil", path)
	}

	return children, nil
}

// ExistW to judge watch whether it exists or not by @zkPath
func (c *Client) ExistW(zkPath string) (<-chan zk.Event, error) {
	conn := c.getConn()
	if conn == nil {
		return nil, ErrNilZkClientConn
	}
	_, _, watcher, err := conn.ExistsW(zkPath)

	if err != nil {
		return nil, errors.WithMessagef(err, "zk.ExistsW(path:%s)", zkPath)
	}

	return watcher.EvtCh, nil
}

// GetContent gets content by @zkPath
func (c *Client) GetContent(zkPath string) ([]byte, *zk.Stat, error) {
	return c.conn.Get(zkPath)
}

// SetContent set content of zkPath
func (c *Client) SetContent(zkPath string, content []byte, version int32) (*zk.Stat, error) {
	return c.conn.Set(zkPath, content, version)
}

// getConn gets zookeeper connection safely
func (c *Client) getConn() *zk.Conn {
	if c == nil {
		return nil
	}
	c.RLock()
	defer c.RUnlock()
	return c.conn
}

// Reconnect gets zookeeper reconnect event
func (c *Client) Reconnect() <-chan struct{} {
	return c.reconnectCh
}

// GetEventHandler gets zookeeper event handler
func (c *Client) GetEventHandler() EventHandler {
	return c.eventHandler
}

func (c *Client) Close() {
	if c.share {
		zkClientPool.Lock()
		defer zkClientPool.Unlock()
		c.activeNumber--
		if c.activeNumber == 0 {
			c.conn.Close()
			delete(zkClientPool.zkClient, c.name)
		}
	} else {
		c.Lock()
		conn := c.conn
		c.activeNumber--
		c.conn = nil
		c.Unlock()
		if conn != nil {
			conn.Close()
		}
	}
}
