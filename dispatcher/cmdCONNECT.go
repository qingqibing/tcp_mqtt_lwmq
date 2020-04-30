package dispatcher

import (
	"container/list"
	"lwmq/iface"
	"lwmq/mlog"
	"sync"
	"time"
)

func respCONNACK(cl iface.Iclient, resp1 byte, resp2 byte) uint32 {
	var resp = []byte{CONNACK << 4, 0x02, resp1, resp2}

	cl.Send(resp, 4)

	return Success
}

// HandleCONNECT handle CONNECT command
func HandleCONNECT(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("CONNECT")

	var resp1 byte = 0
	var resp2 byte = 0

	flag := buff[0] & 0x0f
	if flag != 0 {
		return ArgumentError
	}

	// Check header
	leftLenSize, leftLen, sts := DecodeLen(buff, size)
	if sts != Success {
		return sts
	}

	if leftLen < 10 {
		return LenError
	}

	varStart := 1 + leftLenSize
	protocolLen := uint32(buff[varStart])<<8 + uint32(buff[varStart+1])
	if protocolLen != 0x04 {
		return LenError
	}

	protocolName := string(buff[varStart+2 : varStart+2+protocolLen])
	if protocolName != "MQTT" {
		return ArgumentError
	}

	protocolLevel := buff[varStart+2+protocolLen]
	if protocolLevel != 0x04 {
		resp1 = 0x00
		resp2 = 0x01
		respCONNACK(cl, resp1, resp2)

		return ArgumentError
	}

	connectFlag := buff[varStart+2+protocolLen+1]
	keepAlive := uint32(buff[varStart+2+protocolLen+2])<<8 + uint32(buff[varStart+2+protocolLen+3])

	// Check payload
	payloadStart := varStart + 2 + protocolLen + 4
	clientIDLen := uint32(buff[payloadStart])<<8 + uint32(buff[payloadStart+1])
	// TODO: zero byte client ID
	clientID := string(buff[payloadStart+2 : payloadStart+2+clientIDLen])

	// TODO: Will topic, user name, password

	mlog.Debug("Protocol Name:", protocolName)
	mlog.Debug("Connect flag:", connectFlag)
	mlog.Debug("Keep alive:", keepAlive)
	mlog.Debug("Client ID:", clientID)

	// Add new client to server
	mclient := &MQTTClient{
		ConnClient:   cl,
		Status:       Connected,
		ProtocolName: protocolName,
		ConnectFlag:  connectFlag,
		KeepAlive:    keepAlive,
		SubList:      list.New(),
		lock:         new(sync.Mutex),
		LastTime:     0,
		CreateTime:   time.Now().Format(time.UnixDate),
	}

	if keepAlive > 0 {
		mclient.LastTime = time.Now().Unix()
	}

	sts = Mserver.AddMQTTClient(clientID, mclient)
	if sts == ClientExist {
		if (connectFlag & 0x02) == 0 {
			resp1 |= 0x01
		}
	} else if sts != Success {
		resp1 = 0x00
		resp2 = 0x02
		respCONNACK(cl, resp1, resp2)

		return sts
	}

	// Send Response
	resp1 = 0
	resp2 = 0
	sts = respCONNACK(cl, resp1, resp2)
	if sts != Success {
		return sts
	}

	return Success
}

// HandleDISCONNECT handle DISCONNECT command
func HandleDISCONNECT(cl iface.Iclient, buff []byte, size uint32) uint32 {
	mlog.Debug("DISCONNECT")

	if cl == nil {
		return Success
	}

	flag := buff[0] & 0x0f
	if flag != 0 {
		return ArgumentError
	}

	if buff[1] != 0 {
		return LenError
	}

	clientID := Mserver.GetMQTTClientID(cl)
	if len(clientID) > 0 {
		Mserver.DelMQTTClient(clientID)
	}

	cl.Stop()

	return Success
}
