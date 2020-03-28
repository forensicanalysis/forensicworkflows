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
""" This module thinly wraps some dfVFS functionality for easier use """

import os
import os.path
import re
import locale
import logging
from abc import ABCMeta, abstractmethod

from dfvfs.lib import definitions
from dfvfs.lib import glob2regex
from dfvfs.lib import errors as dfvfs_errors
from dfvfs.helpers import volume_scanner, file_system_searcher
from dfvfs.resolver import resolver
import pybde

from dfvfs_utils import get_file_handle

LOGGER = logging.getLogger(__name__)


class EncryptionHandler(object, metaclass=ABCMeta):  # pylint: disable=too-few-public-methods
    """ this class defines the interface for handling password prompts """

    @abstractmethod
    def unlock_volume(self, info, credentials):
        """
        Receives information about an encrypted volume and returns a key.
        Method can be called multiple times if "skip" has not been returned!
        :param info: Textual information about the volume
        :param credentials: Methods to unlock
        :return: Tuple (credential_type, key), where credential type is in credentials
                    or None to skip
        """
        pass


class DFVFSHelperMediator(volume_scanner.VolumeScannerMediator):
    # pylint: disable=invalid-name
    """Class that defines a volume scanner mediator."""

    # For context see: http://en.wikipedia.org/wiki/Byte
    _UNITS_1000 = ['B', 'kB', 'MB', 'GB', 'TB', 'EB', 'ZB', 'YB']
    _UNITS_1024 = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'EiB', 'ZiB', 'YiB']

    def __init__(self, encryption, vss, partitions):
        """Initializes the scanner mediator object."""
        super(DFVFSHelperMediator, self).__init__()
        self._encode_errors = 'strict'
        self._preferred_encoding = locale.getpreferredencoding()
        self._vss = vss
        self._partitions = partitions
        self._encryption_handler = encryption

    def _EncodeString(self, string):
        """Encodes a string in the preferred encoding.

        Returns:
          A byte string containing the encoded string.
        """
        try:
            # Note that encode() will first convert string into a Unicode string
            # if necessary.
            encoded_string = string.encode(
                self._preferred_encoding, errors=self._encode_errors)
        except UnicodeEncodeError:
            if self._encode_errors == 'strict':
                # logging.error(
                #    'Unable to properly write output due to encoding error. '
                #    'Switching to error tolerant encoding which can result in '
                #    'non Basic Latin (C0) characters being replaced with "?" or '
                #    '"\\ufffd".')
                self._encode_errors = 'replace'

            encoded_string = string.encode(
                self._preferred_encoding, errors=self._encode_errors)

        return encoded_string

    def _FormatHumanReadableSize(self, size):
        """Formats the size as a human readable string.

        Args:
          size: The size in bytes.

        Returns:
          A human readable string of the size.
        """
        magnitude_1000 = 0
        size_1000 = float(size)
        while size_1000 >= 1000:
            size_1000 /= 1000
            magnitude_1000 += 1

        magnitude_1024 = 0
        size_1024 = float(size)
        while size_1024 >= 1024:
            size_1024 /= 1024
            magnitude_1024 += 1

        size_string_1000 = None
        if 0 < magnitude_1000 <= 7:
            size_string_1000 = '{0:.1f}{1:s}'.format(
                size_1000, self._UNITS_1000[magnitude_1000])

        size_string_1024 = None
        if 0 < magnitude_1024 <= 7:
            size_string_1024 = '{0:.1f}{1:s}'.format(
                size_1024, self._UNITS_1024[magnitude_1024])

        if not size_string_1000 or not size_string_1024:
            return '{0:d} B'.format(size)

        return '{0:s} / {1:s} ({2:d} B)'.format(
            size_string_1024, size_string_1000, size)

    @staticmethod
    def _ParseVSSStoresString(vss_stores):
        """Parses the user specified VSS stores string.

        A range of stores can be defined as: 3..5. Multiple stores can
        be defined as: 1,3,5 (a list of comma separated values). Ranges
        and lists can also be combined as: 1,3..5. The first store is 1.
        All stores can be defined as "all".

        Args:
          vss_stores (str): user specified VSS stores.

        Returns:
          list[int|str]: Individual VSS stores numbers or the string "all".

        Raises:
          ValueError: if the VSS stores option is invalid.
        """
        if not vss_stores:
            return []

        if vss_stores == 'all':
            return ['all']

        stores = []
        for vss_store_range in vss_stores.split(','):
            # Determine if the range is formatted as 1..3 otherwise it indicates
            # a single store number.
            if '..' in vss_store_range:
                first_store, last_store = vss_store_range.split('..')
                try:
                    first_store = int(first_store, 10)
                    last_store = int(last_store, 10)
                except ValueError:
                    raise ValueError('Invalid VSS store range: {0:s}.'.format(
                        vss_store_range))

                for store_number in range(first_store, last_store + 1):
                    if store_number not in stores:
                        stores.append(store_number)
            else:
                if vss_store_range.startswith('vss'):
                    vss_store_range = vss_store_range[3:]

                try:
                    store_number = int(vss_store_range, 10)
                except ValueError:
                    raise ValueError('Invalid VSS store range: {0:s}.'.format(
                        vss_store_range))

                if store_number not in stores:
                    stores.append(store_number)

        return sorted(stores)

    def GetPartitionIdentifiers(self, volume_system, volume_identifiers):
        """Retrieves partition identifiers that should be scanned

        Args:
          volume_system: the volume system (instance of dfvfs.TSKVolumeSystem).
          volume_identifiers: a list of strings containing the volume identifiers.

        Returns:
          A list of strings containing the selected partition identifiers or None.

        Raises:
          ScannerError: if the source cannot be processed.
        """

        LOGGER.info('The following partitions were found:')
        LOGGER.info('Identifier\tOffset (in bytes)\tSize (in bytes)')

        for volume_identifier in sorted(volume_identifiers):
            volume = volume_system.GetVolumeByIdentifier(volume_identifier)
            if not volume:
                raise dfvfs_errors.ScannerError(
                    'Volume missing for identifier: {0:s}.'.format(volume_identifier))

            volume_extent = volume.extents[0]
            LOGGER.info('{0:s}\t\t{1:d} (0x{1:08x})\t{2:s}'.format(
                volume.identifier, volume_extent.offset,
                self._FormatHumanReadableSize(volume_extent.size)))

        selected_volume_identifier = self._partitions
        selected_volume_identifier = selected_volume_identifier.strip()

        if not selected_volume_identifier.startswith('p'):
            try:
                partition_number = int(selected_volume_identifier, 10)
                selected_volume_identifier = 'p{0:d}'.format(partition_number)
            except ValueError:
                pass

        LOGGER.info("Selected partition(s): %s", selected_volume_identifier)
        if selected_volume_identifier == 'all':
            return volume_identifiers

        return [selected_volume_identifier]

    def GetVSSStoreIdentifiers(self, volume_system, volume_identifiers):
        """Retrieves VSS store identifiers.

        This method can be used to prompt the user to provide VSS store identifiers.

        Args:
          volume_system (VShadowVolumeSystem): volume system.
          volume_identifiers (list[str]): volume identifiers.

        Returns:
          list[int]: selected VSS store numbers or None.

        Raises:
          ScannerError: if the source cannot be processed.
        """
        normalized_volume_identifiers = []
        for volume_identifier in volume_identifiers:
            volume = volume_system.GetVolumeByIdentifier(volume_identifier)
            if not volume:
                raise dfvfs_errors.ScannerError(
                    'Volume missing for identifier: {0:s}.'.format(volume_identifier))

            try:
                volume_identifier = int(volume.identifier[3:], 10)
                normalized_volume_identifiers.append(volume_identifier)
            except ValueError:
                pass

        LOGGER.info('The following Volume Shadow Snapshots (VSS) were found:')
        LOGGER.info('Identifier\tVSS store identifier')

        for volume_identifier in volume_identifiers:
            volume = volume_system.GetVolumeByIdentifier(volume_identifier)
            if not volume:
                raise dfvfs_errors.ScannerError(
                    'Volume missing for identifier: {0:s}.'.format(
                        volume_identifier))

            vss_identifier = volume.GetAttribute('identifier')
            LOGGER.info('{0:s}\t\t{1:s}'.format(
                volume.identifier, vss_identifier.value))

        selected_vss_stores = self._vss

        selected_vss_stores = selected_vss_stores.strip()
        if not selected_vss_stores:
            selected_vss_stores = []

        try:
            selected_vss_stores = self._ParseVSSStoresString(selected_vss_stores)
        except ValueError:
            selected_vss_stores = []

        if selected_vss_stores == ['all']:
            # We need to set the stores to cover all vss stores.
            selected_vss_stores = list(range(1, volume_system.number_of_volumes + 1))

        LOGGER.info("Selected vss stores: %s", selected_vss_stores)
        return selected_vss_stores

    def UnlockEncryptedVolume(
            self, source_scanner_object, scan_context, locked_scan_node, credentials):
        """Unlocks an encrypted volume.

        This method can be used to prompt the user to provide encrypted volume
        credentials.

        Args:
          source_scanner_object: the source scanner (instance of SourceScanner).
          scan_context: the source scanner context (instance of
                        SourceScannerContext).
          locked_scan_node: the locked scan node (instance of SourceScanNode).
          credentials: the credentials supported by the locked scan node (instance
                       of dfvfs.Credentials).

        Returns:
          A boolean value indicating whether the volume was unlocked.
        """
        # pylint: disable=too-many-locals

        volume_info = ""

        if locked_scan_node.type_indicator == definitions.TYPE_INDICATOR_BDE:
            volume_info = "Bitlocker volume"
            parent_pathspec = locked_scan_node.path_spec.parent
            bde_file_obj = get_file_handle(parent_pathspec)

            check = pybde.check_volume_signature_file_object(bde_file_obj)
            if check:
                bde = pybde.open_file_object(bde_file_obj)

                label = bde.get_description()
                if label:
                    volume_info = "{} ({})".format(volume_info, label)

                uuid = bde.get_volume_identifier()
                if uuid:
                    volume_info = "{}: {}".format(volume_info, uuid)

        else:
            volume_info = "Encrypted volume"

        credentials_list = list(credentials.CREDENTIALS)

        unlocked = False

        while not unlocked:
            credential_type, credential_data = self._encryption_handler.unlock_volume(
                volume_info, credentials_list)

            if not credential_type or not credential_data:
                LOGGER.warning("No key supplied: Skipping decryption of %s", volume_info)
                break

            try:
                unlocked = source_scanner_object.Unlock(
                    scan_context, locked_scan_node.path_spec, credential_type,
                    credential_data)
            except IOError as error:
                LOGGER.warning("Encountered dfVFS exception during decrypt: %s", error)

            if not unlocked:
                LOGGER.warning('Unable to unlock volume using credential "%s" (type %s)',
                               credential_data, credential_type)
            else:
                LOGGER.info('Unlocked volume using credential "%s" (type %s)',
                            credential_data, credential_type)


        return unlocked


class DFVFSHelper(volume_scanner.VolumeScanner):
    """ A helper object for a particular image file or folder """

    def __init__(self, evidence, encryption_handler, vss='all', partitions='all'):
        mediator = DFVFSHelperMediator(encryption=encryption_handler,
                                       vss=vss, partitions=partitions)
        super(DFVFSHelper, self).__init__(mediator=mediator)

        self.evidence = evidence
        if not os.path.exists(self.evidence):
            raise RuntimeError("Source does not exist: %s" % self.evidence)

        # if input is a folder, look for split files
        if os.path.isdir(evidence):
            for evidence_file in os.listdir(evidence):
                if os.path.splitext(evidence_file)[1].lower() in ['.e01', '.001']:
                    LOGGER.info("Using split file %s as dfVFS input", evidence_file)
                    self.evidence = os.path.join(evidence, evidence_file)
                    break
            # if none are found, fallback to processing the dir

        if self.evidence.endswith('.zip'):
            # the VolumeScanner does not open ZIP files, do it manually
            os_spec = self.GetBasePathSpecs(self.evidence)[0]
            from dfvfs.path import zip_path_spec
            zip_spec = zip_path_spec.ZipPathSpec('/', parent=os_spec)
            self.base_path_specs = [zip_spec]
        else:
            self.base_path_specs = self.GetBasePathSpecs(self.evidence)
        if not self.base_path_specs:
            raise RuntimeError('No supported file system found in source.')

    @staticmethod
    def clean_up():
        """
        Cleans up references and open file descriptors. Should be called
        after work has finished on long-running processes.
        """
        # TODO: Find out if we can use our own context everywhere so we do not need to access protected members
        default_context = resolver.Resolver._resolver_context
        file_cache = default_context._file_object_cache
        filesystem_cache = default_context._file_system_cache
        if file_cache._values:
            LOGGER.warning("Found open references in file_cache: %s", file_cache._values.__str__())
            for key in list(file_cache._values.keys())[:]:
                try:
                    file_cache._values[key].vfs_object.close()
                except (IOError, KeyError, AttributeError) as error:
                    LOGGER.warning(error)
                    pass
        if filesystem_cache._values:
            LOGGER.warning("Found open references in filesystem_cache: %s",
                           list(filesystem_cache._values.keys()).__str__())
            for key in list(filesystem_cache._values.keys())[:]:
                try:
                    filesystem_cache._values[key].vfs_object.Close()
                except (IOError, KeyError, AttributeError) as error:
                    LOGGER.warning(error)
                    pass

        default_context.Empty()
        LOGGER.debug("Cleanup completed")

    def partitions(self):
        """
        Get all "root" file system object of the dfvfs-instance. This includes
        partitions, volume shadow snapshots, etc..
        :return: List[PathSpec]
        """
        return self.base_path_specs

    def all(self, partitions=None):
        """
        Recursively walk over every entry in the source
        :return: tuple of full path and file_entry object
        """
        stack = []
        if not partitions:
            partitions = self.base_path_specs

        for base_path_spec in partitions:
            try:
                file_system = resolver.Resolver.OpenFileSystem(base_path_spec)
                file_entry = resolver.Resolver.OpenFileEntry(base_path_spec)
            except dfvfs_errors.BackEndError as err:
                LOGGER.warning("Unable to open partition %s: %s", base_path_spec.comparable, err)
                continue

            if file_entry is None:
                logging.warning('Unable to open base path specification:\n{0:s}'
                                .format(base_path_spec.comparable))
                continue

            stack.append((file_system, file_entry, ''))

        while stack:
            file_system, file_entry, parent = stack.pop()
            full_path = file_system.JoinPath([parent, file_entry.name])

            yield full_path, file_entry

            for sub_file_entry in file_entry.sub_file_entries:
                stack.append((file_system, sub_file_entry, full_path))

    def all_files(self, partitions=None):
        """
        Return all files (and not folders) in the source
        :return: full path and file_entry object
        """
        for full_path, file_entry in self.all(partitions=partitions):
            if file_entry.IsFile():
                yield full_path, file_entry

    def find_paths(self, locations, case_sensitive=False, regex=False, partitions=None):
        """
        Fast search for paths inside the volume. Must match path depth exactly!
        :param locations: list of search strings
        :param case_sensitive: bool
        :param regex: bool if search strings contain regex
        :param partitions: optional: limit search to certain partitions (from partitions() )
        :return: list of PathSpec results
        """
        if not partitions:
            partitions = self.base_path_specs

        if not locations:
            LOGGER.debug("dfvfs.find_paths called with empty search specs")
            return

        specs = []
        for search in locations:
            if regex:
                spec = file_system_searcher.FindSpec(
                    location_regex=search, case_sensitive=case_sensitive)
            elif '*' in search or '?' in search:
                spec = file_system_searcher.FindSpec(
                    location_glob=search, case_sensitive=case_sensitive)
            else:
                spec = file_system_searcher.FindSpec(
                    location=search, case_sensitive=case_sensitive)
            specs.append(spec)

        for base_path_spec in partitions:
            file_system = None
            try:
                file_system = resolver.Resolver.OpenFileSystem(base_path_spec)
                searcher = file_system_searcher.FileSystemSearcher(file_system, base_path_spec)
                for result in searcher.Find(specs):
                    yield result
            except dfvfs_errors.Error as error:
                LOGGER.warning("Encountered exception in dfVFS: %s", error)
                continue
            finally:
                if file_system:
                    resolver.Resolver._resolver_context.ReleaseFileSystem(file_system)

    def find_subpaths(self, names, case_sensitive=False, regex=False, partitions=None):
        """
        Match any part of the full path of a file within the volume.
        Note: Slowest possibility!
        :param names: List of patterns
        :param case_sensitive: bool
        :param regex: bool
        :return: list of FileEntry results
        """
        plain_search, regex_search = self._prepare_search_terms(names, case_sensitive, regex)
        for full_path, file_entry in self.all_files(partitions=partitions):
            if self._check_search_terms(full_path, case_sensitive, plain_search, regex_search,
                                        match_substring=True):
                yield file_entry

    def find_filename(self, names, case_sensitive=False, regex=False, partitions=None):
        """
        Match the filename part only.
        Note: Slower than find_paths, but faster than find_subpaths!
        :param names: List of patterns
        :param case_sensitive: bool
        :param regex: bool
        :param partitions: Optional list of partitions to restrict search (from .partitions())
        :return: list of FileEntry results
        """
        plain_search, regex_search = self._prepare_search_terms(names, case_sensitive, regex)

        # we do not need the path calculation in self.all() here
        # might save some time, but this duplicates some code..
        if not partitions:
            partitions = self.base_path_specs
        stack = []
        for base_path_spec in partitions:
            file_entry = resolver.Resolver.OpenFileEntry(base_path_spec)
            if file_entry is None:
                logging.warning('Unable to open base path specification:\n{0:s}'
                                .format(base_path_spec.comparable))
                continue
            stack.append(file_entry)

        while stack:
            file_entry = stack.pop()
            try:
                name = file_entry.name
                if self._check_search_terms(name, case_sensitive, plain_search, regex_search):
                    yield file_entry

            except AttributeError:
                pass

            for sub_file_entry in file_entry.sub_file_entries:
                stack.append(sub_file_entry)

    @staticmethod
    def _prepare_search_terms(terms, case_sensitive, regex):
        if case_sensitive:
            re_flags = 0
        else:
            re_flags = re.IGNORECASE
        plain_search = []
        regex_search = []
        for term in terms:
            if regex:
                regex_search.append(re.compile(term, re_flags))
            elif '*' in term or '?' in term:
                glob2reg = glob2regex.Glob2Regex(term)
                regex_search.append(re.compile(glob2reg, re_flags))
            else:
                if not case_sensitive:
                    plain_search.append(term.lower())
                else:
                    plain_search.append(term)
        return plain_search, regex_search

    @staticmethod
    def _check_search_terms(haystack, case_sensitive, plain_search,
                            regex_search, match_substring=False):
        if not case_sensitive:
            haystack = haystack.lower()
        for plain in plain_search:
            if match_substring and plain in haystack:
                return True
            if plain == haystack:
                return True
        for regex in regex_search:
            if match_substring and regex.find(haystack):
                return True
            if regex.match(haystack):
                return True
        return False
