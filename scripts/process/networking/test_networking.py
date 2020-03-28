import os
import shutil
import tempfile

import forensicstore
import pytest
from .networking import main


@pytest.fixture
def data():
    tmpdir = tempfile.mkdtemp()
    shutil.copytree("test", os.path.join(tmpdir, "data"))
    return os.path.join(tmpdir, "data")


def test_networking(data):
    cwd = os.getcwd()
    os.chdir(os.path.join(data, "data", "example1.forensicstore"))

    main()

    store = forensicstore.connect(os.path.join(data, "data", "example1.forensicstore"))
    items = list(store.select("known_network"))
    store.close()
    assert len(items) == 9

    os.chdir(cwd)
    shutil.rmtree(data)
