package sourcemap

import (
	"bufio"
	"bytes"
	"gopkg.in/sourcemap.v1/base64vlq"
	"io"
	"strings"
)

type (
	Map struct {
		Version         int      `json:"version"`
		File            string   `json:"file,omitempty"`
		SourceRoot      string   `json:"sourceRoot,omitempty"`
		Sources         []string `json:"sources,omitempty"`
		SourcesContent  []string `json:"sourcesContent,omitempty"`
		Names           []string `json:"names,omitempty"`
		Mappings        string   `json:"mappings"`
		decodedMappings []Mapping
	}
	Mapping struct {
		GeneratedLine   int
		GeneratedColumn int
		SourceIndex     int
		SourceLine      int
		SourceColumn    int
		NameIndex       int
	}
)

func encodeMappings(ms []Mapping, includeNames bool) string {
	var buf bytes.Buffer
	enc := base64vlq.NewEncoder(&buf)
	var lastM Mapping
	var lastGC int
	for i, m := range ms {
		if i > 0 {
			if m.GeneratedLine != lastM.GeneratedLine {
				buf.WriteByte(';')
				lastGC = 0
			} else {
				buf.WriteByte(',')
			}
		}
		enc.Encode(m.GeneratedColumn - lastGC)
		enc.Encode(m.SourceIndex - lastM.SourceIndex)
		enc.Encode(m.SourceLine - lastM.SourceLine)
		enc.Encode(m.SourceColumn - lastM.SourceColumn)
		if includeNames {
			enc.Encode(m.NameIndex - lastM.NameIndex)
		}
		lastM = m
		lastGC = m.GeneratedColumn
	}
	return buf.String()
}

func decodeMappings(raw string) []Mapping {
	ms := make([]Mapping, 0)
	var lastM Mapping
	for i, segments := range strings.Split(raw, ";") {
		lastGC := 0
		for _, segment := range strings.Split(segments, ",") {
			dec := base64vlq.NewDecoder(strings.NewReader(segment))
			m := Mapping{
				GeneratedLine: i,
			}
			gc, err := dec.Decode()
			if err == nil {
				m.GeneratedColumn = lastGC + gc
			}
			si, err := dec.Decode()
			if err == nil {
				m.SourceIndex = lastM.SourceIndex + si
			}
			sl, err := dec.Decode()
			if err == nil {
				m.SourceLine = lastM.SourceLine + sl
			}
			sc, err := dec.Decode()
			if err == nil {
				m.SourceColumn = lastM.SourceColumn + sc
			}
			ni, err := dec.Decode()
			if err == nil {
				m.NameIndex = lastM.NameIndex + ni
			}
			ms = append(ms, m)
			lastGC = m.GeneratedColumn
			lastM = m
		}
	}
	return ms
}

func (m *Map) DecodedMappings() []Mapping {
	if m.decodedMappings == nil {
		m.decodedMappings = decodeMappings(m.Mappings)
	}
	return m.decodedMappings
}

func Generate(name string, r io.Reader) (*Map, error) {
	m := &Map{
		Version:         3,
		File:            name,
		Sources:         []string{name},
		SourcesContent:  []string{""},
		decodedMappings: []Mapping{},
	}

	i := 0
	br := bufio.NewReader(r)
	for {
		bs, err := br.ReadSlice('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		m.SourcesContent[0] += string(bs)
		m.decodedMappings = append(m.decodedMappings, Mapping{
			GeneratedLine:   i,
			GeneratedColumn: 0,
			SourceIndex:     0,
			SourceLine:      i,
			SourceColumn:    0,
		})
		if err == io.EOF {
			break
		}
		i++
	}
	m.Mappings = encodeMappings(m.decodedMappings, false)
	return m, nil
}

func Merge(name string, maps ...*Map) *Map {
	m := &Map{
		Version:         3,
		File:            name,
		Sources:         []string{},
		SourcesContent:  []string{},
		decodedMappings: []Mapping{},
	}

	sourceIndexOffset := 0
	for _, tm := range maps {
		m.Sources = append(m.Sources, tm.Sources...)
		m.SourcesContent = append(m.SourcesContent, tm.SourcesContent...)
		ms := tm.DecodedMappings()
		for _, mm := range ms {
			mm.SourceIndex += sourceIndexOffset
			m.decodedMappings = append(m.decodedMappings, mm)
		}
		sourceIndexOffset += len(m.Sources)
	}
	m.Mappings = encodeMappings(m.decodedMappings, false)
	return m
}
