package iface

// Irequest request interface
type Irequest interface {
	GetCid() uint32
	GetConn() Iconn
	GetData() []byte
	GetSize() uint32
}
