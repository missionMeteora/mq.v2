package utilities

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/missionMeteora/mq/conn"
)

func TestBasicAuth(t *testing.T) {
	var wg sync.WaitGroup
	ba := NewBasicAuth("foo", "bar")
	wg.Add(2)

	go func() {
		var (
			s   = conn.New()
			nc  net.Conn
			err error
		)

		defer wg.Done()

		if nc, err = Listen(":16777"); err != nil {
			t.Fatal(err)
		}

		s.OnConnect(ba.Check)
		if err = s.Connect(nc); err != nil {
			t.Fatal(err)
		}

		if err = s.Close(); err != nil {
			return
		}
	}()

	time.Sleep(time.Millisecond * 10)

	go func() {
		var (
			c   = conn.New()
			nc  net.Conn
			err error
		)

		defer wg.Done()

		if nc, err = Dial(":16777"); err != nil {
			t.Fatal(err)
		}

		c.OnConnect(ba.Auth)
		if err = c.Connect(nc); err != nil {
			t.Fatal(err)
		}

		if err = c.Close(); err != nil {
			return
		}
	}()

	wg.Wait()
}

func TestBasicAuthFail(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		var (
			s   = conn.New()
			ba  = NewBasicAuth("foo", "bar")
			nc  net.Conn
			err error
		)

		defer wg.Done()

		if nc, err = Listen(":16777"); err != nil {
			t.Fatal(err)
		}

		s.OnConnect(ba.Check)
		if err = s.Connect(nc); err == nil {
			t.Fatal("expected invalid credentials")
		}
	}()

	time.Sleep(time.Millisecond * 10)

	go func() {
		var (
			c   = conn.New()
			ba  = NewBasicAuth("foo", "baz")
			nc  net.Conn
			err error
		)

		defer wg.Done()

		if nc, err = Dial(":16777"); err != nil {
			t.Fatal(err)
		}

		c.OnConnect(ba.Auth)
		if err = c.Connect(nc); err == nil {
			t.Fatal("expected invalid credentials")
		}
	}()

	wg.Wait()
}
