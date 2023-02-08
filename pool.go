package molix

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/molikatty/spinlock"
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
	tasks       []*MTask
	capacity    int32
	runnings    int32
	PanicHandle func(v interface{})
	status      Status
	spin           sync.Locker
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
		tasks:    make([]*MTask, 0, 1),
		spin:        spinlock.NewSpinLock(),
	}
	mp.cond = *sync.NewCond(mp.spin)


	return mp, nil
}

func (m *MPool) addRunnins() {
	atomic.AddInt32(&m.runnings, 1)
}

func (m *MPool) decRunnins() {
	atomic.AddInt32(&m.runnings, -1)
}

func (m *MPool) Running() int {
	return int(atomic.LoadInt32(&m.runnings))
}

func (m *MPool) Cap() int {
	return int(m.capacity)
}

func (m *MPool) Free() int {
	c := m.Cap()
	if c < 0 {
		return -1
	}

	return c - m.Running()
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

func (m *MPool) makeMTask() *MTask {
	return &MTask{
		p:  m,
		mt: make(chan func(), 1),
	}
}

func (m *MPool) getMTask() (mt *MTask) {
	m.spin.Lock()

	if mt = m.detach(); mt != nil {
		m.spin.Unlock()
	} else if m.Free() > 0 {
		m.spin.Unlock()
		mt = m.makeMTask()
		mt.run()
	} else {
	again:
		m.cond.Wait()
		if m.isStoped() {
			m.spin.Unlock()
			return
		}

		if mt = m.detach(); mt == nil {
			goto again
		}
		m.spin.Unlock()
	}

	return
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

	m.spin.Lock()
	defer m.spin.Unlock()
	m.tasks = append(m.tasks, mt)
	m.cond.Signal()

	return true
}

func (m *MTask) run() {
	m.p.addRunnins()
	go func() {
		defer func() {
			m.p.decRunnins()
			if err := recover(); err != nil {
				if m.p.PanicHandle == nil {
					panic(err)
				}
				m.p.PanicHandle(err)
			}
			m.p.cond.Signal()
		}()

		for task := range m.mt {
			if task == nil {
				return
			}

			task()

			if !m.p.insert(m) {
				break
			}
		}
	}()
}

func (m *MPool) Stop() {
	m.setStatus(RUNNING, STOPED)

	for i := range m.tasks {
		m.tasks[i].mt <- nil
		m.tasks[i] = nil
	}
	m.tasks = nil
}
