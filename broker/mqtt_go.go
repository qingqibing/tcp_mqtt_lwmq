package main

import (
	"lwmq/deviceview"
	"lwmq/dispatcher"
	"lwmq/manager"
	"lwmq/mlog"
	"lwmq/service"
)

func main() {
	mlog.SetMinLevel(mlog.WARNING)

	deviceview.Startservice()

	manager.ClientManager.SetWorkInQueue(true)
	manager.ClientManager.SetOnAdd(dispatcher.OnAddClient)
	manager.ClientManager.StartWorkers(15)

	lwmq := service.NewLwmq()
	// setConnHandler
	lwmq.SetOnConnect(manager.ClientOnConn)
	lwmq.Start()
}
