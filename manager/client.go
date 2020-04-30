package manager

import (
	"lwmq/iface"
	"lwmq/mlog"
)

// Idle status
const (
	Idle = iota
	GetHead
	WaitDataDone
	ProcessData
	Err
	Closed
	Removed
)

// RingBuff ring buff
type RingBuff struct {
	Buff     []byte
	Size     uint32
	ReadIdx  uint32
	WriteIdx uint32
}

// GetLen get data length in ring buff
func (ring *RingBuff) GetLen() uint32 {
	if ring.WriteIdx >= ring.ReadIdx {
		return ring.WriteIdx - ring.ReadIdx
	}

	return (ring.Size - ring.ReadIdx + ring.WriteIdx)
}

// PutData put data to ring buffer
func (ring *RingBuff) PutData(buff []byte) {
	var i uint32
	length := uint32(len(buff))

	if ring.Size-ring.WriteIdx > length {
		for i = 0; i < length; i++ {
			ring.Buff[ring.WriteIdx+i] = buff[i]
		}

		ring.WriteIdx += length
	} else {
		pre := ring.Size - ring.WriteIdx

		for i = 0; i < pre; i++ {
			ring.Buff[ring.WriteIdx+i] = buff[i]
		}

		for i = 0; i < (length - pre); i++ {
			ring.Buff[i] = buff[pre+i]
		}

		ring.WriteIdx = length - pre
	}

	if ring.WriteIdx >= ring.Size {
		ring.WriteIdx = 0
	}
}

// Clear Clear buffer
func (ring *RingBuff) Clear() {
	ring.ReadIdx = ring.WriteIdx
}

// Pickdata pick data, not move readIdx
func (ring *RingBuff) Pickdata(offset uint32) byte {
	if ring.Size-ring.ReadIdx > offset {
		return ring.Buff[ring.ReadIdx+offset]
	}

	pre := ring.Size - ring.ReadIdx
	return ring.Buff[offset-pre]
}

// GetData get data from ring buffer, if round over return new slice
func (ring *RingBuff) GetData(length uint32) (uint32, []byte, bool) {
	var rData []byte
	var i uint32
	var newSlice bool

	if ring.GetLen() < length {
		return 0, nil, false
	}

	if ring.Size-ring.ReadIdx >= length {
		rData = ring.Buff[ring.ReadIdx : ring.ReadIdx+length]
		newSlice = false
		ring.ReadIdx += length
	} else {
		rData = make([]byte, length)

		pre := ring.Size - ring.ReadIdx
		for i = 0; i < pre; i++ {
			rData[i] = ring.Buff[ring.ReadIdx+i]
		}

		for i = 0; i < (length - pre); i++ {
			rData[pre+i] = ring.Buff[i]
		}

		newSlice = true
		ring.ReadIdx = length - pre
	}

	if ring.ReadIdx >= ring.Size {
		ring.ReadIdx = 0
	}

	return length, rData, newSlice
}

// Client a connected client
type Client struct {
	Cid          uint32
	Conn         iface.Iconn
	Status       byte
	buffPool     [][]byte
	buffIdx      uint32
	ringbuff     RingBuff
	WaitDataSize uint32
	checkData    func(iface.Iclient, uint32) uint32
	dispathData  func(iface.Iclient, uint32, []byte, uint32) uint32
	requestCnt   uint32
	workIndo     bool
}

// ClientOnConn client on connect callback
func ClientOnConn(cid uint32, conn iface.Iconn) {
	client := NewClient(cid, conn)

	go client.Start()
}

// NewClient add an new clent
func NewClient(cid uint32, conn iface.Iconn) iface.Iclient {
	return &Client{
		Cid:          cid,
		Conn:         conn,
		Status:       Idle,
		WaitDataSize: 0,
		requestCnt:   0,
		workIndo:     false,
	}
}

// GetCid get cid
func (c *Client) GetCid() uint32 {
	return c.Cid
}

// GetStatus get Status
func (c *Client) GetStatus() byte {
	return c.Status
}

// SetStatus set Status
func (c *Client) SetStatus(sts byte) {
	if (c.Status != Closed) && (c.Status != Removed) {
		// Not set if closed
		c.Status = sts
	}
}

// GetConn get connection
func (c *Client) GetConn() iface.Iconn {
	return c.Conn
}

// PickBuff pick byte from buffer
func (c *Client) PickBuff(offset uint32) byte {
	return c.ringbuff.Pickdata(offset)
}

// GetWaitDataSize get WaitDataSize
func (c *Client) GetWaitDataSize() uint32 {
	return c.WaitDataSize
}

// SetWaitDataSize set WaitDataSize
func (c *Client) SetWaitDataSize(wds uint32) {
	c.WaitDataSize = wds
}

// SetHandler set client handler
func (c *Client) SetHandler(checkData iface.CheckHandler, dispatch iface.DispatchHandler) {
	c.checkData = checkData
	c.dispathData = dispatch
}

// DispathData dispatch client handler
func (c *Client) DispathData(cl iface.Iclient, cid uint32, buff []byte, size uint32) uint32 {
	return c.dispathData(cl, cid, buff, size)
}

// ClearBuff clear data buffer
func (c *Client) ClearBuff() {
	c.WaitDataSize = 0
}

func (c *Client) parseData(buffChan chan []byte) {
	for {
		for c.ringbuff.GetLen() > 0 {
			// CHeck data if complete request
			c.checkData(c, c.ringbuff.GetLen())
			if c.Status == WaitDataDone {
				reqLen := c.WaitDataSize
				getLen, reqBuff, isNew := c.ringbuff.GetData(reqLen)
				if getLen != reqLen {
					mlog.Error("Data size error!")
					break
				}

				var reqeuestBuff []byte

				if isNew {
					reqeuestBuff = reqBuff
				} else {
					reqeuestBuff = make([]byte, reqLen)
					copy(reqeuestBuff, reqBuff)
				}

				// If data is complete, queue a request to client manager
				c.requestCnt++
				request := &Request{
					cid:  c.Cid,
					conn: c.Conn,
					data: reqeuestBuff,
					size: uint32(len(reqeuestBuff)),
				}
				ClientManager.Queue(request)

				c.Status = Idle
			} else if c.Status == Err {
				mlog.Error("Data format error!")
				c.ringbuff.Clear()
				c.Status = Idle
				break
			} else {
				break
			}
		}

		select {
		case buff := <-buffChan:
			mlog.Info("Get data:", len(buff))
			c.ringbuff.PutData(buff)

			//mlog.Info("Ring buff:", c.ringbuff.Buff)
		}
	}
}

// ReadHandler wait and read data
func (c *Client) ReadHandler() {
	mlog.Info("ReadHandler started")
	defer mlog.Debug("ReadHandler exit:", c.Cid)
	defer c.Stop()

	c.ringbuff.Size = 102400 * 2
	c.ringbuff.Buff = make([]byte, c.ringbuff.Size)
	c.ringbuff.ReadIdx = 0
	c.ringbuff.WriteIdx = 0

	var sizeBuf uint32 = 8192
	c.buffPool = make([][]byte, 10) //10 rows
	for i := 0; i < 10; i++ {
		c.buffPool[i] = make([]byte, sizeBuf)
	}
	c.buffIdx = 0

	buffChan := make(chan []byte, 10)
	go c.parseData(buffChan)

	for {
		var readBuf []byte = c.buffPool[c.buffIdx]

		// Read data from connection
		rlen, err := c.Conn.Read(readBuf, sizeBuf)
		if err != nil {
			mlog.Error("Read exit!")
			break
		}

		mlog.Info("read data:", readBuf[:rlen])

		if rlen > 0 {
			buffChan <- readBuf[:rlen]
		}

		c.buffIdx++
		if c.buffIdx >= 10 {
			c.buffIdx = 0
		}
	}
}

// Send handle write data
func (c *Client) Send(data []byte, size uint32) {
	if c.Status != Closed {
		mlog.Info("Send data:", data)
		c.WriteHandler(data, size)
	}
}

// WriteHandler handle write data
func (c *Client) WriteHandler(buff []byte, size uint32) {
	mlog.Debug("WriteHandler started")

	c.Conn.Write(buff, size)
}

// Dequeue start client
func (c *Client) Dequeue() {
	mlog.Debug("Dequeue")

	c.requestCnt--
}

// IsInWork check if client has request in work
func (c *Client) IsInWork() bool {
	return c.workIndo
}

// SetInWork set if client has request in work
func (c *Client) SetInWork(sts bool) {
	c.workIndo = sts
}

// Start start client
func (c *Client) Start() {
	mlog.Debug("Client start")

	ClientManager.AddClient(c.Cid, c)

	go c.ReadHandler()
	//go c.WriteHandler()
}

//Stop stop client
func (c *Client) Stop() {
	mlog.Debug("Client stop:", c.Cid)

	if (c.Status != Closed) && (c.Status != Removed) {
		c.Status = Closed
		c.Conn.Close()
	}

	if (c.requestCnt == 0) && (c.Status != Removed) {
		c.Conn.FreeCid()
		ClientManager.RemoveClient(c.Cid)
		c.Status = Removed
	}
}
