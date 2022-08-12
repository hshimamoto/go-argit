// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"time"
)

// TARFile
type TARFile struct {
	rw io.ReadWriter
	rd *tar.Reader
	wr *tar.Writer
}

func NewTARFile(rw io.ReadWriter) *TARFile {
	return &TARFile{rw: rw}
}

func (f *TARFile) WriteRegFile(path string, info os.FileInfo, rd io.Reader) error {
	if f.wr == nil {
		f.wr = tar.NewWriter(f.rw)
	}
	err := f.wr.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path,
		Size:     info.Size(),
		Mode:     0644,
		ModTime:  time.Now(),
	})
	if err != nil {
		return err
	}
	n, err := io.Copy(f.wr, rd)
	if n != info.Size() {
		return fmt.Errorf("Copied %d != %d expected", n, info.Size())
	}
	return f.wr.Flush()
}

func (f *TARFile) ReadRegFile() (*tar.Header, io.Reader, error) {
	if f.rd == nil {
		f.rd = tar.NewReader(f.rw)
	}
	for {
		hdr, err := f.rd.Next()
		if err != nil {
			return nil, nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		return hdr, f.rd, nil
	}
}
