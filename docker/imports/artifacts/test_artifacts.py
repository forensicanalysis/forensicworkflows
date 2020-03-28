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
