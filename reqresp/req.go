package reqresp

import (
	"github.com/missionMeteora/mq/conn"
)

// Request is the request type
type Request struct {
	c *conn.Conn
}

// Request is the request type
func (r *Request) Request(b []byte, fn func([]byte)) (err error) {
	if err = r.c.Put(b); err != nil {
		return
	}

	return r.c.Get(fn)
}
