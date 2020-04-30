package manager

import (
	"container/list"
	"lwmq/iface"
	"lwmq/mlog"
	"sync"
)

// Manager clinet manager
type Manager struct {
	clients     map[uint32]iface.Iclient
	works       *list.List // All request in list
	lock        *sync.Mutex
	wakelock    *sync.Mutex
	cond        *sync.Cond
	workinqueue bool // True: request of one client in sequence
	onAddClient func(iface.Iclient)
}

// ClientManager manager
var ClientManager *Manager

// AddClient add one client
func (m *Manager) AddClient(cid uint32, clt iface.Iclient) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, exist := m.clients[cid]
	if exist {
		mlog.Error("Client exist error!")
		return
	}

	m.onAddClient(clt)
	m.clients[cid] = clt
	mlog.Debug("Add client:", cid)
}

// RemoveClient remove client
func (m *Manager) RemoveClient(cid uint32) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, exist := m.clients[cid]
	if !exist {
		mlog.Error("Client not exist!")
		return
	}

	delete(m.clients, cid)
	mlog.Debug("Remove client:", cid)
}

// GetClient get client
func (m *Manager) GetClient(cid uint32) (iface.Iclient, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	cl, exist := m.clients[cid]

	return cl, exist
}

// SetWorkInQueue set if do requests of one client in sequence
func (m *Manager) SetWorkInQueue(sts bool) {
	m.workinqueue = sts
}

// Wakeup workers
func (m *Manager) wakeup() {
	m.cond.Broadcast()
}

// Queue send request
func (m *Manager) Queue(request iface.Irequest) {
	m.lock.Lock()
	defer m.wakeup()
	defer m.lock.Unlock()

	m.works.PushBack(request)

	mlog.Info("Queue:", request)
}

// Retrieval a request from work pool
func (m *Manager) getOneWork() (iface.Irequest, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.works.Len() == 0 {
		return nil, false
	}

	if !m.workinqueue {
		Onerequest := m.works.Front()
		m.works.Remove(Onerequest)

		return Onerequest.Value.(iface.Irequest), true
	}

	var request iface.Irequest

	item := m.works.Front()
	inReqeuest := item.Value.(iface.Irequest)
	cid := inReqeuest.GetCid()
	cl, exist := m.clients[cid]
	if !exist {
		mlog.Warning("Client not exist, do request!")
		request = inReqeuest
	} else {
		if cl.IsInWork() {
			mlog.Info("Client in work, wait!")
		} else {
			cl.SetInWork(true)
			request = inReqeuest
			m.works.Remove(item)
		}
	}

	if request == nil {
		return nil, false
	}

	return request, true
}

// Single worker
func (m *Manager) oneWorker(idx int) {
	mlog.Debug("Worker:", idx, " started!")

	defer func() {
		if err := recover(); err != nil {
			mlog.Debug("Worker:", idx, " exit:", err)
		}
	}()

	for {
		request, ok := m.getOneWork()
		if ok {
			mlog.Info("Worker:", idx, " Get data:", request.GetData())

			if len(request.GetData()) == 0 {
				mlog.Warning("Worker:", idx, " request has no data!")
			}

			cid := request.GetCid()
			cl, exist := m.GetClient(cid)
			if !exist {
				mlog.Error("Client not exist, data may lost!")
				continue
			}

			// start real work
			cl.Dequeue()
			status := cl.DispathData(cl, cid, request.GetData(), request.GetSize())
			if m.workinqueue {
				m.lock.Lock()
				cl.SetInWork(false)
				m.lock.Unlock()

				m.wakeup()
			}
			if status != 0 {
				mlog.Error("Process error! Close connection:", status)
				cl.Stop()
			}

		} else {
			// If no request, go to sleep
			m.cond.L.Lock()
			m.cond.Wait()

			mlog.Info("Worker:", idx, " wakeup")

			m.cond.L.Unlock()
		}
	}
}

// StartWorkers start work pool
func (m *Manager) StartWorkers(workerCnt int) {
	for i := 0; i < workerCnt; i++ {
		go m.oneWorker(i)
	}
}

// SetOnAdd add client callback
func (m *Manager) SetOnAdd(onAdd func(iface.Iclient)) {
	m.onAddClient = onAdd
}

func init() {
	ClientManager = &Manager{
		clients:     make(map[uint32]iface.Iclient),
		works:       list.New(),
		lock:        new(sync.Mutex),
		wakelock:    new(sync.Mutex),
		workinqueue: false,
	}

	ClientManager.cond = sync.NewCond(ClientManager.wakelock)
}
