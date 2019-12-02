package plcconnector

import (
	"encoding/json"
	"io/ioutil"
)

type jsSymbols struct {
	Instance int    `json:"instance"`
	Array    bool   `json:"array"`
	Struct   bool   `json:"struct"`
	Type     string `json:"type"`
	TypeInt  int    `json:"type_int"`
	TypeSize int    `json:"type_size"`
	Size     int    `json:"size"`
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
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	var db JS
	err = json.Unmarshal(data, &db)
	if err != nil {
		return err
	}
	for name, t := range db.Templates {
		var tmpl []T
		for _, m := range t.Member {
			var tx T
			tx.N = m.Name
			tx.T = m.Type
			tx.C = m.Size
			tmpl = append(tmpl, tx)
		}
		p.newUDT(tmpl, name, t.Handle)
	}
	for name, s := range db.Symbols {
		var tag Tag
		if s.Size > 0 {
			tag.Count = s.Size
		} else {
			tag.Count = 1
		}
		tag.Name = name
		if s.TypeInt < TypeStruct {
			tag.Type = s.TypeInt & TypeType
			tag.data = make([]uint8, s.TypeSize*tag.Count)
		} else {
			st, ok := p.tids[s.Type]
			if !ok {
				panic("")
			}
			tag.st = &st
			tag.Type = int(st.h) | TypeStructHead
			tag.data = make([]uint8, st.l*tag.Count)
		}
		p.AddTag(tag)

	}
	return nil
}
