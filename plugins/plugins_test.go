package plugins

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
)

func setup() (storeDir string, err error) {
	tempDir, err := ioutil.TempDir("", "forensicstoreprocesstest")
	if err != nil {
		return "", err
	}
	storeDir = filepath.Join(tempDir, "test")
	err = os.MkdirAll(storeDir, 0755)
	if err != nil {
		return "", err
	}

	err = copy.Copy(filepath.Join("..", "test"), storeDir)
	if err != nil {
		return "", err
	}

	return storeDir, nil
}

func cleanup(folders ...string) (err error) {
	for _, folder := range folders {
		err := os.RemoveAll(folder)
		if err != nil {
			return err
		}
	}
	return nil
}
