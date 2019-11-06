package plcconnector

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func handleUDPConnection(conn *net.UDPConn) {

}

func (p *PLC) handleUDPRequest(conn *net.UDPConn, dt []byte, n int, addr *net.UDPAddr) {
	r := req{}
	r.p = p
	r.readBuf = bufio.NewReader(bytes.NewReader(dt))
	r.writeBuf = new(bytes.Buffer)

	err := r.read(&r.encHead)
	if err != nil {
		return
	}

	switch r.encHead.Command {
	case ecListIdentity:
		if r.eipListIdentity() != nil {
			return
		}

	case ecListServices:
		if r.eipListServices() != nil {
			return
		}

	case ecListInterfaces:
		r.write(uint16(0)) // ItemCount

	default:
		p.debug("unknown command:", r.encHead.Command)

		data := make([]uint8, r.encHead.Length)
		err = r.read(&data)
		if err != nil {
			return
		}
		r.encHead.Status = eipInvalid

		r.write(data)
	}

	err = conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err != nil {
		fmt.Println(err)
		return
	}

	r.encHead.Length = uint16(r.writeBuf.Len())
	var buf bytes.Buffer

	err = binary.Write(&buf, binary.LittleEndian, r.encHead)
	if err != nil {
		fmt.Println(err)
		return
	}
	buf.Write(r.writeBuf.Bytes())

	_, err = conn.WriteToUDP(buf.Bytes(), addr)
	if err != nil {
		fmt.Println(err)
	}
}

func (p *PLC) serveUDP(host string) error {
	udpAddr, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		buffer := make([]byte, 0x8000)

		conn.SetDeadline(time.Now().Add(time.Second))

		n, addr, err := conn.ReadFromUDP(buffer)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			p.closeMut.RLock()
			endP := p.closeI
			p.closeMut.RUnlock()
			if endP {
				break
			}
		} else if err != nil {
			return err
		} else {
			go p.handleUDPRequest(conn, buffer, n, addr)
		}
	}
	return nil
}
