һ��Paxos���ݽṹ

��������Ƶ���Paxosʵ����������Ҫ����ʱ���������ݽṹ��

PaxosData {
	Lock sync.Mutex
	Np int		// Acceptor�����������
	Na int		// Acceptor���ܵ������
	Va interface{}	// ���Ѿ�decide����vaΪdecide��ֵ
}

    ����һ��Paxosʵ�������в�ͬ��go routine�ڷ�����Щ���ݣ����ÿ����������ʹ��Go�����ṩ��sync.atomic.Value���洢������һ������atomic register�Ĳ�����������Щ������Ҫ������PaxosData��������ԭ�Ӵ�����ˣ������ṩһ��Lock�����������PaxosData������

    PaxosData��Ĭ��ֵ�ǣ�Np:-1, Na:-1, Va:nil��



����Զ��Paxosʵ�����������һ���������������ڴ��һ���ڴ������PaxosAllocator������һ��seq int��*PaxosData��һ��map�����ڿ����в�ͬ��go routine�ڷ������������������������������������һ�����������Դﵽԭ���ԡ����⣬���ǵ�Done()�����������������ڴ����������һ���½�bound��ֻ����������½���ڴ�����Ż������������ڴ���������ṩ�Ľӿ����£�

	Create(seq int)	*PaxosData	// ����һ�����Ϊseq��Paxosʵ�����ڴ棬���Ѿ������򲻴���
					// ��seq <= bound�򷵻�һ���½���Ĭ�ϵ�PaxosData
	Done(newbound int)		// ��newbound > bound���������½磬����������в����µ��½��е��ڴ�
					// �����ָ�룬ֻ����ʹ����Щ�ڴ��Paxosʵ�����н�����Ż���GC�������



�����Paxos�����н�����������һ������Paxos��������Ľ����PaxosResult������һ����seq��interface{}��map�����ڿ����в�ͬ��go routine�ڷ�������������������������������һ�����������Դﵽԭ���ԡ����⣬���ǵ�Done()������������������������һ���½�bound��ֻ����������½�Ľ���Żᱻ��¼����������ṩ�Ľӿ����£�

	Read(seq int) (bool, interface{})
					// ����seq��Paxos��ʵ������ʱ������(true,data)�����򷵻�(false,garbage)
	Write(seq int, interface{})	// ��seq <= bound��ִ�У�����д������
	Done(newbound int)		// ��newbound > bound���������½磬����������в����µ��½��е�����

�������ݲο�paxosutility.go��



����Paxos�㷨

��Done()��Min()�Ĵ���ͨ��HandlePrepare��HandleAccept˳�㴫���Լ���ֵ��ʹ���ܹ�����֪��������Done()��ֵ��

��Max()�Ĵ���ÿ�η����µ�seq������һ�¡�

��PaxosPeerȫ�ֱ�����

	min []int		// Done(seq)���õ�seq��¼
	minlock sync.Mutex	// �����漰��min�Ĳ�������Done(),Min()��������minlock
	max int			// �Ѿ����ֵ����seq
	maxlock sync.Mutex	// �����漰��max�Ĳ����������max��ֵ��Max()��������maxlock

Ϊ�˷�ֹlivelock�ĳ��֣�����������paxos peer��ʼ�µ�һ��propose֮ǰ����sleep 500ms��

�������ݲο�paxos.go��



����KVPaxosMap

���������һ�����ڵ���������ʹ�õ�distributed map����ΪKVPaxosMap���ڲ�ʹ��Paxos��ﵽһ���ԡ��ṩPut(),Get(),Delete(),Update(),Dump(),Countkey(),Shutdown()�Ƚӿڡ�

�������ݲο�kvpaxos.go��



�ġ�server

����ʵ���server�������ʵ���server�ܹ�������ͬ����ο�start_server.go,stop_server.go��

��Ҫע����ǣ��ڵ���Shutdown()֮��server����ֱ�ӹص������ǻ�ܾ��û���һ���������緵��false�Ϳ�ֵ�ȵȣ������ҽ��ײ��paxos peer����Ϊdead��



�塢client

client�ṩ�ӿ�Insert,Get,Delete,Update,Countkey,Dump��ֻ�Ǽ򵥵Ľ��û��ĵ������������������ѯ��ֱ���������з�����������ʧ�ܣ���client�ܾ�������

�������ݲο�kvclient.go��