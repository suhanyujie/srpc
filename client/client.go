package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
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

func main() {
	serverAddrChan := make(chan string)
	go startServer(serverAddrChan)
	serverAddr := <-serverAddrChan
	// 客户端发起连接
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Printf("[main] err: %v", err)
	}
	defer func() {
		conn.Close()
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
		coder.Write(h, fmt.Sprintf("[main] srpc client req: %d", h.TraceId))
		// 从连接中读取服务端返回的 header
		coder.ReadHeader(h)
		var reply string
		// 从连接中读取服务端返回的 body
		coder.ReadBody(&reply)
		log.Printf("[main] reply: %s", reply)
	}
}
