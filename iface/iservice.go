package iface

// Iservicer lwmq service interface
type Iservicer interface {
	Start()
	Stop()
	SetOnConnect(func(cid uint32, conn Iconn))
	AllocCid() (uint32, byte)
	FreeCid(uint32)
}
