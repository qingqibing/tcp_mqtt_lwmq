package service

import (
	"lwmq/iface"
	"lwmq/mlog"
	"net"
	"sync"
)

// Connection as a connection
type Connection struct {
	Server iface.Iservicer
	Conn   *net.TCPConn
	cid    uint32
	lock   *sync.Mutex
}

// Read read data from connection
func (c *Connection) Read(buff []byte, size uint32) (uint32, error) {
	len, err := c.Conn.Read(buff)
	if err != nil {
		mlog.Error("Read error:", err)
		return 0, err
	}

	return uint32(len), err
}

// Write write data to connection
func (c *Connection) Write(buff []byte, size uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	mlog.Debug("Write data")

	c.Conn.Write(buff)
}

// Close close connection
func (c *Connection) Close() {
	mlog.Debug("Connection closed")
	c.Conn.Close()
}

// FreeCid free cid
func (c *Connection) FreeCid() {
	c.Server.FreeCid(c.cid)
}

// NewConn new connection
func NewConn(server iface.Iservicer, conn *net.TCPConn, cid uint32) iface.Iconn {
	return &Connection{
		Server: server,
		Conn:   conn,
		cid:    cid,
		lock:   new(sync.Mutex),
	}
}
