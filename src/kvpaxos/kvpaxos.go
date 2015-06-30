package kvpaxos

import (
    "paxos"

    "encoding/gob"
    "sync"
    "time"
)

type KVPaxosMap struct {
    lock sync.Mutex
    px *paxos.Paxos
    done int
    data map[string]string
    dead bool
}

func NewKVPaxosMap(peers []string, me int) *KVPaxosMap {
    m := &KVPaxosMap{}
    m.lock = sync.Mutex{}
    m.px = paxos.Make(peers, me, nil)
    m.done = 0
    m.data = make(map[string]string)
    m.dead = false
    return m
}

//--------------------------------------------------------------//

type Proposal struct {
    Type string
    Key string
    Value string
}

func (p *Proposal) equalsTo(p_ *Proposal) bool {
    return p.Type == p_.Type && p.Key == p_.Key && p.Value == p_.Value
}

func init() {
    gob.RegisterName("Proposal", Proposal{})
}

//--------------------------------------------------------------//

// Must acquire m.lock
func (m *KVPaxosMap) waitDecide(seq int) {
    to := 10 * time.Millisecond
    for {
        decided, _ := m.px.Status(seq)
        if decided {
            return
        }
        time.Sleep(to)
        if to < 10 * time.Second {
            to *= 2
        }
    }
}

// Must acquire m.lock
func (m *KVPaxosMap) submitProposal(p Proposal) int {
    for {
        seq := m.px.Max() + 1
        m.px.Start(seq, p)
        m.waitDecide(seq)
        _, tmp := m.px.Status(seq)
        p_, _ := tmp.(Proposal)
        if p.equalsTo(&p_) {
            return seq
        }
    }
}

// Must acquire m.lock
func (m *KVPaxosMap) doAStep() {
    seq := m.done
    if decided, _ := m.px.Status(seq); !decided {
        m.px.Start(seq, Proposal{"Null", "", ""})
        m.waitDecide(seq)
    }

    _, tmp := m.px.Status(seq)
    p, _ := tmp.(Proposal)
    if p.Type == "Put" {
        if _, ok := m.data[p.Key]; !ok {
            m.data[p.Key] = p.Value
        }
    } else if p.Type == "Update" {
        if _, ok := m.data[p.Key]; ok {
            m.data[p.Key] = p.Value
        }
    } else if p.Type == "Delete" {
        delete(m.data, p.Key)
    }
    m.px.Done(m.done)
    m.done++
}

//--------------------------------------------------------------//

func (m *KVPaxosMap) Put(key string, value string) bool {
    m.lock.Lock()
    defer m.lock.Unlock()

    if m.dead {
        return false
    }

    seq := m.submitProposal(Proposal{"Put", key, value})
    for m.done < seq {
        m.doAStep()
    }

    result := false
    if _, ok := m.data[key]; !ok {
        m.data[key] = value
        result = true
    }
    m.px.Done(m.done)
    m.done++
    return result
}

func (m *KVPaxosMap) Get(key string) (bool, string) {
    m.lock.Lock()
    defer m.lock.Unlock()

    if m.dead {
        return false, ""
    }

    seq := m.submitProposal(Proposal{"Get", key, ""})
    for m.done < seq {
        m.doAStep()
    }

    v, ok := m.data[key]
    m.px.Done(m.done)
    m.done++
    return ok, v
}

func (m *KVPaxosMap) Update(key string, value string) bool {
    m.lock.Lock()
    defer m.lock.Unlock()

    if m.dead {
        return false
    }

    seq := m.submitProposal(Proposal{"Update", key, value})
    for m.done < seq {
        m.doAStep()
    }

    result := false
    if _, ok := m.data[key]; ok {
        m.data[key] = value
        result = true
    }
    m.px.Done(m.done)
    m.done++
    return result
}

func (m *KVPaxosMap) Delete(key string) (bool, string) {
    m.lock.Lock()
    defer m.lock.Unlock()

    if m.dead {
        return false, ""
    }

    seq := m.submitProposal(Proposal{"Delete", key, ""})
    for m.done < seq {
        m.doAStep()
    }

    v, ok := m.data[key]
    delete(m.data, key)
    m.px.Done(m.done)
    m.done++
    return ok, v
}

func (m *KVPaxosMap) Count() int {
    m.lock.Lock()
    defer m.lock.Unlock()

    if m.dead {
        return -1
    }

    seq := m.submitProposal(Proposal{"Count", "", ""})
    for m.done < seq {
        m.doAStep()
    }

    m.px.Done(m.done)
    m.done++
    return len(m.data)
}

func (m *KVPaxosMap) Dump() string {
    m.lock.Lock()
    defer m.lock.Unlock()

    if m.dead {
        return ""
    }

    seq := m.submitProposal(Proposal{"Dump", "", ""})
    for m.done < seq {
        m.doAStep()
    }

    cnt := 0
    size := len(m.data)
    info := "["
    for key, value := range m.data {
        info += `["` + key + `","` + value + `"]`
        cnt++
        if cnt < size {
            info += ","
        }
    }
    info += "]"

    m.px.Done(m.done)
    m.done++
    return info
}

func (m *KVPaxosMap) Shutdown() {
    m.lock.Lock()
    defer m.lock.Unlock()

    m.dead = true
    m.px.Kill()
}