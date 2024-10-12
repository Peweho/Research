package index_service

import (
	"Research/ServiceHub"
	"Research/types/doc"
	"Research/types/index"
	"Research/types/raft"
	"Research/types/term_query"
	"Research/util"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Sentinel struct {
	ServiceHub.IServiceHub          // 从Hub上获取IndexServiceWorker集合。可能是直接访问ServiceHub，也可能是走代理
	connPool               sync.Map // 与各个IndexServiceWorker建立的连接。把连接缓存起来，避免每次都重建连接
}

// 创建哨兵
func NewSentinel(hub ServiceHub.IServiceHub) *Sentinel {
	return &Sentinel{
		IServiceHub: hub,
		connPool:    sync.Map{},
	}
}

func (s *Sentinel) GetGrpcConn(endpoint *ServiceHub.EndPoint) *grpc.ClientConn {
	//1、判断是否存在连接
	if value, ok := s.connPool.Load(endpoint.SelfAddr); ok {
		conn := value.(*grpc.ClientConn)
		//1.1、判断连接是否可用
		if conn.GetState() == connectivity.TransientFailure || conn.GetState() == connectivity.Shutdown {
			// 连接不可用，关闭和删除连接
			util.Log.Printf("connection status to endpoint %s is %s", endpoint, conn.GetState())
			conn.Close()
			s.connPool.Delete(endpoint)
		} else {
			//可用，直接返回
			return conn
		}
	}
	// 2、创建连接
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		endpoint.SelfAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // tls安全链接，传递了一个空证书
		grpc.WithBlock()) // Dial是异步链接，设置Block会变为同步，（异步状态下ctx超时不会生效）
	if err != nil {
		util.Log.Printf("dial %s failed: %s", endpoint, err)
		return nil
	}

	util.Log.Printf("connect to grpc server %s", endpoint.SelfAddr)
	s.connPool.Store(endpoint.SelfAddr, conn)
	return conn
}

// 将文档添加到一个节点上
func (s *Sentinel) AddDoc(document *doc.Document) (int, error) {
	//1、获取服务器
	endpoint := s.GetServiceEndpoint(INDEX_SERVICE)
	if endpoint == nil {
		return 0, fmt.Errorf("there is no alive index worker")
	}
	//2、获取grpc连接
	conn := s.GetGrpcConn(endpoint)
	if conn == nil {
		return 0, fmt.Errorf("connect to worker %s failed", endpoint)
	}
	////3、创建客户端
	//client := index.NewIndexServiceClient(conn)
	////4、发送grpc请求
	//affected, err := client.AddDoc(context.Background(), document)
	resp, err := RaftClintRequest(conn, 1, document, nil, nil, nil, nil, nil)
	affected := resp.(*index.AffectedCount)
	if err != nil {
		return 0, err
	}
	util.Log.Printf("add %d doc to worker %s", affected.Count, endpoint)
	return int(affected.Count), nil

}

// 从集群中删除一个文档，需要遍历集群所有节点，异步完成，
func (s *Sentinel) DeleteDoc(docId string) int {
	//1、获取服务器
	endpoints := s.GetServiceEndpoints(INDEX_SERVICE)
	if len(endpoints) == 0 {
		return 0
	}

	var res int32 //统计结果
	wg := sync.WaitGroup{}

	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint *ServiceHub.EndPoint) {
			defer wg.Done()
			//2、获取grpc连接,并发发送删除请求
			conn := s.GetGrpcConn(endpoint)
			if conn == nil {
				return
			}
			////3、创建客户端
			//client := index.NewIndexServiceClient(conn)
			////4、发送grpc请求
			//affected, err := client.DeleteDoc(context.Background(), &index.DocId{DocId: docId})
			resp, err := RaftClintRequest(conn, 2, nil, &index.DocId{DocId: docId}, nil, nil, nil, nil)
			affected := resp.(*index.AffectedCount)
			if err != nil {
				util.Log.Printf("delete doc %s from worker %s failed: %s", docId, endpoint, err)
			} else if affected.Count > 0 {
				atomic.StoreInt32(&res, 1)
				util.Log.Printf("delete %d doc from worker %s", affected.Count, endpoint)
			}
		}(endpoint)
	}
	wg.Wait()
	return int(res)
}

// 从集群上查找，合并查询的结果
func (s *Sentinel) Search(query *term_query.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []*doc.Document {
	//1、获取服务器
	endpoints := s.GetServiceEndpoints(INDEX_SERVICE)
	if len(endpoints) == 0 {
		return nil
	}

	res := make([]*doc.Document, 100)         //统计结果
	docsChan := make(chan *doc.Document, 100) // 存储并发结果
	wg := sync.WaitGroup{}

	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint *ServiceHub.EndPoint) {
			defer wg.Done()
			//2、获取grpc连接,并发发送删除请求
			conn := s.GetGrpcConn(endpoint)
			if conn == nil {
				return
			}
			////3、创建客户端
			//client := index.NewIndexServiceClient(conn)
			////4、发送grpc请求
			//affected, err := client.Search(context.Background(), &index.SearchRequest{
			//	OnFlag:  onFlag,
			//	OffFlag: offFlag,
			//	OrFlags: orFlags,
			//})
			resp, err := RaftClintRequest(conn, 3, nil, nil, query, onFlag, offFlag, orFlags)
			affected := resp.(*index.SearchResult)
			if err != nil {
				util.Log.Printf("search from cluster failed: %s", err)
			} else {
				if len(affected.Results) > 0 {
					//5、合并结果
					util.Log.Printf("search %d doc from worker %s", len(affected.Results), endpoint)
					for _, document := range affected.Results {
						docsChan <- document
					}
				}
			}
		}(endpoint)
	}

	signal := make(chan struct{})
	go func() {
		for {
			val, ok := <-docsChan
			if !ok {
				break
			}
			res = append(res, val)
		}
		//读取完毕之后会等待接收主协程信号
		<-signal
	}()

	wg.Wait()
	close(docsChan)      // 写入完毕，关闭管道，可以继续读
	signal <- struct{}{} // 发送信号，退出协程（不接受信号就阻塞）
	return res
}

func (s *Sentinel) Count() int {
	//1、获取服务器
	endpoints := s.GetServiceEndpoints(INDEX_SERVICE)
	if len(endpoints) == 0 {
		return 0
	}

	var res int32 //统计结果
	wg := sync.WaitGroup{}

	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint *ServiceHub.EndPoint) {
			defer wg.Done()
			//2、获取grpc连接,并发发送删除请求
			conn := s.GetGrpcConn(endpoint)
			if conn == nil {
				return
			}
			//3、创建客户端
			client := index.NewIndexServiceClient(conn)
			//4、发送grpc请求
			affected, err := client.Count(context.Background(), &index.CountRequest{})
			if err != nil {
				util.Log.Printf("get doc count from worker %s failed: %s", endpoint, err)
			} else {
				atomic.StoreInt32(&res, affected.Count)
				util.Log.Printf("worker %s have %d documents", endpoint, affected.Count)
			}
		}(endpoint)
	}
	wg.Wait()
	return int(res)
}

func (s *Sentinel) Close() error {
	s.connPool.Range(func(key, value any) bool {
		conn := value.(*grpc.ClientConn)
		_ = conn.Close()
		return true
	})
	s.Close()
	return nil
}

func RaftClintRequest(conn *grpc.ClientConn, reqType int64, document *doc.Document, docId *index.DocId, query *term_query.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) (any, error) {
	client := raft.NewResearchClientServiceClient(conn)

	var args []string
	switch reqType {
	case 1:
		args = make([]string, 1)
		str, err := Serialize[*doc.Document](document)
		if err != nil {
			return nil, err
		}
		args[0] = str
		research, err := client.Research(context.Background(), &raft.ResearchRequest{
			ReqType: &reqType,
			Args:    args,
		})
		if err != nil || len(research.Values) == 0 {
			return nil, err
		}
		target, err := DeserializeAffectedCount(research.Values[0])
		return target, err
	case 2:
		args = make([]string, 1)
		str, err := Serialize[*index.DocId](docId)
		if err != nil {
			return nil, err
		}
		args[0] = str
		research, err := client.Research(context.Background(), &raft.ResearchRequest{
			ReqType: &reqType,
			Args:    args,
		})
		target, err := DeserializeAffectedCount(research.Values[0])
		return target, err
	case 3:
		args = make([]string, 4)
		str1, err := Serialize[*term_query.TermQuery](query)
		if err != nil {
			return nil, err
		}
		args[0] = str1
		str2, err := Serialize[*util.Bitmap](onFlag)
		if err != nil {
			return nil, err
		}
		args[1] = str2
		str3, err := Serialize[*util.Bitmap](offFlag)
		if err != nil {
			return nil, err
		}
		args[2] = str3
		str4, err := Serialize[[]*util.Bitmap](orFlags)
		if err != nil {
			return nil, err
		}
		args[3] = str4
		research, err := client.Research(context.Background(), &raft.ResearchRequest{
			ReqType: &reqType,
			Args:    args,
		})
		target, err := DeserializeSearchResult(research.Values)
		if err != nil {
			return nil, err
		}
		return target, err
	}
	return nil, errors.New("unknown request type")
}

func DeserializeAffectedCount(s string) (target *index.AffectedCount, err error) {
	atoi, err := strconv.Atoi(s)
	if err != nil {
		return
	}
	target = &index.AffectedCount{
		Count: int32(atoi),
	}
	return
}

func DeserializeSearchResult(s []string) (target *index.SearchResult, err error) {
	reader := bytes.NewReader([]byte{})
	target.Results = make([]*doc.Document, len(s))
	for i := 0; i < len(s); i++ {
		reader.Reset([]byte(s[i]))
		decoder := gob.NewDecoder(reader)
		err = decoder.Decode(target.Results[i])
		if err != nil {
			return
		}
	}
	return
}

func Serialize[T *doc.Document | *index.DocId | *term_query.TermQuery | *util.Bitmap | []*util.Bitmap](input T) (str string, err error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err = encoder.Encode(input)
	if err != nil {
		return
	}
	str = buffer.String()
	return
}
