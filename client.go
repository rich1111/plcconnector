package plcconnector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Client .
type Client struct{}

// Connect .
func Connect(host string) Client {
	var c Client

	return c
}

// Identity .
type Identity struct {
	Addr         string
	VendorID     int    // UINT
	DeviceType   int    // UINT
	ProductCode  int    // UINT
	Revision     string // UINT
	Status       int    // UINT
	SerialNumber uint   // UDINT
	Name         string // SHORTSTRING
	State        int    // USINT
}

// Discover .
func Discover(bc string) ([]Identity, error) {
	var (
		buf bytes.Buffer
		ids []Identity
	)

	bwrite(&buf, encapsulationHeader{
		Command: ecListIdentity,
	})

	laddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:44818")
	if err != nil {
		return nil, err
	}

	raddr, err := net.ResolveUDPAddr("udp4", bc)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second))

	_, err = conn.WriteToUDP(buf.Bytes(), raddr)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 0x8000)
	for {
		ln, err := conn.Read(buffer)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return ids, nil
		} else if err != nil {
			return nil, err
		}
		if ln >= 62 {
			var id Identity
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
			id.Addr = net.JoinHostPort(net.IPv4(byte(data.SocketAddr), byte(data.SocketAddr>>8), byte(data.SocketAddr>>16), byte(data.SocketAddr>>24)).String(),
				strconv.Itoa(int(htons(data.SocketPort))))
			id.VendorID = int(attrs[0]) + int(attrs[1])<<8
			id.DeviceType = int(attrs[2]) + int(attrs[3])<<8
			id.ProductCode = int(attrs[4]) + int(attrs[5])<<8
			id.Revision = fmt.Sprintf("%d.%d", attrs[6], attrs[7])
			id.Status = int(attrs[8]) + int(attrs[9])<<8
			id.SerialNumber = uint(attrs[10]) + uint(attrs[11])<<8 + uint(attrs[12])<<16 + uint(attrs[13])<<24
			id.Name = string(attrs[15 : 15+attrs[14]])
			id.State = int(attrs[len(attrs)-1])
			ids = append(ids, id)
		}
	}
}
