// Copyright (c) 2020 Siemens AG
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// Author(s): Jonas Plum

package cmd

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
	dockerFileReader, err := os.Open(src) // #nosec
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
