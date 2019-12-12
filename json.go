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

	in := p.Class[0xAC].inst[1]
	in.attr[1] = TagINT(int16(db.AC[0]), "Attr1")
	in.attr[2] = TagINT(int16(db.AC[1]), "Attr2")
	in.attr[3] = TagDINT(int32(db.AC[2]), "Attr3")
	in.attr[4] = TagDINT(int32(db.AC[3]), "Attr4")
	in.attr[10] = TagDINT(int32(db.AC[4]), "Attr5")

	for name, t := range db.Templates { // FIXME kolejnosc tak jak w jsonie
		var tmpl []T
		for _, m := range t.Member {
			var tx T
			tx.N = m.Name
			tx.T = m.Type
			tx.C = m.Size
			tx.O = m.Offset
			tmpl = append(tmpl, tx)
		}
		p.newUDT(tmpl, name, t.Handle, t.Size)
	}
	for name, s := range db.Symbols { // FIXME kolejnosc tak jak w jsonie
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
		p.addTag(tag, s.Instance)
	}
	return nil
}
