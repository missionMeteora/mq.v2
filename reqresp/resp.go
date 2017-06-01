package reqresp

import (
	"github.com/missionMeteora/mq.v2/conn"
)

// Response is the response type
type Response struct {
	c  conn.Conn
	fn func(byte) (resp []byte, err error)
}

func (r *Response) listen() {
	var err error
	for {
		var resp []byte
		if err = r.c.Get(func(b []byte) {
			resp, err = r.fn(b)
		}); err != nil {
			// Handle error
		}

		if err = r.c.Put(resp); err != nil {
			return
		}
	}
}
