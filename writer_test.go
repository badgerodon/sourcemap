package sourcemap

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	assert := assert.New(t)

	f1 := "alert('f1-1');\nalert('f1-2');\nalert('f1-3');\n;\nalert('f1-4');\n\n"
	f2 := "alert('f2-1');\nalert('f2-2');\n"

	var out bytes.Buffer

	w, err := NewWriter(&out, "out.js", []string{"f1.js", "f2.js"})
	assert.Nil(err)

	for i, f := range []string{f1, f2} {
		w.NextSource()
		ln := 0
		sf := bufio.NewScanner(strings.NewReader(f))
		for sf.Scan() {
			w.WriteSourceLine(sf.Text())
			w.WriteGeneratedLine(Segment{0, i, ln, 0})
			ln++
		}
		w.WriteSourceLine("")
		w.WriteGeneratedLine(Segment{0, i, ln, 0})
	}
	w.Close()

	type Result struct {
		Mappings string `json:"mappings"`
	}
	var result Result
	err = json.NewDecoder(&out).Decode(&result)
	assert.Nil(err)
	assert.Equal("AAAA;AACA;AACA;AACA;AACA;AACA;AACA;ACNA;AACA;AACA", result.Mappings)
}
