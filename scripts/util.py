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
