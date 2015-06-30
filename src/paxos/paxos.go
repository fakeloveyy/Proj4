package paxos

import (
    "paxos/paxosutility"

    "container/list"
    "net"
    "net/rpc"
    "log"
    "sync"
    "syscall"
    "fmt"
    "math/rand"
    "time"
)

type Paxos struct {
    l net.Listener
    dead bool
    unreliable bool
    rpcCount int
    peers []string
    me int // index into peers[]

    total int
    min []int
    minlock sync.Mutex
    max int
    maxlock sync.Mutex
    alloc *paxosutility.PaxosAllocator
    result *paxosutility.PaxosResult
}

//--------------------------------------------------//

// Maintain min and free memory

func __min(arr []int) int {
    result := arr[0]
    for i := 1; i < len(arr); i++ {
        if arr[i] < result {
            result = arr[i]
        }
    }
    return result
}

func (px *Paxos) refreshMin(seq int, index int) {
    px.minlock.Lock()
    defer px.minlock.Unlock()

    if seq <= px.min[index] {
        return
    }
    px.min[index] = seq
    newbound := __min(px.min)
    px.alloc.Done(newbound)
    px.result.Done(newbound)
}

func (px *Paxos) getMin(index int) int {
    px.minlock.Lock()
    defer px.minlock.Unlock()

    return px.min[index]
}

func (px *Paxos) Min() int {
    px.minlock.Lock()
    defer px.minlock.Unlock()

    return __min(px.min) + 1
}

func (px *Paxos) Done(seq int) {
    px.refreshMin(seq, px.me)
}

//--------------------------------------------------//

// Maintain max

func (px *Paxos) refreshMax(seq int) {
    px.maxlock.Lock()
    defer px.maxlock.Unlock()

    if seq > px.max {
        px.max = seq
    }
}

func (px *Paxos) Max() int {
    px.maxlock.Lock()
    defer px.maxlock.Unlock()

    return px.max
}

//--------------------------------------------------//

// Syscall and constructor

func call(srv string, name string, args interface{}, reply interface{}) bool {
    c, err := rpc.Dial("tcp", srv)
    if err != nil {
        err1 := err.(*net.OpError)
        if err1.Err != syscall.ENOENT && err1.Err != syscall.ECONNREFUSED {
            fmt.Printf("paxos Dial() failed: %v\n", err1)
        }
        return false
    }
    defer c.Close()

    err = c.Call(name, args, reply)
    if err == nil {
        return true
    }

    //fmt.Println(err)
    return false
}

func (px *Paxos) Kill() {
    px.dead = true
    if px.l != nil {
        px.l.Close()
    }
}

func Make(peers []string, me int, rpcs *rpc.Server) *Paxos {
    px := &Paxos{}
    px.peers = peers
    px.me = me

    // Your initialization code here.
    px.total = len(peers)
    px.min = make([]int, px.total)
    for i := 0; i < px.total; i++ {
        px.min[i] = -1
    }
    px.minlock = sync.Mutex{}
    px.max = -1
    px.maxlock = sync.Mutex{}
    px.alloc = paxosutility.NewPaxosAllocator()
    px.result = paxosutility.NewPaxosResult()

    if rpcs != nil {
        // caller will create socket &c
        rpcs.Register(px)
    } else {
        rpcs = rpc.NewServer()
        rpcs.Register(px)
        // prepare to receive connections from clients.
        // change "unix" to "tcp" to use over a network.
        // os.Remove(peers[me]) // only needed for "unix"
        l, e := net.Listen("tcp", peers[me]);
        if e != nil {
            log.Fatal("listen error: ", e);
        }
        px.l = l

        // please do not change any of the following code,
        // or do anything to subvert it.

        // create a thread to accept RPC connections
        go func() {
            for px.dead == false {
                conn, err := px.l.Accept()
                if err == nil && px.dead == false {
                    if px.unreliable && (rand.Int63() % 1000) < 100 {
                        // discard the request.
                        conn.Close()
                    } else if px.unreliable && (rand.Int63() % 1000) < 200 {
                        // process the request but force discard of reply.
                        c1 := conn.(*net.UnixConn)
                        f, _ := c1.File()
                        err := syscall.Shutdown(int(f.Fd()), syscall.SHUT_WR)
                        if err != nil {
                            fmt.Printf("shutdown: %v\n", err)
                        }
                        px.rpcCount++
                        go rpcs.ServeConn(conn)
                    } else {
                        px.rpcCount++
                        go rpcs.ServeConn(conn)
                    }
                } else if err == nil {
                    conn.Close()
                }
                if err != nil && px.dead == false {
                    fmt.Printf("Paxos(%v) accept: %v\n", me, err.Error())
                }
            }
        }()
    }

    return px
}

//--------------------------------------------------//

// HandlePrepare RPC

type PrepareArgs struct {
    Seq int
    N int
    Doneseq int
    Index int
}

type PrepareReplys struct {
    Doneseq int
    Decided bool
    Decidedval interface{}
    Ok bool
    Na int
    Va interface{}
}

func (px *Paxos) HandlePrepare(args PrepareArgs, replys *PrepareReplys) error {
    if px.dead {
        return nil
    }

    px.refreshMax(args.Seq)
    px.refreshMin(args.Doneseq, args.Index)

    replys.Doneseq = px.getMin(px.me)

    if exist, val := px.result.Read(args.Seq); exist {
        replys.Decided = true
        replys.Decidedval = val
        return nil
    }
    replys.Decided = false

    data := px.alloc.Create(args.Seq)
    data.Lock.Lock()
    defer data.Lock.Unlock()

    if Np, _ := data.Np.Load().(int); args.N > Np {
        data.Np.Store(args.N)
        replys.Ok = true
        replys.Na, _ = data.Na.Load().(int)
        replys.Va = data.Va.Load()
    } else {
        replys.Ok = false
    }
    return nil
}

//--------------------------------------------------//

type AcceptArgs struct {
    Seq int
    N int
    V interface{}
    Doneseq int
    Index int
}

type AcceptReplys struct {
    Doneseq int
    Decided bool
    Decidedval interface{}
    Ok bool
}

func (px *Paxos) HandleAccept(args AcceptArgs, replys *AcceptReplys) error {
    if px.dead {
        return nil
    }

    px.refreshMax(args.Seq)
    px.refreshMin(args.Doneseq, args.Index)

    replys.Doneseq = px.getMin(px.me)

    if exist, val := px.result.Read(args.Seq); exist {
        replys.Decided = true
        replys.Decidedval = val
        return nil
    }
    replys.Decided = false

    data := px.alloc.Create(args.Seq)
    data.Lock.Lock()
    defer data.Lock.Unlock()

    if Np, _ := data.Np.Load().(int); args.N >= Np {
        data.Np.Store(args.N)
        data.Na.Store(args.N)
        data.Va.Store(args.V)
        replys.Ok = true
    } else {
        replys.Ok = false
    }
    return nil
}

//--------------------------------------------------//

type DecideArgs struct {
    Seq int
    V interface{}
    Doneseq int
    Index int
}

type DecideReplys struct {
    Doneseq int
}

func (px *Paxos) HandleDecide(args DecideArgs, replys *DecideReplys) error {
    if px.dead {
        return nil
    }

    px.refreshMax(args.Seq)
    px.refreshMin(args.Doneseq, args.Index)

    replys.Doneseq = px.getMin(px.me)

    px.result.Write(args.Seq, args.V)
    return nil
}

//--------------------------------------------------//

func (px *Paxos) propose(seq int, v interface{}) {
    // Refresh the max
    px.refreshMax(seq)

    // Allocate memory for proposer instance
    data := px.alloc.Create(seq)

    for ; !px.dead; time.Sleep(500 * time.Millisecond) {
        // If the instance is already abandoned, then break
        if seq < px.Min() {
            return
        }

        // If the instance already decides, then break
        if exist, _ := px.result.Read(seq); exist {
            break
        }

        // Find a large enough n to be a new proposal round
        // n = me (mod total)
        n, _ := data.Np.Load().(int)
        {
            tmp := (n / px.total) * px.total + px.me
            for tmp <= n {
                tmp += px.total
            }
            n = tmp
        }

        // Send prepare(n) to all servers including itself
        success := 0
        na := -1
        va := v
        for i := 0; i < px.total; i++ {
            args := PrepareArgs{seq, n, px.getMin(px.me), px.me}
            replys := &PrepareReplys{}
            if px.dead {
                return
            }
            if i != px.me {
                ok := call(px.peers[i], "Paxos.HandlePrepare", args, replys)
                if !ok {
                    // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "prepare_ack from", i, "failed")
                    continue
                }
                // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "prepare_ack from", i, replys.Ok)
            } else {
                px.HandlePrepare(args, replys)
                // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "prepare_ack from", i, replys.Ok)
            }

            // Update the Done seq
            px.refreshMin(replys.Doneseq, i)
            if seq < px.Min() {
                return
            }

            // If the instance seq is already decided, then just learns it and returns
            if replys.Decided {
                px.result.Write(seq, replys.Decidedval)
                go px.broadcast(seq, replys.Decidedval)
                return
            }

            if replys.Ok {
                success++
                if replys.Na > na {
                    na = replys.Na
                    va = replys.Va
                }
            }
        }

        // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "prepare success number", success)
        // If not prepare_ok from majority, then restart
        if success + success <= px.total {
            continue
        }

        // Send accept(n, va) to all servers including itself
        success = 0
        for i := 0; i < px.total; i++ {
            if px.dead {
                return
            }

            args := AcceptArgs{seq, n, va, px.getMin(px.me), px.me}
            replys := &AcceptReplys{}

            if i != px.me {
                ok := call(px.peers[i], "Paxos.HandleAccept", args, replys)
                if !ok {
                    // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "accept_ack from", i, "failed")
                    continue
                }
                // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "accept_ack from", i, replys.Ok)
            } else {
                px.HandleAccept(args, replys)
                // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "accept_ack from", i, replys.Ok)
            }

            // Update the Done seq
            px.refreshMin(replys.Doneseq, i)

            // If the instance seq is already decided, then just learns it and returns
            if replys.Decided {
                px.result.Write(seq, replys.Decidedval)
                go px.broadcast(seq, replys.Decidedval)
                return
            }

            if replys.Ok {
                success++
            }
        }

        // After update several doneseq, check whether current seq is done or not
        if seq < px.Min() {
            return
        }

        // fmt.Println("Paxos", seq, "Peer", px.me, "Round", n, "accept success number", success)
        // If not prepare_ok from majority, then restart
        if success + success <= px.total {
            continue
        }

        // Decide va
        px.result.Write(seq, va)

        // Send decided(va) to all servers exclude itself
        go px.broadcast(seq, va)
    }
}

//--------------------------------------------------//

func (px *Paxos) broadcast(seq int, v interface{}) {
    l := list.New()
    for i := 0; i < px.total; i++ {
        if i != px.me {
            l.PushBack(i)
        }
    }

    for seq >= px.Min() && l.Len() > 0 {
        for e := l.Front(); e != nil; {
            if px.dead {
                return
            }
            i, _ := e.Value.(int)
            args := DecideArgs{seq, v, px.getMin(px.me), px.me}
            replys := &DecideReplys{}
            ok := call(px.peers[i], "Paxos.HandleDecide", args, replys)
            if !ok {
                e = e.Next()
            } else {
                px.refreshMin(replys.Doneseq, i)
                tmp := e.Next()
                l.Remove(e)
                e = tmp
            }
        }
    }
}

//--------------------------------------------------//

func (px *Paxos) Start(seq int, v interface{}) {
    if px.dead {
        return
    }
    go px.propose(seq, v)
}

func (px *Paxos) Status(seq int) (bool, interface{}) {
    if px.dead {
        return false, nil
    }
    return px.result.Read(seq)
}
