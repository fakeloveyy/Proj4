一、Paxos数据结构

①首先设计单个Paxos实例运行所需要的临时变量的数据结构：

PaxosData {
	Lock sync.Mutex
	Np int		// Acceptor见过的最大编号
	Na int		// Acceptor接受的最大编号
	Va interface{}	// 若已经decide，则va为decide的值
}

    由于一个Paxos实例可能有不同的go routine在访问这些数据，因此每个数据我们使用Go语言提供的sync.atomic.Value来存储，这是一个类似atomic register的部件。由于有些操作需要对整个PaxosData的数据做原子处理，因此，我们提供一个Lock来方便对整个PaxosData上锁。

    PaxosData的默认值是：Np:-1, Na:-1, Va:nil。



②针对多个Paxos实例，我们设计一个管理它们运行内存的一个内存管理器PaxosAllocator。它是一个seq int到*PaxosData的一个map。由于可能有不同的go routine在访问这个管理器，因此我们让这个管理器带有一个互斥锁，以达到原子性。另外，考虑到Done()操作，我们设计这个内存管理器带有一个下界bound，只有满足这个下界的内存请求才会真正被处理。内存管理器所提供的接口如下：

	Create(seq int)	*PaxosData	// 创建一个编号为seq的Paxos实例的内存，若已经存在则不创建
					// 若seq <= bound则返回一个新建的默认的PaxosData
	Done(newbound int)		// 若newbound > bound则设置新下界，并且清空所有不在新的下界中的内存
					// 仅清空指针，只有在使用这些内存的Paxos实例运行结束后才会由GC真正清除



③针对Paxos的运行结果，我们设计一个管理Paxos决定结果的结果表PaxosResult。它是一个从seq到interface{}的map。由于可能有不同的go routine在访问这个结果表，因此我们让这个结果表带有一个互斥锁，以达到原子性。另外，考虑到Done()操作，我们设计这个结果表带有一个下界bound，只有满足这个下界的结果才会被记录。结果表所提供的接口如下：

	Read(seq int) (bool, interface{})
					// 当第seq个Paxos的实例存在时，返回(true,data)，否则返回(false,garbage)
	Write(seq int, interface{})	// 若seq <= bound则不执行，否则写入数据
	Done(newbound int)		// 若newbound > bound则设置新下界，并且情况所有不在新的下界中的数据

以上内容参考paxosutility.go。



二、Paxos算法

②Done()和Min()的处理：通过HandlePrepare和HandleAccept顺便传递自己的值，使得能够互相知道所有人Done()的值。

③Max()的处理：每次发现新的seq都更新一下。

④PaxosPeer全局变量：

	min []int		// Done(seq)调用的seq记录
	minlock sync.Mutex	// 凡是涉及到min的操作，如Done(),Min()，需获得锁minlock
	max int			// 已经发现的最大seq
	maxlock sync.Mutex	// 凡是涉及到max的操作，如更新max的值，Max()，需获得锁maxlock

为了防止livelock的出现，我们让所有paxos peer开始新的一轮propose之前都会sleep 500ms。

以上内容参考paxos.go。



三、KVPaxosMap

我们设计了一个用于单个服务器使用的distributed map，称为KVPaxosMap，内部使用Paxos库达到一致性。提供Put(),Get(),Delete(),Update(),Dump(),Countkey(),Shutdown()等接口。

以上内容参考kvpaxos.go。



四、server

本次实验的server与第三次实验的server架构基本相同，请参考start_server.go,stop_server.go。

需要注意的是，在调用Shutdown()之后，server不会直接关掉，而是会拒绝用户的一切请求（例如返回false和空值等等），并且将底层的paxos peer设置为dead。



五、client

client提供接口Insert,Get,Delete,Update,Countkey,Dump，只是简单的将用户的调用向各个服务器依次询问直到成若所有服务器都返回失败，则client拒绝该请求。

以上内容参考kvclient.go。