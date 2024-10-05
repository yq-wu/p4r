package control

import (
	"github.com/p4lang/p4runtime/go/p4/v1"
)

type ControlTable interface {
	Table(string) TableControl
	Digest(string) DigestControl
	Counter(string) CounterControl
}

type Control interface {
	ControlTable
	PerformArbitration()
	IsMaster() bool
	SetMastershipStatus(bool)
	Run()
	InstallProgram(string, string) error
}

type CounterData struct {
	ByteCount   int64
	PacketCount int64
	Index       int64
}

type TableEntry v1.TableEntry

type DirectCounterData struct {
	TableEntry  *TableEntry
	ByteCount   int64
	PacketCount int64
}
