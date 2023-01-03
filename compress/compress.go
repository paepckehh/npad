// package compress ...
package compress

// import
import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// const
const (
	_dot   = "."
	_empty = ""
)

//
// EXTERNAL INTERFACE
//

// Compress ..
func Compress(algo string, level int, in []byte) []byte {
	switch algo {
	case "ZSTD", "GZIP", "DEFLATE":
		return compressGO(algo, level, in)
	}
	errOut("internal error: unsupported compress algo [" + algo + "]")
	return nil
}

// Decompress ...
func Decompress(algo string, in []byte) []byte {
	switch algo {
	case "ZSTD", "GZIP", "DEFLATE":
		return decompressGO(algo, in)
	}
	errOut("internal error: unsupported decompress algo [" + algo + "]")
	return nil
}

// GetFileExtension ...
func GetFileExtension(algo string) string {
	switch algo {
	case "ZSTD":
		return ".zst"
	case "GZIP":
		return ".gz"
	case "XZ":
		return ".xz"
	case "LZ4":
		return ".lz4"
	case "BR":
		return ".br"
	case "DEFLATE":
		return ".deflate"
	case _empty:
		return _empty
	}
	errOut("unsupported file compression algo [" + algo + "]")
	return _empty
}

// GetFileAlgo ...
func GetFileAlgo(filename string) string {
	s := strings.Split(filename, ".")
	l := len(s)
	if l < 1 {
		errOut("[decompress] [" + filename + "] [unknown extension]")
		return _empty
	}
	switch s[l-1] {
	case "zst", "zstd":
		return "ZSTD"
	case "gz":
		return "GZIP"
	case "xz", "txz":
		return "XZ"
	case "lz4":
		return "LZ4"
	case "br":
		return "BR"
	case "deflate":
		return "DEFLATE"
	case _empty:
		return _empty
	}
	errOut("[decompress] [" + filename + "] [" + s[l-1] + "] [unknown extension]")
	return _empty
}

// ReadFile ...
func ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, errors.New("[compress] [scanner] unable to write file [" + name + "] [unknown extension]")
	}
	return Decompress(GetFileAlgo(name), data), nil
}

// WriteFile ...
func WriteFile(name string, data []byte, perm os.FileMode, level int) error {
	s := strings.Split(name, ".")
	l := len(s)
	if l < 1 {
		return errors.New("[compress] [scanner] unable to write file [" + name + "] [unknown extension]")
	}
	return os.WriteFile(name, Compress(GetFileAlgo(s[l-1]), level, data), perm)
}

// GetReader ...
func GetReader(name string) (io.Reader, error) {
	var r io.Reader
	l := len(name)
	if l < 3 {
		return r, errors.New("[compress] [scanner] unable to read file [" + name + "] [unknown extension]")
	}
	f, err := os.Open(name)
	if err != nil {
		return r, errors.New("[compress] [scanner] unable to read file [" + name + "] [" + err.Error() + "]")
	}
	switch name[len(name)-3:] {
	case "zst":
		r, err = zstd.NewReader(f)
	case ".gz":
		r, err = gzip.NewReader(f)
	case "tsv", "txt", "csv":
		r = f
	default:
		return r, errors.New("[compress] [internal] unsupported format")
	}
	if err != nil {
		return r, errors.New("[compress] [scanner] unable to read file [" + name + "] [" + err.Error() + "]")
	}
	return r, nil
}

// GetFileScanner ...
func GetFileScanner(name string) (s *bufio.Scanner, err error) {
	r, err := GetReader(name)
	if err != nil {
		return s, errors.New("[compress] [scanner] unable to read file [" + name + "] [" + err.Error() + "]")
	}
	return bufio.NewScanner(r), nil
}

// WriteTransportCompressedPage ...
func WriteTransportCompressedPage(page string, r http.ResponseWriter, q *http.Request, tryCompress bool) {
	p := []byte(page)
	var err error
	if tryCompress && len(page) > 1400 { // skip compression attempt if data fits uncompressed into one TCP frame [MTU]
		// compression auto-negoation optimized for size  [zstd|br|gzip]
		accept := strings.Join(q.Header["Accept-Encoding"], " ")
		switch {
		case strings.Contains(accept, "zstd"):
			r.Header().Set("Content-Encoding", "zstd")
			p = Compress("ZSTD", 19, p)
		case strings.Contains(accept, "gzip"):
			r.Header().Set("Content-Encoding", "gzip")
			p = Compress("GZIP", 9, p)
		case strings.Contains(accept, "deflate"):
			r.Header().Set("Content-Encoding", "deflate")
			p = Compress("DEFLATE", 9, p)
		}
		_, err = fmt.Fprint(r, string(p))
	} else {
		_, err = fmt.Fprint(r, page)
	}
	if err != nil {
		errOut("[handler] [out] [" + err.Error() + "]")
	}
}

//
// INTERNAL BACKENDS: NATIVE GO
//

func decompressGO(algo string, data []byte) []byte {
	if algo == "" {
		return data
	}
	var err error
	var r io.Reader
	br := bytes.NewReader(data)
	switch algo {
	case "ZSTD":
		r, err = zstd.NewReader(br)
	case "GZIP":
		r, err = gzip.NewReader(br)
	case "DEFLATE":
		r, err = zlib.NewReader(br)
	default:
		errOut("unsupported de-compress algo [" + algo + "]")
		return nil
	}
	if err != nil {
		errOut("unable to create new de-compress reader [" + algo + "]")
		return nil
	}
	out, err := io.ReadAll(r)
	if err != nil {
		errOut("[decompress] [" + algo + "] [" + err.Error() + "]")
		return nil
	}
	return out
}

func compressGO(algo string, level int, data []byte) []byte {
	if algo == "" || level == 0 {
		return data
	}
	var buf bytes.Buffer
	switch algo {
	case "ZSTD":
		if level > 19 {
			level = 19
		}
		w, err := zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
			zstd.WithEncoderCRC(false),
			zstd.WithZeroFrames(false),
			zstd.WithSingleSegment(true),
			zstd.WithLowerEncoderMem(false),
			zstd.WithAllLitEntropyCompression(true),
			zstd.WithNoEntropyCompression(false))
		if err != nil {
			errOut("unable to create new zstd writer [" + err.Error() + "]")
			return nil
		}
		out := w.EncodeAll(data, nil)
		w.Close()
		return out
	case "GZIP":
		if level > 9 {
			level = 9
		}
		w, err := gzip.NewWriterLevel(&buf, level)
		if err != nil {
			errOut("unable to create new gzip writer [" + err.Error() + "]")
			return nil
		}
		if _, err = w.Write(data); err != nil {
			errOut("unable to write via gzip writer [" + err.Error() + "]")
			return nil
		}
		w.Close()
	case "DEFLATE":
		if level > 9 {
			level = 9
		}
		w, err := zlib.NewWriterLevel(&buf, level)
		if err != nil {
			errOut("unable to create new deflate writer [" + err.Error() + "]")
			return nil
		}
		if _, err = w.Write(data); err != nil {
			errOut("unable to write via deflate writer [" + err.Error() + "]")
			return nil
		}
		w.Close()
	default:
		errOut("unsupported compression algo [requested:" + algo + "]")
		return nil
	}
	r := io.Reader(&buf)
	out, err := io.ReadAll(r)
	if err != nil {
		errOut("[compress] [algo:" + algo + "] [" + err.Error() + "]")
		return nil
	}
	return out
}

//
// INTERNAL BACKENDS: CGO BINDINGS
//

//
// INTERNAL BACKENDS: CMD PIPE WRAPPER
//

//
// LITTLE HELPER
//

// out ...
func out(in string) { os.Stdout.Write([]byte(in)) }

// errOut ...
func errOut(in string) { out("[compress] [error] " + in) }
