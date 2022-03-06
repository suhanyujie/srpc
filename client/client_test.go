package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/suhanyujie/srpc/core"
	"github.com/suhanyujie/srpc/srpc"
)

func startServer(serverAddrChan chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("err: %v", err)
		return
	}
	log.Println("start rpc server on ", l.Addr())
	serverAddrChan <- l.Addr().String()
	server := srpc.DefaultServer
	server.Accept(l)
}

func TestSend1(t *testing.T) {
	serverAddrChan := make(chan string)
	go startServer(serverAddrChan)
	serverAddr := <-serverAddrChan
	// 客户端发起连接
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Printf("[main] err: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	time.Sleep(2 * time.Second)
	// assembly option of header
	json.NewEncoder(conn).Encode(srpc.DefaultOption)
	coder := core.NewGobCode(conn)
	for i := 0; i < 5; i++ {
		h := &core.Header{
			Method:  "Foo.Sum",
			TraceId: "traceId-" + strconv.Itoa(i),
		}
		coder.Write(h, fmt.Sprintf("[main] srpc client req: %s", h.TraceId))
		// 从连接中读取服务端返回的 header
		coder.ReadHeader(h)
		var reply string
		// 从连接中读取服务端返回的 body
		coder.ReadBody(&reply)
		log.Printf("[main] reply: %s", reply)
	}
}

func TestSend2(t *testing.T) {
	log.SetFlags(0)
	addr := make(chan string)

	// 启动服务器
	go startServer(addr)

	// 客户端发起连接
	client, _ := Dial("tcp", <-addr)
	defer func() {
		_ = client.Close()
	}()

	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("[TestSend2] srpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatalf("[TestSend2]call Foo.Sum err: %v", err)
			}
			log.Printf("[TestSend2] reply: %v", reply)
		}(i)
	}
	wg.Wait()
}
