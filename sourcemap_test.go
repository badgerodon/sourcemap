package sourcemap

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestSourceMap(t *testing.T) {
	assert := assert.New(t)

	m1, err := Generate("f1.js", strings.NewReader("alert('f1-1');\nalert('f1-2');\nalert('f1-3');\n;\nalert('f1-4');\n\n"))
	assert.Nil(err)

	m2, err := Generate("f2.js", strings.NewReader("alert('f2-1');\nalert('f2-2');\n"))
	assert.Nil(err)

	m3 := Merge("combined.js", m1, m2)

	assert.Equal("AAAA;AACA;AACA;AACA;AACA;AACA;AACA;ACNA;AACA;AACA", m3.Mappings)
}
