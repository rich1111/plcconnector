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

const (
	tableStyle = "style='font-family:\"Courier New\", Courier, monospace; border-spacing: 20px 0;'"
)

func (p *PLC) tagsIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var toSend strings.Builder

	toSend.WriteString("<!DOCTYPE html>\n<html><title>plcconnector</title><h3>PLC connector</h3><p>Wersja: 11</p>\n<table " + tableStyle + "><tr><th>Nazwa</th><th>Rozmiar</th><th>Typ</th><th>ASCII</th></tr>\n")

	p.tMut.RLock()
	arr := make([]string, 0, len(p.tags))

	for n := range p.tags {
		arr = append(arr, n)
	}

	sort.Strings(arr)

	for _, n := range arr {
		toSend.WriteString("<tr><td><a href=\"/" + n + "\">" + n + "</a></td><td>" + strconv.Itoa(p.tags[n].Count) + "</td><td>" + typeToString(p.tags[n].Type) + "</td><td>")
		var ascii strings.Builder
		if p.tags[n].Type != TypeREAL && p.tags[n].Type != TypeBOOL {
			ascii.Grow(p.tags[n].Count)
			ln := int(typeLen(uint16(p.tags[n].Type)))
			for i := 0; i < len(p.tags[n].data); i += ln {
				tmp := int64(p.tags[n].data[i])
				for j := 1; j < ln; j++ {
					tmp += int64(p.tags[n].data[i+j]) << uint(8*j)
				}
				switch p.tags[n].Type {
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
	tj.Count = int(t.Count)
	ln := int(typeLen(uint16(t.Type)))
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
		} else {
			tj.Data = append(tj.Data, float64(tmp))
		}
		if t.Type != TypeREAL && t.Type != TypeBOOL {
			if tmp <= 256 && ((t.Type == TypeSINT && tmp >= -128) || (t.Type != TypeSINT && tmp >= 0)) {
				tj.ASCII = append(tj.ASCII, asciiCode(uint8(tmp)))
			} else {
				tj.ASCII = append(tj.ASCII, "")
			}
		}
	}
	tj.Typ = typeToString(t.Type)

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

func floatToString(f uint32) string {
	s := f >> 31
	e := (f & 0x7f800000) >> 23
	m := f & 0x007fffff
	return fmt.Sprintf("<td>%v</td><td>%v</td><td>%.8b</td><td>%.23b</td></tr>\n", math.Float32frombits(f), s, e, m)
}

func tagToHTML(t *Tag) string {
	var toSend strings.Builder

	ln := int(typeLen(uint16(t.Type)))

	toSend.WriteString("<!DOCTYPE html>\n<html><title>" + t.Name + "</title><h3>" + t.Name + "</h3>")
	toSend.WriteString("<table " + tableStyle + "><tr><th>N</th><th>" + typeToString(t.Type) + "</th>")
	if t.Type == TypeBOOL {
		toSend.WriteString("</tr>\n")
	} else if t.Type == TypeREAL {
		toSend.WriteString("<th>SIGN</th><th>EXPONENT</th><th>MANTISSA</th></tr>\n")
	} else {
		toSend.WriteString("<th>HEX</th><th>ASCII</th><th>BIN</th></tr>\n")
		toSend.WriteString(fmt.Sprintf("<td></td><td></td><td></td><td></td><td>%s</td></tr>\n", hexTr(ln)))
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
		if t.Type != TypeREAL && t.Type != TypeBOOL {
			ascii := ""
			if err == nil {
				hx = hex.EncodeToString(buf.Bytes())
			}
			bin := bytesToBinString(buf.Bytes())
			if tmp <= 256 && ((t.Type == TypeSINT && tmp >= -128) || (t.Type != TypeSINT && tmp >= 0)) {
				ascii = asciiCode(uint8(tmp))
			}
			toSend.WriteString(fmt.Sprintf("<td>%d</td><td>%v</td><td>%v</td><td>%v</td><td>%v</td></tr>\n", n, tmp, hx, ascii, bin))
		} else if t.Type == TypeBOOL {
			toSend.WriteString(fmt.Sprintf("<td>%d</td><td>%v</td></tr>\n", n, tmp))
		} else {
			toSend.WriteString(fmt.Sprintf("<td>%d</td>%s</tr>\n", n, floatToString(uint32(tmp))))
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
