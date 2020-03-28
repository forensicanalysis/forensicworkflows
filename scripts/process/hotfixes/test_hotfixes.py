import os
import shutil
import tempfile

import forensicstore
import pytest

from .hotfixes import main


@pytest.fixture
def data():
    tmpdir = tempfile.mkdtemp()
    shutil.copytree("test", os.path.join(tmpdir, "data"))
    return os.path.join(tmpdir, "data")


def test_hotfixes(data):
    cwd = os.getcwd()
    os.chdir(os.path.join(data, "data", "example1.forensicstore"))

    main()

    store = forensicstore.connect(os.path.join(data, "data", "example1.forensicstore"))
    items = list(store.select("hotfix"))
    store.close()
    assert len(items) == 14

    os.chdir(cwd)
    shutil.rmtree(data)
