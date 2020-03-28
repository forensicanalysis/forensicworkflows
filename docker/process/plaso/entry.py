#!/usr/bin/env python3

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
