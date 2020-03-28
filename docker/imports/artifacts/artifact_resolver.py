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
""" Classes and logic to build a knowledge base and extract artifacts """

import logging
import re
import sqlite3
from datetime import datetime
from typing import List, Tuple, Optional, Iterable, Dict

import dfvfs.lib.definitions as dfvfs_defs
import dfvfs_helper
import dfvfs_utils
from definitions import PartitionInfo
from dfvfs.path.path_spec import PathSpec
from dfwinreg.interface import WinRegistryKey
from forensicstore import ForensicStore
from misc_utils import get_file_infos, CasePreservingSet
from os_base import OperatingSystemBase
from os_unknown import UnknownOS

from pyartifacts import ArtifactDefinition, Registry, ArtifactSource, KnowledgeBase
from pyartifacts import definitions as artifact_defs

LOGGER = logging.getLogger(__name__)

TEMPORARY_RESOLVE = 'tmp_resolve_4_variable'


class ResolvedArtifact(object):  # pylint: disable=too-few-public-methods,too-many-arguments
    """ class to save resolved paths for an artifact """

    # pylint: disable=too-many-arguments

    def __init__(self, artifact: ArtifactDefinition,
                 files: List[PathSpec], dirs: List[PathSpec], paths: List[PathSpec],
                 registry_keys: List[WinRegistryKey], registry_vals: List[Tuple[WinRegistryKey, List[str]]],
                 sub_artifacts: List['ResolvedArtifact']):
        self.artifact = artifact
        self.files = files
        self.dirs = dirs
        self.paths = paths
        self.registry_keys = registry_keys
        self.registry_vals = registry_vals
        self.sub_artifacts = sub_artifacts

    def empty(self) -> bool:
        """ check if anything is contained """
        return not (self.files or self.dirs or self.registry_keys
                    or self.registry_vals or self.sub_artifacts)


class ArtifactResolver:
    """
    This class converts artifacts to actual bits of information
    """

    def __init__(self, partinfo: PartitionInfo, artifacts_registry: Registry, system: OperatingSystemBase = None):
        """
        Initializes the class and loads artifacts
        :param dfvfs: [DFVFSHelper]: DFVFSHelper-object to access data
        :param partition: [PathSpec]: A DFVFS-PathSpec object identifying the root of the
                system partition
        :param partition_name: Name of the current partition
        :param artifacts_registry: [Registry]: Database of forensic artifacts definitions
        :param system: [OperatingSystemBase]: optional reference to a OperatingSystem-instance for
                further variable resolving
        """
        # pylint: disable=invalid-name,too-many-instance-attributes,too-many-arguments
        self.dfvfs = partinfo.helper
        self.partition = partinfo.path_spec
        self.partition_name = partinfo.name
        if system:
            self.system = system
        else:
            self.system = UnknownOS()
        if system:
            self.os = system.get_os_name()
        else:
            self.os = None
        if not self.os:
            self.artifacts = artifacts_registry.artifacts
        else:
            self.artifacts = {name: artifact for name, artifact in artifacts_registry.artifacts.items()
                              if not artifact.supported_os or self.os in artifact.supported_os}
        LOGGER.debug("Picked %d matching artifact definitions", len(self.artifacts))
        self.knowledge_base = KnowledgeBase(self.artifacts)
        # hard code values that cannot be resolved with their provides-definition for implementation reasons
        self.knowledge_cache: Dict[str, Iterable[str]] = {'environ_systemdrive': '/'}

    def _resolve_artifact(self, artifact: ArtifactDefinition) -> ResolvedArtifact:
        # pylint: disable=too-many-locals,too-many-branches,too-many-nested-blocks
        """
        Resolve actual paths from artifact path definitions
        :param artifact[ArtifactDefinition]: An artifact as defined by the artifacts python module
        :return: A ResolvedArtifact instance where all variables are resolved and wildcards expanded
        """

        if artifact.name != TEMPORARY_RESOLVE:
            LOGGER.info("Resolving artifact %s", artifact.name)
        else:
            LOGGER.debug("\tresolving variables..")

        files = []
        dirs = []
        paths = []
        registry_keys = []
        registry_vals = []
        sub_artifacts = []

        for source in artifact.sources:
            supported_os = getattr(source, "supported_os", None)
            if self.os and supported_os and self.os not in supported_os:
                continue

            # FILE: Single file(s): Might be wildcarded
            if source.type == artifact_defs.SOURCE_TYPE_FILE:
                files.extend(self.glob_file_paths(self._expand_paths(source.paths, source.separator)))

            # DIRECTORY: An (absolute) directory (might be wildcarded)
            # Used to obtain a directory listing
            elif source.type == artifact_defs.SOURCE_TYPE_DIRECTORY:
                dirs.extend(self.glob_file_paths(self._expand_paths(source.paths, source.separator)))

            # A path specification
            elif source.type == artifact_defs.SOURCE_TYPE_PATH:
                paths.extend(self.glob_file_paths(self._expand_paths(source.paths, source.separator)))

            # REGISTRY_KEY: Whole key with all values (may be wildcarded)
            elif source.type == artifact_defs.SOURCE_TYPE_REGISTRY_KEY:
                for k in source.keys:
                    expanded_keys = self._expand_path(k)
                    for kk in expanded_keys:
                        registry_keys.extend(self.glob_registry_path(kk))

            # REGISTRY_VALUE: Single key-value-pair(s) (may be wildcarded)
            elif source.type == artifact_defs.SOURCE_TYPE_REGISTRY_VALUE:
                for pair in source.key_value_pairs:
                    try:
                        reg_paths = self._expand_path(pair["key"])
                    except RuntimeError as err:
                        LOGGER.info("Failed to resolve path [%s]: %s", pair["key"], err)
                        continue

                    for path in reg_paths:
                        globbed_paths = self.glob_registry_path(path)  # we need to glob key paths first
                        for reg_key in globbed_paths:
                            value_names = self.glob_registry_value(reg_key, pair['value'])  # .. and values second
                            registry_vals.append((reg_key, value_names))

            # ARTIFACT_GROUP: A list of other artifacts
            elif source.type == artifact_defs.SOURCE_TYPE_ARTIFACT_GROUP:
                for s_artifact_name in source.names:
                    s_artifact = self.get_resolved_artifact(s_artifact_name)
                    if s_artifact:
                        sub_artifacts.append(s_artifact)
                    else:
                        LOGGER.warning("Skipping unknown sub-artifact %s in %s", s_artifact_name, artifact.name)

        return ResolvedArtifact(artifact, files, dirs, paths, registry_keys, registry_vals, sub_artifacts)

    def _resolve_source(self, source: ArtifactSource) -> Iterable[str]:
        """ This is used as the callback function for KnowledgeBase to do variable resolving"""
        LOGGER.debug('Resolving source: %s:%s', source.type, source.__dict__)
        results = set()
        artifact = ArtifactDefinition(TEMPORARY_RESOLVE, [source])
        resolved = self._resolve_artifact(artifact)
        # For keys, the key path is of interest
        for key_path in [key.path for key in resolved.registry_keys]:
            results.add(key_path)
        # For registry values, the content of the value is relevant
        for key, values in resolved.registry_vals:
            for value in values:
                value = key.GetValueByName(value)
                if value.DataIsInteger() or value.DataIsString():
                    data = str(value.GetDataAsObject())
                    results.add(data)
                else:
                    LOGGER.warning("Not adding value from %s since it has unparseable type", key)
        # paths are added as strings
        # .. as are directories (should not be source-provides, but might happen)
        for path in resolved.paths + resolved.dirs:
            path_str = dfvfs_utils.get_relative_path(path)
            results.add(path_str)
        # for files, we want the content (line based)
        for filename in resolved.files:
            handle = dfvfs_utils.get_file_handle(filename)
            results.add(''.join(handle.readlines()))
            handle.close()
        LOGGER.debug('Resolved to: %s', results)
        return results

    def _expand_paths(self, paths: List[str], separator: str = None):
        """
        Resolve a list of paths. See _expand_path
        :param paths[List[str]]: List of paths to resolve
        :param separator[str]:  optional, separator char from the artifact
        :return: List of resolved path(s), flattened
        """
        results = []
        for path in paths:
            try:
                resolved_paths = self._expand_path(path)
            except RuntimeError as err:
                LOGGER.info("Failed to resolve path [%s]: %s", path, err)
                continue

            for resolved in resolved_paths:
                if separator:
                    resolved = resolved.replace(separator, '/')
                    if self.system.get_os_name() == 'Windows':
                        resolved = resolved.replace('\\', '/')
                results.append(resolved)

        return results

    def _expand_path(self, path: str) -> List[str]:
        """
        This method will resolve artifact source paths that have variables.
        Some variables might refer to lists (e.g. users), so multiple
        paths can be returned after resolving.

        :param path: The artifact definition path to resolve
        :return: A list of paths with resolved sources. They can still contain wildcards like '*'!
        """
        # pylint: disable=too-many-nested-blocks,too-many-branches
        variable_regex = '(%?%([a-zA-Z0-9_.-]+)%?%)'
        variables = re.findall(variable_regex, path)  # will return tuples: (with surrounding %s, without)
        results = [path]
        for var, stripped_var in variables:  # replace variables one by one, updating all paths in results
            subst = self.get_var(stripped_var)
            if not subst:
                LOGGER.warning("Cannot resolve path \"%s\", \"%s\" is unknown", path, var)
                return []
            new_results = []
            for result in results:
                for value in subst:
                    # some variables contain variables themselves, so recurse if necessary
                    # these variables should resolve to exactly one value though
                    if re.match(variable_regex, value):
                        replacement_list = self._expand_path(value)
                        if len(replacement_list) > 1:
                            LOGGER.error("Nested variable replacement in \"%s\" found. Aborting...", value)
                            return []
                        replacement = replacement_list[0]
                    else:
                        replacement = value
                    new_results.append(result.replace(var, replacement))
            results = new_results

        return results

    def get_var(self, key: str) -> Iterable[str]:
        """
        Retrieves the value of a specified variable from the variable database.
        :param key: str: The name of the variable
        :return: The value of the variable or None if the variable does not exist
        """
        if not key:
            return []

        if key in self.knowledge_cache:
            return self.knowledge_cache[key]

        real_key = key.replace('%', '')
        try:
            results = self.knowledge_base.get_value(real_key, self._resolve_source)
        except ValueError as first_err:
            # maybe this was a windows variable, try some magic
            try:
                results = self.get_var('environ_' + real_key.lower())
            except ValueError:
                LOGGER.warning("Could not resolve %s: %s", key, first_err)
                return []

        variable_regex = '(%?%([a-zA-Z0-9_.-]+)%?%)'
        real_results = CasePreservingSet()  # we do not want the same path in different case
        for result in results:
            more_vars = re.search(variable_regex, result)
            if result.startswith('C:\\'):
                LOGGER.debug("Fixing absolute path %s", result)
                result = result.replace('C:\\', '/').replace('\\', '/')
            if more_vars:
                real_results.update(self._expand_path(result))
            else:
                real_results.add(result)
        self.knowledge_cache[key] = real_results
        return real_results

    def get_resolved_artifact(self, artifact_name: str) -> Optional[ResolvedArtifact]:
        """
        Retrieves an artifact from the database, resolves variables and returns it.
        This method will also check if the artifact is supported on the OS and if the
        dependencies (from the artifacts "conditions" field) are met.
        :param artifact_name: [str]: Name of the artifact
        :return: ResolvedArtifact instance or None if the artifact was not found or is not supported
        """
        artifact = self.artifacts.get(artifact_name, None)
        if not artifact:
            LOGGER.warning("Unknown or non-applicable artifact: %s", artifact_name)
            return None

        if self.os and artifact.supported_os and self.os not in artifact.supported_os:
            LOGGER.info("Artifact %s not supported for OS %s", artifact_name, self.os)
            return None

        if not self.os and artifact.supported_os:
            LOGGER.warning("Trying optimistic extract of %s, no OS known for current partition", artifact_name)

        return self._resolve_artifact(artifact)

    def get_all_valid(self) -> List[ResolvedArtifact]:
        """
        :return: List[ResolvedArtifact] of all valid artifacts that can be extracted
        """
        for artifact in self.artifacts:
            try:
                resolved = self.get_resolved_artifact(artifact)
                if resolved:
                    yield resolved
            except RuntimeError as err:
                LOGGER.info("Caught error while trying to load artifact [%s]: %s", artifact, err)

    def extract(self, artifact: ResolvedArtifact, store: ForensicStore) -> bool:
        """
        Extract an artifact to an export directory by creating a new ForensicStore there
        :return: True on success, false if nothing was extracted, raises RuntimeError otherwise
        """

        if not artifact:
            return False

        artifact_name = artifact.artifact.name

        if artifact.empty():
            LOGGER.debug("Nothing to extract found for %s", artifact_name)
            return False

        extracted = self._extract_files(artifact, store)
        extracted |= self._extract_registry(artifact, store)

        # handle sub-artifacts for ARTIFACT_GROUP and extract them to our output folder
        for sub in artifact.sub_artifacts:
            LOGGER.debug("Attemping extract of sub-artifact %s", sub.artifact.name)
            extracted |= self.extract(sub, store)

        if extracted:
            LOGGER.info("Extracted %s", artifact_name)
        else:
            LOGGER.debug("Nothing extracted for %s", artifact_name)
        return extracted

    def _extract_files(self, artifact: ResolvedArtifact, artifact_output: ForensicStore) -> bool:
        """
        Extract an artifact's files and folders to the forensic store
        :param artifact: artifact to extract
        :param artifact_output: Output forensic store
        :type artifact_output: ForensicStore
        :type artifact: ResolvedArtifact
        :return: True on success, False if nothing was written
        """

        artifact_name = artifact.artifact.name
        success = False

        for export_file in artifact.files:
            success = True
            file_infos = get_file_infos(export_file)
            if not file_infos:
                LOGGER.warning("Could not get file infos for \"%s\". Skipping",
                               dfvfs_utils.reconstruct_full_path(export_file))
                continue

            if file_infos['type'] != dfvfs_defs.FILE_ENTRY_TYPE_FILE:
                LOGGER.debug("Not exporting entry of wrong type: %s",
                             dfvfs_utils.reconstruct_full_path(export_file))
                continue

            store_obj_id = artifact_output.add_file_item(artifact_name, file_infos['name'],
                                                         created=file_infos.get('created', None),
                                                         modified=file_infos.get('modified', None),
                                                         accessed=file_infos.get('accessed', None),
                                                         origin={
                                                             'path': file_infos['path'],
                                                             'partition': self.partition_name
                                                         },
                                                         errors=None)
            output_name = f"{self.partition_name}_" \
                          f"{dfvfs_utils.get_relative_path(export_file).replace('/', '_').strip('_')}"
            file_contents = dfvfs_helper.get_file_handle(export_file)
            with artifact_output.add_file_item_export(store_obj_id, export_name=output_name) as file_export:
                chunk_size = 65536
                data = file_contents.read(chunk_size)
                while data:
                    file_export.write(data)
                    data = file_contents.read(chunk_size)
            file_contents.close()

        return success

    def _extract_registry(self, artifact: ResolvedArtifact, store: ForensicStore):
        """
        Extract an artifact's registry keys and values to an export directory.
        :type artifact: ResolvedArtifact
        :type store: ForensicStore
        :return: True on success, False if nothing was written
        """
        if not artifact:
            return False
        if not artifact.registry_keys and not artifact.registry_vals:
            return False

        if not self.system.get_registry():
            LOGGER.info("Cannot glob registry path since system does not support registry operations.")
            return False

        artifact_name = artifact.artifact.name

        exported = False
        for key in artifact.registry_keys:
            exported = True
            self._key_to_forensicstore(store, key, artifact=artifact_name)

        for key, values in artifact.registry_vals:
            exported = True
            self._key_to_forensicstore(store, key, values, artifact=artifact_name)

        return exported

    @staticmethod
    def _key_to_forensicstore(store: ForensicStore, key: WinRegistryKey, values: Optional[List[str]] = None,
                              artifact: str = '') -> None:
        """ Export a registry key to the forensicstore, optionally only picking certain values from the key """
        last_write_tuple = key.last_written_time.CopyToStatTimeTuple()
        if last_write_tuple[0]:
            last_write_date = datetime.utcfromtimestamp(last_write_tuple[0])
            last_write_date = last_write_date.replace(microsecond=(int(last_write_tuple[1] / 10)))
        else:
            last_write_date = datetime.utcfromtimestamp(0)
        try:
            key_item_id = store.add_registry_key_item(artifact=artifact, modified=last_write_date,
                                                      key=key.path, errors=None)
        except TypeError as err:
            LOGGER.exception("Error adding registry key: %s", err)
            return

        for value in key.GetValues():
            name = value.name
            if values and name not in values:
                continue
            if name is None:
                name = '(Default)'
            type_str = value.data_type_string
            if type_str == 'REG_DWORD_LE':
                type_str = 'REG_DWORD'
            # if value.data and (value.DataIsBinaryData() or value.data_type_string == "REG_NONE"):
            #     data = value.data
            # elif value.data:
            #     data = '{}'.format(value.GetDataAsObject())
            # else:
            #     data = b""

            try:
                store.add_registry_value_item(key_item_id, type_str, value.data, name)
            except sqlite3.OperationalError:
                LOGGER.exception("Error updating value")
                continue

    def resolve_superglobs(self, paths: List[str], separator: str = '/') -> List[str]:
        """
        Convert the forensicartifacts "superglob" operator (**) to a list of
        normal wildcarded paths.
        """
        paths_to_find = []
        simple_globs = [f for f in paths if '**' not in f]
        super_globs = [f for f in paths if '**' in f]

        paths_to_find.extend(simple_globs)

        for file_path in super_globs:
            # Artifacts can have superglobs with a number behind them.
            # Since we need to glob a max-number for unbound depths to be fast, use 3 as default
            match = re.search(r'\*\*(\d+)', file_path)
            if match:
                max_depth = int(match.group(1))
            else:
                max_depth = 3
                match = re.search(r'\*\*', file_path)

            sub = '*'
            for _ in range(max_depth):
                paths_to_find.append(file_path.replace(match.group(0), sub))
                sub = sub + separator + '*'
        return paths_to_find

    def glob_file_paths(self, paths: List[str]) -> List[PathSpec]:
        """
        Resolves file level globbing patterns to actual dfvfs file references.
        This is mostly used to extract artifacts, but can also be called externally.
        :param paths: [List[str]]: List of file paths with globbing characters (*, **)
        :return: List[PathSpec] of results from globbing on this system's partition
        """
        result = []
        paths_to_find = self.resolve_superglobs(paths)

        if paths_to_find:
            result.extend(self.dfvfs.find_paths(paths_to_find, case_sensitive=False,
                                                partitions=[self.partition]))
        return result

    def glob_registry_path(self, registry_key: str, ignore_trailing_wildcard=False) -> List[WinRegistryKey]:
        """
        This will glob a registry path with one ore more wildcards to all the registry keys that
        match.
        This is mostly used to extract artifacts, but can also be called externally.
        :param ignore_trailing_wildcard: Skip the last '*' if it is at the very end
        :param key
        """
        paths_to_find = self.resolve_superglobs([registry_key], separator='\\')

        registry = self.system.get_registry()
        if not registry:
            return

        for key in paths_to_find:
            # we will work like in dfwinreg's regsearcher, but try to skip the first few steps until
            # we see a wildcard. This means we don't need to use a virtual root key
            keyparts = key.split('\\')

            if ignore_trailing_wildcard and keyparts[-1] == '*':
                keyparts = keyparts[:-1]

            # find first location of a wildcard
            tail = None
            tail_idx = -1
            i = 0
            for i, val in enumerate(keyparts):
                if '*' in val:
                    tail = '\\'.join(keyparts[0:i])
                    tail_idx = i
                    break
            if not tail:  # no wildcard at all
                tail = '\\'.join(keyparts)
                tail_idx = i + 1

            reg_key = registry.GetKeyByPath(tail)
            if not reg_key:
                return

            for result in self._glob_registry_path_rec(keyparts, tail_idx, reg_key):
                yield result

    def _glob_registry_path_rec(self, keyparts: List[str], level: int, reg_key: WinRegistryKey) -> List[WinRegistryKey]:
        """
        Recursive part of glob_registry_path. Only used internally.
        :param keyparts[List[str]]: List of registry key parts
        :param level: current level in the matching (index into keyparts)
        :param reg_key[RegistryKey]: regf parent key object
        :return: Recursively calculates all matches and returns List[RegistryKey]
        """

        # catch edge case where no globbing is needed at all
        if level >= len(keyparts):
            yield self.system.get_registry().GetKeyByPath('\\'.join(keyparts))
            return

        matches = []
        subkeys = reg_key.GetSubkeys()
        curr_part = keyparts[level]
        if '*' in curr_part:  # this is a location where globbing is needed
            regex = re.compile(curr_part.replace('*', '.*').lower())
            for subkey in subkeys:
                if regex.match(subkey.name.lower()):
                    matches.append(subkey)
        else:  # no globbing at this point, simple string match
            for subkey in subkeys:
                if subkey.name.lower() == curr_part.lower():
                    matches.append(subkey)

        # determine if we have checked all parts
        if level == (len(keyparts) - 1):
            for match in matches:
                yield match

        else:
            for match in matches:
                for submatch in self._glob_registry_path_rec(keyparts, level + 1, match):
                    yield submatch

    @staticmethod
    def glob_registry_value(key: WinRegistryKey, search_value: str) -> List[str]:
        """
        Globs wildcards in registry value names to a list of names
        :param key: regf-registry key
        :param search_value: str, can contain wildcard
        :return: list of str
        """
        if not key or not search_value:
            return []
        matches = []
        value_regex = re.compile(search_value.replace('*', '.*').lower())
        for value in key.GetValues():
            if value is None or value.name is None:
                continue
            if value_regex.match(value.name.lower()):
                matches.append(value.name)
        return matches

    def process_artifact(self, artifact_name: str, store: ForensicStore) -> bool:
        """
        Start the extraction process of a particular artifact
        :param artifact_name: Name of the artifact
        :param store: ForensicStorage output
        :return: True if success, False otherwise
        """
        LOGGER.debug("Attempting extract of %s", artifact_name)
        artifact = self.get_resolved_artifact(artifact_name)
        if not artifact:
            return False

        return self.extract(artifact, store)
