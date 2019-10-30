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
	for _, ch := range f {
		switch ch {
		case '$':
			if !str {
				comment = true
			}
		case 0xD, 0xA:
			comment = false
			section = false
		case '[':
			if !str && !comment {
				section = true
				sectionName = ""
			}
		case ']':
			if !str && !comment {
				section = false
				// fmt.Println("section", sectionName)
				p.eds[sectionName] = make(map[string]string)
				item = true
				itemName = ""
			}
		case '=':
			if !str && !comment {
				value = true
				valueName = ""
				item = false
				itemName = strings.TrimSpace(itemName)
				// fmt.Println("item", itemName)
			}
		case ';':
			if !str && !comment {
				valueName = strings.TrimSpace(valueName)
				// fmt.Println("value", valueName)
				s, ok := p.eds[sectionName]
				if ok {
					s[itemName] = valueName
				}
				item = true
				itemName = ""
				value = false
			}
		case '"':
			if !comment { // TODO /" ?, sklejanie stringow
				str = !str
			}
		// case ',': TODO
		default:
			if !comment {
				if section {
					sectionName += string(ch)
				} else if item {
					itemName += string(ch)
				} else if value {
					valueName += string(ch)
				}
			}
		}
	}
	// fmt.Println(p.eds)

	p.Class[1] = defaultIdentityClass() // TODO parse eds
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
		i.Attr[1] = AttrUINT(uint16(v), "VendorID")
	}
	v, err = p.getEDSInt("Device", "ProdType")
	if err == nil {
		i.Attr[2] = AttrUINT(uint16(v), "DeviceType")
	}
	v, err = p.getEDSInt("Device", "ProdCode")
	if err == nil {
		i.Attr[3] = AttrUINT(uint16(v), "ProductCode")
	}
	i.Attr[4] = AttrUINT(majRev+minRev<<8, "Revision")
	vs, err := p.getEDS("Device", "ProdName")
	if err == nil {
		i.Attr[7] = AttrShortString(vs, "ProductName")
	}

	p.Class[0x37] = NewClass("File", 32)

	in := NewInstance(11) // EDS.gz

	chksum := uint(0)
	// in.data = buf.Bytes()
	in.data = f
	for _, x := range in.data {
		chksum += uint(x)
		chksum |= 0xFFFF
	}
	chksum = 0x10000 - chksum

	in.Attr[1] = AttrUSINT(stateLoaded, "State")
	in.Attr[2] = AttrStringI("EDS and Icon Files", "InstanceName")
	in.Attr[3] = AttrUINT(1, "InstanceFormatVersion")
	in.Attr[4] = AttrStringI("EDS.txt", "FileName")
	in.Attr[5] = AttrUINT(majRev+minRev<<8, "FileRevision")
	in.Attr[6] = AttrUDINT(uint32(len(f)), "FileSize")
	in.Attr[7] = AttrINT(int16(chksum), "FileChecksum")
	in.Attr[8] = AttrUSINT(255, "InvocationMethod")  // not aplicable
	in.Attr[9] = AttrUSINT(1, "FileSaveParameters")  // BYTE
	in.Attr[10] = AttrUSINT(1, "FileType")           // read only
	in.Attr[11] = AttrUSINT(0, "FileEncodingFormat") // uncompressed

	p.Class[0x37].Inst[0xC8] = in
	return nil
}
