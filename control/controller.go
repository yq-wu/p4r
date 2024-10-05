package control

import (
	"errors"
	"log"

	"github.com/p4lang/p4runtime/go/p4/v1"
	"p4r/client"
	"p4r/entity"
)

// Controller 结构体
//   - Client: P4RClient 实例，用于与 P4Runtime 交换机通信。
//   - DigestChannel: 用于处理来自 P4 交换机的 Digest 消息的通道。
//   - ArbitrationChannel: 用于处理仲裁消息的通道，用于管理控制器的主控权。
//   - setupNotifChannel: 用于通知仲裁是否成功。
type Controller struct {
	Client             client.P4RClient
	DigestChannel      chan *v1.StreamMessageResponse_Digest
	ArbitrationChannel chan *v1.StreamMessageResponse_Arbitration
	setupNotifChannel  chan bool
}

// StartMessageRouter 该方法启动了一个 goroutine，监听 IncomingMessageChannel，
// 并根据消息类型将消息发送到相应的处理通道（例如仲裁消息发送到 ArbitrationChannel，Digest 消息发送到 DigestChannel）。
func (sc *Controller) StartMessageRouter() {
	IncomingMessageChannel := sc.Client.GetMessageChannels().IncomingMessageChannel
	go func() {
		for {
			in := <-IncomingMessageChannel
			update := in.GetUpdate()

			switch update.(type) {
			case *v1.StreamMessageResponse_Arbitration:
				sc.ArbitrationChannel <- update.(*v1.StreamMessageResponse_Arbitration)
			case *v1.StreamMessageResponse_Digest:
				sc.DigestChannel <- update.(*v1.StreamMessageResponse_Digest)
			default:
				log.Println("Message has unknown type")
			}
		}
	}()
}

// SetMastershipStatus 该方法设置控制器的主控权状态。
// 调用 P4RClient 的 SetMastershipStatus 方法，并通过 setupNotifChannel 通知主控权状态的变化。
func (sc *Controller) SetMastershipStatus(status bool) {
	sc.setupNotifChannel <- status
	sc.Client.SetMastershipStatus(status)
}

func (sc *Controller) IsMaster() bool {
	return sc.Client.IsMaster()
}

// Run 方法是控制器启动的主要入口，负责运行一系列操作：
//  1. 启动客户端。
//  2. 启动消息路由。
//  3. 启动仲裁更新监听。
//  4. 执行仲裁以参与主控权竞争。
//  5. 等待仲裁结果。
func (sc *Controller) Run() {
	sc.Client.Run()
	sc.StartMessageRouter()
	sc.StartArbitrationUpdateListener()
	sc.PerformArbitration()
	<-sc.setupNotifChannel
}

// InstallProgram 该方法用于安装 P4 编译后的二进制程序到设备上。
//   - 它首先检查是否拥有主控权，只有在成为主控设备时才能执行安装操作，否则会返回错误。
func (sc *Controller) InstallProgram(binPath, p4InfoPath string) error {
	if !sc.IsMaster() {
		return errors.New("Control does not have mastership, cannot install program on device")
	}
	return sc.Client.SetFwdPipe(binPath, p4InfoPath)
}

func NewController(addr string, deviceID uint64, electionID v1.Uint128) (Control, error) {
	Client, err := client.NewClient(addr, deviceID, electionID)
	if err != nil {
		return nil, err
	}
	digestChan := make(chan *v1.StreamMessageResponse_Digest, 10)
	arbitrationChan := make(chan *v1.StreamMessageResponse_Arbitration)
	setupNotifChan := make(chan bool)

	controller := Controller{
		Client:             Client,
		DigestChannel:      digestChan,
		ArbitrationChannel: arbitrationChan,
		setupNotifChannel:  setupNotifChan,
	}

	return &controller, nil
}

// Table 返回 TableControl
func (sc *Controller) Table(tableName string) TableControl {
	tables := *sc.Client.GetEntities("TABLE")
	table := tables[tableName].(*entity.Table)

	return TableControl{
		table:   table,
		control: sc,
	}
}

// Digest 返回 DigestControl
func (sc *Controller) Digest(digestName string) DigestControl {
	digests := *sc.Client.GetEntities("DIGEST")
	digest := digests[digestName].(*entity.Digest)

	return DigestControl{
		digest:  digest,
		control: sc,
	}
}

// Counter 返回 CounterControl
func (sc *Controller) Counter(counterName string) CounterControl {
	counters := *sc.Client.GetEntities("COUNTER")
	counter := counters[counterName].(*entity.Counter)

	return CounterControl{
		counter: counter,
		control: sc,
	}
}
