#!/usr/bin/env python
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
# Author(s): Demian Kellermann
""" Some basic tests for the extraction extractor """
# pylint: skip-file

import hashlib
import logging
import os
import shutil
from os.path import dirname, realpath, join, isdir

from forensicstore import ForensicStore

logging.getLogger('jsonlite.jsonlite').setLevel(logging.WARNING)

TEST_CASE_NAME = 'test_case'

CASES_LOCAL_PATH = realpath(join(dirname(realpath(__file__)), '..', 'test_images'))


def md5_file(file_path):
    CHUNK = 4096
    md5 = hashlib.md5()  # nosec
    with open(file_path, 'rb') as infile:
        data = infile.read(CHUNK)
        while data:
            md5.update(data)
            data = infile.read(CHUNK)
    return md5.hexdigest()


def teardown_function():
    """ Runs after every test and cleans output folder so the evidence can be recreated by the extractor """
    output_base = join(CASES_LOCAL_PATH, TEST_CASE_NAME)
    folders = [join(output_base, f) for f in os.listdir(output_base) if
               isdir(join(output_base, f)) and f.startswith('artifacts')]
    for folder in folders:
        shutil.rmtree(folder)


def find_store(local_path):
    store = None
    for root, dirs, files in os.walk(local_path):
        if root.endswith('.forensicstore'):
            assert store is None  # There is only one partition here
            store = ForensicStore(root)
    assert store is not None
    return store
