package dispatcher

import (
	"lwmq/iface"
	"lwmq/mlog"
)

func respSUBACK(cl iface.Iclient, pid uint32, respSub []byte, respCnt uint32) uint32 {
	var resp = []byte{}
	var pidEncode = []byte{byte(pid >> 8), byte(pid & 0xff)}

	resp = append(resp, SUBACK<<4)

	respLen := 2 + respCnt
	respLenEncode := EncodeLen(respLen)

	resp = append(resp, respLenEncode...)
	resp = append(resp, pidEncode...)
	resp = append(resp, respSub...)

	cl.Send(resp, uint32(len(resp)))

	return Success
}

// HandleSUBSCRIBE handle SUBSCRIBE command
func HandleSUBSCRIBE(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("HandleSUBSCRIBE")

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

	// Parse subscribe
	var i uint32
	var subCnt uint32 = 0
	var subResp = []byte{}
	mclient := Mserver.GetMQTTClient(cl)

	for i = (varStart + 2); i < (1 + leftLenSize + leftLen); {
		topicLen := uint32(buff[i])<<8 + uint32(buff[i+1])
		topicFilter := string(buff[i+2 : i+2+topicLen])
		topicQos := buff[i+2+topicLen]

		if (topicQos & 0xfc) != 0 {
			return ArgumentError
		}

		subscribe := &SubTopic{
			Topic: topicFilter,
			Qos:   topicQos,
		}

		mclient.AddSubscribe(subscribe)
		subResp = append(subResp, topicQos)
		subCnt++

		i += 2 + topicLen + 1

		mlog.Debug("Len:", topicLen)
		mlog.Debug("Topic:", topicFilter)
		mlog.Debug("Qos:", topicQos)
	}

	sts = respSUBACK(cl, pid, subResp, subCnt)
	if sts != Success {
		return sts
	}

	return Success
}
