package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/missionMeteora/mq.v2/conn"

	"github.com/go-mangos/mangos"
	mpair "github.com/go-mangos/mangos/protocol/pair"
	mtcp "github.com/go-mangos/mangos/transport/tcp"

	"github.com/pkg/profile"
)

var setVal []byte

const reportFmt = "%s sent %d messages with a size of %d bytes in an average of %v\n"

func main() {
	p := profile.Start(profile.CPUProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	//	reportMQ(32, 1000000)
	//	reportMQ(128, 1000000)
	//	reportMQ(1024, 1000000)
	reportMQ(1024*4, 1000000)
	p.Stop()
	//	reportMQ(1024*32, 1000000)

	//	reportMangos(32, 1000000)
	//	reportMangos(128, 1000000)
	//	reportMangos(1024, 1000000)
	//	reportMangos(1024*4, 1000000)
	//	reportMangos(1024*32, 1000000)
}

func reportMQ(sz, n int) {
	var d time.Duration
	val := make([]byte, sz)
	d += testMQ(val, n)
	d += testMQ(val, n)
	d += testMQ(val, n)
	fmt.Printf(reportFmt, "mq", n, len(val), d/3)
}

func reportMangos(sz, n int) {
	var d time.Duration
	val := make([]byte, sz)
	d += testMangos(val, n)
	d += testMangos(val, n)
	d += testMangos(val, n)
	fmt.Printf(reportFmt, "mangos", n, len(val), d/3)
}

func testMQ(val []byte, n int) time.Duration {
	var (
		l   net.Listener
		wg  sync.WaitGroup
		err error
	)

	s := conn.New()
	c := conn.New()
	ready := make(chan struct{}, 1)

	wg.Add(n)

	go func() {
		var (
			nc  net.Conn
			err error
		)

		if l, err = net.Listen("tcp", ":1337"); err != nil {
			panic(err)
		}

		ready <- struct{}{}

		if nc, err = l.Accept(); err != nil {
			panic(err)
		}

		if err = s.Connect(nc); err != nil {
			panic(err)
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
			panic(err)
		}

		if err = c.Connect(nc); err != nil {
			panic(err)
		}

		ready <- struct{}{}

		var n int
		for {
			if c.Get(func(b []byte) {
				setVal = b
				n++
				wg.Done()
			}) != nil {
				return
			}
		}
	}()

	<-ready
	<-ready

	start := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		if err = s.Put(val); err != nil {
			panic(err)
		}
	}

	wg.Wait()
	end := time.Now().UnixNano()

	s.Close()
	c.Close()
	l.Close()
	return time.Duration(end - start)
}

func testMangos(val []byte, n int) time.Duration {
	var (
		s  mangos.Socket
		c  mangos.Socket
		wg sync.WaitGroup
	)

	ready := make(chan struct{}, 1)
	wg.Add(n)

	go func() {
		var err error
		if s, err = mpair.NewSocket(); err != nil {
			panic(err)
		}

		s.AddTransport(mtcp.NewTransport())

		ready <- struct{}{}
		if err = s.Listen("tcp://127.0.0.1:1337"); err != nil {
			panic(err)
		}
	}()

	<-ready

	go func() {
		var err error
		if c, err = mpair.NewSocket(); err != nil {
			panic(err)
		}

		c.AddTransport(mtcp.NewTransport())
		if err = c.Dial("tcp://127.0.0.1:1337"); err != nil {
			panic(err)
		}

		ready <- struct{}{}

		for {
			if setVal, err = c.Recv(); err != nil {
				break
			}

			wg.Done()
		}
	}()

	<-ready
	start := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		s.Send(val)
	}

	wg.Wait()
	end := time.Now().UnixNano()

	s.Close()
	c.Close()
	return time.Duration(end - start)
}
