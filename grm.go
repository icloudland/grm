package grm


import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"math/rand"
	"time"
	"bytes"
)

const (
	STOP    = "__P:"
)

type GoroutineChannel struct {
	gid  uint64
	name string
	msg  chan string
}

type GoroutineChannelMap struct {
	mutex      sync.Mutex
	grchannels map[string]*GoroutineChannel
}

func (m *GoroutineChannelMap) unregister(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.grchannels[name]; !ok {
		return fmt.Errorf("goroutine channel not find: %q", name)
	}
	fmt.Println("goroutine[" + name + "] quit")
	delete(m.grchannels, name)
	return nil
}

func (m *GoroutineChannelMap) register(name string) error {
	gchannel := &GoroutineChannel{
		gid:  uint64(rand.Int63()),
		name: name,
	}
	gchannel.msg = make(chan string)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.grchannels == nil {
		m.grchannels = make(map[string]*GoroutineChannel)
	} else if _, ok := m.grchannels[gchannel.name]; ok {
		return fmt.Errorf("goroutine channel already defined: %q", gchannel.name)
	}
	m.grchannels[gchannel.name] = gchannel
	return nil
}

func (m *GoroutineChannelMap) stop() {

	grNames := make([]string, 0)

	m.mutex.Lock()
	for name, _ := range m.grchannels {
		grNames = append(grNames, name)
	}
	m.mutex.Unlock()

	for _, name := range grNames {
		if v, ok := m.get(name); ok {
			v.msg <- STOP + strconv.Itoa(int(v.gid))
		}
	}

	//m.grchannels = make(map[string]*GoroutineChannel)
}

func (m *GoroutineChannelMap) view() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var buf bytes.Buffer
	fmt.Println("***********current gorountine***********")
	for k, _ := range m.grchannels {
		fmt.Println(k)
		buf.WriteString(k)
		buf.WriteString(";")
	}
	fmt.Println("****************************************")

	return buf.String()

}

func (m *GoroutineChannelMap) hasGrchannels(name string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.grchannels == nil {
		return false
	}
	_, ok := m.grchannels[name]
	return ok
}

func (m *GoroutineChannelMap) get(name string) (*GoroutineChannel, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.grchannels == nil {
		return nil, false
	}
	if _, ok := m.grchannels[name]; ok {
		return m.grchannels[name], true
	}

	return nil, false
}

func (m *GoroutineChannelMap) get1(name string) (*GoroutineChannel) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.grchannels == nil {
		return nil
	}
	if _, ok := m.grchannels[name]; ok {
		return m.grchannels[name]
	}

	return nil
}

type GrManager struct {
	grchannelMap *GoroutineChannelMap
}

func NewGrManager() *GrManager {
	gm := &GoroutineChannelMap{}
	return &GrManager{grchannelMap: gm}
}

func (gm *GrManager) StopLoopGoroutine(name string) error {
	stopChannel, ok := gm.grchannelMap.get(name)
	if !ok {
		return fmt.Errorf("not found goroutine name :" + name)
	}
	stopChannel.msg <- STOP + strconv.Itoa(int(stopChannel.gid))
	return nil
}

func (gm *GrManager) Stop() {
	gm.grchannelMap.stop()
}

func (gm *GrManager) View() string {
	return gm.grchannelMap.view()
}

func (gm *GrManager) HasGrchannel(name string) bool {
	return gm.grchannelMap.hasGrchannels(name)
}

func (gm *GrManager) Register(name string) error {
	return gm.grchannelMap.register(name)
}

func (gm *GrManager) UnRegister(name string) error {
	return gm.grchannelMap.unregister(name)
}

func (gm *GrManager) Get1(name string) (*GoroutineChannel) {
	return gm.grchannelMap.get1(name)
}

func (gm *GrManager) GetMsg(name string) (chan string) {
	return gm.grchannelMap.get1(name).msg
}

func (gm *GrManager) NewLoopGr(name string, fc interface{}, args ...interface{}) {
	go func(this *GrManager, n string, fc interface{}, args ...interface{}) {
		//register channel
		err := this.grchannelMap.register(n)
		if err != nil {
			return
		}
		for {
			select {
			case info := <-this.grchannelMap.get1(name).msg:
				taskInfo := strings.Split(info, ":")
				signal, gid := taskInfo[0], taskInfo[1]
				if gid == strconv.Itoa(int(this.grchannelMap.grchannels[name].gid)) {
					if signal == "__P" {
						this.grchannelMap.unregister(name)
						return
					} else {
						fmt.Println("unknown signal")
					}
				}
			default:
				fmt.Println("no signal")
			}

			if len(args) > 1 {
				fc.(func(...interface{}))(args)
			} else if len(args) == 1 {
				fc.(func(interface{}))(args[0])
			} else {
				fc.(func())()
			}
		}
	}(gm, name, fc, args...)
}

func (gm *GrManager) NewTimerGr(name string, fc interface{}, d time.Duration, args ...interface{}) {
	go func(this *GrManager, n string, fc interface{}, args ...interface{}) {
		//register channel
		err := this.grchannelMap.register(n)
		if err != nil {
			return
		}
		t := time.NewTimer(0)

		for {
			select {
			case info := <-this.grchannelMap.get1(name).msg:
				taskInfo := strings.Split(info, ":")
				signal, gid := taskInfo[0], taskInfo[1]
				if gid == strconv.Itoa(int(this.grchannelMap.grchannels[name].gid)) {
					if signal == "__P" {
						this.grchannelMap.unregister(name)
						return
					}else {
						fmt.Println("unknown signal")
					}
				}
			case <-t.C:
				if len(args) > 1 {
					fc.(func(...interface{}))(args)
				} else if len(args) == 1 {
					fc.(func(interface{}))(args[0])
				} else {
					fc.(func())()
				}
				t.Reset(d)
			}
		}

	}(gm, name, fc, args...)
}

func (gm *GrManager) NewTimer1Gr(name string, fc interface{}, start time.Duration, internal time.Duration, deadLine time.Duration, args ...interface{}) {
	if len(args) > 1 {
		gm.NewTimer2Gr(name, fc, nil, start, internal, deadLine, args)
	} else if len(args) == 1 {
		gm.NewTimer2Gr(name, fc, nil, start, internal, deadLine, args[0])
	} else {
		gm.NewTimer2Gr(name, fc, nil, start, internal, deadLine)
	}

}

func (gm *GrManager) NewTimer2Gr(name string, fc interface{}, dfc interface{}, start time.Duration, internal time.Duration, deadLine time.Duration, args ...interface{}) {
	go func(this *GrManager, n string, fc interface{}, args ...interface{}) {
		//register channel
		err := this.grchannelMap.register(n)
		if err != nil {
			return
		}
		t := time.NewTimer(start)
		dl := time.Now().Add(deadLine)

		for {
			select {
			case info := <-this.grchannelMap.get1(name).msg:
				taskInfo := strings.Split(info, ":")
				signal, gid := taskInfo[0], taskInfo[1]
				if gid == strconv.Itoa(int(this.grchannelMap.grchannels[name].gid)) {
					if signal == "__P" {
						this.grchannelMap.unregister(name)
						return
					}else {
						fmt.Println("unknown signal")
					}
				}
			case <-t.C:
				if len(args) > 1 {
					fc.(func(...interface{}))(args)
				} else if len(args) == 1 {
					ret := fc.(func(interface{}) interface{})(args[0])
					if ret == nil {
						this.grchannelMap.unregister(name)
						return
					}
				} else {
					ret := fc.(func() interface{})()
					if ret == nil {
						this.grchannelMap.unregister(name)
						return
					}

				}
				if time.Now().After(dl) {
					this.grchannelMap.unregister(name)
					if dfc != nil {
						dfc.(func(string))(name)
					}
					return
				}
				t.Reset(internal)
			}
		}

	}(gm, name, fc, args...)
}

func (gm *GrManager) NewGr(name string, fc interface{}, args ...interface{}) {
	go func(n string, fc interface{}, args ...interface{}) {
		//register channel
		err := gm.grchannelMap.register(n)
		if err != nil {
			return
		}
		if len(args) > 1 {
			fc.(func(...interface{}))(args)
		} else if len(args) == 1 {
			fc.(func(interface{}))(args[0])
		} else {
			fc.(func())()
		}
		gm.grchannelMap.unregister(name)
	}(name, fc, args...)

}
