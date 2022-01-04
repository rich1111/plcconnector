package plcconnector

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Client .
type Client struct {
	c       net.Conn
	rd      *bufio.Reader
	wr      *bytes.Buffer
	handle  uint32
	context uint64

	Timeout uint16
}

func (c *Client) read(data interface{}) error {
	err := binary.Read(c.rd, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (c *Client) write(data interface{}) {
	err := binary.Write(c.wr, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

// Connect .
func Connect(host string) (*Client, error) {
	var (
		c   Client
		h   encapsulationHeader
		ct  uint16
		it  itemType
		ser listServicesData
		rs  registerSessionData
	)

	conn, err := net.Dial("tcp4", host)
	if err != nil {
		return nil, err
	}
	c.c = conn
	c.wr = new(bytes.Buffer)
	c.Timeout = 20

	conn.SetDeadline(time.Now().Add(time.Second))

	// ListServices
	c.write(encapsulationHeader{
		Command: ecListServices,
	})
	_, err = conn.Write(c.wr.Bytes())
	if err != nil {
		return nil, err
	}
	c.rd = bufio.NewReader(conn)

	err = c.read(&h)
	if err != nil {
		return nil, err
	}
	err = c.read(&ct)
	if err != nil {
		return nil, err
	}
	err = c.read(&it)
	if err != nil {
		return nil, err
	}
	err = c.read(&ser)
	if err != nil {
		return nil, err
	}
	if ser.NameOfService[0] != 67 || ser.CapabilityFlags&lscfTCP != lscfTCP {
		return nil, errors.New("tcp encapsulation not supported")
	}

	// RegisterSession
	c.reset()
	defer c.reset()
	c.write(encapsulationHeader{
		Command: ecRegisterSession,
		Length:  uint16(binary.Size(rs)),
	})
	c.write(registerSessionData{
		ProtocolVersion: 1,
	})
	_, err = conn.Write(c.wr.Bytes())
	if err != nil {
		return nil, err
	}
	err = c.read(&h)
	if err != nil {
		return nil, err
	}
	err = c.read(&rs)
	if err != nil {
		return nil, err
	}
	if rs.ProtocolVersion != 1 {
		return nil, errors.New("unsupported protocol version")
	}
	c.handle = h.SessionHandle

	return &c, nil
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
func Discover() ([]Identity, error) {
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

	raddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:44818")
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

// Close .
func (c *Client) Close() error {
	c.c.SetDeadline(time.Now().Add(time.Second))

	defer c.reset()
	// UnregisterSession
	c.write(encapsulationHeader{
		Command:       ecUnRegisterSession,
		SessionHandle: c.handle,
	})

	c.c.Write(c.wr.Bytes())

	c.context = 0
	c.handle = 0

	return c.c.Close()
}

// GetAttributesAll
func (c *Client) GetAttributesAll(class, instance int) ([]byte, error) {
	path := pathCIA(class, instance, -1, -1)

	defer c.reset()
	c.writeHead(path, GetAttrAll, 0)
	_, err := c.c.Write(c.wr.Bytes())
	if err != nil {
		return nil, err
	}

	_, ln, err := c.readHead()
	if err != nil {
		return nil, err
	}

	d := make([]byte, ln)
	err = c.read(&d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// GetAttributeList
func (c *Client) GetAttributeList(class, instance int, list []int) ([]byte, error) {
	path := pathCIA(class, instance, -1, -1)
	count := len(list)

	defer c.reset()
	c.writeHead(path, GetAttrList, 2+count*2)
	c.write(uint16(count))
	for _, v := range list {
		c.write(uint16(v))
	}

	_, err := c.c.Write(c.wr.Bytes())
	if err != nil {
		return nil, err
	}

	_, ln, err := c.readHead()
	if err != nil {
		return nil, err
	}

	d := make([]byte, ln)
	err = c.read(&d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// GetAttributeSingle
func (c *Client) GetAttributeSingle(class, instance, attr int) ([]byte, error) {
	path := pathCIA(class, instance, attr, -1)

	defer c.reset()
	c.writeHead(path, GetAttr, 0)
	_, err := c.c.Write(c.wr.Bytes())
	if err != nil {
		return nil, err
	}

	_, ln, err := c.readHead()
	if err != nil {
		return nil, err
	}

	d := make([]byte, ln)
	err = c.read(&d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// ReadTag .
func (c *Client) ReadTag(tag string, count int) (*Tag, error) {
	path := constructPath(parsePath(tag))
	if path == nil {
		return nil, errors.New("path parse error")
	}

	defer c.reset()
	c.writeHead(path, ReadTag, 2)
	c.write(uint16(count))

	_, err := c.c.Write(c.wr.Bytes())
	if err != nil {
		return nil, err
	}

	_, ln, err := c.readHead()
	if err != nil {
		return nil, err
	}

	var t uint16
	err = c.read(&t)
	if err != nil {
		return nil, err
	}
	typlen := 2
	if t == TypeStructHead>>16 {
		err = c.read(&t)
		if err != nil {
			return nil, err
		}
		typlen = 4
	}
	d := make([]uint8, ln-typlen)
	err = c.read(&d)
	if err != nil {
		return nil, err
	}

	// fmt.Println(typeToString(int(t)))
	// fmt.Println(d)

	return &Tag{Name: tag, Type: int(t), data: d}, nil
}

func (c *Client) reset() {
	c.wr.Reset()
	c.rd.Reset(c.c)
}

func (c *Client) readHead() (int, int, error) {
	var (
		h encapsulationHeader
		s sendData
		i itemType
		r response
	)

	err := c.read(&h)
	if err != nil {
		return 0, 0, err
	}
	err = c.read(&s)
	if err != nil {
		return 0, 0, err
	}
	if s.ItemCount != 2 {
		return 0, 0, errors.New("itemCount != 2")
	}
	err = c.read(&i)
	if err != nil {
		return 0, 0, err
	}
	if i.Type != itNullAddress || i.Length != 0 {
		return 0, 0, errors.New("connected addres item not supported")
	}
	err = c.read(&i)
	if err != nil {
		return 0, 0, err
	}
	if i.Type != itUnconnData || i.Length == 0 {
		return 0, 0, errors.New("connected data item not supported")
	}
	err = c.read(&r)
	if err != nil {
		return 0, 0, err
	}
	if r.Status != Success { // TODO additional status size
		return int(r.Status), int(i.Length) - 4, errors.New("status not Success")
	}
	return int(r.Status), int(i.Length) - 4, nil
}

func (c *Client) writeHead(path []uint8, service uint8, dataLen int) {
	c.c.SetDeadline(time.Now().Add(time.Second * time.Duration(c.Timeout)))

	c.context++
	c.write(encapsulationHeader{
		Command:       ecSendRRData,
		Length:        uint16(16 + 2 + len(path) + dataLen),
		SessionHandle: c.handle,
		SenderContext: c.context,
	})
	c.write(sendData{
		Timeout:   c.Timeout,
		ItemCount: 2,
	})
	c.write(itemType{Type: itNullAddress, Length: 0})
	c.write(itemType{Type: itUnconnData, Length: uint16(2 + len(path) + dataLen)})
	c.write(protocolData{
		Service:  service,
		PathSize: uint8(len(path) / 2),
	})
	c.write(path)
}
