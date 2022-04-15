package plcconnector

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type jsSymbols struct {
	Instance int    `json:"instance"`
	Array    bool   `json:"array"`
	Struct   bool   `json:"struct"`
	Type     string `json:"type"`
	TypeInt  int    `json:"type_int"`
	TypeSize int    `json:"type_size"`
	Dim      []int  `json:"dim"`
}

type jsMember struct {
	Size     int    `json:"size"`
	Type     string `json:"type"`
	TypeInt  int    `json:"type_int"`
	TypeSize int    `json:"type_size"`
	Offset   int    `json:"offset"`
	Name     string `json:"name"`
}

type jsTemplates struct {
	Handle int        `json:"handle"`
	Size   int        `json:"size"`
	Member []jsMember `json:"member"`
}

// JS .
type JS struct {
	AC        [5]int                 `json:"ac"`
	Symbols   map[string]jsSymbols   `json:"symbols"`
	Templates map[string]jsTemplates `json:"templates"`
}

// ImportJSON .
func (p *PLC) ImportJSON(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var db JS
	err = json.Unmarshal(data, &db)
	if err != nil {
		return err
	}

	tt := db.Templates
	for len(tt) > 0 {
		newtt := make(map[string]jsTemplates)
		for name, t := range tt {
			var tmpl []udtT
			sis := false
			for _, m := range t.Member {
				var tx udtT
				tx.N = m.Name
				tx.T = m.Type
				tx.C = m.Size
				tx.O = m.Offset
				if m.TypeInt > TypeStruct {
					_, ok := p.tids[m.Type]
					if !ok {
						sis = true
						break
					}
				}
				tmpl = append(tmpl, tx)
			}
			if sis {
				newtt[name] = t
			} else {
				p.newUDT(tmpl, name, t.Handle, t.Size)
			}
		}
		tt = newtt
	}

	for name, s := range db.Symbols {
		var tag Tag
		if len(s.Dim) != 3 {
			return errors.New("dim.length != 3")
		}
		tag.Dim[0] = s.Dim[0]
		tag.Dim[1] = s.Dim[1]
		tag.Dim[2] = s.Dim[2]
		tag.Name = name
		if s.TypeInt < TypeStruct {
			tag.Type = s.TypeInt & TypeType
			tag.data = make([]uint8, s.TypeSize*tag.Dims())
		} else {
			st, ok := p.tids[s.Type]
			if !ok {
				panic("symbols " + s.Type)
			}
			tag.st = &st
			tag.Type = int(st.h) | TypeStructHead
			tag.data = make([]uint8, st.l*tag.Dims())
		}
		p.addTag(tag, s.Instance)
	}

	in := p.Class[0xAC].inst[1]
	in.SetAttrINT(1, int16(db.AC[0]))
	in.SetAttrINT(2, int16(db.AC[1]))
	in.SetAttrDINT(3, int32(db.AC[2]))
	in.SetAttrDINT(4, int32(db.AC[3]))
	in.SetAttrDINT(10, int32(db.AC[4]))
	p.Class[SymbolClass].inst[0].SetAttrUDINT(8, uint32(db.AC[2]))

	return nil
}

type memJS struct {
	Rx   []uint8 `json:"rx"`
	Read bool    `json:"read"`
}

// ImportMemoryJSON .
func (p *PLC) ImportMemoryJSON(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	db := make(map[string]memJS)
	err = json.Unmarshal(data, &db)
	if err != nil {
		return err
	}
	for n, c := range db {
		tag, ok := p.tags[strings.ToLower(n)]
		if !ok {
			return errors.New("no tag " + n)
		}
		if c.Read && tag.prot == 0 {
			tag.prot = 2
			tag.in.attr[10] = TagUSINT(tag.prot, "External Acces")
		}
		if c.Read {
			if len(c.Rx) != len(tag.data) {
				if p.Verbose {
					fmt.Println(file, "data length mismatch", n)
				}
				continue
				// return errors.New("data length mismatch " + n)
			}
			copy(tag.data, c.Rx)
		}
	}

	return nil
}
