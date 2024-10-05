package client

import (
	configv1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	"github.com/p4lang/p4runtime/go/p4/v1"
	"p4r/entity"
)

// MessageChannels 其中包含两个通道：
//   - IncomingMessageChannel 接收来自 P4 交换机的消息。
//   - OutgoingMessageChannel 发送消息到 P4 交换机。
type MessageChannels struct {
	IncomingMessageChannel chan *v1.StreamMessageResponse
	OutgoingMessageChannel chan *v1.StreamMessageRequest
}

// ArbitrationData 结构体，包含两个字段：
//   - DeviceID 用于标识客户端所连接的特定 P4 设备。
//   - ElectionID 标识客户端在流控制中的主控权。
type ArbitrationData struct {
	DeviceID   uint64
	ElectionID v1.Uint128
}

// EntityClient defines any client that can interact with P4 switch entities such
// as tables, actions, counters, etc
type EntityClient interface {
	GetEntities(string) *map[string]entity.Entity

	// WriteUpdate is used to update an entity on the switch. Refer to the P4Runtime spec to know more.
	WriteUpdate(update *v1.Update) error

	ReadEntities(entities []*v1.Entity) (chan *v1.Entity, error)

	ReadEntitiesSync(entities []*v1.Entity) ([]*v1.Entity, error)
}

// P4RClient represents a p4Runtime client. Most methods are just getters since Go's
// interface implementation does not allow non-function members
type P4RClient interface {
	EntityClient
	// To initialize the client
	Init(addr string, deviceID uint64, electionID v1.Uint128) error

	// Run will do whatever is needed to ensure that the client is active
	// once it is initialized.
	Run()

	SetFwdPipe(binPath string, p4InfoPath string) error

	// GetMessageChannels will return the message channels used by the client
	GetMessageChannels() MessageChannels

	// GetArbitrationData will return the data required to perform arbitration
	// for the client
	GetArbitrationData() ArbitrationData

	// GetStreamChannel will return the StreamChannel instance associated with the client
	GetStreamChannel() v1.P4Runtime_StreamChannelClient

	// P4Info will return the P4Info struct associated to the client
	P4Info() *configv1.P4Info

	// IsMaster returns true if the client is master
	IsMaster() bool

	// SetMastershipStatus sets the mastership status of the client
	SetMastershipStatus(bool)
}
