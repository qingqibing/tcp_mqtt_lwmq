package iface

// Iconn connection interface
type Iconn interface {
	Read(buff []byte, size uint32) (uint32, error)
	Write(buff []byte, size uint32)
	Close()
	FreeCid()
}
