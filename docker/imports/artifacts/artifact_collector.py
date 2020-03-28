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
""" Main artifact collector module """
# pylint: disable=fixme,invalid-name

import logging
import os.path
import sys
from typing import List

import definitions
import dfvfs.lib.definitions as dfvfs_defs
import dfvfs_helper
import dfvfs_utils
import fs.base
from artifact_resolver import ArtifactResolver
from definitions import PartitionInfo
from forensicstore import ForensicStore
from os_unknown import UnknownOS
from os_windows import WindowsSystem

from pyartifacts import Registry

LOGGER = logging.getLogger(__name__)


class ArtifactExtractor(object):
    """
    This is the main class that manages the artifact extraction from any dfVFS-supported input
    """

    # pylint: disable=too-few-public-methods

    def __init__(self, source_paths: List[str], result_root: fs.base.FS, artifact_registry: Registry,
                 encryption_handler: dfvfs_helper.EncryptionHandler):
        self.dfvfs_list = []
        self.artifact_registry = artifact_registry
        self.tmp_dir = None
        new_source_path = source_paths[0]
        self.dfvfs_list = [dfvfs_helper.DFVFSHelper(new_source_path, encryption_handler)]
        self.encryption_handler = encryption_handler
        # noinspection PyTypeChecker
        self.store = ForensicStore(result_root)

    def clean_up(self):
        """ Called when no more actions will be called in this object """
        for d in self.dfvfs_list:
            d.clean_up()
        self.dfvfs_list = None
        self.encryption_handler = None
        if self.tmp_dir:
            self.tmp_dir.cleanup()
            self.tmp_dir = None

    def extract_artifact(self, artifact_name):  # pylint: disable=invalid-name
        """
        Extract a particular artifact from all possible locations within this dfvfs-image
        """
        real_partitions: List[PartitionInfo] = []

        # forensic images can have more than one partition, but we always only process one image at a time
        real_partitions = [PartitionInfo(helper=self.dfvfs_list[0], path_spec=partition, name=chr(ord('c') + i))
                           for i, partition in enumerate(self.dfvfs_list[0].partitions())
                           if not dfvfs_utils.is_on_filesystem(partition, dfvfs_defs.TYPE_INDICATOR_VSHADOW)]
        LOGGER.info("Found %d partitions", len(real_partitions))
        for partinfo in real_partitions:
            current_os = self._guess_os(partinfo.helper, partinfo.path_spec)
            try:
                if current_os == definitions.OPERATING_SYSTEM_WINDOWS:
                    system = WindowsSystem(partinfo.helper, partinfo.path_spec)

                elif current_os == definitions.OPERATING_SYSTEM_UNKNOWN:
                    system = UnknownOS()
                    LOGGER.warning("Operating system not detected on partition %s. Only basic extraction possible.",
                                   dfvfs_utils.reconstruct_full_path(partinfo.path_spec))
                else:
                    LOGGER.warning("Operating system %s is not yet supported on %s. Using basic extraction.",
                                   dfvfs_utils.reconstruct_full_path(partinfo.path_spec), current_os)
                    system = UnknownOS()

                LOGGER.info("=== Starting processing of partition")
                resolver = ArtifactResolver(partinfo, self.artifact_registry, system)
                resolver.process_artifact(artifact_name, self.store)

                if current_os == definitions.OPERATING_SYSTEM_WINDOWS:
                    system._reg_reader._cleanup_open_files("")  # TODO

            except RuntimeError as err:
                LOGGER.exception("Encountered exception during processing of %s: %s",
                                 dfvfs_utils.reconstruct_full_path(partinfo.path_spec), err)
                if 'pytest' in sys.modules:
                    raise  # we want to see what exactly is failing when tests are running

        self.store.close()

    @staticmethod
    def _guess_os(dfvfs, partition):
        """Tries to determine the underlying operating system.
        Adapted from plaso/engine.py
        Returns:
          str: operating system, for example "Windows". This should be one of
              the values in definitions.OPERATING_SYSTEMS.
        """
        find_specs = [
            '/etc',
            '/System/Library',
            '/Windows/System32',
            '/WINNT/System32',
            '/WINNT35/System32',
            '/WTSRV/System32',
        ]

        locations = []
        for path_spec in dfvfs.find_paths(find_specs, partitions=[partition]):
            path = dfvfs_utils.get_relative_path(path_spec)
            if path:
                locations.append(path.lower().rstrip('/'))

        # We need to check for both forward and backward slashes since the path
        # spec will be OS dependent, as in running the tool on Windows will return
        # Windows paths (backward slash) vs. forward slash on *NIX systems.
        windows_locations = {'/windows/system32', '\\windows\\system32', '/winnt/system32', '\\winnt\\system32',
                             '/winnt35/system32', '\\winnt35\\system32', '\\wtsrv\\system32', '/wtsrv/system32'}

        if windows_locations.intersection(set(locations)):
            return definitions.OPERATING_SYSTEM_WINDOWS
        if '/system/library' in locations:
            return definitions.OPERATING_SYSTEM_MACOSX
        if '/etc' in locations:
            return definitions.OPERATING_SYSTEM_LINUX
        return definitions.OPERATING_SYSTEM_UNKNOWN
