package sourcemap

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestReader(t *testing.T) {
	assert := assert.New(t)

	tf := filepath.Join(os.TempDir(), "map.map")
	defer os.Remove(tf)

	ioutil.WriteFile(tf, []byte(`{"version":3,"sources":["f1.js","f2.js"],"names":[],"mappings":"AAAA;AACA;AACA;AACA;AACA;AACA;AACA;ACNA;AACA;AACA","file":"public/javascripts/app.js","sourcesContent":["alert('f1-1');\nalert('f1-2');\nalert('f1-3');\n;\nalert('f1-4');\n\n","alert('f2-1');\nalert('f2-2');\n"]}`), 0666)

	_, err := Read(tf)
	assert.Nil(err)
}
