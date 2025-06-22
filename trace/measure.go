package trace

import (
	"os"
	"strconv"
	"sync"
	"time"
)

type MeasureTime struct {
	file *os.File
	lock *sync.Mutex
}

func NewMeasureTime() *MeasureTime {
	file, err := os.Create("browser.trace")
	if err != nil {
		panic(err)
	}
	file.WriteString("{\"traceEvents\": [")
	ts := time.Now().UnixMicro()
	file.WriteString(
		`{ "name": "process_name",` +
			`"ph": "M",` +
			`"ts":` + strconv.Itoa(int(ts)) + `,` +
			`"pid": 1, "cat": "__metadata",` +
			`"args": {"name": "Browser"}}`)
	file.Sync()
	return &MeasureTime{file: file, lock: &sync.Mutex{}}
}

func (m *MeasureTime) Time(name string) {
	m.lock.Lock()
	ts := time.Now().UnixMicro()
	// note: no threadId for tid available
	m.file.WriteString(
		`, { "ph": "B", "cat": "_",` +
			`"name": "` + name + `",` +
			`"ts": ` + strconv.Itoa(int(ts)) + `,` +
			`"pid": 1, "tid": 1}`)
	m.file.Sync()
	m.lock.Unlock()
}

func (m *MeasureTime) Stop(name string) {
	m.lock.Lock()
	ts := time.Now().UnixMicro()
	// note: no threadId for tid available
	m.file.WriteString(
		`, { "ph": "E", "cat": "_",` +
			`"name": "` + name + `",` +
			`"ts": ` + strconv.Itoa(int(ts)) + `,` +
			`"pid": 1, "tid": 1}`)
	m.file.Sync()
	m.lock.Unlock()
}

func (m *MeasureTime) Finish() {
	m.lock.Lock()
	m.file.WriteString("]}")
	m.file.Close()
	m.lock.Unlock()
}
