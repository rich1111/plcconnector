// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Copyright 2020 github.com/podeszfa All rights reserved.
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

	_ "embed"
)

//go:embed web/main.css
var mainCSS string

//go:embed web/main.js
var mainJS string

//go:embed web/tag.js
var tagJS string

const version = "2021.04"

func (p *PLC) tagsIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var toSend strings.Builder

	toSend.WriteString("<!DOCTYPE html>\n<html><style>" + mainCSS + "</style><script>" + mainJS + "</script><title>" + p.Name + "</title><h3>" + p.Name + "</h3><p>Wersja biblioteki: " + version + "</p>\n<input type=checkbox id=showbtn name=showbtn><label for=showbtn>Pokaż wszystkie</label><table><tr><th>Nazwa</th><th>Rozmiar</th><th>Typ</th><th>Odczyt</th><th>ASCII</th></tr>\n")

	p.tMut.RLock()
	arr := make([]string, 0, len(p.tags))

	for _, t := range p.tags {
		arr = append(arr, t.Name)
	}

	sort.Strings(arr)

	for _, n := range arr {
		t := p.tags[strings.ToLower(n)]
		toSend.WriteString("<tr class=\"" + iif(t.prot, "pr", "rw") + "\"><td><a href=\"/" + n + "\" id=\"" + n + "\">" + n + "</a></td><td>" + strconv.Itoa(t.Dims()*t.Len()) + " B</td><td>" + t.TypeString() + t.DimString() + "</td><td>" + iif(t.prot, "☐", "☑") + "</td><td>")
		var ascii strings.Builder
		if t.BasicType() != TypeREAL && t.BasicType() != TypeLREAL && t.BasicType() != TypeBOOL {
			ascii.Grow(t.Dims())
			ln := t.ElemLen()
			startI := 0
			if t.BasicType() == TypeSTRING {
				startI = 2
			}
			for i := startI; i < len(t.data); i += ln {
				tmp := int64(t.data[i])
				for j := 1; j < ln; j++ {
					tmp += int64(t.data[i+j]) << uint(8*j)
				}
				switch t.BasicType() {
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
		switch t.BasicType() {
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
		if t.BasicType() == TypeREAL {
			tj.Data = append(tj.Data, float64(math.Float32frombits(uint32(tmp))))
		} else if t.BasicType() == TypeLREAL {
			tj.Data = append(tj.Data, math.Float64frombits(uint64(tmp)))
		} else {
			tj.Data = append(tj.Data, float64(tmp))
		}
		if t.BasicType() != TypeREAL && t.BasicType() != TypeLREAL && t.BasicType() != TypeBOOL {
			if tmp <= 256 && ((t.BasicType() == TypeSINT && tmp >= -128) || (t.BasicType() != TypeSINT && tmp >= 0)) {
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
		fmt.Fprintf(&buf, "%.8b ", b)
	}
	return buf.String()
}

func hexTr(ln int) string {
	var r strings.Builder
	r.WriteString("<pre style='font-family:\"Courier New\", Courier, monospace; margin: 0px;'>")
	for i := ln; i > 0; i-- {
		fmt.Fprintf(&r, "%v       ", i*8)
	}
	r.WriteString("</pre>")
	return r.String()
}

func float32ToString(f uint32) string {
	s := f >> 31
	e := (f & 0x7f800000) >> 23
	m := f & 0x007fffff
	return fmt.Sprintf("%v</td><td>%v</td><td>%08b</td><td>%023b</td></tr>\n", math.Float32frombits(f), s, e, m)
}

func float64ToString(f uint64) string {
	s := f >> 63
	e := (f & 0x7FF0000000000000) >> 52
	m := f & 0xFFFFFFFFFFFFF
	return fmt.Sprintf("%v</td><td>%v</td><td>%011b</td><td>%052b</td></tr>\n", math.Float64frombits(f), s, e, m)
}

func structToHTML(t *Tag, data []uint8, n int, N bool, prevName string, b *strings.Builder) {
	off := n * t.ElemLen()
	for i := 0; i < len(t.st.d); i++ {
		if strings.HasPrefix(t.st.d[i].Name, "ZZZZZZZZZZ") {
			continue
		}
		var val strings.Builder
		ln := t.st.d[i].ElemLen()
		if t.st.d[i].Type > TypeStructHead {
			if t.st.d[i].Dim[0] > 0 {
				val.WriteString("<td><table><tr><th>N</th><th>Nazwa</th><th>Typ</th><th>Wartość</th></tr>")
				for k := 0; k < t.st.d[i].Dim[0]; k++ {
					structToHTML(&t.st.d[i], data[t.st.d[i].offset+off:t.st.d[i].offset+off+ln*t.st.d[i].Dim[0]], k, true, t.PathString(n), &val)
				}
			} else {
				val.WriteString("<td><table><tr><th>Nazwa</th><th>Typ</th><th>Wartość</th></tr>")
				structToHTML(&t.st.d[i], data[t.st.d[i].offset+off:t.st.d[i].offset+off+ln], 0, false, t.PathString(n), &val)
			}
			val.WriteString("</table>")
		} else if t.st.d[i].BasicType() == TypeBOOL { // FIXME: array of BOOL
			clic := prevName + "." + t.PathString(n) + "." + t.st.d[i].Name
			fmt.Fprintf(&val, "<td onclick=clicBOOL(event) class=clic tag='%s'>", clic)
			if (data[t.st.d[i].offset+off]>>t.st.d[i].Dim[0])&1 == 1 {
				val.WriteString("1")
			} else {
				val.WriteString("0")
			}
		} else {
			val.WriteString("<td>")
			for x := 0; x < t.st.d[i].Dims(); x++ {
				if x != 0 {
					val.WriteString(", ")
				}
				tmp := int64(data[t.st.d[i].offset+off+ln*x])
				for j := 1; j < ln; j++ {
					tmp += int64(data[t.st.d[i].offset+off+ln*x+j]) << uint(8*j)
				}
				if t.st.d[i].BasicType() == TypeREAL {
					fmt.Fprintf(&val, "%v", math.Float32frombits(uint32(tmp)))
				} else if t.st.d[i].BasicType() == TypeLREAL {
					fmt.Fprintf(&val, "%v", math.Float64frombits(uint64(tmp)))
				} else {
					switch t.st.d[i].BasicType() {
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
					fmt.Fprintf(&val, "%v", tmp)
				}
			}
		}
		b.WriteString("<tr><td>")
		if N {
			fmt.Fprintf(b, "%s</td><td>", t.NString(n))
		}
		fmt.Fprintf(b, "%s</td><td>%s</td>%s</td></tr>", t.st.d[i].Name, t.st.d[i].TypeString()+t.st.d[i].DimString(), val.String())
	}
	if len(t.st.d) == 2 && strings.EqualFold(t.st.d[0].Name, "len") && strings.EqualFold(t.st.d[1].Name, "data") {
		b.WriteString("<tr><td>")
		if N {
			b.WriteString("</td><td>")
		}
		strLen := (int(data[t.st.d[0].offset+off+3]) << 24) + (int(data[t.st.d[0].offset+off+2]) << 16) + (int(data[t.st.d[0].offset+off+1]) << 8) + int(data[t.st.d[0].offset+off])
		fmt.Fprintf(b, "</td><td><td>ASCII: %s</td></tr>", string(data[t.st.d[1].offset+off:t.st.d[1].offset+off+strLen]))
	}
	if N {
		b.WriteString(`<tr style="height: 25px;"/>`)
	}
}

func tagToHTML(t *Tag) string {
	var toSend strings.Builder

	ln := t.ElemLen()

	toSend.WriteString("<!DOCTYPE html>\n<html><style>" + mainCSS + "</style><script>" + tagJS + "</script><title>" + t.Name + "</title><a href=\"/#" + t.Name + "\">powrót</a> <a href=\"\">odśwież</a><h3>" + t.Name + "</h3>")
	if t.Type > TypeStructHead {
		if t.Dim[0] > 0 {
			toSend.WriteString("<h4>" + t.TypeString() + t.DimString() + "</h4><table><tr><th>N</th><th>Nazwa</th><th>Typ</th><th>Wartość</th></tr>")
			for i := 0; i < len(t.data)/ln; i++ {
				structToHTML(t, t.data, i, true, "", &toSend)
			}
		} else {
			toSend.WriteString("<h4>" + t.TypeString() + "</h4><table><tr><th>Nazwa</th><th>Typ</th><th>Wartość</th></tr>")
			structToHTML(t, t.data, 0, false, "", &toSend)
		}
		toSend.WriteString("</table></html>")
		return toSend.String()
	}
	if t.Dim[0] > 0 {
		toSend.WriteString("<table><tr><th>N</th><th>" + t.TypeString() + "</th>")
	} else {
		toSend.WriteString("<table><tr><th>" + t.TypeString() + "</th>")
	}
	if t.BasicType() == TypeBOOL {
		toSend.WriteString("</tr>\n")
	} else if t.BasicType() == TypeREAL || t.BasicType() == TypeLREAL {
		toSend.WriteString("<th>SIGN</th><th>EXPONENT</th><th>MANTISSA</th></tr>\n")
	} else {
		toSend.WriteString("<th>HEX</th><th>ASCII</th><th>BIN</th></tr>\n")
		if t.Dim[0] > 0 {
			toSend.WriteString("<td></td>")
		}
		fmt.Fprintf(&toSend, "<td></td><td></td><td></td><td>%s</td></tr>\n", hexTr(ln))
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

		switch t.BasicType() {
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
		toSend.WriteString("<tr>")
		if t.Dim[0] > 0 {
			fmt.Fprintf(&toSend, "<td>%s</td>", t.NString(n))
		}
		if t.BasicType() != TypeREAL && t.BasicType() != TypeLREAL && t.BasicType() != TypeBOOL {
			ascii := ""
			if err == nil {
				hx = hex.EncodeToString(buf.Bytes())
			}
			bin := bytesToBinString(buf.Bytes())
			if tmp <= 256 && ((t.BasicType() == TypeSINT && tmp >= -128) || (t.BasicType() != TypeSINT && tmp >= 0)) {
				ascii = asciiCode(uint8(tmp))
			}
			if t.BasicType() == TypeULINT {
				fmt.Fprintf(&toSend, "<td onclick=clicINT(event) class=clic tag='%s' size='%d'>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>\n", t.PathString(n), typeLen(uint16(t.Type)), uint64(tmp), hx, ascii, bin)
			} else {
				fmt.Fprintf(&toSend, "<td onclick=clicINT(event) class=clic tag='%s' size='%d'>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>\n", t.PathString(n), typeLen(uint16(t.Type)), tmp, hx, ascii, bin)
			}
		} else if t.BasicType() == TypeBOOL {
			fmt.Fprintf(&toSend, "<td onclick=clicBOOL(event) class=clic tag='%s'>%v</td></tr>\n", t.PathString(n), tmp)
		} else if t.BasicType() == TypeREAL {
			fmt.Fprintf(&toSend, "<td onclick=clicREAL(event) class=clic tag='%s' size='4'>%s</tr>\n", t.PathString(n), float32ToString(uint32(tmp)))
		} else if t.BasicType() == TypeLREAL {
			fmt.Fprintf(&toSend, "<td onclick=clicREAL(event) class=clic tag='%s' size='8'>%s</tr>\n", t.PathString(n), float64ToString(uint64(tmp)))
		}
		n++
	}

	toSend.WriteString("</table></html>")

	return toSend.String()
}

func (p *PLC) handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		p.tagsIndexHTML(w, r)
	} else if r.URL.Path == "/.tagSet" && r.Method == http.MethodPost {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "tagSet error", http.StatusBadRequest)
			return
		}
		ps := string(b)
		ind := strings.Index(ps, "=")
		name := strings.TrimSpace(ps[:ind])
		pth := parsePath(name)

		val := strings.FieldsFunc(strings.TrimSpace(ps[ind+1:]), func(r rune) bool { return r == ',' })
		arr := make([]byte, len(val))

		for i, s := range val {
			x, _ := strconv.Atoi(strings.TrimSpace(s))
			arr[i] = byte(x)
		}

		ok := p.saveTag(pth, 0, 0, arr, 0)

		if ok {
			io.WriteString(w, "ok")
		} else {
			io.WriteString(w, "fail")
		}
		// fmt.Println(ps, pth, ok, len(arr))
	} else {
		p.tMut.RLock()
		t, ok := p.tags[strings.ToLower(path.Base(r.URL.Path))]
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
