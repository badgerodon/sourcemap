package sourcemap

import (
	"encoding/json"
	"gopkg.in/sourcemap.v1/base64vlq"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type (
	SourceMap struct {
		srcName        string
		File           string
		Mappings       [][]Segment
		Sources        []string
		SourcesContent []string
		Names          []string
	}
)

func Read(srcName string) (*SourceMap, error) {
	sm := &SourceMap{srcName: srcName}
	bs, err := ioutil.ReadFile(srcName)
	if err != nil {
		return nil, err
	}
	return sm, sm.UnmarshalJSON(bs)
}

func (sm *SourceMap) UnmarshalJSON(data []byte) error {
	type Raw struct {
		Version        int      `json:"version"`
		File           string   `json:"file,omitempty"`
		Mappings       string   `json:"mappings"`
		Sources        []string `json:"sources,omitempty"`
		SourceRoot     string   `json:"sourceRoot,omitempty"`
		SourcesContent []string `json:"sourcesContent,omitempty"`
		Names          []string `json:"names,omitempty"`
	}
	var raw Raw
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	sm.File = raw.File
	sm.Sources = raw.Sources
	sm.SourcesContent = raw.SourcesContent
	if sm.SourcesContent == nil {
		sm.SourcesContent = make([]string, len(sm.Sources))
		for i, s := range sm.Sources {
			fn := filepath.Join(filepath.Dir(sm.srcName), "./"+raw.SourceRoot+s)
			bs, err := ioutil.ReadFile(fn)
			if err != nil {
				return err
			}
			sm.SourcesContent[i] = string(bs)
		}
	}
	sm.Names = raw.Names

	lines := strings.Split(raw.Mappings, ";")
	lastSegment := Segment{}
	for i, line := range lines {
		segments := strings.Split(line, ",")
		lastGC := 0
		ss := []Segment{}
		for _, segment := range segments {
			s := Segment{
				GeneratedLine:   i,
				GeneratedColumn: 0,
				SourceIndex:     0,
				SourceLine:      0,
				SourceColumn:    0,
				NameIndex:       0,
			}

			d := base64vlq.NewDecoder(strings.NewReader(segment))
			gc, err := d.Decode()
			if err == nil {
				s.GeneratedColumn = lastGC + gc
			}
			si, err := d.Decode()
			if err == nil {
				s.SourceIndex = lastSegment.SourceIndex + si
			}
			sl, err := d.Decode()
			if err == nil {
				s.SourceLine = lastSegment.SourceLine + sl
			}
			sc, err := d.Decode()
			if err == nil {
				s.SourceColumn = lastSegment.SourceColumn + sc
			}
			ni, err := d.Decode()
			if err == nil {
				s.NameIndex = lastSegment.NameIndex + ni
			}

			ss = append(ss, s)
			lastGC = s.GeneratedColumn
			lastSegment = s
		}
		sm.Mappings = append(sm.Mappings, ss)
	}
	return nil
}
