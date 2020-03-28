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
""" Glue logic to enable command line mode """
# pylint: disable=missing-class-docstring,missing-function-docstring

import argparse
import logging
import os
import sys

import encryption_handlers
import fs
import fs.path
from artifact_collector import ArtifactExtractor

from pyartifacts import Registry

LOGGER = logging.getLogger(__name__)


def parse_args():
    parser = argparse.ArgumentParser(description="Process forensic images and extract artifacts")
    parser.add_argument(
        "-a",
        "--artifacts-path",
        default='artifacts',
        dest="artifacts_path",
        help="Path where to search for artifact definitions"
    )
    parser.add_argument(
        "-k",
        "--keys",
        dest="keyfile",
        help="Keyfile for decryption"
    )
    parser.add_argument(
        "-o",
        "--output-dir",
        dest="output_dir",
        help="Output location (will be created)"
    )
    parser.add_argument(
        "-e",
        "--extract",
        nargs='+',
        dest="artifact_names",
        help="Which artifact to extract"
    )
    parser.add_argument(
        "-i",
        "--input",
        nargs='+',
        dest="input_evidence",
        help="Input file(s) (or folders) to process"
    )
    parser.add_argument('-v', '--verbose', action='count', default=0)
    my_args = parser.parse_args()
    if not all([my_args.input_evidence, my_args.artifact_names]):
        parser.error("The following arguments are required: -e/--extract, -i/--input")
    return my_args


class ArtifactExtractionCommand:
    # pylint: disable=too-few-public-methods

    def __init__(self, args):
        self.artifact_registry = Registry()
        self.artifact_registry.read_folder(args.artifacts_path)
        if not self.artifact_registry.artifacts:
            LOGGER.warning("Could not read any artifact definition from %s", args.artifacts_path)
        artifact_names = list([a.name for a in self.artifact_registry.artifacts.values()])
        artifact_names.sort()
        self.args = args

    def run(self):
        print("run", self.args)
        # create output evidence folder using pyfs
        os.makedirs(self.args.output_dir, exist_ok=True)
        out_fs = fs.open_fs(self.args.output_dir)

        # do we have a key list for decryption?
        encryption_keys = []
        if args.keyfile:
            with open(args.keyfile, 'r') as keyfile:
                encryption_keys = encryption_handlers.read_key_list(keyfile)

        extractor = None
        try:
            handler = encryption_handlers.ConsoleEncryptionHandler(encryption_keys)
            in_files = [f for f in self.args.input_evidence if f]
            extractor = ArtifactExtractor(in_files, out_fs,
                                          self.artifact_registry, handler)
            for artifact in self.args.artifact_names:
                print("Extract %s" % artifact)
                extractor.extract_artifact(artifact)
        except Exception as error:
            LOGGER.exception("Uncaught exception during job: %s", error)
        finally:
            if extractor:
                extractor.clean_up()


def cmd_mode(args):
    if not os.path.isdir(args.artifacts_path):
        print(f"Not a directory: {args.artifacts_path}")
        sys.exit(1)
    for infile in args.input_evidence:
        if infile:
            if not os.path.exists(infile):
                print(f"Input does not exist: {infile}")
                sys.exit(1)
    # levels = [logging.WARNING, logging.INFO, logging.DEBUG]
    level = logging.DEBUG  # levels[min(len(levels) - 1, args.verbose)]
    logging.basicConfig(format="[%(asctime)s] %(message)s", datefmt='%Y-%m-%d %H:%M:%S', level=level)
    logging.getLogger('dfvfs_helper.dfvfs_helper').setLevel(logging.ERROR)
    extractor = ArtifactExtractionCommand(args)
    extractor.run()


if __name__ == '__main__':
    args = parse_args()
    print(args)
    cmd_mode(args)
