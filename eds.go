package plcconnector

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	_ "embed"
)

//go:embed example/test.eds
var defEDS []byte

const (
	stateNonExistent  = 0
	stateEmpty        = 1
	stateLoaded       = 2
	stateUploadInit   = 3
	stateDownloadInit = 4
	stateUpload       = 5
	stateDownload     = 6
	stateStoring      = 7
)

var (
	errNotFound = errors.New("not found")
	errBadICO   = errors.New("malformed ICO")
)

func (p *PLC) getEDS(section string, item string) (string, error) {
	s, ok := p.eds[section]
	if ok {
		v, vok := s[item]
		if vok {
			fmt.Println(section, item, v)
			return v, nil
		}
	}
	return "", errNotFound
}

func (p *PLC) getEDSInt(section string, item string) (int, error) {
	v, err := p.getEDS(section, item)
	if err == nil {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	return 0, errNotFound
}

func (p *PLC) loadEDS(fn string) error {
	var (
		orig_f []byte
		f      []byte
		err    error
		gz     = false
	)

	if fn == "" {
		orig_f = defEDS
	} else {
		orig_f, err = os.ReadFile(fn)
		if err != nil {
			return err
		}
	}

	if len(orig_f) >= 10 && orig_f[0] == 0x1f && orig_f[1] == 0x8b {
		gz = true
		f, err = loadGzip(orig_f)
		if err != nil {
			return nil
		}
	} else {
		f = make([]byte, len(orig_f))
		copy(f, orig_f)
	}

	p.eds = make(map[string]map[string]string)
	comment := false
	section := false
	sectionName := ""
	str := false
	item := false
	value := false
	valueName := ""
	itemName := ""
	valueString := false
	lf := len(f)

	for i := 0; i < lf; i++ {
		ch := f[i]
		switch ch {
		case 0:
			if lf > i+5 && f[i+1] == 0 && f[i+2] == 1 && f[i+3] == 0 {
				icons := (int(f[i+5]) << 8) | int(f[i+4])
				hdrSize := 6
				icoSize := 0

				for k := 0; k < icons; k++ {
					if i+hdrSize+11 >= lf {
						return errBadICO
					}
					icoSize += (int(f[i+hdrSize+11]) << 24) | (int(f[i+hdrSize+10]) << 16) | (int(f[i+hdrSize+9]) << 8) | int(f[i+hdrSize+8])
					hdrSize += 16
				}
				icoSize += hdrSize
				if i+icoSize > lf {
					return errBadICO
				}

				p.favicon = f[i : i+icoSize]
				i += icoSize

				fmt.Printf("ICO: %d icon(s), %d bytes\n", icons, icoSize)
			} else {
				return errBadICO
			}
		case '$':
			if !str {
				comment = true
			} else {
				valueName += string(ch)
			}
		case 0xD, 0xA:
			comment = false
			section = false
		case '[':
			if !comment {
				if !str {
					section = true
					sectionName = ""
				} else {
					valueName += string(ch)
				}
			}
		case ']':
			if !comment {
				if !str {
					section = false
					// fmt.Println("section", sectionName)
					p.eds[sectionName] = make(map[string]string)
					item = true
					itemName = ""
				} else {
					valueName += string(ch)
				}
			}
		case '=':
			if !comment {
				if !str {
					value = true
					valueName = ""
					item = false
					itemName = strings.TrimSpace(itemName)
					// fmt.Println("item", itemName)
				} else {
					valueName += string(ch)
				}
			}
		case ';':
			if !comment {
				if !str {
					valueName = strings.TrimSpace(valueName)
					// fmt.Println("value", valueName)
					s, ok := p.eds[sectionName]
					if ok {
						s[itemName] = valueName
					}
					item = true
					itemName = ""
					value = false
					valueString = false
				} else {
					valueName += string(ch)
				}
			}
		case '"':
			if !comment { // TODO \" \n Vol1 7-3.5.4
				str = !str
				if str {
					valueString = true
				}
			}
		// case ',': TODO
		default:
			if !comment {
				if section {
					sectionName += string(ch)
				} else if item {
					itemName += string(ch)
				} else if value {
					if valueString && !str {
						break
					}
					valueName += string(ch)
				}
			}
		}
	}
	// fmt.Println(p.eds["Device"]["IconContents"])
	// fmt.Println(p.eds["Port"]["Port1"])

	p.Class[IdentityClass] = defaultIdentityClass()
	i := p.Class[IdentityClass].inst[1]
	i.getall = []int{1, 2, 3, 4, 5, 6, 7} // FIXME communication device 10

	majRev := uint16(1)
	minRev := uint16(1)

	v, err := p.getEDSInt("Device", "MajRev")
	if err == nil {
		majRev = uint16(v)
	}
	v, err = p.getEDSInt("Device", "MinRev")
	if err == nil {
		minRev = uint16(v)
	}

	v, err = p.getEDSInt("Device", "VendCode")
	if err == nil {
		i.SetAttrUINT(1, uint16(v))
	}
	v, err = p.getEDSInt("Device", "ProdType")
	if err == nil {
		i.SetAttrUINT(2, uint16(v))
	}
	v, err = p.getEDSInt("Device", "ProdCode")
	if err == nil {
		i.SetAttrUINT(3, uint16(v))
	}
	i.SetAttrUINT(4, majRev+minRev<<8)
	vs, err := p.getEDS("Device", "ProdName")
	if err == nil {
		p.Name = vs
		i.attr[7] = TagShortString(vs, "ProductName")
		i.attr[13] = TagStringI(vs, "InternationalProductName")
	}

	p.Class[FileClass] = NewClass("File", 32)
	p.Class[FileClass].inst[0].SetAttrUINT(1, 2)

	in := NewInstance(11) // EDS.gz

	if !gz {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Name = "0001000E00601800.eds"
		gz.Comment = "ODVA File Encoding V1.0"
		gz.OS = 0x0B // NTFS
		gz.Write(f)
		gz.Close()
		in.data = buf.Bytes()
	} else {
		in.data = orig_f
	}

	chksum := 0
	for _, x := range in.data {
		chksum += int(x)
	}
	chksum = 0x10000 - (chksum & 0xFFFF)

	in.attr[1] = TagUSINT(stateLoaded, "State")
	in.attr[2] = TagStringI("EDS and Icon Files", "InstanceName")
	in.attr[3] = TagUINT(1, "InstanceFormatVersion")
	in.attr[4] = TagStringI("EDS.gz", "FileName")
	in.attr[5] = TagUINT(majRev+minRev<<8, "FileRevision")
	in.attr[6] = TagUDINT(uint32(len(in.data)), "FileSize")
	in.attr[7] = TagINT(int16(chksum), "FileChecksum")
	in.attr[8] = TagUSINT(255, "InvocationMethod") // not aplicable
	in.attr[9] = TagUSINT(0, "FileSaveParameters") // BYTE
	in.attr[10] = TagUSINT(1, "FileType")          // read only
	in.attr[11] = TagUSINT(1, "FileEncodingFormat")

	dir := []uint8{0xC8, 0x00}
	dir = append(dir, in.attr[2].data...)
	dir = append(dir, in.attr[4].data...)
	p.Class[FileClass].inst[0].attr[32] = &Tag{Name: "Directory", data: dir}

	p.Class[FileClass].SetInstance(0xC8, in)

	p.Class[MessageRouter] = NewClass("Message Router", 7)
	in = NewInstance(0)
	p.Class[MessageRouter].SetInstance(1, in)

	p.Class[ConnManager] = NewClass("Connection Manager", 7)
	in = NewInstance(0)
	p.Class[ConnManager].SetInstance(1, in)

	p.Class[0xAC] = NewClass("AC", 0) // unknown class, values from 1756-pm020_-en-p.pdf p. 57
	in = NewInstance(10)
	in.attr[1] = TagINT(5, "Attr1")
	in.attr[2] = TagINT(1, "Attr2")
	in.attr[3] = TagDINT(0, "TagCRC") // guess
	in.attr[4] = TagDINT(0, "UDTCRC") // guess
	in.attr[10] = &Tag{Name: "Attr10", Type: TypeDINT, data: []uint8{0xF8, 0xDE, 0x47, 0xB8}}
	p.Class[0xAC].SetInstance(1, in)

	p.Class[ProgramClass] = NewClass("Program", 0)
	in = NewInstance(1)
	in.attr[1] = TagString("main", "Program Name")
	p.Class[ProgramClass].SetInstance(1, in)

	p.Class[SymbolClass] = NewClass("Symbol", 8)
	p.Class[SymbolClass].inst[0].SetAttrUINT(1, 4)
	p.Class[SymbolClass].inst[0].attr[8] = TagUDINT(0, "Symbol UID")
	p.symbols = p.Class[SymbolClass]

	p.Class[TemplateClass] = NewClass("Template", 0)
	p.template = p.Class[TemplateClass]

	p.Class[ClockClass] = NewClass("Clock", 0)
	p.Class[ClockClass].inst[0].SetAttrUINT(1, 3)
	in = NewInstance(11)
	in.attr[5] = &Tag{Name: "DateAndTime"}
	in.attr[5].getter = func() []uint8 {
		t := time.Now().Add(p.timOff)
		x := make([]uint8, 7*4)
		binary.LittleEndian.PutUint64(x, uint64(t.Year()))
		binary.LittleEndian.PutUint64(x[4:], uint64(t.Month()))
		binary.LittleEndian.PutUint64(x[8:], uint64(t.Day()))
		binary.LittleEndian.PutUint64(x[12:], uint64(t.Hour()))
		binary.LittleEndian.PutUint64(x[16:], uint64(t.Minute()))
		binary.LittleEndian.PutUint64(x[20:], uint64(t.Second()))
		// microsecond
		return x
	}
	in.attr[6] = TagLINT(0, "CurrentUTCValue")
	in.attr[6].getter = func() []uint8 {
		x := make([]uint8, 8)
		binary.LittleEndian.PutUint64(x, uint64(time.Now().Add(p.timOff).UnixNano()/1000))
		return x
	}
	in.attr[6].write = true
	in.attr[6].setter = func(dt []uint8) uint8 {
		if len(dt) != 8 {
			return TooMuchData
		}
		x := int64(binary.LittleEndian.Uint64(dt))
		p.timOff = -time.Since(time.Unix(x/1_000_000, x%1_000_000))
		return Success
	}
	in.attr[7] = &Tag{Name: "UTCDateAndTime"}
	in.attr[7].getter = func() []uint8 {
		t := time.Now().Add(p.timOff).UTC()
		x := make([]uint8, 7*4)
		binary.LittleEndian.PutUint64(x, uint64(t.Year()))
		binary.LittleEndian.PutUint64(x[4:], uint64(t.Month()))
		binary.LittleEndian.PutUint64(x[8:], uint64(t.Day()))
		binary.LittleEndian.PutUint64(x[12:], uint64(t.Hour()))
		binary.LittleEndian.PutUint64(x[16:], uint64(t.Minute()))
		binary.LittleEndian.PutUint64(x[20:], uint64(t.Second()))
		// microsecond
		return x
	}
	in.attr[8] = &Tag{Name: "TimeZoneString", getter: func() []uint8 {
		_, offset := time.Now().Add(p.timOff).Zone()
		str := fmt.Sprintf("GMT%+03d:%02d", offset/3600, int(math.Abs(float64((offset%3600)/60))))
		strLen := len(str)
		x := make([]byte, 4+strLen)
		binary.LittleEndian.PutUint32(x, uint32(strLen))
		copy(x[4:], str)
		return x
	}}
	in.attr[9] = TagINT(0, "DSTAdjustment")
	in.attr[10] = TagUSINT(0, "EnableDST")
	in.attr[11] = TagLINT(0, "CurrentValue")
	in.attr[11].getter = func() []uint8 {
		x := make([]uint8, 8)
		t := time.Now().Add(p.timOff)
		_, off := t.Zone()
		binary.LittleEndian.PutUint64(x, uint64((t.UnixNano()/1000)+int64(off)*1_000_000))
		return x
	}
	p.Class[ClockClass].SetInstance(1, in)

	p.Class[PortClass] = NewClass("Port", 9)
	p.Class[PortClass].inst[0].SetAttrUINT(1, 2)
	p.Class[PortClass].inst[0].attr[8] = TagUINT(1, "EntryPort")
	p.Class[PortClass].inst[0].attr[9] = &Tag{Name: "PortInstanceInfo", data: []uint8{0, 0, 0, 0, 4, 0, 1, 0}} // uint 4 - Ethernet/IP , uint 1 - CIP port number
	in = NewInstance(10)
	in.attr[1] = TagUINT(4, "PortType")
	in.attr[2] = TagUINT(1, "PortNumber")
	in.attr[3] = &Tag{Name: "LinkObject", Type: TypeEPATH, data: []uint8{0x02, 0x00, 0x20, 0xF5, 0x24, 0x01}}
	in.attr[4] = TagShortString("EtherNet/IP port", "PortName")
	in.attr[7] = &Tag{Name: "NodeAddress", Type: TypeEPATH, data: []uint8{0x01, 0x00, 0x10, 0x01}}
	in.attr[10] = &Tag{Name: "Port Routing Capabilities", Type: TypeDWORD, data: []uint8{0x00, 0x00, 0x00, 0x00}}
	p.Class[PortClass].SetInstance(1, in)

	p.Class[TCPClass] = NewClass("TCP Interface", 0)
	p.Class[TCPClass].inst[0].SetAttrUINT(1, 4)
	in = NewInstance(13)
	in.attr[1] = TagUDINT(1, "Status")
	in.attr[2] = TagUDINT(0b1_1_0, "ConfigurationCapabality")
	in.attr[3] = TagUDINT(0b1_0010, "ConfigurationControl")
	in.attr[4] = &Tag{Name: "PhysicalLinkObject", Type: TypeEPATH, data: []uint8{0x02, 0x00, 0x20, 0xF6, 0x24, 0x01}}
	ip, mac := getNetIf()
	in.attr[5] = &Tag{Name: "InterfaceConfiguration", data: []uint8{ // TODO
		uint8(ip >> 24), uint8(ip >> 16), uint8(ip >> 8), uint8(ip), // IP address
		0xFF, 0, 0, 0, // network mask
		0xA, 0xA, 0, 1, // gateway address
		8, 8, 8, 8, // name server
		1, 1, 1, 1, // name server 2
		0, 0, // string domain name
	}}
	hostname, _ := os.Hostname()
	in.attr[6] = TagString(hostname, "HostName")
	in.attr[10] = TagBOOL(false, "SelectACD")
	in.attr[11] = &Tag{Name: "LastConflictDetected", data: make([]byte, 1+6+28)}
	in.attr[13] = TagUINT(120, "EncapsulationInactivityTimeout")
	p.Class[TCPClass].SetInstance(1, in)

	p.Class[EthernetClass] = NewClass("Ethernet Link", 0)
	p.Class[EthernetClass].inst[0].SetAttrUINT(1, 4)
	in = NewInstance(11)
	in.attr[1] = TagUDINT(1000, "InterfaceSpeed")
	in.attr[2] = TagUDINT(0b0_1_011_1_1, "InterfaceFlags")
	in.attr[3] = &Tag{Name: "PhysicalAddress", data: mac}
	in.attr[4] = &Tag{Name: "InterfaceCounters", data: make([]byte, 11*4), Dim: [3]int{11, 0, 0}, st: &structData{l: 4}}
	in.attr[5] = &Tag{Name: "MediaCounters", data: make([]byte, 12*4), Dim: [3]int{12, 0, 0}, st: &structData{l: 4}}
	in.attr[6] = &Tag{Name: "InterfaceControl", data: []byte{1, 0, 0, 0}} // WORD ControlBits, UINT ForcedInternetSpeed
	in.attr[7] = TagUSINT(2, "InterfaceType")                             // 2: twisted-pair, 3: optical fiber
	in.attr[8] = TagUSINT(1, "InterfaceState")
	in.attr[9] = TagUSINT(1, "AdminState")
	in.attr[10] = TagShortString("eth0", "InterfaceLabel")
	in.attr[11] = &Tag{Name: "InterfaceCapability", data: []byte{0, 0, 0, 0, 3, 0, 10, 0, 1, 100, 0, 1, 0xE8, 0x03, 1}} // DWORD Capability Bits, USINT Speed/Duplex Array Count: UINT Interface Speed, USINT Inferface Duplex Mode (1: full duplex)

	p.Class[EthernetClass].SetInstance(1, in)

	// FIXME communication device
	// service 4B verify a fault location, no par, resp 2 bytes zeroes
	// service 4C clear rapid faults 2 bytes, no par, resp status 0xC Object State Conflict if dlr not enabled
	// service 4E clear gateway partial fault 2 bytes, no par, resp status 0xC Object State Conflict if dlr not enabled

	// p.Class[DLRClass] = NewClass("Device Level Ring", 0)
	// p.Class[DLRClass].inst[0].SetAttrUINT(1, 3)
	// in = NewInstance(9)
	// in.getall = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	// in.attr[1] = TagUSINT(0, "Network Topology") // 0: liner, 1: ring
	// in.attr[2] = TagUSINT(0, "Network Status")   // 0: normal, 1: ring fault, 2: unexpected loop detected, 3: partial network fault, 4: rapid fault / restore cycle

	// in.attr[3] = TagUSINT(3, "Unknown")
	// in.attr[4] = &Tag{Name: "Unknown", data: []byte{0, 0, 144, 1, 0, 0, 168, 7, 0, 0, 0, 0}}

	// in.attr[5] = TagUINT(0, "Ring Fault Count")
	// in.attr[6] = &Tag{Name: "Last Active Node on Port 1", data: make([]byte, 10)} // ip address, mac address
	// in.attr[7] = &Tag{Name: "Last Active Node on Port 2", data: make([]byte, 10)} // ip address, mac address
	// in.attr[8] = TagUINT(0, "Ring Participants Count")
	// // attr 9 hole Ring Protocol Participants List
	// in.attr[10] = &Tag{Name: "Active Supervisor	Address", data: make([]byte, 10)} // ip address, mac address
	// in.attr[11] = TagUSINT(0, "Active Supervisor Precedence")
	// in.attr[12] = &Tag{Name: "Capability Flags", data: []byte{162, 0, 0, 0}}

	// p.Class[DLRClass].SetInstance(1, in)

	return nil
}
