// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package plcconnector

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

const mainCSS = `
:root {
	--hide: none;
}
.pr {
	display: var(--hide);
}
table {
	font-family: "Courier New", Courier, monospace;
	border-spacing: 20px 0;
	text-align: left;
}
`

const mainJS = `
<script>
document.addEventListener("DOMContentLoaded", function(e) {
	var b = document.getElementById("showbtn");
	b.addEventListener("click", function(e){
		document.documentElement.style.setProperty('--hide', b.checked ? 'table-row' : 'none');
	});
});
</script>
`

func (p *PLC) tagsIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var toSend strings.Builder

	toSend.WriteString("<!DOCTYPE html>\n<html><style>" + mainCSS + "</style>" + mainJS + "<title>" + p.Name + "</title><h3>" + p.Name + "</h3><p>Wersja biblioteki: 2020.01</p>\n<input type=checkbox id=showbtn name=showbtn><label for=showbtn>Pokaż wszystkie</label><table><tr><th>Nazwa</th><th>Rozmiar</th><th>Typ</th><th>Odczyt</th><th>ASCII</th></tr>\n")

	p.tMut.RLock()
	arr := make([]string, 0, len(p.tags))

	for n := range p.tags {
		arr = append(arr, n)
	}

	sort.Strings(arr)

	for _, n := range arr {
		t := p.tags[n]
		toSend.WriteString("<tr class=\"" + iif(t.prot, "pr", "rw") + "\"><td><a href=\"/" + n + "\" id=\"" + n + "\">" + n + "</a></td><td>" + strconv.Itoa(t.Dims()*t.Len()) + " B</td><td>" + t.TypeString() + t.DimString() + "</td><td>" + iif(t.prot, "☐", "☑") + "</td><td>")
		var ascii strings.Builder
		if t.Type != TypeREAL && t.Type != TypeLREAL && t.Type != TypeBOOL {
			ascii.Grow(t.Dims())
			ln := t.ElemLen()
			startI := 0
			if t.Type == TypeSTRING {
				startI = 2
			}
			for i := startI; i < len(t.data); i += ln {
				tmp := int64(t.data[i])
				for j := 1; j < ln; j++ {
					tmp += int64(t.data[i+j]) << uint(8*j)
				}
				switch t.Type {
				case TypeSINT:
					tmp = int64(int8(tmp))
				case TypeINT:
					tmp = int64(int16(tmp))
				case TypeDINT:
					tmp = int64(int32(tmp))
				case TypeDWORD:
					tmp = int64(int32(tmp))
				case TypeUSINT:
					tmp = int64(uint8(tmp))
				case TypeUINT:
					tmp = int64(uint16(tmp))
				case TypeUDINT:
					tmp = int64(uint32(tmp))
				case TypeULINT:
					tmp = int64(uint64(tmp))
				}
				if tmp < 256 && tmp >= 32 {
					ascii.WriteRune(rune(tmp))
				} else {
					break
				}
			}
			toSend.WriteString(ascii.String())
			toSend.WriteString("</td></tr>\n")
		}
	}
	p.tMut.RUnlock()

	toSend.WriteString("</table></html>")

	io.WriteString(w, toSend.String())
}

type tagJSON struct {
	Typ   string    `json:"type"`
	Count int       `json:"count"`
	Data  []float64 `json:"data"`
	ASCII []string  `json:"ascii,omitempty"`
}

func tagToJSON(t *Tag) string {
	var tj tagJSON
	tj.Count = one(t.Dim[0])
	ln := t.ElemLen()
	for i := 0; i < len(t.data); i += ln {
		tmp := int64(t.data[i])
		for j := 1; j < ln; j++ {
			tmp += int64(t.data[i+j]) << uint(8*j)
		}
		switch t.Type {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			tmp = int64(int8(tmp))
		case TypeINT:
			tmp = int64(int16(tmp))
		case TypeDINT:
			tmp = int64(int32(tmp))
		case TypeDWORD:
			tmp = int64(int32(tmp))
		case TypeUSINT:
			tmp = int64(uint8(tmp))
		case TypeUINT:
			tmp = int64(uint16(tmp))
		case TypeUDINT:
			tmp = int64(uint32(tmp))
		case TypeULINT:
			tmp = int64(uint64(tmp))
		}
		if t.Type == TypeREAL {
			tj.Data = append(tj.Data, float64(math.Float32frombits(uint32(tmp))))
		} else if t.Type == TypeLREAL {
			tj.Data = append(tj.Data, math.Float64frombits(uint64(tmp)))
		} else {
			tj.Data = append(tj.Data, float64(tmp))
		}
		if t.Type != TypeREAL && t.Type != TypeLREAL && t.Type != TypeBOOL {
			if tmp <= 256 && ((t.Type == TypeSINT && tmp >= -128) || (t.Type != TypeSINT && tmp >= 0)) {
				tj.ASCII = append(tj.ASCII, asciiCode(uint8(tmp)))
			} else {
				tj.ASCII = append(tj.ASCII, "")
			}
		}
	}
	tj.Typ = t.TypeString()

	b, err := json.Marshal(tj)
	if err != nil {
		fmt.Println(err)
		return "{}"
	}
	return string(b)
}

func bytesToBinString(bs []byte) string {
	var buf strings.Builder
	for _, b := range bs {
		buf.WriteString(fmt.Sprintf("%.8b ", b))
	}
	return buf.String()
}

func hexTr(ln int) string {
	var r strings.Builder
	r.WriteString("<pre style='font-family:\"Courier New\", Courier, monospace; margin: 0px;'>")
	for i := ln; i > 0; i-- {
		r.WriteString(fmt.Sprintf("%v       ", i*8))
	}
	r.WriteString("</pre>")
	return r.String()
}

func float32ToString(f uint32) string {
	s := f >> 31
	e := (f & 0x7f800000) >> 23
	m := f & 0x007fffff
	return fmt.Sprintf("<td>%v</td><td>%v</td><td>%08b</td><td>%023b</td></tr>\n", math.Float32frombits(f), s, e, m)
}

func float64ToString(f uint64) string {
	s := f >> 63
	e := (f & 0x7FF0000000000000) >> 52
	m := f & 0xFFFFFFFFFFFFF
	return fmt.Sprintf("<td>%v</td><td>%v</td><td>%011b</td><td>%052b</td></tr>\n", math.Float64frombits(f), s, e, m)
}

func structToHTML(t *Tag, b *strings.Builder) {
	for i := 0; i < len(t.st.d); i++ {
		tmp := int64(t.data[t.st.d[i].offset])
		ln := t.st.d[i].ElemLen()
		for j := 1; j < ln; j++ {
			tmp += int64(t.data[t.st.d[i].offset+j]) << uint(8*j)
		}
		switch t.st.d[i].Type {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			tmp = int64(int8(tmp))
		case TypeINT:
			tmp = int64(int16(tmp))
		case TypeDINT:
			tmp = int64(int32(tmp))
		case TypeDWORD:
			tmp = int64(int32(tmp))
		case TypeUSINT:
			tmp = int64(uint8(tmp))
		case TypeUINT:
			tmp = int64(uint16(tmp))
		case TypeUDINT:
			tmp = int64(uint32(tmp))
		case TypeULINT:
			tmp = int64(uint64(tmp))
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%d</td></tr>", t.st.d[i].Name, t.st.d[i].TypeString(), tmp))
	}
}

func tagToHTML(t *Tag) string {
	var toSend strings.Builder

	ln := t.ElemLen()

	toSend.WriteString("<!DOCTYPE html>\n<html><style type=\"text/css\">" + mainCSS + "</style><title>" + t.Name + "</title><a href=\"/#" + t.Name + "\">powrót</a> <a href=\"\">odśwież</a><h3>" + t.Name + "</h3>")
	if t.Type > TypeStructHead { // TODO array of struct, struct in struct
		toSend.WriteString("<h4>" + t.TypeString() + "</h4><table><tr><th>Nazwa</th><th>Typ</th><th>Wartość</th></tr>")
		structToHTML(t, &toSend)
		toSend.WriteString("</table></html>")
		return toSend.String()
	}
	if t.Dim[0] > 0 {
		toSend.WriteString("<table><tr><th>N</th><th>" + t.TypeString() + "</th>")
	} else {
		toSend.WriteString("<table><tr><th>" + t.TypeString() + "</th>")
	}
	if t.Type == TypeBOOL {
		toSend.WriteString("</tr>\n")
	} else if t.Type == TypeREAL || t.Type == TypeLREAL {
		toSend.WriteString("<th>SIGN</th><th>EXPONENT</th><th>MANTISSA</th></tr>\n")
	} else {
		toSend.WriteString("<th>HEX</th><th>ASCII</th><th>BIN</th></tr>\n")
		if t.Dim[0] > 0 {
			toSend.WriteString(fmt.Sprintf("<td></td>"))
		}
		toSend.WriteString(fmt.Sprintf("<td></td><td></td><td></td><td>%s</td></tr>\n", hexTr(ln)))
	}

	n := 0
	for i := 0; i < len(t.data); i += ln {
		tmp := int64(t.data[i])
		for j := 1; j < ln; j++ {
			tmp += int64(t.data[i+j]) << uint(8*j)
		}
		hx := ""
		var err error
		buf := new(bytes.Buffer)

		switch t.Type {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			err = binary.Write(buf, binary.BigEndian, int8(tmp))
			tmp = int64(int8(tmp))
		case TypeINT:
			err = binary.Write(buf, binary.BigEndian, int16(tmp))
			tmp = int64(int16(tmp))
		case TypeDINT:
			err = binary.Write(buf, binary.BigEndian, int32(tmp))
			tmp = int64(int32(tmp))
		case TypeDWORD:
			err = binary.Write(buf, binary.BigEndian, int32(tmp))
			tmp = int64(int32(tmp))
		case TypeLINT:
			err = binary.Write(buf, binary.BigEndian, tmp)
		case TypeUSINT:
			err = binary.Write(buf, binary.BigEndian, int8(tmp))
			tmp = int64(uint8(tmp))
		case TypeUINT:
			err = binary.Write(buf, binary.BigEndian, int16(tmp))
			tmp = int64(uint16(tmp))
		case TypeUDINT:
			err = binary.Write(buf, binary.BigEndian, int32(tmp))
			tmp = int64(uint32(tmp))
		case TypeULINT:
			err = binary.Write(buf, binary.BigEndian, uint64(tmp))
		}
		if t.Dim[0] > 0 {
			toSend.WriteString(fmt.Sprintf("<td>%s</td>", t.NString(n)))
		}
		if t.Type != TypeREAL && t.Type != TypeLREAL && t.Type != TypeBOOL {
			ascii := ""
			if err == nil {
				hx = hex.EncodeToString(buf.Bytes())
			}
			bin := bytesToBinString(buf.Bytes())
			if tmp <= 256 && ((t.Type == TypeSINT && tmp >= -128) || (t.Type != TypeSINT && tmp >= 0)) {
				ascii = asciiCode(uint8(tmp))
			}
			if t.Type == TypeULINT {
				toSend.WriteString(fmt.Sprintf("<td>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>\n", uint64(tmp), hx, ascii, bin))
			} else {
				toSend.WriteString(fmt.Sprintf("<td>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>\n", tmp, hx, ascii, bin))
			}
		} else if t.Type == TypeBOOL {
			toSend.WriteString(fmt.Sprintf("<td>%v</td></tr>\n", tmp))
		} else if t.Type == TypeREAL {
			toSend.WriteString(fmt.Sprintf("%s</tr>\n", float32ToString(uint32(tmp))))
		} else if t.Type == TypeLREAL {
			toSend.WriteString(fmt.Sprintf("%s</tr>\n", float64ToString(uint64(tmp))))
		}
		n++
	}

	toSend.WriteString("</table></html>")

	return toSend.String()
}

func (p *PLC) handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		p.tagsIndexHTML(w, r)
	} else {
		p.tMut.RLock()
		t, ok := p.tags[path.Base(r.URL.Path)]
		if ok {
			_, json := r.URL.Query()["json"]
			if json {
				str := tagToJSON(t)
				p.tMut.RUnlock()
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				io.WriteString(w, str)
			} else {
				str := tagToHTML(t)
				p.tMut.RUnlock()
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				io.WriteString(w, str)
			}
		} else {
			p.tMut.RUnlock()
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "not found")
		}
	}
}

var server *http.Server

// ServeHTTP listens on the TCP network address host.
func (p *PLC) ServeHTTP(host string) *http.Server {
	server = &http.Server{Addr: host, Handler: http.HandlerFunc(p.handler)}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println("plcconnector ServeHTTP: ", err)
		}
	}()
	return server
}

// CloseHTTP shutdowns the HTTP server
func (p *PLC) CloseHTTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var err error
	if err = server.Shutdown(ctx); err != nil {
		fmt.Println("plcconnector CloseHTTP: ", err)
	}
	p.debug("server.Shutdown")
	return err
}
