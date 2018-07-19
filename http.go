// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package plcconnector

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
)

func typeToString(t uint16) string {
	switch t {
	case TypeBOOL:
		return "BOOL"
	case TypeSINT:
		return "SINT"
	case TypeINT:
		return "INT"
	case TypeDINT:
		return "DINT"
	case TypeREAL:
		return "REAL"
	case TypeDWORD:
		return "DWORD"
	case TypeLINT:
		return "LINT"
	default:
		return "UNKNOWN"
	}
}

func tagsIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var toSend strings.Builder

	toSend.WriteString("<!DOCTYPE html>\n<html><h3>PLC connector</h3><p>Wersja: 1</p>\n<table><tr><th>Nazwa</th><th>Rozmiar</th><th>Typ</th></tr>\n")

	tMut.RLock()
	arr := make([]string, 0, len(tags))

	for n := range tags {
		arr = append(arr, n)
	}

	sort.Strings(arr)

	for _, n := range arr {
		toSend.WriteString("<tr><td><a href=\"/" + n + "\">" + n + "</a></td><td>" + strconv.Itoa(int(tags[n].Count)) + "</td><td>" + typeToString(tags[n].Typ) + "</td></tr>\n")
	}
	tMut.RUnlock()

	toSend.WriteString("</table></html>")

	io.WriteString(w, toSend.String())
}

type tagJSON struct {
	Typ   string    `json:"type"`
	Count int       `json:"count"`
	Data  []float64 `json:"data"`
}

func tagToJSON(t *Tag) string {
	var tj tagJSON
	tj.Count = int(t.Count)
	ln := int(typeLen(t.Typ))
	for i := 0; i < len(t.Data); i += ln {
		tmp := int(t.Data[i])
		for j := 1; j < ln; j++ {
			tmp += int(t.Data[i+j]) << uint(8*j)
		}
		switch t.Typ {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			tmp = int(int8(tmp))
		case TypeINT:
			tmp = int(int16(tmp))
		case TypeDINT:
			tmp = int(int32(tmp))
		case TypeDWORD:
			tmp = int(int32(tmp))
		case TypeLINT:
			tmp = int(int64(tmp))
		}
		if t.Typ == TypeREAL {
			tj.Data = append(tj.Data, float64(math.Float32frombits(uint32(tmp))))
		} else {
			tj.Data = append(tj.Data, float64(tmp))
		}
	}
	tj.Typ = typeToString(t.Typ)

	b, err := json.Marshal(tj)
	if err != nil {
		fmt.Println(err)
		return "{}"
	}
	return string(b)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		tagsIndexHTML(w, r)
	} else {
		tMut.RLock()
		t, ok := tags[path.Base(r.URL.Path)]
		if ok {
			str := tagToJSON(&t)
			tMut.RUnlock()
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			io.WriteString(w, str)
		} else {
			tMut.RUnlock()
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "not found")
		}
	}
}

// ServeHTTP listens on the TCP network address host.
func ServeHTTP(host string) error {
	return http.ListenAndServe(host, http.HandlerFunc(handler))
}
