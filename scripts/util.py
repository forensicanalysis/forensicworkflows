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
import sys


def merge_conditions(list_a, list_b):
    if list_a is None:
        return list_b
    if list_b is None:
        return list_a
    list_c = []
    for item_a in list_a:
        for item_b in list_b:
            list_c.append({**item_a, **item_b})
    return list_c


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


def combined_conditions(conditions):
    parser = argparse.ArgumentParser(description='parse key pairs into a dictionary')
    parser.add_argument("--filter", dest="filter", action=StoreDictKeyPair, metavar="type=file,name=System.evtx...")
    args, _ = parser.parse_known_args(sys.argv[1:])

    return merge_conditions(args.filter, conditions)
