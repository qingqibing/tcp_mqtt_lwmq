package dispatcher

import (
	"lwmq/iface"
	"lwmq/mlog"
)

func respPINGRESP(cl iface.Iclient) uint32 {
	var resp = []byte{PINGRESP << 4, 0x00}

	cl.Send(resp, uint32(len(resp)))

	return Success
}

// HandlePINGREQ handle PINGREQ command
func HandlePINGREQ(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("HandlePINGREQ")

	if cl == nil {
		return ConnErr
	}

	flag := buff[0] & 0x0f
	if flag != 0x00 {
		return ArgumentError
	}

	// Check header
	if buff[1] != 0 {
		return LenError
	}

	sts := respPINGRESP(cl)
	if sts != Success {
		return sts
	}

	return Success
}
