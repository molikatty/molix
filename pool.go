package molix

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Status int32

const (
	RUNNING Status = iota
	STOPED
)

var (
	ErrPoolCap          = errors.New("invalid size for pool")
	ErrPoolAlreadyStope = errors.New("pool already stoped")
)

type MTask struct {
	p  *MPool
	mt chan func()
}

type MPool struct {
	capacity    int32
	runnings    int32
	tasks       []*MTask
	PanicHandle func(v interface{})
	p           *sync.Pool
	status      Status
	l           sync.Mutex
	once        sync.Once
	cond        sync.Cond
}

func NewMPool(size int32) (*MPool, error) {
	if size <= 0 {
		return nil, ErrPoolCap
	}

	mp := &MPool{
		status:   RUNNING,
		capacity: size,
	}
	mp.cond = *sync.NewCond(&mp.l)

	mp.p = &sync.Pool{
		New: func() interface{} {
			return &MTask{
				p:  mp,
				mt: make(chan func(), 1),
			}
		},
	}

	return mp, nil
}

func (m *MPool) addRunnins() {
	atomic.AddInt32(&m.runnings, 1)
}

func (m *MPool) decRunnins() {
	atomic.AddInt32(&m.runnings, -1)
}

func (m *MPool) GetRunnings() int {
	return int(atomic.LoadInt32(&m.runnings))
}

func (m *MPool) GetCap() int {
	return int(m.capacity)
}

func (m *MPool) isStoped() bool {
	return atomic.LoadInt32((*int32)(&m.status)) == int32(STOPED)
}

func (m *MPool) setStatus(o, t Status) bool {
	return atomic.CompareAndSwapInt32((*int32)(&m.status), int32(o), int32(t))
}

func (m *MPool) Submit(t func()) error {
	if m.isStoped() {
		return ErrPoolAlreadyStope
	}

	m.getMTask().mt <- t

	return nil
}

func (m *MPool) getMTask() *MTask {
	var mt *MTask
	m.l.Lock()

	if mt = m.detach(); mt != nil {
		m.l.Unlock()
	} else if m.GetCap() > m.GetRunnings() {
		m.l.Unlock()
		mt = m.p.Get().(*MTask)
		mt.run()
	} else {
	again:
		m.cond.Wait()
		if mt = m.detach(); mt == nil {
			goto again
		}
		m.l.Unlock()
	}

	return mt
}

func (m *MPool) detach() *MTask {
	n := len(m.tasks)
	if n == 0 {
		return nil
	}

	mt := m.tasks[n-1]
	m.tasks[n-1] = nil
	m.tasks = m.tasks[:n-1]
	return mt
}

func (m *MPool) insert(mt *MTask) bool {
	if m.isStoped() {
		m.cond.Broadcast()
		return false
	}
	m.l.Lock()
	defer m.l.Unlock()
	m.tasks = append(m.tasks, mt)
	m.cond.Signal()

	return true
}

func (m *MTask) run() {
	m.p.addRunnins()
	go func() {
		defer func() {
			m.p.decRunnins()
			m.p.p.Put(m)
			if err := recover(); err != nil {
				if m.p.PanicHandle == nil {
					panic(err)
				}
				m.p.PanicHandle(err)
			}
			m.p.cond.Signal()
		}()

		for task := range m.mt {
			task()

			if !m.p.insert(m) {
				break
			}
		}
	}()
}

func (m *MPool) Close() error {
	m.once.Do(func() {
		m.setStatus(RUNNING, STOPED)
		for i := range m.tasks {
			for len(m.tasks[i].mt) > 0 {
				time.Sleep(1e6)
			}
			close(m.tasks[i].mt)
			m.tasks[i] = nil
		}
		m.tasks = nil
	})

	return nil
}
