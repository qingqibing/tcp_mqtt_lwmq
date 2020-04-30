package manager

import (
	"lwmq/iface"
)

// Request request
type Request struct {
	cid  uint32
	conn iface.Iconn
	data []byte
	size uint32
}

// GetCid get cid
func (r *Request) GetCid() uint32 {
	return r.cid
}

// GetConn get connection
func (r *Request) GetConn() iface.Iconn {
	return r.conn
}

// GetData get data
func (r *Request) GetData() []byte {
	return r.data
}

// GetSize get data
func (r *Request) GetSize() uint32 {
	return r.size
}
