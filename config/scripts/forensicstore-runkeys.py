#!/usr/bin/env python
# Copyright (c) 2019 Siemens AG
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

import sys

import forensicstore
import storeutil


def transform(items):
    results = []
    for item in items:
        if "values" in item:
            for value in item["values"]:
                results.append({
                    "Key": item["key"],
                    "Command": value["data"],
                    "Name": value["name"],
                    "SID": item["key"].split("\\")[1],
                    "type": "runkey",
                })

    return results


def main(args):
    parser = storeutil.ScriptArgumentParser(
        'run-keys',
        description='Process windows run keys',
        store_arg=True,
        filter_arg=True,
    )
    args = parser.parse_args(args)

    for url in args.forensicstore:
        store = forensicstore.connect(url)
        hklmsw = "HKEY_LOCAL_MACHINE\\Software\\"
        hkusw = "HKEY_USERS\\%\\Software\\"
        conditions = [
            {'key': hklmsw + r"Microsoft\Windows\CurrentVersion\Policies\Explorer\Run"},
            {'key': hklmsw + r"Microsoft\Windows\CurrentVersion\Run"},
            {'key': hklmsw + r"Microsoft\Windows\CurrentVersion\RunOnce"},
            {'key': hklmsw + r"Microsoft\Windows\CurrentVersion\RunOnce\Setup"},
            {'key': hklmsw + r"Microsoft\Windows\CurrentVersion\RunOnceEx"},
            {'key': hklmsw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\Run"},
            {'key': hklmsw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\RunOnce"},
            {'key': hklmsw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\RunOnce\Setup"},
            {'key': hklmsw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\RunOnceEx"},
            {'key': hklmsw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\Policies\Explorer\Run"},
            {'key': hkusw + r"Microsoft\Windows\CurrentVersion\Policies\Explorer\Run"},
            {'key': hkusw + r"Microsoft\Windows\CurrentVersion\Run"},
            {'key': hkusw + r"Microsoft\Windows\CurrentVersion\RunOnce"},
            {'key': hkusw + r"Microsoft\Windows\CurrentVersion\RunOnce\Setup"},
            {'key': hkusw + r"Microsoft\Windows\CurrentVersion\RunOnceEx"},
            {'key': hkusw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\Policies\Explorer\Run"},
            {'key': hkusw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\Run"},
            {'key': hkusw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\RunOnce"},
            {'key': hkusw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\RunOnce\Setup"},
            {'key': hkusw + r"Wow6432Node\Microsoft\Windows\CurrentVersion\RunOnceEx"}
        ]
        combined_conditions = storeutil.merge_conditions(args.filter, conditions)
        items = store.select("windows-registry-key", combined_conditions)
        results = transform(items)
        for result in results:
            store.insert(result)
        store.close()


if __name__ == '__main__':
    main(sys.argv[1:])
