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
""" This module contains some additional helper functions """
# pylint: disable=line-too-long

import logging
import hashlib

import fs.path

from dfvfs.lib  import definitions
from dfvfs.resolver import resolver
from dfvfs.resolver.context import Context

LOGGER = logging.getLogger(__name__)
CHUNK_SIZE = 32768


def reconstruct_full_path(entry):
    """
    Create a unique string representation of a PathSpec object, starting at the root of the
    dfvfs input object
    :param entry: A dfvfs path_spec object
    :return: [str] Representation of the object's location as file path
    """
    if not entry:
        return None

    curr = entry
    path = ''
    while curr:
        if getattr(curr, 'parent', None) is None:
            # skip the last level, as this is the storage path on the evidence store which has no
            # relevance
            break
        newpath = getattr(curr, 'location', None)
        if newpath is None:
            newpath = '/' + getattr(curr, 'type_indicator', '')
        path = newpath + path
        curr = getattr(curr, 'parent', None)
    return path.rstrip('/')


def is_on_filesystem(entry, filesystem):
    """
    Check if a certain element is on a certain type of filesystem
    :param entry: a dfvfs PathSpec object
    :param filesystem: dfvfs type indicator string
    :return: True if the specified filesystem is somewhere in the path-chain of the element
    """
    path = entry
    while True:
        if path.type_indicator == filesystem:
            return True
        if path.parent:
            path = path.parent
        else:
            return False


def get_relative_path(path_spec):
    """
    Get the path of an element up to the next container
    :param file_entry[PathSpec]: an dfvfs element
    :return: [str] representation of the relative path of the element within the innermost container
    """
    try:
        return path_spec.location.rstrip('/')
    except AttributeError:
        return None


def get_file_handle(path_spec, data_stream_name=None, context=None):
    """
    Get a file-like object to the contents of the specified element
    :param path_spec: A dfvfs PathSpec element
    :param data_stream_name: Optional number of desired datastream
    :param context: Use this context instead of the default one, or create a new one if == 'new'
    :return: File-like object or None if there are no data-streams (e.g. for folders)
    """
    # make new file_entry with the same path_spec to we can create a new context
    my_entry = pathspec_to_fileentry(path_spec, context)

    if my_entry.number_of_data_streams < 1:
        LOGGER.error("Cannot get file handle for %s: No data streams", my_entry.name)
        return None
    return my_entry.GetFileObject(data_stream_name=data_stream_name)


def pathspec_to_fileentry(path_spec, context=None):
    """
    Get the corresponding FileEntry for a PathSpec
    :param path_spec: A dfvfs PathSpec object
    :param context: Use this context instead of the default one, or create a new one if == 'new'
    :return: The FileSpec object whose path is pathspec
    """
    if context and context == 'new':
        my_entry = resolver.Resolver.OpenFileEntry(path_spec, resolver_context=Context())
    elif context:
        my_entry = resolver.Resolver.OpenFileEntry(path_spec, resolver_context=context)
    else:
        my_entry = resolver.Resolver.OpenFileEntry(path_spec)
    return my_entry


def _get_file_paths(file_entry, output_dir, filename, prepend_path_hash, with_path):
    if with_path:
        filepath = get_relative_path(file_entry)
        output_path = fs.path.dirname(filepath)
        output_dir.makedirs(output_path, recreate=True)
        real_outputdir = output_dir.opendir(output_path)
    else:
        real_outputdir = output_dir

    if filename:
        basename = filename
    else:
        basename = file_entry.name

    if prepend_path_hash:
        filepath = get_relative_path(file_entry)
        path_hash = hashlib.md5(filepath.encode())  # nosec
        basename = "{}_{}".format(path_hash.hexdigest(), basename)

    return real_outputdir, basename


def export_file(path_spec, output_dir, filename=None, prepend_pathhash=False, with_path=False):
    """
    Write a file to a filesystem. If the object has multiple data streams, all of them are extracted
    and numbered.
    :param path_spec: [PathSpec]: The dfvfs path spec pointing to the file to be extracted
    :param output_dir: [fs.base.FS]: The output base path
    :param filename: [str]: Optional alternative file name for the output file
    :param prepend_pathhash: [bool]: optional: If True, prepends the md5-hash of the file's storage
    path to the filename
    :param with_path: optional: If True, also recreate the folder structure to the file
    :return: A dict of file hashes of the contents of the first datastream or False on error
    """
    # pylint: disable=too-many-branches
    file_entry = pathspec_to_fileentry(path_spec)
    if not file_entry:
        LOGGER.error("Invalid file entry: %s", file_entry)
        return False
    if file_entry.number_of_data_streams < 1:
        if file_entry.entry_type == definitions.FILE_ENTRY_TYPE_DIRECTORY:
            LOGGER.debug("Skipping extraction of directory entry %s", file_entry.name)
        else:
            LOGGER.error("Cannot export %s: No data streams. Type: %s", file_entry.name, file_entry.entry_type)
        return False

    output_dir, basename = _get_file_paths(file_entry, output_dir, filename, prepend_pathhash, with_path)

    stream = 0
    hasher = HashHelper()
    for data_stream in file_entry.data_streams:
        stream += 1
        if file_entry.number_of_data_streams > 1 and stream > 1:
            output_filename = "{}-{}".format(basename, stream)
        else:
            output_filename = basename
        with output_dir.open(output_filename, 'wb') as outfile:
            file_object = file_entry.GetFileObject(data_stream_name=data_stream.name)
            if not file_object:
                LOGGER.error("Could not get file object for %s, datastream=%s",
                             output_filename, data_stream.name)
                continue
            try:
                data = file_object.read(CHUNK_SIZE)
                while data:
                    outfile.write(data)
                    if stream == 1:  # we are writing the first stream
                        hasher.update(data)
                    data = file_object.read(CHUNK_SIZE)
            except IOError:
                LOGGER.exception("Error reading file contents")
            finally:
                file_object.close()
    return hasher.get_hashes()


class HashHelper(object):
    """ A class to generate multiple hashes at once """
    def __init__(self):
        self.calculate = {"MD5": "md5", "SHA-1": "sha1"}
        self.hashers = {}
        for algo in self.calculate:
            self.hashers[algo] = hashlib.new(self.calculate[algo])

    def update(self, data):
        """ Feed data into the hashing algorithms """
        for algo in self.hashers:
            self.hashers[algo].update(data)

    def get_hashes(self):
        """ Return all hashes as hex strings """
        hashes = {}
        for algo in self.hashers:
            hashes[algo] = self.hashers[algo].hexdigest()
        return hashes
