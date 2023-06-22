package proxycommands

import (
	"bytes"
	"io"

	"golang.org/x/text/transform"
)

type Mapping struct {
	From string
	To   string
}

func NewRewriter(reader io.Reader, mappings []Mapping) io.Reader {
	transformers := []transform.Transformer{}
	for _, mapping := range mappings {
		if mapping.From == "" || mapping.To == "" {
			continue
		} else if mapping.From == "/" {
			continue
		}

		transformers = append(transformers, String(mapping.From, mapping.To))
	}

	return transform.NewReader(reader, transform.Chain(transformers...))
}

// Transformer replaces text in a stream
// See: http://golang.org/x/text/transform
type Transformer struct {
	transform.NopResetter

	old, new []byte
	oldlen   int
}

var _ transform.Transformer = (*Transformer)(nil)

// Bytes returns a transformer that replaces all instances of old with new.
// Unlike bytes.Replace, empty old values don't match anything.
func Bytes(old, new []byte) Transformer {
	return Transformer{old: old, new: new, oldlen: len(old)}
}

// String returns a transformer that replaces all instances of old with new.
// Unlike strings.Replace, empty old values don't match anything.
func String(old, new string) Transformer {
	return Bytes([]byte(old), []byte(new))
}

// Transform implements golang.org/x/text/transform#Transformer
func (t Transformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	var n int
	// don't do anything for empty old string. We're forced to do this because an optimization in
	// transform.String prevents us from generating any output when the src is empty.
	// see: https://github.com/golang/text/blob/master/transform/transform.go#L570-L576
	if t.oldlen == 0 {
		n, err = fullcopy(dst, src)
		return n, n, err
	}
	// replace all instances of old with new
	for {
		i := bytes.Index(src[nSrc:], t.old)
		if i == -1 {
			break
		}
		// copy everything up to the match
		n, err = fullcopy(dst[nDst:], src[nSrc:nSrc+i])
		nSrc += n
		nDst += n
		if err != nil {
			return nSrc, nDst, err
		}
		// copy the new value
		n, err = fullcopy(dst[nDst:], t.new)
		if err != nil {
			return nSrc, nDst, err
		}
		nDst += n
		nSrc += t.oldlen
	}
	// if we're at the end, tack on any remaining bytes
	if atEOF {
		n, err = fullcopy(dst[nDst:], src[nSrc:])
		nDst += n
		nSrc += n
		return nSrc, nDst, err
	}
	// skip everything except the trailing len(r.old) - 1
	// we do this because there could be a match straddling
	// the boundary
	if skip := len(src[nSrc:]) - t.oldlen + 1; skip > 0 {
		n, err = fullcopy(dst[nDst:], src[nSrc:nSrc+skip])
		nSrc += n
		nDst += n
		if err != nil {
			return nSrc, nDst, err
		}
	}
	err = transform.ErrShortSrc
	return nDst, nSrc, err
}

func fullcopy(dst, src []byte) (n int, err error) {
	n = copy(dst, src)
	if n < len(src) {
		err = transform.ErrShortDst
	}
	return
}
