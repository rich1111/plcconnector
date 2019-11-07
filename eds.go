package plcconnector

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

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
	f, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
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

	p.Class[1] = defaultIdentityClass()
	i := p.Class[1].Inst[1]

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
		i.attr[1] = AttrUINT(uint16(v), "VendorID")
	}
	v, err = p.getEDSInt("Device", "ProdType")
	if err == nil {
		i.attr[2] = AttrUINT(uint16(v), "DeviceType")
	}
	v, err = p.getEDSInt("Device", "ProdCode")
	if err == nil {
		i.attr[3] = AttrUINT(uint16(v), "ProductCode")
	}
	i.attr[4] = AttrUINT(majRev+minRev<<8, "Revision")
	vs, err := p.getEDS("Device", "ProdName")
	if err == nil {
		i.attr[7] = AttrShortString(vs, "ProductName")
	}

	p.Class[0x37] = NewClass("File", 32)

	in := NewInstance(11) // EDS.gz

	chksum := uint(0)
	in.data = f
	for _, x := range in.data {
		chksum += uint(x)
		chksum &= 0xFFFF
	}
	chksum = 0x10000 - chksum

	in.attr[1] = AttrUSINT(stateLoaded, "State")
	in.attr[2] = AttrStringI("EDS and Icon Files", "InstanceName")
	in.attr[3] = AttrUINT(1, "InstanceFormatVersion")
	in.attr[4] = AttrStringI("EDS.txt", "FileName")
	in.attr[5] = AttrUINT(majRev+minRev<<8, "FileRevision")
	in.attr[6] = AttrUDINT(uint32(len(f)), "FileSize")
	in.attr[7] = AttrINT(int16(chksum), "FileChecksum")
	in.attr[8] = AttrUSINT(255, "InvocationMethod")  // not aplicable
	in.attr[9] = AttrUSINT(1, "FileSaveParameters")  // BYTE
	in.attr[10] = AttrUSINT(1, "FileType")           // read only
	in.attr[11] = AttrUSINT(0, "FileEncodingFormat") // uncompressed

	p.Class[0x37].SetInstance(0xC8, in)

	p.Class[0xAC] = NewClass("AC", 0) // unknown class, values from 1756-pm020_-en-p.pdf p. 57
	in = NewInstance(10)
	in.attr[1] = AttrINT(5, "Attr1")
	in.attr[2] = AttrINT(1, "Attr2")
	in.attr[3] = &Attribute{Name: "Attr3", data: []uint8{0x03, 0xB2, 0x80, 0xC5}}   // DINT
	in.attr[4] = &Attribute{Name: "Attr4", data: []uint8{0x03, 0xB2, 0x80, 0xC5}}   // DINT
	in.attr[10] = &Attribute{Name: "Attr10", data: []uint8{0xF8, 0xDE, 0x47, 0xB8}} // DINT
	p.Class[0xAC].SetInstance(1, in)

	p.Class[0x6B] = NewClass("Symbol", 1)
	p.symbols = p.Class[0x6B]

	p.Class[0x6C] = NewClass("Template", 1)
	p.Class[0x6C].Inst[0].attr[1] = AttrUINT(1, "Revision")

	p.Class[0xF4] = NewClass("Port", 9)
	p.Class[0xF4].Inst[0].attr[1] = AttrUINT(1, "Revision")
	p.Class[0xF4].Inst[0].attr[2] = AttrUINT(1, "MaxInstance")
	p.Class[0xF4].Inst[0].attr[3] = AttrUINT(1, "NumInstances")
	p.Class[0xF4].Inst[0].attr[8] = AttrUINT(1, "EntryPort")
	p.Class[0xF4].Inst[0].attr[9] = &Attribute{Name: "PortInstanceInfo", data: []uint8{0, 0, 0, 0, 4, 0, 1, 0}} // uint 4 - Ethernet/IP , uint 1 - CIP port number
	in = NewInstance(7)
	in.attr[1] = AttrUINT(4, "PortType")
	in.attr[2] = AttrUINT(1, "PortNumber")
	in.attr[3] = &Attribute{Name: "LinkObject", data: []uint8{0x02, 0x00, 0x20, 0xF5, 0x24, 0x01}}
	in.attr[4] = AttrShortString("EtherNet/IP port", "PortName")
	in.attr[7] = &Attribute{Name: "NodeAddress", data: []uint8{0x01, 0x00, 0x10, 0x01}}
	p.Class[0xF4].SetInstance(1, in)

	p.Class[0xF5] = NewClass("TCP Interface", 0)
	in = NewInstance(6)
	in.attr[1] = AttrUDINT(1, "Status")
	in.attr[2] = AttrUDINT(0b1_1_0, "ConfigurationCapabality")
	in.attr[3] = AttrUDINT(0b1_0010, "ConfigurationControl")
	in.attr[4] = &Attribute{Name: "PhysicalLinkObject", data: []uint8{0x02, 0x00, 0x20, 0xF6, 0x24, 0x01}}
	ip := getIP4()
	in.attr[5] = &Attribute{Name: "InterfaceConfiguration", data: []uint8{ // TODO
		uint8(ip >> 24), uint8(ip >> 16), uint8(ip >> 8), uint8(ip), // IP address
		0xFF, 0, 0, 0, // network mask
		0xA, 0xA, 0, 1, // gateway address
		8, 8, 8, 8, // name server
		1, 1, 1, 1, // name server 2
		0, 0, // string domain name
	}}
	in.attr[6] = AttrString("", "HostName") // TODO
	p.Class[0xF5].SetInstance(1, in)

	p.Class[0xF6] = NewClass("Ethernet Link", 1)
	p.Class[0xF6].Inst[0].attr[1] = AttrUINT(3, "Revision")
	in = NewInstance(3)
	in.attr[1] = AttrUDINT(1000, "InterfaceSpeed")
	in.attr[2] = AttrUDINT(0b0_1_011_1_1, "InterfaceFlags")
	in.attr[3] = &Attribute{Name: "PhysicalAddress", data: []uint8{0x02, 0x00, 0x20, 0xF5, 0x24, 0x01}} // TODO MAC
	p.Class[0xF6].SetInstance(1, in)

	return nil
}
