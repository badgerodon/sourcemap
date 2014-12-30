package sourcemap

import (
	"bufio"
	"bytes"
	"encoding/json"
	"gopkg.in/sourcemap.v1/base64vlq"
	"io"
	"os"
	"strings"
)

type (
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
)

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
	gc := 0
	for i, segment := range segments {
		if i > 0 {
			w.mappings.WriteByte(',')
		}
		enc.Encode(segment.GeneratedColumn - gc)
		enc.Encode(segment.SourceIndex - w.lastSegment.SourceIndex)
		enc.Encode(segment.SourceLine - w.lastSegment.SourceLine)
		enc.Encode(segment.SourceColumn - w.lastSegment.SourceColumn)
		w.lastSegment = segment
		gc = segment.GeneratedColumn
	}
	w.lineCount++
	return nil
}

func Write(dstName string, sm *SourceMap) error {
	fw, err := os.Create(dstName)
	if err != nil {
		return err
	}
	defer fw.Close()

	type Raw struct {
		Version        int      `json:"version"`
		File           string   `json:"file,omitempty"`
		Mappings       string   `json:"mappings"`
		Sources        []string `json:"sources,omitempty"`
		SourcesContent []string `json:"sourcesContent,omitempty"`
		Names          []string `json:"names,omitempty"`
	}
	raw := Raw{
		Version:        3,
		File:           sm.File,
		Mappings:       "",
		Sources:        sm.Sources,
		SourcesContent: sm.SourcesContent,
		Names:          sm.Names,
	}
	var buf bytes.Buffer
	var lastSegment Segment
	enc := base64vlq.NewEncoder(&buf)
	for i, m := range sm.Mappings {
		if i > 0 {
			buf.WriteByte(';')
		}
		lastGC := 0
		for j, s := range m {
			if j > 0 {
				buf.WriteByte(',')
			}
			enc.Encode(s.GeneratedColumn - lastGC)
			enc.Encode(s.SourceIndex - lastSegment.SourceIndex)
			enc.Encode(s.SourceLine - lastSegment.SourceLine)
			enc.Encode(s.SourceColumn - lastSegment.SourceColumn)
			if len(raw.Names) > 0 {
				enc.Encode(s.NameIndex - lastSegment.NameIndex)
			}
			lastGC = s.GeneratedColumn
			lastSegment = s
		}
	}
	raw.Mappings = buf.String()

	return json.NewEncoder(fw).Encode(raw)
}
