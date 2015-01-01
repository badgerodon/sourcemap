package sourcemap

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceMap(t *testing.T) {
	assert := assert.New(t)

	tf := filepath.Join(os.TempDir(), "test-source-map.map")
	defer os.Remove(tf)

	m1, err := Generate("f1.js", strings.NewReader("alert('f1-1');\nalert('f1-2');\nalert('f1-3');\n;\nalert('f1-4');\n\n"))
	assert.Nil(err)

	m2, err := Generate("f2.js", strings.NewReader("alert('f2-1');\nalert('f2-2');\n"))
	assert.Nil(err)

	m3 := Merge("combined.js", m1, m2)

	assert.Equal("AAAA;AACA;AACA;AACA;AACA;AACA;AACA;ACNA;AACA;AACA", m3.Mappings)

	err = WriteFile(tf, m3)
	assert.Nil(err)

	m4, err := ReadFile(tf)
	assert.Nil(err)

	assert.Equal(m3.DecodedMappings(), m4.DecodedMappings())
}
