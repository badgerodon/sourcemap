package sourcemap

import (
	"bufio"
	"bytes"
	"encoding/json"
	"gopkg.in/sourcemap.v1/base64vlq"
	"io"
	"strings"
)

type (
	RuneReader interface {
		ReadRune() (r rune, size int, err error)
	}
	RuneWriter interface {
		WriteRune(r rune) (size int, err error)
	}

	Writer struct {
		underlying         io.Writer
		dst                *bufio.Writer
		mappings           *bufio.Writer
		underlyingMappings *bytes.Buffer
		lastSegment        Segment
		file               string
		sources            []string
		sourceIndex        int
		lineCount          int
	}
	Segment struct {
		GeneratedColumn int
		SourceIndex     int
		SourceLine      int
		SourceColumn    int
	}
)

var (
	hexc        = "0123456789abcdef"
	ZeroSegment Segment
)

func encodeString(rw RuneWriter, rr RuneReader) error {
	for {
		r, _, err := rr.ReadRune()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		switch r {
		case '\\', '"':
			rw.WriteRune('\\')
			_, err = rw.WriteRune(r)
		case '\n':
			rw.WriteRune('\\')
			_, err = rw.WriteRune('n')
		case '\r':
			rw.WriteRune('\\')
			_, err = rw.WriteRune('r')
		case '\t':
			rw.WriteRune('\\')
			_, err = rw.WriteRune('t')
		case '\u2028':
			rw.WriteRune('\\')
			rw.WriteRune('u')
			rw.WriteRune('2')
			rw.WriteRune('0')
			rw.WriteRune('2')
			_, err = rw.WriteRune('8')
		case '\u2029':
			rw.WriteRune('\\')
			rw.WriteRune('u')
			rw.WriteRune('2')
			rw.WriteRune('0')
			rw.WriteRune('2')
			_, err = rw.WriteRune('9')
		default:
			if r < 0x20 {
				rw.WriteRune('\\')
				rw.WriteRune('u')
				rw.WriteRune('0')
				rw.WriteRune('0')
				rw.WriteRune(rune(hexc[byte(r)>>4]))
				_, err = rw.WriteRune(rune(hexc[byte(r)&0xF]))
			} else {
				_, err = rw.WriteRune(r)
			}
		}

		if err != nil {
			return err
		}
	}
}

func NewWriter(w io.Writer, file string, sources []string) (*Writer, error) {
	var um bytes.Buffer
	sm := &Writer{
		underlying:         w,
		dst:                bufio.NewWriter(w),
		mappings:           bufio.NewWriter(&um),
		underlyingMappings: &um,
		file:               file,
		sources:            sources,
	}
	return sm, sm.writeHeader()
}

func (w *Writer) Close() error {
	w.mappings.Flush()
	w.writeFooter()
	return w.dst.Flush()
}

func (w *Writer) writeHeader() error {
	_, err := io.WriteString(w.dst, `{ "version": 3, "file": `)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w.dst).Encode(w.file)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w.dst, `, "sources": `)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w.dst).Encode(w.sources)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w.dst, `, "sourcesContent": [`)
	if err != nil {
		return err
	}
	return nil
}

func (w *Writer) writeFooter() error {
	if len(w.sources) > 0 {
		_, err := io.WriteString(w.dst, `"`)
		if err != nil {
			return err
		}
	}
	_, err := io.WriteString(w.dst, `], "mappings": "`)
	if err != nil {
		return err
	}
	err = encodeString(w.dst, w.underlyingMappings)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w.dst, `" }`)
	if err != nil {
		return err
	}
	return nil
}

func (w *Writer) NextSource() error {
	var err error
	if w.sourceIndex == 0 {
		_, err = io.WriteString(w.dst, `"`)
	} else {
		_, err = io.WriteString(w.dst, `","`)
	}
	w.sourceIndex++
	return err
}

func (w *Writer) WriteSourceLine(line string) error {
	rr := bufio.NewReader(strings.NewReader(line))
	err := encodeString(w.dst, rr)
	if err != nil {
		return err
	}
	_, err = w.dst.WriteString("\\n")
	return err
}

func (w *Writer) WriteGeneratedLine(segments ...Segment) error {
	if w.lineCount > 0 {
		err := w.mappings.WriteByte(';')
		if err != nil {
			return err
		}
	}
	enc := base64vlq.NewEncoder(w.mappings)
	for i, segment := range segments {
		if i > 0 {
			w.mappings.WriteByte(',')
		}
		enc.Encode(segment.GeneratedColumn - segments[0].GeneratedColumn)
		enc.Encode(segment.SourceIndex - w.lastSegment.SourceIndex)
		enc.Encode(segment.SourceLine - w.lastSegment.SourceLine)
		enc.Encode(segment.SourceColumn - w.lastSegment.SourceColumn)
		w.lastSegment = segment
	}
	w.lineCount++
	return nil
}
