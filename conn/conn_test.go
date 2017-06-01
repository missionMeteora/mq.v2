package conn

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"bytes"
	"github.com/go-mangos/mangos"
	mpair "github.com/go-mangos/mangos/protocol/pair"
	mtcp "github.com/go-mangos/mangos/transport/tcp"
)

var (
	testVal    = []byte("hello world")
	testSetVal []byte
)

func TestConn(t *testing.T) {
	var l net.Listener
	s := New()
	c := New()
	done := make(chan error, 1)

	defer s.Close()
	defer c.Close()

	go func() {
		var (
			nc  net.Conn
			msg string
			err error
		)

		if l, err = net.Listen("tcp", ":1337"); err != nil {
			return
		}

		if nc, err = l.Accept(); err != nil {
			return
		}

		if err = s.Connect(nc); err != nil {
			return
		}

		if err = s.Put([]byte("hello 0")); err != nil {
			done <- err
		}

		if msg, err = s.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 1" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		if err = s.Put([]byte("hello 2")); err != nil {
			done <- err
		}

		if msg, err = s.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 3" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		done <- nil
	}()

	time.Sleep(time.Millisecond * 10)

	go func() {
		var (
			nc  net.Conn
			msg string
			err error
		)

		if nc, err = net.Dial("tcp", ":1337"); err != nil {
			done <- err
		}

		c = New()
		c.Connect(nc)

		if msg, err = c.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 0" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		if err = c.Put([]byte("hello 1")); err != nil {
			done <- err
		}

		if msg, err = c.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 2" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		if err = c.Put([]byte("hello 3")); err != nil {
			done <- err
		}

		// Do close shit here
	}()

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	l.Close()
}

func TestBuffer(t *testing.T) {
	var b buffer
	buf := bytes.NewBuffer(nil)
	buf.WriteString("hello world")

	b.ReadN(buf, 4)
	if string(b.Bytes()) != "hell" {
		t.Fatal("invalid value")
	}

	b.ReadN(buf, 4)
	if string(b.Bytes()) != "o wo" {
		t.Fatal("invalid value")
	}

	b.ReadN(buf, 4)
	if string(b.Bytes()) != "rld" {
		t.Fatal("invalid value")
	}
}

func BenchmarkMQ_32B(b *testing.B) {
	benchmarkMQ(b, make([]byte, 32))
}

func BenchmarkMQ_128B(b *testing.B) {
	benchmarkMQ(b, make([]byte, 128))
}

func BenchmarkMQ_1KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024))
}

func BenchmarkMQ_4KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*4))
}

func BenchmarkMQ_16KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*16))
}

func BenchmarkMQ_32KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*32))
}

func BenchmarkMQ_64KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*64))
}

func BenchmarkMQ_256KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*256))
}

func BenchmarkMQ_512KB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*512))
}

func BenchmarkMQ_1MB(b *testing.B) {
	benchmarkMQ(b, make([]byte, 1024*1024))
}

func BenchmarkMangos_32B(b *testing.B) {
	benchmarkMangos(b, make([]byte, 32))
}

func BenchmarkMangos_128B(b *testing.B) {
	benchmarkMangos(b, make([]byte, 128))
}

func BenchmarkMangos_1KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024))
}

func BenchmarkMangos_4KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*4))
}

func BenchmarkMangos_16KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*16))
}

func BenchmarkMangos_32KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*32))
}

func BenchmarkMangos_64KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*64))
}

func BenchmarkMangos_256KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*256))
}

func BenchmarkMangos_512KB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*512))
}

func BenchmarkMangos_1MB(b *testing.B) {
	benchmarkMangos(b, make([]byte, 1024*1024))
}

func benchmarkMQ(b *testing.B, val []byte) {
	var (
		s = New()
		c = New()

		ready = make(chan struct{}, 1)

		l   net.Listener
		wg  sync.WaitGroup
		err error
	)

	wg.Add(b.N)

	go func() {
		var (
			nc  net.Conn
			err error
		)

		if l, err = net.Listen("tcp", ":1337"); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}

		if nc, err = l.Accept(); err != nil {
			b.Fatal(err)
		}

		if err = s.Connect(nc); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}
	}()

	<-ready

	go func() {
		var (
			nc  net.Conn
			err error
		)

		if nc, err = net.Dial("tcp", ":1337"); err != nil {
			b.Fatal(err)
		}

		if err = c.Connect(nc); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}

		var n int
		for {
			if c.Get(func(b []byte) {
				testSetVal = b
				n++
				wg.Done()
			}) != nil {
				return
			}
		}
	}()

	<-ready
	<-ready
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err = s.Put(val); err != nil {
			b.Fatal(err)
		}
	}

	wg.Wait()
	b.ReportAllocs()
	s.Close()
	c.Close()
	l.Close()
}

func benchmarkMangos(b *testing.B, val []byte) {
	var (
		s  mangos.Socket
		c  mangos.Socket
		wg sync.WaitGroup
	)

	ready := make(chan struct{}, 1)

	go func() {
		var err error
		if s, err = mpair.NewSocket(); err != nil {
			b.Fatal(err)
		}

		s.AddTransport(mtcp.NewTransport())

		ready <- struct{}{}
		if err = s.Listen("tcp://127.0.0.1:1337"); err != nil {
			b.Fatal(err)
		}
	}()

	<-ready

	go func() {
		var err error
		if c, err = mpair.NewSocket(); err != nil {
			b.Fatal(err)
		}

		c.AddTransport(mtcp.NewTransport())
		if err = c.Dial("tcp://127.0.0.1:1337"); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}

		for {
			if testSetVal, err = c.Recv(); err != nil {
				break
			}

			wg.Done()
		}
	}()

	<-ready
	wg.Add(b.N)

	for i := 0; i < b.N; i++ {
		s.Send(val)
	}

	wg.Wait()
	b.ReportAllocs()
	s.Close()
	c.Close()
}
