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
""" various helper methods """

import collections
import logging
import os
import os.path
from collections.abc import MutableSet
from datetime import datetime

import dfvfs_utils
import six

LOGGER = logging.getLogger(__name__)


class CasePreservingSet(MutableSet):
    """ https://stackoverflow.com/questions/27531211/how-to-get-case-insensitive-python-set """

    def __init__(self, *values):
        self._values = {}
        self._fold = str.casefold  # Python 3
        for val in values:
            self.add(val)

    def __repr__(self):
        return '<{}{} at {:x}>'.format(
            type(self).__name__, tuple(self._values.values()), id(self))

    def __contains__(self, value):
        return self._fold(value) in self._values

    def __iter__(self):
        return iter(self._values.values())

    def __len__(self):
        return len(self._values)

    def add(self, value):
        """ Add a value """
        self._values[self._fold(value)] = value

    def discard(self, value):
        """ Remove a value """
        try:
            del self._values[self._fold(value)]
        except KeyError:
            pass

    def update(self, values):
        """ Add multiple values """
        for value in values:
            self.add(value)


def ensure_dir(path, raise_=False):  # pragma: no cover
    """
    Ensure a given path is a directory by creating it if necessary and erroring out if it is a file
    :param path: [str]: A path
    :return: True if path is a folder or was created. False if it is a file
    """
    if not os.path.isdir(path):
        if os.path.exists(path):
            LOGGER.error("Output dir %s exists and is not a folder!")
            if raise_:
                raise RuntimeError("Output dir %s exists and is not a folder!")
            return False
        os.makedirs(path)
    return True


def iterable(arg):
    """
    We need to distinguish if a variable value is a list or a string. Since strings
    are also iterable in Python, this helper will decide if something is truely iterable
    """
    return isinstance(arg, collections.Iterable) and not isinstance(arg, six.string_types)


def get_file_infos(path_spec):
    """
    Returns metadata about files as a STIX 2.0 compliant dict.
    :param path_spec: PathSpec: dfVFS file_entry object
    :return: dict
    """

    file_entry = dfvfs_utils.pathspec_to_fileentry(path_spec)
    stat = file_entry.GetStat()
    if not stat:
        LOGGER.warning("Could not get stat object for %s", file_entry.name)

    entry = {
        "size": getattr(stat, 'size', 0),
        "name": file_entry.name,
        "type": file_entry.entry_type,
    }
    for time in [('atime', 'accessed'), ('mtime', 'modified'), ('crtime', 'created')]:
        secs = getattr(stat, time[0], 0)
        nanos = getattr(stat, time[0] + '_nano', 0)
        if secs and secs != 0:
            datetime_entry = datetime.utcfromtimestamp(secs)
            datetime_entry = datetime_entry.replace(microsecond=int(nanos / 10))
            entry[time[1]] = datetime_entry.isoformat(timespec='milliseconds') + 'Z'

    # the path is not part of STIX 2.0 for file objects, but is very useful to have,
    # so we make it a custom attribute
    entry["path"] = path_spec.location

    return entry
