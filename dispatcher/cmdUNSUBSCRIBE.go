package dispatcher

import (
	"lwmq/iface"
	"lwmq/mlog"
)

func respUNSUBACK(cl iface.Iclient, pid uint32) uint32 {
	var resp = []byte{}
	var pidEncode = []byte{byte(pid >> 8), byte(pid & 0xff)}

	resp = append(resp, UNSUBACK<<4)

	respLen := 2
	respLenEncode := EncodeLen(uint32(respLen))

	resp = append(resp, respLenEncode...)
	resp = append(resp, pidEncode...)

	cl.Send(resp, uint32(len(resp)))

	return Success
}

// HandleUNSUBSCRIBE handle UNSUBSCRIBE command
func HandleUNSUBSCRIBE(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("HandleUNSUBSCRIBE")

	if cl == nil {
		return ConnErr
	}

	flag := buff[0] & 0x0f
	if flag != 0x02 {
		return ArgumentError
	}

	// Check header
	leftLenSize, leftLen, sts := DecodeLen(buff, size)
	if sts != Success {
		return sts
	}

	varStart := 1 + leftLenSize
	pid := uint32(buff[varStart])<<8 + uint32(buff[varStart+1])
	mlog.Debug("Pid:", pid)

	// Parse unsubscribe
	var i uint32
	mclient := Mserver.GetMQTTClient(cl)

	for i = (varStart + 2); i < (1 + leftLenSize + leftLen); {
		topicLen := uint32(buff[i])<<8 + uint32(buff[i+1])
		topicFilter := string(buff[i+2 : i+2+topicLen])
		mclient.DelSubscribe(topicFilter)

		i += 2 + topicLen

		mlog.Debug("Len:", topicLen)
		mlog.Debug("Topic:", topicFilter)
	}

	sts = respUNSUBACK(cl, pid)
	if sts != Success {
		return sts
	}

	return Success
}
