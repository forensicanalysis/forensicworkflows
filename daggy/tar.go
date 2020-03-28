package daggy

import (
	"archive/tar"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func tarFolder(srcDir string, tw *tar.Writer) error {
	infos, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return errors.Wrap(err, "reading directory failed")
	}

	for _, info := range infos {
		if info.IsDir() {
			err = tarFolder(filepath.Join(srcDir, info.Name()), tw)
		} else {
			err = tarWrite(filepath.Join(srcDir, info.Name()), info.Name(), tw)
		}
		if err != nil {
			return errors.Wrap(err, "packing tars failed")
		}
	}
	return nil
}
func tarWrite(src string, dest string, tw *tar.Writer) error {
	dockerFileReader, err := os.Open(src)
	if err != nil {
		return err
	}
	readDockerFile, err := ioutil.ReadAll(dockerFileReader)
	if err != nil {
		return err
	}
	tarHeader := &tar.Header{
		Name: dest,
		Size: int64(len(readDockerFile)),
	}
	err = tw.WriteHeader(tarHeader)
	if err != nil {
		return err
	}
	_, err = tw.Write(readDockerFile)
	if err != nil {
		return err
	}
	return nil
}
