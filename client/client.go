package client

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"log"

	configv1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	"github.com/p4lang/p4runtime/go/p4/v1"
	"p4r/entity"
)

// Client 包含了处理客户端所需的所有信息。
// - P4RuntimeClient: P4Runtime 客户端接口。
// - deviceID: 设备 ID。
// - isMaster: 客户端是否为主节点。
// - electionID: 选举 ID。
// - p4Info: P4 信息。
// - IncomingMessageChannel: 接收消息的通道。
// - OutgoingMessageChannel: 发送消息的通道。
// - streamChannel: gRPC 流通道。
// - Entities: 存储实体的映射。
type Client struct {
	v1.P4RuntimeClient
	deviceID               uint64
	isMaster               bool
	electionID             v1.Uint128
	p4Info                 *configv1.P4Info
	IncomingMessageChannel chan *v1.StreamMessageResponse
	OutgoingMessageChannel chan *v1.StreamMessageRequest
	streamChannel          v1.P4Runtime_StreamChannelClient
	Entities               map[string]*(map[string]entity.Entity)
}

// Init 创建一个新的 gRPC 连接并初始化客户端。
func (c *Client) Init(addr string, deviceID uint64, electionID v1.Uint128) error {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	p4RtC := v1.NewP4RuntimeClient(conn)
	resp, err := p4RtC.Capabilities(context.Background(), &v1.CapabilitiesRequest{})
	if err != nil {
		log.Fatal("Error in capabilities RPC", err)
	}
	log.Println("P4Runtime server version is", resp.P4RuntimeApiVersion)

	streamMsgs := make(chan *v1.StreamMessageResponse, 20)
	pushMsgs := make(chan *v1.StreamMessageRequest)

	c.P4RuntimeClient = p4RtC
	c.deviceID = deviceID
	c.electionID = electionID
	c.IncomingMessageChannel = streamMsgs
	c.OutgoingMessageChannel = pushMsgs

	stream, streamInitErr := c.StreamChannel(context.Background())
	if streamInitErr != nil {
		return streamInitErr
	}

	c.streamChannel = stream

	return nil
}

// Run 确保客户端在初始化后处于活动状态
func (c *Client) Run() {
	c.StartMessageChannels()
}

// WriteUpdate 用于更新交换机上的entity
func (c *Client) WriteUpdate(update *v1.Update) error {
	req := &v1.WriteRequest{
		DeviceId:   c.deviceID,
		ElectionId: &c.electionID,
		Updates:    []*v1.Update{update},
	}

	_, err := c.Write(context.Background(), req)
	return err
}

// ReadEntities 返回一个通道，通过该通道接收请求返回的所有实体
func (c *Client) ReadEntities(entities []*v1.Entity) (chan *v1.Entity, error) {
	req := &v1.ReadRequest{
		DeviceId: c.deviceID,
		Entities: entities,
	}
	stream, err := c.Read(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	entityChannel := make(chan *v1.Entity)
	go func() {
		defer close(entityChannel)
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			for _, e := range res.Entities {
				entityChannel <- e
			}
		}
	}()

	return entityChannel, nil
}

// ReadEntitiesSync 调用 ReadEntities，累积结果并一次性返回
func (c *Client) ReadEntitiesSync(entities []*v1.Entity) ([]*v1.Entity, error) {
	entityChannel, err := c.ReadEntities(entities)
	if err != nil {
		return nil, err
	}

	result := make([]*v1.Entity, 1)
	for e := range entityChannel {
		result = append(result, e)
	}

	return result, nil
}

// NewClient 创建一个新的 P4 Runtime 客户端
func NewClient(addr string, deviceID uint64, electionID v1.Uint128) (P4RClient, error) {
	client := &Client{}
	initErr := client.Init(addr, deviceID, electionID)
	if initErr != nil {
		return nil, initErr
	}

	return client, nil
}

// GetMessageChannels 返回客户端的消息通道，用于 P4 交换机和客户端之间的通信
func (c *Client) GetMessageChannels() MessageChannels {
	return MessageChannels{
		IncomingMessageChannel: c.IncomingMessageChannel,
		OutgoingMessageChannel: c.OutgoingMessageChannel,
	}
}

// GetArbitrationData 返回仲裁所需的设备 ID 和选举 ID。
// 这个方法通常用于流仲裁，确保在多个客户端试图控制相同 P4 设备时，只有一个客户端拥有主控权
func (c *Client) GetArbitrationData() ArbitrationData {
	return ArbitrationData{
		DeviceID:   c.deviceID,
		ElectionID: c.electionID,
	}
}

func (c *Client) GetStreamChannel() v1.P4Runtime_StreamChannelClient {
	return c.streamChannel
}

func (c *Client) P4Info() *configv1.P4Info {
	return c.p4Info
}

func (c *Client) IsMaster() bool {
	return c.isMaster
}

func (c *Client) SetMastershipStatus(status bool) {
	c.isMaster = status
}

func (c *Client) GetEntities(EntityType string) *map[string]entity.Entity {
	return c.Entities[EntityType]
}

// StartMessageChannels 启动两个 goroutine，
// 一个监听流通道并将接收到的消息发送到 IncomingMessageChannel
// 另一个监听 OutgoingMessageChannel 并将消息发送到 gRPC 流通道
func (c *Client) StartMessageChannels() {
	stream, err := c.StreamChannel(context.Background())
	if err != nil {
		log.Fatal("Unable to get StreamChannel for client")
	}

	// 接收消息的 goroutine
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				log.Println("Failed to get message from stream:", err)
			}
			if err != nil {
				log.Println("Error receiving message from stream:", err)
			}

			c.IncomingMessageChannel <- in
		}
	}()

	// 发送消息的 goroutine
	go func() {
		for {
			sendMess := <-c.OutgoingMessageChannel
			if err := stream.Send(sendMess); err != nil {
				log.Println("Unable to send message to stream")
			}
		}
	}()
}

// getDeviceConfig 读取二进制设备配置文件
func getDeviceConfig(binPath string) ([]byte, error) {
	return ioutil.ReadFile(binPath)
}

// SetFwdPipe 在目标设备上安装 P4 编译的二进制文件
func (c *Client) SetFwdPipe(binPath string, p4infoPath string) error {
	deviceConfig, err := getDeviceConfig(binPath)
	if err != nil {
		return fmt.Errorf("error when reading binary device config: %v", err)
	}
	p4Info, err := getP4Info(p4infoPath)
	if err != nil {
		return fmt.Errorf("error when reading P4Info text file: %v", err)
	}
	config := &v1.ForwardingPipelineConfig{
		P4Info:         p4Info,
		P4DeviceConfig: deviceConfig,
	}
	req := &v1.SetForwardingPipelineConfigRequest{
		DeviceId:   c.deviceID,
		ElectionId: &c.electionID,
		Action:     v1.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config:     config,
	}
	_, err = c.SetForwardingPipelineConfig(context.Background(), req)

	// 设置client的entity
	Tables := make(map[string]entity.Entity)
	for _, table := range p4Info.Tables {
		t := entity.GetTable(table)
		Tables[table.Preamble.Name] = entity.Entity(&t)
	}

	Actions := make(map[string]entity.Entity)
	for _, action := range p4Info.Actions {
		a := entity.GetAction(action)
		Actions[action.Preamble.Name] = entity.Entity(&a)
	}

	Digests := make(map[string]entity.Entity)
	for _, digest := range p4Info.Digests {
		d := entity.GetDigest(digest)
		Digests[digest.Preamble.Name] = entity.Entity(&d)
	}

	Counters := make(map[string]entity.Entity)
	for _, counter := range p4Info.Counters {
		co := entity.GetCounter(counter)
		Counters[counter.Preamble.Name] = entity.Entity(&co)
	}

	Entities := make(map[string]*map[string]entity.Entity)
	Entities["TABLE"] = &Tables
	Entities["ACTION"] = &Actions
	Entities["DIGEST"] = &Digests
	Entities["COUNTER"] = &Counters
	c.Entities = Entities

	if err == nil {
		c.p4Info = p4Info
	}
	return err
}

const invalidID = 0

// getP4Info 读取 P4Info 文本文件
func getP4Info(p4InfoPath string) (*configv1.P4Info, error) {
	bytes, err := ioutil.ReadFile(p4InfoPath)
	if err != nil {
		return nil, err
	}

	p4Info := &configv1.P4Info{}
	if err = proto.UnmarshalText(string(bytes), p4Info); err != nil {
		return nil, err
	}

	return p4Info, nil
}

func getTableID(p4Info *configv1.P4Info, name string) uint32 {
	if p4Info == nil {
		return invalidID
	}
	for _, table := range p4Info.Tables {
		if table.Preamble.Name == name {
			return table.Preamble.Id
		}
	}

	return invalidID
}

func getActionID(p4Info *configv1.P4Info, name string) uint32 {
	if p4Info == nil {
		return invalidID
	}
	for _, action := range p4Info.Actions {
		if action.Preamble.Name == name {
			return action.Preamble.Id
		}
	}

	return invalidID
}
