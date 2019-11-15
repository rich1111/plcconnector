package plcconnector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Client .
type Client struct{}

// Connect .
func Connect(host string) Client {
	var c Client

	return c
}

// Discover .
func Discover(bc string) error {
	var buf bytes.Buffer

	bwrite(&buf, encapsulationHeader{
		Command: ecListIdentity,
	})

	laddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:44818")
	if err != nil {
		return err
	}

	raddr, err := net.ResolveUDPAddr("udp4", bc)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.WriteToUDP(buf.Bytes(), raddr)
	if err != nil {
		return err
	}

	conn.SetDeadline(time.Now().Add(time.Second))

	buffer := make([]byte, 0x8000)
	for {
		ln, err := conn.Read(buffer)
		if err != nil {
			return err
		}
		if ln > 24 {
			rd := bytes.NewReader(buffer)
			var (
				head  encapsulationHeader
				count uint16
				typ   itemType
				data  listIdentityData
			)
			bread(rd, &head)
			bread(rd, &count)
			bread(rd, &typ)
			bread(rd, &data)
			attrs := make([]byte, int(typ.Length)-binary.Size(data))
			bread(rd, &attrs)
			fmt.Println(head, count, typ, data, attrs)
			break
		}
	}

	return nil
}
