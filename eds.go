package plcconnector

import (
	"io/ioutil"
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

// TODO fix gz
func (p *PLC) loadEDS(fn string) error {
	f, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	// var buf bytes.Buffer
	// w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	// if err != nil {
	// 	return err
	// }
	// _, err = w.Write(f)
	// if err != nil {
	// 	return err
	// }
	// err = w.Close()
	// if err != nil {
	// 	return err
	// }

	p.Class[1] = defaultIdentityClass() // TODO parse eds

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
	in.Attr[5] = AttrUINT(1+1<<8, "FileRevision") // TODO parse
	// in.Attr[6] = AttrUDINT(uint32(buf.Len()), "FileSize")
	in.Attr[6] = AttrUDINT(uint32(len(f)), "FileSize")
	in.Attr[7] = AttrINT(int16(chksum), "FileChecksum")
	in.Attr[8] = AttrUSINT(255, "InvocationMethod")  // not aplicable
	in.Attr[9] = AttrUSINT(1, "FileSaveParameters")  // BYTE
	in.Attr[10] = AttrUSINT(1, "FileType")           // read only
	in.Attr[11] = AttrUSINT(0, "FileEncodingFormat") // compressed

	p.Class[0x37].Inst[0xC8] = in
	return nil
}
