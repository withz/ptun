package proto

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const (
	maxPayloadSize = 8180
	headerSize     = 4
)

type header struct {
	tag    PacketTag
	length uint16
}

const (
	Empty PacketTag = 10
	Raw   PacketTag = 11
	Req   PacketTag = 12
	Resp  PacketTag = 13
	Ping  PacketTag = 14
	Pong  PacketTag = 15
)

type PacketTag uint16

type Packet struct {
	header header
	body   []byte
	buf    *[]byte
}

func (pkt *Packet) Body() []byte {
	return pkt.body
}

func (pkt *Packet) SetBody(p []byte) error {
	pkt.body = p
	if len(p) > maxPayloadSize+headerSize {
		return fmt.Errorf("packet is too long")
	}
	pkt.header.length = uint16(len(p))
	return nil
}

func (pkt *Packet) Tag() PacketTag {
	return pkt.header.tag
}

func (pkt *Packet) SetTag(pt PacketTag) {
	pkt.header.tag = pt
}

func (pkt *Packet) Release() {
	if pkt == nil || pkt.buf == nil {
		return
	}
	bytesPool.Put(pkt.buf)
}

func PackRaw(pt PacketTag, p []byte) (*Packet, error) {
	pkt := &Packet{
		header: header{
			tag: pt,
		},
		body: p,
	}
	if len(p) > maxPayloadSize {
		return nil, fmt.Errorf("packet is too long")
	}
	if len(p) > 0 {
		pkt.header.length = uint16(len(p))
		pkt.body = p
	}
	return pkt, nil
}

func PackInto(pt PacketTag, payload []byte, w io.Writer) error {
	pkt, err := PackRaw(pt, payload)
	if err != nil {
		return err
	}
	writer := bufio.NewWriterSize(w, headerSize+maxPayloadSize)
	payloads := []any{pkt.header.tag, pkt.header.length}
	for _, payload := range payloads {
		if err = binary.Write(writer, binary.BigEndian, payload); err != nil {
			return err
		}
	}
	if pkt.body == nil {
		return writer.Flush()
	}
	if err = binary.Write(writer, binary.BigEndian, pkt.body); err != nil {
		return err
	}
	return writer.Flush()
}

func Unpack(p []byte) (pkt *Packet, err error) {
	reader := bytes.NewReader(p)
	return UnpackFrom(reader)
}

func UnpackFrom(r io.Reader) (pkt *Packet, err error) {
	pbuf := bytesPool.Get().(*[]byte)
	buf := *pbuf
	h := buf[:headerSize]
	n, err := r.Read(h)
	if err != nil {
		return nil, err
	}
	if n != headerSize {
		return nil, fmt.Errorf("malformed packet")
	}
	pkt = &Packet{
		buf: pbuf,
	}
	defer func() {
		if err != nil {
			bytesPool.Put(pbuf)
		}
	}()
	reader := bytes.NewReader(h)
	if err = binary.Read(reader, binary.BigEndian, &pkt.header.tag); err != nil {
		return nil, err
	}
	if err = binary.Read(reader, binary.BigEndian, &pkt.header.length); err != nil {
		return nil, err
	}
	if pkt.header.length == 0 {
		return pkt, nil
	}
	if pkt.header.length > maxPayloadSize {
		return nil, fmt.Errorf("malformed packet")
	}

	p := buf[headerSize : headerSize+pkt.header.length]
	n, err = r.Read(p)
	if err != nil {
		return nil, err
	}
	if n != int(pkt.header.length) {
		return nil, fmt.Errorf("malformed packet")
	}
	pkt.body = p
	return pkt, nil
}

var (
	bytesPool *sync.Pool
)

func init() {
	bytesPool = &sync.Pool{
		New: func() any {
			p := make([]byte, headerSize+maxPayloadSize)
			return &p
		},
	}
}
