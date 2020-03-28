from .util import merge_conditions


def test_merge_1():
    filters = [{'type': 'file'}, {'type': 'dictionary'}]

    conditions = [{
        'key': "HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\%"
    }, {
        'key': "HKEY_USERS\\%\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\%"
    }]

    result = merge_conditions(filters, conditions)

    expected = [{
        'type': 'file',
        'key': "HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\%"
    }, {
        'type': 'file',
        'key': "HKEY_USERS\\%\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\%"
    }, {
        'type': 'dictionary',
        'key': "HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\%"
    }, {
        'type': 'dictionary',
        'key': "HKEY_USERS\\%\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\%"
    }]

    assert result == expected
