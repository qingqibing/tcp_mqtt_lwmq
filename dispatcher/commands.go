package dispatcher

// Response status
const (
	Success = iota
	Fail
	CmdNotFound
	ArgumentError
	LenError
	DataError
	MoreData
	ConnErr
	ConnExist
	ClientExist
)

// Status
const (
	Idle = iota
	Connected
	Disconnected
)

// Command
const (
	Reserved = iota
	CONNECT
	CONNACK
	PUBLISH
	PUBACK
	PUBREC
	PUBREL
	PUBCOMP
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ
	PINGRESP
	DISCONNECT
	Reserved2
)
