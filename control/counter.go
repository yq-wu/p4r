package control

import (
	"errors"

	"github.com/p4lang/p4runtime/go/p4/v1"
	"p4r/entity"
)

// CounterControl
// 用于操作 P4 中的计数器。主要功能包括读取指定索引的计数器值、读取所有计数器值，以及异步读取所有计数器值。
type CounterControl struct {
	control *Controller
	counter *entity.Counter
}

// ，用于从 v1.Entity 类型中提取计数器数据，返回 CounterData 结构体
func getCounterData(entity *v1.Entity) CounterData {
	counterEntry := entity.GetCounterEntry()
	return CounterData{
		ByteCount:   counterEntry.Data.ByteCount,
		PacketCount: counterEntry.Data.PacketCount,
		Index:       counterEntry.Index.Index,
	}
}

// ReadValueAtIndex 方法用于读取计数器在指定索引处的值。
// 它通过索引读取 entity 并调用客户端同步读取方法 ReadEntitiesSync，返回一个 CounterData 实例。
func (cc *CounterControl) ReadValueAtIndex(index int64) (*CounterData, error) {
	Entity := cc.counter.ReadValueWithIndex(index)
	entityList := []*v1.Entity{Entity}

	res, err := cc.control.Client.ReadEntitiesSync(entityList)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("No counter entries found at given index")
	}

	result := getCounterData(res[0])
	return &result, nil
}

// ReadValues 该方法用于读取计数器的所有值。它调用 ReadEntitiesSync，读取并返回计数器的所有条目。
func (cc *CounterControl) ReadValues() ([]*CounterData, error) {
	entity := cc.counter.ReadValue()
	entityList := []*v1.Entity{entity}

	res, err := cc.control.Client.ReadEntitiesSync(entityList)

	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("Target counter does not have any entries")
	}

	result := make([]*CounterData, 0)
	for _, item := range res {
		if item == nil {
			continue
		}
		counterData := getCounterData(item)
		result = append(result, &counterData)
	}

	return result, nil
}

// StreamValues 该方法用于异步读取计数器的所有值，并将结果通过通道 (channel) 发送出去。它适用于需要异步操作的场景。
// 一个 goroutine 被启动，用于从 counterEntityCh 读取数据，并将其转换为 CounterData 后发送到通道 cdataChannel 中。
func (cc *CounterControl) StreamValues() (chan *CounterData, error) {
	entity := cc.counter.ReadValue()
	entityList := []*v1.Entity{entity}

	counterEntityCh, err := cc.control.Client.ReadEntities(entityList)
	if err != nil {
		return nil, err
	}

	cdataChannel := make(chan *CounterData, cc.counter.Size)
	go func() {
		defer close(cdataChannel)
		for e := range counterEntityCh {
			counterData := getCounterData(e)
			cdataChannel <- &counterData
		}
	}()

	return cdataChannel, nil
}
