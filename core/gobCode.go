package core

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCode 编解码实现

type GobCode struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

// 确保 GobCode 实现了 CodeIf
var _ CodeIf = (*GobCode)(nil)

func (g GobCode) Close() error {
	return g.conn.Close()
}

// ReadHeader 从 buffer 中读取并解码 header
func (g GobCode) ReadHeader(header *Header) error {
	return g.dec.Decode(header)
}

// ReadBody 从 buffer 中读取并解码 body
func (g GobCode) ReadBody(body interface{}) error {
	return g.dec.Decode(body)
}

// Write 将准备好的 header 和 body 编码后写入连接
func (g GobCode) Write(header *Header, body interface{}) (err error) {
	defer func() {
		_ = g.buf.Flush()
		if err != nil {
			_ = g.Close()
			log.Printf("[Write] flush err: %v", err)
		}
	}()
	if err = g.enc.Encode(header); err != nil {
		log.Printf("[Write] srpc header Encode err: %v", err)
		return err
	}
	if err = g.enc.Encode(body); err != nil {
		log.Printf("[Write] srpc body Encode err: %v", err)
		return err
	}
	return nil
}

// NewGobCode 实例化 gob 编解码器
func NewGobCode(conn io.ReadWriteCloser) CodeIf {
	buf := bufio.NewWriter(conn)

	return &GobCode{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn), // 实例化解码器，用于从连接中读取字节并解码
		enc:  gob.NewEncoder(buf),  // 实例化编码器，用于向 buf 中写入编码后的字节
	}
}
