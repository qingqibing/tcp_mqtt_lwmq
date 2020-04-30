package iface

// CheckHandler handler for client
type CheckHandler func(Iclient, uint32) uint32

// DispatchHandler handler for client
type DispatchHandler func(Iclient, uint32, []byte, uint32) uint32

// Iclient connection interface
type Iclient interface {
	GetCid() uint32
	GetStatus() byte
	SetStatus(byte)
	GetConn() Iconn

	PickBuff(uint32) byte

	GetWaitDataSize() uint32
	SetWaitDataSize(uint32)

	Send([]byte, uint32)

	SetHandler(CheckHandler, DispatchHandler)
	DispathData(Iclient, uint32, []byte, uint32) uint32
	ClearBuff()
	Start()
	Stop()
	Dequeue()

	IsInWork() bool
	SetInWork(bool)
}
