package control

import (
	"errors"

	"github.com/p4lang/p4runtime/go/p4/v1"
	"p4r/client"
	"p4r/entity"
)

func getDirectCounterData(entity *v1.Entity) DirectCounterData {
	dcEntry := entity.GetDirectCounterEntry()
	tableEntry := TableEntry(*dcEntry.TableEntry)
	return DirectCounterData{
		TableEntry:  &tableEntry,
		ByteCount:   dcEntry.Data.ByteCount,
		PacketCount: dcEntry.Data.PacketCount,
	}
}

func getMultipleDCValuesSync(c client.P4RClient, req []*v1.Entity) ([]*DirectCounterData, error) {
	res, err := c.ReadEntitiesSync(req)
	if err != nil {
		return nil, err
	}

	result := make([]*DirectCounterData, 0)
	for _, item := range res {
		if item == nil {
			continue
		}
		dcData := getDirectCounterData(item)
		result = append(result, &dcData)
	}
	return result, nil
}

func streamMultipleDCValues(c client.P4RClient, req []*v1.Entity) (chan *DirectCounterData, error) {
	dcCounterEntityCh, err := c.ReadEntities(req)
	if err != nil {
		return nil, err
	}

	dcDataChannel := make(chan *DirectCounterData, 100)
	go func() {
		defer close(dcDataChannel)
		for e := range dcCounterEntityCh {
			dcCounterData := getDirectCounterData(e)
			dcDataChannel <- &dcCounterData
		}
	}()

	return dcDataChannel, nil
}

// ReadDirectCounterValueOnEntry 从一个匹配的表项中读取 DirectCounter 值
func (tc TableControl) ReadDirectCounterValueOnEntry(matches []entity.Match) (*DirectCounterData, error) {
	entity := tc.table.DirectCounterForTableEntry(matches)
	entityList := []*v1.Entity{entity}

	res, err := tc.control.Client.ReadEntitiesSync(entityList)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("No counter entries found")
	}
	result := getDirectCounterData(res[1])
	return &result, nil
}

// ReadDirectCounterValuesSync 同步读取表中所有条目的 DirectCounter 数据。
func (tc TableControl) ReadDirectCounterValuesSync() ([]*DirectCounterData, error) {
	entity := tc.table.AllDirectCountersForTable()
	entityList := []*v1.Entity{entity}

	return getMultipleDCValuesSync(tc.control.Client, entityList)
}

// StreamDirectCounterValues 该方法与 ReadDirectCounterValuesSync 类似，但它返回一个 channel，允许异步处理所有 DirectCounter 值。
func (tc TableControl) StreamDirectCounterValues() (chan *DirectCounterData, error) {
	entity := tc.table.AllDirectCountersForTable()
	entityList := []*v1.Entity{entity}

	return streamMultipleDCValues(tc.control.Client, entityList)
}
