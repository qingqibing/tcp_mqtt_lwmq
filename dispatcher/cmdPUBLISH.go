package dispatcher

import (
	"lwmq/iface"
	"lwmq/mlog"
)

func respPUBACK(cl iface.Iclient, pid uint32) uint32 {
	var resp = []byte{}
	var pidEncode = []byte{byte(pid >> 8), byte(pid & 0xff)}

	resp = append(resp, PUBACK<<4)

	respLen := uint32(2)
	respLenEncode := EncodeLen(respLen)

	resp = append(resp, respLenEncode...)
	resp = append(resp, pidEncode...)

	cl.Send(resp, uint32(len(resp)))

	return Success
}

// HandlePUBLISH handle PUBLISH command
func HandlePUBLISH(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("HandlePUBLISH")

	flag := buff[0] & 0x0f
	//retain := flag & 0x0f
	Qos := (flag & 0x06) >> 1
	//dup := (flag & 0x08) >> 3

	if Qos == 0x03 {
		if cl == nil {
			return ConnErr
		}

		cl.Stop()
		return ArgumentError
	}

	// Check header
	leftLenSize, _, sts := DecodeLen(buff, size)
	if sts != Success {
		return sts
	}

	// Parse header
	varStart := 1 + leftLenSize
	topicLen := uint32(buff[varStart])<<8 + uint32(buff[varStart+1])
	topic := string(buff[varStart+2 : varStart+2+topicLen])

	mlog.Debug("Publish to:", topic, " Qos:", Qos)

	// Parse payload
	publish := &PubTopic{
		Topic: topic,
		Qos:   Qos,
		Pid:   0,
	}

	if Qos > 0 {
		pid := uint32(buff[varStart+2+topicLen])<<8 + uint32(buff[varStart+2+topicLen+1])
		publish.Pid = pid
	}
	publish.Payload = buff

	Mserver.PubToClient(publish)

	if Qos == 1 {
		if cl == nil {
			return ConnErr
		}

		sts = respPUBACK(cl, publish.Pid)
		if sts != Success {
			return sts
		}
	} else if Qos == 2 {
		// TODO
	}

	return Success
}

// HandlePUBACK handle PUBACK command
func HandlePUBACK(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("HandlePUBACK")

	if cl == nil {
		return ConnErr
	}

	flag := buff[0] & 0x0f
	if flag != 0x00 {
		return ArgumentError
	}

	// Check header
	if buff[1] != 2 {
		return LenError
	}

	pid := uint32(buff[2])<<8 + uint32(buff[3])
	Mserver.PubAck(cl, pid)

	return Success
}
