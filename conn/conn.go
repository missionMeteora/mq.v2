package conn

import (
	"bytes"
	"net"
	"sync"
	"time"

	"github.com/missionMeteora/toolkit/errors"
	"github.com/missionMeteora/uuid"
)

const (
	// ErrCannotConnect is returned when a connection is not idle when Connect() is called
	ErrCannotConnect = errors.Error("cannot connect to connected or closed connections")
	// ErrIsIdle is returned when an action is attempted on an idle connection
	ErrIsIdle = errors.Error("cannot perform action on idle connection")
)

const (
	stateIdle uint8 = iota
	stateConnected
	stateClosed
)

const (
	// noCopySize is the size limit for copying values into the write buffer on puts
	noCopySize = 1024 * 32
)

// New will return a new connection
func New() Conn {
	var c conn
	c.key = uuid.New()
	c.wbuf = bytes.NewBuffer(nil)
	return &c
}

// Conn is a connection interface
type Conn interface {
	Connect(nc net.Conn) (err error)
	Key() string
	Created() time.Time
	OnConnect(fns ...OnConnectFn) Conn
	OnDisconnect(fns ...OnDisconnectFn) Conn
	Get(fn func([]byte)) (err error)
	GetStr() (msg string, err error)
	Put(b []byte) (err error)
	Close() (err error)
}

// conn is a connection
type conn struct {
	mux sync.RWMutex
	nc  net.Conn

	key  uuid.UUID
	rbuf buffer
	wbuf *bytes.Buffer
	l    lengthy

	onC []OnConnectFn
	onD []OnDisconnectFn

	mlen uint64

	state uint8
}

// get is a raw internal call for getting a message, does not handle locking nor post-get cleanup
func (c *conn) get(fn func([]byte)) (err error) {
	// Let's ensure our connection is not closed or idle
	switch c.state {
	case stateClosed:
		return errors.ErrIsClosed
	case stateIdle:
		return ErrIsIdle
	}

	// Read message length
	if c.mlen, err = c.l.Read(c.nc); err != nil {
		return
	}

	// Read message
	if err = c.rbuf.ReadN(c.nc, c.mlen); err != nil {
		return
	}

	if fn != nil {
		// Please do not use the bytes outside of the called functions\
		// I'll be a sad panda if you create a race condition
		fn(c.rbuf.Bytes())
	}

	return
}

// put is the raw internal call for sending a message, does not handle locking
func (c *conn) put(b []byte) (err error) {
	// Let's ensure our connection is not closed or idle
	switch c.state {
	case stateClosed:
		return errors.ErrIsClosed
	case stateIdle:
		return ErrIsIdle
	}

	blen := uint64(len(b))
	if blen < noCopySize {
		return c.smallWrite(b, blen)
	}

	return c.largeWrite(b, blen)

}

func (c *conn) smallWrite(b []byte, blen uint64) (err error) {
	// Write the message length
	if err = c.l.Write(c.wbuf, blen); err != nil {
		return
	}

	c.wbuf.Write(b)

	// Write message to net.Conn
	_, err = c.nc.Write(c.wbuf.Bytes())
	c.wbuf.Reset()
	return
}

func (c *conn) largeWrite(b []byte, blen uint64) (err error) {
	// Write the message length
	if err = c.l.Write(c.nc, blen); err != nil {
		return
	}

	// Write message to net.Conn
	_, err = c.nc.Write(b)
	return
}

func (c *conn) onConnect() (err error) {
	for _, fn := range c.onC {
		if err = fn(c); err != nil {
			return
		}
	}

	return
}

func (c *conn) onDisconnect() {
	for _, fn := range c.onD {
		fn(c)
	}
}

func (c *conn) setConnection(nc net.Conn) (err error) {
	c.mux.Lock()
	if c.state != stateIdle {
		err = ErrCannotConnect
	} else {
		c.nc = nc
		c.state = stateConnected
	}
	c.mux.Unlock()
	return
}

func (c *conn) setIdle() {
	if c.state == stateIdle {
		// conn is already idle, return early
		return
	}

	if c.nc != nil {
		// net.Conn exists, let's close it
		c.nc.Close()
	}

	// Update conn values
	c.nc = nil
	c.state = stateIdle
}

// close will set the state to closed
func (c *conn) close() (err error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.state == stateClosed {
		// conn is already closed
		return errors.ErrIsClosed
	}

	c.state = stateClosed
	return
}

// Connect will connect a connection to a net.Conn
func (c *conn) Connect(nc net.Conn) (err error) {
	if err = c.setConnection(nc); err != nil {
		return
	}

	return c.onConnect()
}

// Key will return the generated key for a connection
func (c *conn) Key() string {
	// We don't need to lock because the key never changes after creation
	return c.key.String()
}

// Created will return the created time for a connection
func (c *conn) Created() time.Time {
	// We don't need to lock because the key never changes after creation
	return c.key.Time()
}

// OnConnect will append an OnConnect func, referenced conn is returned for chaining
// Note: This function is intended to be called before connection, it is NOT thread-safe
func (c *conn) OnConnect(fns ...OnConnectFn) Conn {
	c.onC = append(c.onC, fns...)
	return c
}

// OnDisconnect will append an onDisconnect func, referenced conn is returned for chaining
// Note: This function is intended to be called before connection, it is NOT thread-safe
func (c *conn) OnDisconnect(fns ...OnDisconnectFn) Conn {
	c.onD = append(c.onD, fns...)
	return c
}

// Get will get a message
// Note: If fn is nil, the message will be read and discarded
func (c *conn) Get(fn func([]byte)) (err error) {
	c.mux.Lock()
	if err = c.get(fn); err != nil {
		c.setIdle()
	}
	c.mux.Unlock()
	return
}

// GetStr will get a message as a string
// Note: This is just a helper utility
func (c *conn) GetStr() (msg string, err error) {
	err = c.Get(func(b []byte) {
		msg = string(b)
	})

	return
}

// Put will put a message
func (c *conn) Put(b []byte) (err error) {
	c.mux.Lock()
	err = c.put(b)
	c.mux.Unlock()
	return
}

// Close will close a connection
func (c *conn) Close() (err error) {
	if err = c.close(); err != nil {
		// Connection is already closed, return early
		return
	}

	// Call onDisconnect before we close the net.Conn
	c.onDisconnect()

	c.mux.Lock()
	if c.nc != nil {
		// Close net.Conn
		err = c.nc.Close()
	}
	c.mux.Unlock()

	return
}

// OnConnectFn is called when a connection occurs
type OnConnectFn func(Conn) error

// OnDisconnectFn is called when a connection ends
type OnDisconnectFn func(Conn)
