package compression

import (
	"compress/gzip"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type Compression interface {
	Name() string
	Extension() string
	WrapRead(source io.Reader) (io.Reader, func())
	WrapWrite(dest io.Writer) (io.Writer, func())
}

// In order of download preference. First item is the default for uploading files.
var Compressions = []Compression{&Zstd{}, &Gzip{}, &None{}}

type None struct{}

func (n *None) Name() string                                  { return "none" }
func (n *None) Extension() string                             { return "" }
func (n *None) WrapRead(source io.Reader) (io.Reader, func()) { return source, func() {} }
func (n *None) WrapWrite(dest io.Writer) (io.Writer, func())  { return dest, func() {} }

type Gzip struct{}

func (g *Gzip) Name() string      { return "gzip" }
func (g *Gzip) Extension() string { return ".gz" }
func (g *Gzip) WrapRead(source io.Reader) (io.Reader, func()) {
	r, w := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := func() error {
			defer wg.Done()
			zip, err := gzip.NewWriterLevel(w, gzip.BestCompression)
			if err != nil {
				return err
			}

			_, err = io.Copy(zip, source)
			if err != nil {
				return err
			}

			return zip.Close()
		}(); err != nil {
			w.CloseWithError(err)
		} else {
			w.Close()
		}
	}()
	return r, func() {
		wg.Wait()
		r.Close()
	}
}
func (g *Gzip) WrapWrite(dest io.Writer) (io.Writer, func()) {
	r, w := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := func() error {
			zip, err := gzip.NewReader(r)
			if err != nil {
				return err
			}

			_, err = io.Copy(dest, zip)
			if err != nil {
				return err
			}

			return zip.Close()
		}(); err != nil {
			r.CloseWithError(err)
		} else {
			r.Close()
		}
	}()
	return w, func() {
		w.Close()
		wg.Wait()
	}
}

type Zstd struct{}

func (g *Zstd) Name() string      { return "zstd" }
func (g *Zstd) Extension() string { return ".zstd" }
func (g *Zstd) WrapRead(source io.Reader) (io.Reader, func()) {
	r, w := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := func() error {
			zip, err := zstd.NewWriter(w,
				zstd.WithEncoderLevel(zstd.SpeedBestCompression),
				zstd.WithEncoderCRC(true))
			if err != nil {
				return err
			}

			_, err = io.Copy(zip, source)
			if err != nil {
				return err
			}

			return zip.Close()
		}(); err != nil {
			w.CloseWithError(err)
		} else {
			w.Close()
		}
	}()
	return r, func() {
		wg.Wait()
		r.Close()
	}
}
func (g *Zstd) WrapWrite(dest io.Writer) (io.Writer, func()) {
	r, w := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := func() error {
			zip, err := zstd.NewReader(r)
			if err != nil {
				return err
			}
			defer zip.Close()
			_, err = io.Copy(dest, zip)
			if err != nil {
				return err
			}
			return nil
		}(); err != nil {
			r.CloseWithError(err)
		} else {
			r.Close()
		}
	}()
	return w, func() {
		w.Close()
		wg.Wait()
	}
}
