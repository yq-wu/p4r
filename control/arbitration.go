package control

import (
	"log"

	"github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
)

// PerformArbitration 通过向交换机发送 仲裁请求 来参与仲裁流程。
func (sc *Controller) PerformArbitration() {
	msgChannels := sc.Client.GetMessageChannels()
	outChan := msgChannels.OutgoingMessageChannel
	arbitrationData := sc.Client.GetArbitrationData()

	request := &v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_Arbitration{Arbitration: &v1.MasterArbitrationUpdate{
			DeviceId:   arbitrationData.DeviceID,
			ElectionId: &(arbitrationData.ElectionID),
		}},
	}

	outChan <- request
}

// StartArbitrationUpdateListener 启动了一个 监听仲裁更新 的 goroutine，目的是检查仲裁结果，决定控制器是否获得主控权
func (sc *Controller) StartArbitrationUpdateListener() {
	go func() {
		update := <-sc.ArbitrationChannel
		if update.Arbitration.Status.Code != int32(code.Code_OK) {
			sc.SetMastershipStatus(false)
			log.Println("Arbitration was done. Control did not acquire mastership.")
		} else {
			sc.SetMastershipStatus(true)
			log.Println("Arbitration was done. Control acquired mastership")
		}
	}()
}
