import os
import shutil
import tempfile

import docker
import forensicstore
import pytest


@pytest.fixture
def data():
    tmpdir = tempfile.mkdtemp()
    shutil.copytree("test", os.path.join(tmpdir, "data"))
    return os.path.join(tmpdir, "data")


def test_docker(data):
    client = docker.from_env()

    # build image
    image_tag = "test_docker"
    image, _ = client.images.build(path="docker/process/plaso/", tag=image_tag)

    # run image
    store_path = os.path.abspath(os.path.join(data, "data", "example1.forensicstore"))
    store_path_unix = store_path
    if store_path[1] == ":":
        store_path_unix = "/" + store_path.lower()[0] + store_path[2:].replace("\\", "/")
    volumes = {store_path_unix: {'bind': '/store', 'mode': 'rw'}}
    # plugin_dir: {'bind': '/plugins', 'mode': 'ro'}
    output = client.containers.run(image_tag, command=["--filter", "artifact=WindowsDeviceSetup"], volumes=volumes,
                                   stderr=True)
    print(output)

    # test results
    store = forensicstore.connect(store_path)
    items = list(store.select("event"))
    store.close()
    assert len(items) == 69

    # cleanup
    try:
        shutil.rmtree(data)
    except PermissionError:
        pass


if __name__ == '__main__':
    d = data()
    test_docker(d)
