package paxosutility

import (
    "sync"
    "sync/atomic"
)

type PaxosData struct {
    Lock sync.Mutex
    Np atomic.Value
    Na atomic.Value
    Va atomic.Value
}

func NewPaxosData() *PaxosData {
    data := &PaxosData{sync.Mutex{}, atomic.Value{}, atomic.Value{}, atomic.Value{}}
    data.Np.Store(-1)
    data.Na.Store(-1)
    return data
}

//--------------------------------------------------//

type PaxosAllocator struct {
    data map[int](*PaxosData)
    bound int
    lock sync.Mutex
}

func NewPaxosAllocator() *PaxosAllocator {
    return &PaxosAllocator{make(map[int](*PaxosData)), -1, sync.Mutex{}}
}

func (alloc *PaxosAllocator) Create(seq int) (*PaxosData) {
    alloc.lock.Lock()
    defer alloc.lock.Unlock()

    if seq <= alloc.bound {
        return NewPaxosData()
    }
    val, exist := alloc.data[seq]
    if exist {
        return val
    } else {
        tmp := NewPaxosData()
        alloc.data[seq] = tmp
        return tmp
    }
}

func (alloc *PaxosAllocator) Done(newbound int) {
    alloc.lock.Lock()
    defer alloc.lock.Unlock()

    if newbound > alloc.bound {
        for seq, _ := range(alloc.data) {
            if seq <= newbound {
                delete(alloc.data, seq)
            }
        }
        alloc.bound = newbound
    }
}

//--------------------------------------------------//

type PaxosResult struct {
    data map[int]interface{}
    bound int
    lock sync.Mutex
}

func NewPaxosResult() *PaxosResult {
    return &PaxosResult{make(map[int]interface{}), -1, sync.Mutex{}}
}

func (result *PaxosResult) Read(seq int) (bool, interface{}) {
    result.lock.Lock()
    defer result.lock.Unlock()

    val, exist := result.data[seq]
    if exist {
        return true, val
    } else {
        return false, nil
    }
}

func (result *PaxosResult) Write(seq int, v interface{}) {
    result.lock.Lock()
    defer result.lock.Unlock()

    if seq > result.bound {
        result.data[seq] = v
    }
}

func (result *PaxosResult) Done(newbound int) {
    result.lock.Lock()
    defer result.lock.Unlock()

    if newbound > result.bound {
        for seq, _ := range(result.data) {
            if seq <= newbound {
                delete(result.data, seq)
            }
        }
        result.bound = newbound
    }
}
