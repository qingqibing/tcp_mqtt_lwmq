package dispatcher

import (
	"container/list"
	"lwmq/iface"
	"lwmq/manager"
	"lwmq/mlog"
	"sync"
	"time"
)

// SubTopic subscibe topic
type SubTopic struct {
	Topic string
	Qos   byte
}

// PubTopic publish data to topic
type PubTopic struct {
	Topic   string
	Qos     byte
	Pid     uint32
	Payload []byte
}

// MQTTClient client struct
type MQTTClient struct {
	ConnClient   iface.Iclient
	Status       uint32
	ProtocolName string
	ConnectFlag  byte
	KeepAlive    uint32
	LastTime     int64
	CreateTime   string
	SubList      *list.List
	lock         *sync.Mutex
}

// Refresh refresh last time
func (s *MQTTClient) Refresh() {
	if (s != nil) && (s.Status == Connected) && (s.KeepAlive > 0) {
		s.LastTime = time.Now().Unix()
	}
}

// CheckTmo check timeout
func (s *MQTTClient) CheckTmo() bool {
	// MQTT-3.1.1
	now := time.Now().Unix()

	if (now - s.LastTime) > int64(s.KeepAlive+s.KeepAlive/2) {
		return true
	}

	return false
}

// AddSubscribe add subscribe topic to client
func (s *MQTTClient) AddSubscribe(sub *SubTopic) uint32 {
	s.lock.Lock()
	defer s.lock.Unlock()

	for j := s.SubList.Front(); j != nil; j = j.Next() {
		subscribe := j.Value.(*SubTopic)
		if subscribe.Topic == sub.Topic {
			s.SubList.Remove(j)
			break
		}
	}
	s.SubList.PushBack(sub)

	return Success
}

// DelSubscribe add subscribe topic to client
func (s *MQTTClient) DelSubscribe(topic string) uint32 {
	s.lock.Lock()
	defer s.lock.Unlock()

	for j := s.SubList.Front(); j != nil; j = j.Next() {
		subscribe := j.Value.(*SubTopic)
		if subscribe.Topic == topic {
			s.SubList.Remove(j)
			break
		}
	}

	return Success
}

// HasSubscribe check if subscribe topic
func (s *MQTTClient) HasSubscribe(topic string, qos byte) uint32 {
	if s.Status != Connected {
		return ConnErr
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// Search subscribe list to write data
	for j := s.SubList.Front(); j != nil; j = j.Next() {
		subscribe := j.Value.(*SubTopic)
		if (subscribe.Topic == topic) && (subscribe.Qos >= qos) {
			return Success
		}
	}

	return Fail
}

// PublishData publish data to subscribe topic
func (s *MQTTClient) PublishData(pub *PubTopic, buff []byte, size uint32) uint32 {
	if s.Status != Connected {
		return ConnErr
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// Search subscribe list to write data
	for j := s.SubList.Front(); j != nil; j = j.Next() {
		subscribe := j.Value.(*SubTopic)
		// TODO: support wildcards
		if subscribe.Topic == pub.Topic {
			// set qos
			buff[0] &= 0xf9
			if pub.Qos > subscribe.Qos {
				buff[0] |= (subscribe.Qos << 1)
			} else {
				buff[0] |= (pub.Qos << 1)
			}

			if s.ConnClient != nil {
				s.ConnClient.Send(buff, size)
			}
			break
		}
	}

	return Success
}

// WaitAck wait publish ack
type WaitAck struct {
	Pub      *PubTopic
	Dup      uint32
	WaitAck  *list.List
	waitRec  *list.List
	WaitComp *list.List
}

// MQTTserver server struct
type MQTTserver struct {
	TotalClients  uint32
	OnlineClients uint32
	Lock          *sync.Mutex
	Mclients      map[string]*MQTTClient
	ConnMap       map[uint32]string
	Publist       *list.List
	PubEn         chan byte
	wakelock      *sync.Mutex
	cond          *sync.Cond
}

// Mserver global MQTT server
var Mserver *MQTTserver

// AddMQTTClient add client to server
func (s *MQTTserver) AddMQTTClient(clientID string, mc *MQTTClient) uint32 {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	mqttclient, exist := s.Mclients[clientID]
	if exist {
		if mqttclient.Status == Connected {
			mqttclient.Status = Disconnected
			return ConnExist
		}

		if (mc.ConnectFlag & 0x02) != 0 {
			// Clean session
			s.Mclients[clientID] = mc
			s.ConnMap[mc.ConnClient.GetCid()] = clientID
			s.OnlineClients++
			return Success
		}

		mqttclient.Status = Connected
		s.OnlineClients++
		return ClientExist
	}

	s.Mclients[clientID] = mc
	s.ConnMap[mc.ConnClient.GetCid()] = clientID
	s.TotalClients++
	s.OnlineClients++

	mlog.Info("Add new client:", clientID)
	return Success

}

// GetMQTTClientIDbyCid search client from server
func (s *MQTTserver) GetMQTTClientIDbyCid(cid uint32) string {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	clientID, exist := s.ConnMap[cid]
	if exist {
		return clientID
	}

	return ""
}

// GetMQTTClientID search client from server
func (s *MQTTserver) GetMQTTClientID(cl iface.Iclient) string {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	cid := cl.GetCid()
	clientID, exist := s.ConnMap[cid]
	if exist {
		return clientID
	}

	return ""
}

// GetMQTTClient search client from server
func (s *MQTTserver) GetMQTTClient(cl iface.Iclient) *MQTTClient {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	cid := cl.GetCid()
	clientID, exist := s.ConnMap[cid]
	if exist {
		return s.Mclients[clientID]
	}

	return nil
}

// DelMQTTClient delete client from server
func (s *MQTTserver) DelMQTTClient(clientID string) uint32 {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	mlog.Info("Del MQTT client:", clientID)

	_, exist := s.Mclients[clientID]
	if !exist {
		return Success
	}

	cid := s.Mclients[clientID].ConnClient.GetCid()

	delete(s.Mclients, clientID)
	delete(s.ConnMap, cid)

	s.TotalClients--
	s.OnlineClients--

	return Success
}

// OfflineMQTTClient offline client from server
func (s *MQTTserver) OfflineMQTTClient(clientID string) uint32 {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	mlog.Warning("Offline MQTT client:", clientID)

	mqttclient, exist := s.Mclients[clientID]
	if !exist {
		return Success
	}

	mqttclient.Status = Disconnected
	s.OnlineClients--

	return Success
}

func (s *MQTTserver) checkClient() {
	for {
		for k, v := range s.Mclients {
			client := v

			if (client.Status == Connected) &&
				(client.CheckTmo() || client.ConnClient.GetStatus() == manager.Closed) {
				client.Status = DISCONNECT
				client.ConnClient.Stop()

				// delete client
				s.DelMQTTClient(k)
			}
		}

		time.Sleep(time.Second)
	}
}

func (s *MQTTserver) pubWork() {
	for {
		for {
			// Get one from publish list
			s.Lock.Lock()
			if s.Publist.Len() == 0 {
				s.Lock.Unlock()
				break
			}

			for i := s.Publist.Front(); i != nil; i = i.Next() {
				publishData := i
				waitack := publishData.Value.(*WaitAck)

				if waitack.Dup > 2 {
					// Construct publish data
					publishTopic := waitack.Pub
					if publishTopic.Qos == 1 {
						if (waitack.WaitAck.Len() == 0) || (waitack.Dup >= 10) {
							s.Publist.Remove(publishData)
							break
						}

						// Republish, set DUP
						publishTopic.Payload[0] |= 0x08
						for j := waitack.WaitAck.Front(); j != nil; j = j.Next() {
							v := j.Value.(*MQTTClient)
							v.PublishData(publishTopic, publishTopic.Payload, uint32(len(publishTopic.Payload)))
						}
					} else {
						// QOS 2, TODO
					}
				}

				waitack.Dup++
			}
			s.Lock.Unlock()

			time.Sleep(time.Second)
		}

		// If no request, go to sleep
		s.cond.L.Lock()
		s.cond.Wait()

		mlog.Info("pubWork wakeup")
		s.cond.L.Unlock()
	}
}

// PubAck publish data ACK
func (s *MQTTserver) PubAck(cl iface.Iclient, pid uint32) uint32 {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	for i := s.Publist.Front(); i != nil; i = i.Next() {
		waitack := i.Value.(*WaitAck)

		if waitack.Pub.Pid == pid {
			for j := waitack.WaitAck.Front(); j != nil; j = j.Next() {
				v := j.Value.(*MQTTClient)
				if v.ConnClient.GetCid() == cl.GetCid() {
					s.Publist.Remove(i)
					break
				}
			}
			break
		}
	}

	return Success
}

// PubToClient publish data to client
func (s *MQTTserver) PubToClient(pub *PubTopic) uint32 {
	if pub.Qos == 1 {
		doPublish := false
		// Qos 1 need to wait PUBACK, if not, republish the topic
		s.Lock.Lock()
		waitack := &WaitAck{
			Pub:     pub,
			Dup:     0,
			WaitAck: list.New(),
		}
		// Put waiting clients to list
		for _, v := range s.Mclients {
			if v.HasSubscribe(pub.Topic, pub.Qos) == Success {
				waitack.WaitAck.PushBack(v)
				if !doPublish {
					doPublish = true
				}
			}
		}
		if doPublish {
			s.Publist.PushBack(waitack)
		}
		s.Lock.Unlock()

		for _, v := range s.Mclients {
			v.PublishData(pub, pub.Payload, uint32(len(pub.Payload)))
		}

		if doPublish {
			waitack.Dup = 1

			s.wakePubWork()
		}
	} else if pub.Qos == 2 {
		// TODO
	} else {
		// Qos 0
		for _, v := range s.Mclients {
			v.PublishData(pub, pub.Payload, uint32(len(pub.Payload)))
		}
	}

	return Success
}

func (s *MQTTserver) wakePubWork() {
	s.cond.Signal()
}

// OnAddClient client add callback
func OnAddClient(cl iface.Iclient) {
	cl.SetHandler(checkMQTTdata, dispathMQTTdata)
}

type dispatchHandler func(cl iface.Iclient, buff []byte, size uint32) uint32

// Handlers to handle every command
var dispatchHandlers = map[byte]dispatchHandler{
	CONNECT:     HandleCONNECT,
	DISCONNECT:  HandleDISCONNECT,
	SUBSCRIBE:   HandleSUBSCRIBE,
	UNSUBSCRIBE: HandleUNSUBSCRIBE,
	PINGREQ:     HandlePINGREQ,
	PUBLISH:     HandlePUBLISH,
	PUBACK:      HandlePUBACK,
}

// MIN return min(a, b)
func MIN(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// DecodeLen decode left length
func DecodeLen(buff []byte, size uint32) (uint32, uint32, uint32) {
	// Decode Header, MQTT-3.1.1
	var multiplier uint32 = 1
	var leftLen uint32 = 0
	var leftLenSize uint32

	if size == 1 {
		return 0, 0, MoreData
	} else if size == 2 {
		if buff[1] == 0 {
			return 1, 0, Success
		}

		return 0, 0, MoreData
	}

	var i uint32
	var getHead = false
	var maxHead = MIN(size, 5)

	for i = 1; i < maxHead; i++ {
		encodedByte := buff[i]

		leftLen += uint32(encodedByte&0x7f) * multiplier
		multiplier *= 128
		if multiplier > 128*128*128 {
			return 0, 0, ArgumentError
		}

		if encodedByte == 0 {
			return 0, 0, LenError
		}

		if (encodedByte & 0x80) == 0 {
			getHead = true
			leftLenSize = uint32(i)
			break
		}
	}

	if !getHead {
		return 0, 0, MoreData
	}

	return leftLenSize, leftLen, Success
}

// EncodeLen Encode left length
func EncodeLen(size uint32) []byte {
	encodeLen := []byte{}
	var x = size

	for i := 0; i < 4; i++ {
		encodeLen = append(encodeLen, byte(x%128))
		x = x / 128

		if x > 0 {
			encodeLen[i] |= 0x80
		} else {
			break
		}
	}

	return encodeLen
}

// Check if MQTT data is complete
func checkMQTTdata(cl iface.Iclient, size uint32) uint32 {
	if size == 0 {
		return CmdNotFound
	}

	var buff []byte = make([]byte, 5)
	var i uint32

	for i = 0; i < 5; i++ {
		buff[i] = cl.PickBuff(i)
	}

	mlog.Info("Check MQTT data:", buff, " size:", size)

	command := buff[0] >> 4
	_, exist := dispatchHandlers[command]
	if !exist {
		cl.SetStatus(manager.Err)
		cl.Stop()

		return CmdNotFound
	}

	// If head is ok, no need to decode head again
	if cl.GetStatus() != manager.GetHead {
		leftLenSize, leftLen, sts := DecodeLen(buff, size)
		if sts == MoreData {
			return sts
		} else if sts != Success {
			cl.SetStatus(manager.Err)
			return sts
		}

		cl.SetStatus(manager.GetHead)
		cl.SetWaitDataSize(1 + leftLenSize + leftLen)

		mlog.Debug("Left len:", leftLen)
	}

	if size >= cl.GetWaitDataSize() {
		cl.SetStatus(manager.WaitDataDone)
	}

	return Success
}

// Dispatch data to every command handler
func dispathMQTTdata(cl iface.Iclient, cid uint32, buff []byte, size uint32) uint32 {
	mlog.Info("Dispatch MQTT data")

	var status uint32
	command := buff[0] >> 4
	handler, exist := dispatchHandlers[command]

	if cl != nil {
		if !exist {
			cl.SetStatus(manager.Err)
			return CmdNotFound
		}

		// Start handle data
		cl.SetStatus(manager.ProcessData)
		status = handler(cl, buff, size)
		if Success == status {
			cl.SetStatus(manager.Idle)
		} else {
			cl.SetStatus(manager.Err)
		}

		// Refresh time
		mclient := Mserver.GetMQTTClient(cl)
		if mclient != nil {
			mclient.Refresh()
		}
	} else {
		mlog.Error("Client lost, but still do command.")
		if !exist {
			return CmdNotFound
		}
		handler(cl, buff, size)

		clientID := Mserver.GetMQTTClientIDbyCid(cid)
		if len(clientID) > 0 {
			Mserver.OfflineMQTTClient(clientID)
		}
	}

	return status
}

func init() {
	Mserver = &MQTTserver{
		TotalClients:  0,
		OnlineClients: 0,
		Lock:          new(sync.Mutex),
		Mclients:      make(map[string]*MQTTClient),
		ConnMap:       make(map[uint32]string),
		Publist:       list.New(),
		PubEn:         make(chan byte),
		wakelock:      new(sync.Mutex),
	}
	Mserver.cond = sync.NewCond(Mserver.wakelock)

	go Mserver.pubWork()
	go Mserver.checkClient()
}
