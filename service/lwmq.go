package service

import (
	"fmt"
	"lwmq/iface"
	"lwmq/mlog"
	"net"
	"sync"
)

var lwmqStart = `
---------------------------------------
.____   __      __  _____   ________   
|    | /  \    /  \/     \  \_____  \  
|    | \   \/\/   /  \ /  \  /  / \  \ 
|    |__\        /    Y    \/   \_/.  \
|_______ \__/\  /\____|__  /\_____\ \_/
        \/    \/         \/        \__>
---------------------------------------
`

func init() {
	fmt.Println(lwmqStart)
}

// Lwmq service
type Lwmq struct {
	Name    string
	Type    string
	IP      string
	Port    int
	onConn  func(cid uint32, conn iface.Iconn)
	lock    *sync.Mutex
	cidPool []byte
}

// Start start service
func (s *Lwmq) Start() {
	mlog.Debug("Start LWMQ service!")

	go func() {
		switch s.Type {
		case "tcp", "tcp4":
			addr, err := net.ResolveTCPAddr(s.Type, fmt.Sprintf("%s:%d", s.IP, s.Port))
			if err != nil {
				mlog.Error("Resolve address error:", err)
				return
			}

			listener, err := net.ListenTCP(s.Type, addr)
			if err != nil {
				mlog.Error("Listen address error:", err)
				return
			}

			mlog.Debug("Listen on:", addr.String())

			// Loop wait for connection
			for {
				conn, err := listener.AcceptTCP()
				if err != nil {
					mlog.Error("Accept error:", err)
					continue
				}

				// Alloc one Cid, every connection has different Cid
				cid, sts := s.AllocCid()
				if sts != 0 {
					conn.Close()
					continue
				}

				mlog.Debug("Get connect:", conn.RemoteAddr().String())

				// Get connection and run callback
				connection := NewConn(s, conn, cid)
				s.onConn(cid, connection)
			}
		case "tcp6":

		case "udp":

		default:
			mlog.Error("Address type error!")
			return
		}
	}()

	select {}
}

// SetOnConnect on connect callback
func (s *Lwmq) SetOnConnect(onConn func(cid uint32, conn iface.Iconn)) {
	s.onConn = onConn
}

// AllocCid alloc cid
func (s *Lwmq) AllocCid() (uint32, byte) {
	var i uint32 = 0
	s.lock.Lock()
	defer s.lock.Unlock()

	for i = 0; i < 1024; i++ {
		if s.cidPool[i] != 0x01 {
			break
		}
	}

	if i >= 1024 {
		return 0, 0x10
	}

	s.cidPool[i] = 0x01
	return i, 0
}

// FreeCid free cid
func (s *Lwmq) FreeCid(cid uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cidPool[cid] = 0
}

// Stop stop service
func (s *Lwmq) Stop() {
	mlog.Error("Stop LWMQ service!")
}

// NewLwmq create lwmq service
func NewLwmq() iface.Iservicer {
	return &Lwmq{
		Name:    "LWMQ",
		Type:    "tcp4",
		IP:      "0.0.0.0",
		Port:    1883,
		lock:    new(sync.Mutex),
		cidPool: make([]byte, 1024),
	}
}
