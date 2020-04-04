#!/usr/bin/env python3
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
# Author(s): Jonas Plum

import argparse
import logging
import os
import subprocess
import sys

import forensicstore

LOGGER = logging.getLogger(__name__)


class StoreDictKeyPair(argparse.Action):
    # pylint: disable=too-few-public-methods

    def __call__(self, parser, namespace, values, option_string=None):
        new_dict = {}
        for element in values.split(","):
            key, value = element.split("=")
            new_dict[key] = value
        if hasattr(namespace, self.dest):
            dict_list = getattr(namespace, self.dest)
            if dict_list is not None:
                dict_list.append(new_dict)
                setattr(namespace, self.dest, dict_list)
                return
        setattr(namespace, self.dest, [new_dict])


def main():
    parser = argparse.ArgumentParser(description='parse key pairs into a dictionary')
    parser.add_argument("--filter", dest="filter", action=StoreDictKeyPair, metavar="type=file,name=System.evtx...")
    args, _ = parser.parse_known_args(sys.argv[1:])

    if args.filter is None:
        LOGGER.warning("requires a filter to be set")
        sys.exit(1)

    store = forensicstore.connect(".")
    files = []

    selected = list(store.select("file", args.filter))
    for item in selected:
        if "export_path" in item and os.path.exists(item["export_path"]):
            files.append(item["export_path"])
    store.close()

    os.makedirs("Plaso", exist_ok=True)
    for file in files:
        subprocess.run(
            ["log2timeline.py", "--status_view", "none", "--logfile", "test.log", "Plaso/events.plaso", file],
            check=True
        )

    # TODO: add logfile to forensicstore

    subprocess.run(
        ["psort.py", "--status_view", "none", "-o", "forensicstore", "-w", "/store/", "Plaso/events.plaso"],
        check=True
    )


if __name__ == '__main__':
    main()
