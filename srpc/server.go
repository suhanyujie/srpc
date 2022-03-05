package srpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/suhanyujie/srpc/core"
)

const MagicNumber = 0x1991

type Option struct {
	MagicNumber int
	CodeType    core.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodeType:    core.GobType,
}

// 声明 server 类型，其上挂载了一些方法
type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (s *Server) Accept(lisn net.Listener) {
	for {
		conn, err := lisn.Accept()
		if err != nil {
			log.Println("srpc server, accept err: ", err)
			return
		}
		go s.ServeConn(conn)
	}
}

// ServeConn 处理连接
func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	var err error
	defer func() {
		if err = conn.Close(); err != nil {
			log.Printf("[ServeConn] close err: %v", err)
			return
		}
	}()
	var opt Option
	if err = json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Printf("[ServeConn] Decode err: %v", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("[ServeConn] magicNumber err: %v", err)
		return
	}
	// 使用另一种方式解码字节流
	genCoder, ok := core.CodeFuncMap[opt.CodeType]
	if !ok {
		log.Printf("[ServeConn] get code err")
		return
	}
	s.ServeCoder(genCoder(conn))
}

var invalidRequest = struct{}{}

// ServeCoder 处理 coder（带有连接）
func (s *Server) ServeCoder(coder core.CodeIf) {
	lock := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	for {
		req, err := s.ReadRequest(coder)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			s.sendResp(coder, req.h, invalidRequest, lock)
			log.Printf("[ServeCoder] err: %v", err)
			continue
		}
		wg.Add(1)
		go s.HandleReq(coder, req, lock, wg)
	}
	coder.Close()
}

type request struct {
	h            *core.Header
	argv, replyv reflect.Value
}

func (s *Server) ReadRequestHeader(coder core.CodeIf) (*core.Header, error) {
	var (
		h   core.Header
		err error
	)
	if err = coder.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Printf("srpc server read eof")
			return &h, err
		}
		return &h, err
	}

	return &h, nil
}

func (s *Server) ReadRequest(coder core.CodeIf) (*request, error) {
	var err error
	req := &request{}
	req.h, err = s.ReadRequestHeader(coder)
	if err != nil {
		log.Printf("[ReadRequest] ReadRequestHeader err: %v", err)
		return req, err
	}

	//读取 body
	req.argv = reflect.New(reflect.TypeOf(""))
	if err := coder.ReadBody(req.argv.Interface()); err != nil {
		log.Printf("[ReadRequest] srpc server coder.ReadBody err: %v", err)
	}

	return req, nil
}

// HandleReq 处理请求
func (s *Server) HandleReq(coder core.CodeIf, req *request, lock *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	req.replyv = reflect.ValueOf(fmt.Sprintf("[HandleReq] srpc traceId: %s", req.h.TraceId))
	if err := s.sendResp(coder, req.h, req.replyv.Interface(), lock); err != nil {
		log.Printf("[HandleReq] sendResp err: %v", err)
	}
	log.Printf("[HandleReq] info...")
}

// 向客户端发送响应
func (s *Server) sendResp(coder core.CodeIf, header *core.Header, body interface{}, lock *sync.Mutex) error {
	lock.Lock()
	defer lock.Unlock()
	if err := coder.Write(header, body); err != nil {
		log.Printf("[sendResp] write err: %v", err)
		return err
	}

	return nil
}
