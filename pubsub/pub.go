package pubsub

import (
	"net"
	"sync"
	"time"

	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/mq.v2/conn"
	"github.com/missionMeteora/toolkit/errors"
)

// NewPub will return a new publisher
func NewPub(addr string) (pp *Pub, err error) {
	var p Pub
	if p.l, err = net.Listen("tcp", addr); err != nil {
		return
	}

	p.sm = make(map[string]conn.Conn)
	p.out = journaler.New("Pub", addr)
	p.onDC = append(p.onDC, p.remove)

	pp = &p
	return
}

// Pub is a publisher
type Pub struct {
	mux sync.RWMutex
	out *journaler.Journaler

	l net.Listener

	// Subscriber map
	sm map[string]conn.Conn

	// On connect functions
	onC []conn.OnConnectFn
	// On disconnect functions
	onDC []conn.OnDisconnectFn

	closed bool
}

func (p *Pub) get(key string) (c conn.Conn, ok bool) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	c, ok = p.sm[key]
	return
}

func (p *Pub) getConn(key string) (c conn.Conn, ok bool) {
	p.mux.RLock()
	c, ok = p.get(key)
	p.mux.RUnlock()
	return

}

func (p *Pub) close(wg *sync.WaitGroup) (errs *errors.ErrorList) {
	errs = &errors.ErrorList{}
	if p.closed {
		errs.Push(errors.ErrIsClosed)
		return
	}

	errs.Push(p.l.Close())

	wg.Add(len(p.sm))
	for _, s := range p.sm {
		go func(c conn.Conn) {
			errs.Push(c.Close())
			wg.Done()
		}(s)
	}

	return
}

func (p *Pub) remove(c conn.Conn) {
	p.mux.Lock()
	delete(p.sm, c.Key())
	p.mux.Unlock()
}

// Listen will listen for inbound subscribers
func (p *Pub) Listen() {
	var err error
	for {
		var nc net.Conn
		if nc, err = p.l.Accept(); err != nil {
			return
		}

		p.mux.Lock()
		c := conn.New().OnConnect(p.onC...).OnDisconnect(p.onDC...)
		if err = c.Connect(nc); err != nil {
			p.out.Error("", err)
		} else {
			p.sm[c.Key()] = c
		}
		p.mux.Unlock()
	}
}

// OnConnect will append an OnConnect func
func (p *Pub) OnConnect(fns ...conn.OnConnectFn) {
	p.mux.Lock()
	p.onC = append(p.onC, fns...)
	p.mux.Unlock()
}

// OnDisconnect will append an onDisconnect func
func (p *Pub) OnDisconnect(fns ...conn.OnDisconnectFn) {
	p.mux.Lock()
	p.onDC = append(p.onDC, fns...)
	p.mux.Unlock()
}

// Put will broadcast a message to all subscribers
func (p *Pub) Put(b []byte) {
	p.mux.RLock()
	for _, c := range p.sm {
		c.Put(b)
	}
	p.mux.RUnlock()
}

// Subscribers will provide a map of subscribers with their creation time as the value
func (p *Pub) Subscribers() (sm map[string]time.Time) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	sm = make(map[string]time.Time, len(p.sm))
	for _, c := range p.sm {
		sm[c.Key()] = c.Created()
	}

	return
}

// Remove will remove a subscriber
func (p *Pub) Remove(key string) (err error) {
	c, ok := p.getConn(key)
	if !ok {
		return
	}

	if err = c.Close(); err != nil {
		// We had an error closing the connection. Let's ensure we properly remove the connection from our list
		p.remove(c)
	}

	return
}

// Close will close the Pubber
func (p *Pub) Close() error {
	var wg sync.WaitGroup
	p.mux.Lock()
	errs := p.close(&wg)
	p.mux.Unlock()
	wg.Wait()
	return errs.Err()
}
