package plcconnector

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
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

func (p *PLC) getEDS(section string, item string) (string, error) {
	s, ok := p.eds[section]
	if ok {
		v, vok := s[item]
		if vok {
			fmt.Println(section, item, v)
			return v, nil
		}
	}
	return "", errors.New("not found")
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
	return 0, errors.New("not found")
}

func (p *PLC) loadEDS(fn string) error {
	var (
		f   []byte
		err error
	)

	if fn == "" {
		f = defEDS
	} else {
		f, err = os.ReadFile(fn)
		if err != nil {
			return err
		}
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
	for _, ch := range f {
		switch ch {
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

	gzipped := true

	in := NewInstance(11) // EDS.gz

	if gzipped {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Name = "0001000E00601800.eds"
		gz.Comment = "ODVA File Encoding V1.0"
		gz.OS = 0x0B // NTFS
		gz.Write(f)
		gz.Close()
		in.data = buf.Bytes()
	} else {
		in.data = f
	}

	chksum := 0
	for _, x := range in.data {
		chksum += int(x)
	}
	chksum = 0x10000 - (chksum & 0xFFFF)

	in.attr[1] = TagUSINT(stateLoaded, "State")
	in.attr[2] = TagStringI("EDS and Icon Files", "InstanceName")
	in.attr[3] = TagUINT(1, "InstanceFormatVersion")
	if gzipped {
		in.attr[4] = TagStringI("EDS.gz", "FileName")
		in.attr[11] = TagUSINT(1, "FileEncodingFormat")
	} else {
		in.attr[4] = TagStringI("EDS.txt", "FileName")
		in.attr[11] = TagUSINT(0, "FileEncodingFormat")
	}
	in.attr[5] = TagUINT(majRev+minRev<<8, "FileRevision")
	in.attr[6] = TagUDINT(uint32(len(in.data)), "FileSize")
	in.attr[7] = TagINT(int16(chksum), "FileChecksum")
	in.attr[8] = TagUSINT(255, "InvocationMethod") // not aplicable
	in.attr[9] = TagUSINT(1, "FileSaveParameters") // BYTE
	in.attr[10] = TagUSINT(1, "FileType")          // read only

	dir := []uint8{0xC8, 0x00}
	dir = append(dir, in.attr[2].data...)
	dir = append(dir, in.attr[4].data...)
	p.Class[FileClass].inst[0].attr[32] = &Tag{Name: "Directory", data: dir}

	p.Class[FileClass].SetInstance(0xC8, in)

	p.Class[0xAC] = NewClass("AC", 0) // unknown class, values from 1756-pm020_-en-p.pdf p. 57
	in = NewInstance(10)
	in.attr[1] = TagINT(5, "Attr1")
	in.attr[2] = TagINT(1, "Attr2")
	in.attr[3] = TagDINT(0, "TagCRC") // guess
	in.attr[4] = TagDINT(0, "UDTCRC") // guess
	in.attr[10] = &Tag{Name: "Attr10", Type: TypeDINT, data: []uint8{0xF8, 0xDE, 0x47, 0xB8}}
	p.Class[0xAC].SetInstance(1, in)

	p.Class[SymbolClass] = NewClass("Symbol", 0)
	p.symbols = p.Class[SymbolClass]

	p.Class[TemplateClass] = NewClass("Template", 0)
	p.template = p.Class[TemplateClass]

	p.Class[ClockClass] = NewClass("Clock", 0)
	in = NewInstance(11)
	in.attr[6] = TagULINT(0, "UTSTime")
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
	in.attr[11] = TagULINT(0, "LocalTime")
	in.attr[11].getter = func() []uint8 {
		x := make([]uint8, 8)
		t := time.Now().Add(p.timOff)
		_, off := t.Zone()
		binary.LittleEndian.PutUint64(x, uint64((t.UnixNano()/1000)+int64(off)*1_000_000))
		return x
	}
	p.Class[ClockClass].SetInstance(1, in)

	p.Class[PortClass] = NewClass("Port", 9)
	p.Class[PortClass].inst[0].attr[8] = TagUINT(1, "EntryPort")
	p.Class[PortClass].inst[0].attr[9] = &Tag{Name: "PortInstanceInfo", data: []uint8{0, 0, 0, 0, 4, 0, 1, 0}} // uint 4 - Ethernet/IP , uint 1 - CIP port number
	in = NewInstance(7)
	in.attr[1] = TagUINT(4, "PortType")
	in.attr[2] = TagUINT(1, "PortNumber")
	in.attr[3] = &Tag{Name: "LinkObject", Type: TypeEPATH, data: []uint8{0x02, 0x00, 0x20, 0xF5, 0x24, 0x01}}
	in.attr[4] = TagShortString("EtherNet/IP port", "PortName")
	in.attr[7] = &Tag{Name: "NodeAddress", Type: TypeEPATH, data: []uint8{0x01, 0x00, 0x10, 0x01}}
	p.Class[PortClass].SetInstance(1, in)

	p.Class[TCPClass] = NewClass("TCP Interface", 0)
	in = NewInstance(6)
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
	p.Class[TCPClass].SetInstance(1, in)

	p.Class[EthernetClass] = NewClass("Ethernet Link", 0)
	p.Class[EthernetClass].inst[0].SetAttrUINT(1, 3)
	in = NewInstance(3)
	in.attr[1] = TagUDINT(1000, "InterfaceSpeed")
	in.attr[2] = TagUDINT(0b0_1_011_1_1, "InterfaceFlags")
	in.attr[3] = &Tag{Name: "PhysicalAddress", data: mac}
	p.Class[EthernetClass].SetInstance(1, in)

	return nil
}
