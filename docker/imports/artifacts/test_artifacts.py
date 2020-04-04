# Copyright (c) 2020 Siemens AG
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
#
# Author(s): Jonas Plum

import os
import shutil
import tempfile

import docker
import forensicstore
import pytest


@pytest.fixture
def data():
    return mkdata()


def mkdata():
    tmpdir = tempfile.mkdtemp()
    shutil.copytree("test", os.path.join(tmpdir, "data"))
    return os.path.join(tmpdir, "data")


def to_unix_path(p):
    path_unix = p
    if p[1] == ":":
        path_unix = "/" + p.lower()[0] + p[2:].replace("\\", "/")
    return path_unix


def test_docker(data):
    client = docker.from_env()

    # build image
    image_tag = "test_artifacts"
    image, _ = client.images.build(path="docker/imports/artifacts/", tag=image_tag)

    # run image
    store_path = os.path.abspath(os.path.join(data, "example.forensicstore"))
    store_path_unix = to_unix_path(store_path)
    import_path = os.path.abspath(os.path.join(data, "data", "win10_mock.vhd"))
    import_path_unix = to_unix_path(import_path)
    volumes = {
        store_path_unix: {'bind': '/store', 'mode': 'rw'},
        import_path_unix: {'bind': '/transit', 'mode': 'ro'}
    }
    # plugin_dir: {'bind': '/plugins', 'mode': 'ro'}
    client.containers.run(image_tag, volumes=volumes, stderr=True).decode("ascii")

    # test results
    store = forensicstore.connect(store_path)
    items = list(store.all())
    store.close()

    assert len(items) == 8

    # cleanup
    try:
        shutil.rmtree(data)
    except PermissionError:
        pass


if __name__ == '__main__':
    d = mkdata()
    test_docker(d)
