// +build !build_bindata

package tpl

import (
	"io/ioutil"
	"path/filepath"
)

func Asset(name string) ([]byte, error) {
	path, _ := filepath.Abs(filepath.Join("./tpl", name))

	return ioutil.ReadFile(path)
}
