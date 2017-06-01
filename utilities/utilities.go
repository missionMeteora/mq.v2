package utilities

import (
	"net"

	"github.com/missionMeteora/mq.v2/conn"
	"github.com/missionMeteora/toolkit/errors"
)

const (
	// ErrInvalidCredentials is returned when invalid credentials are provided
	ErrInvalidCredentials = errors.Error("invalid credentials")
)

// NewBasicAuth will return a new basic auth
func NewBasicAuth(user, pass string) *BasicAuth {
	return &BasicAuth{
		user: user,
		pass: pass,
	}
}

// BasicAuth is a basic authentication middleware
type BasicAuth struct {
	user string
	pass string
}

// Check will check credentials for an inbound connection
func (b *BasicAuth) Check(c conn.Conn) (err error) {
	var user, pass string
	if user, err = c.GetStr(); err != nil {
		return
	}

	if pass, err = c.GetStr(); err != nil {
		return
	}

	if user != b.user || pass != b.pass {
		// Send error along the line
		c.Put([]byte(ErrInvalidCredentials.Error()))
		return ErrInvalidCredentials
	}

	c.Put([]byte("OK"))
	return
}

// Auth will send an authentication request to an outbound connection
func (b *BasicAuth) Auth(c conn.Conn) (err error) {
	if err = c.Put([]byte(b.user)); err != nil {
		return
	}

	if err = c.Put([]byte(b.pass)); err != nil {
		return
	}

	var resp string
	if resp, err = c.GetStr(); err != nil {
		return
	}

	if resp != "OK" {
		return errors.Error(resp)
	}

	return
}

// Listen will listen and return a net connection
func Listen(addr string) (nc net.Conn, err error) {
	var l net.Listener
	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}
	defer l.Close()

	return l.Accept()
}

// Dial will dial a requested address and return a
func Dial(addr string) (nc net.Conn, err error) {
	return net.Dial("tcp", addr)
}
