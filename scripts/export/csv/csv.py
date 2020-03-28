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
# Author(s): Demian Kellermann
""" Output table-like data to CSV files, quoting is enabled for every non-numeric field for easier parsing """
import csv
import sys
from io import StringIO

import forensicstore


def transform(store, items, name, header):
    if not items:
        return []
    report_name = name + ".csv"
    with store.store_file("/".join(["Reports", report_name
                                    ])) as (report_path, file_io):
        string_io = StringIO()
        csv_out = csv.DictWriter(string_io,
                                 fieldnames=header,
                                 delimiter=';',
                                 extrasaction='ignore',
                                 quoting=csv.QUOTE_NONNUMERIC)
        csv_out.writeheader()
        csv_out.writerows(items)
        string_io.seek(0)
        file_io.write(string_io.read().encode('utf-8'))
        return [{
            "type": "report",
            "report_path": report_path,
            "format": "csv"
        }]


def main():
    store = forensicstore.connect(".")
    items = list(store.select(sys.argv[1]))
    results = transform(store, items, sys.argv[1], sys.argv[2:])
    for result in results:
        store.insert(result)
    store.close()


if __name__ == '__main__':
    main()
