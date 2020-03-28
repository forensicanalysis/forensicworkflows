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

"""The Forensicstore output module CLI arguments helper."""

from __future__ import unicode_literals

from plaso.lib import errors
from plaso.cli.helpers import interface
from plaso.cli.helpers import manager
from plaso.output import forensicstore


class ForensicstoreOutputArgumentsHelper(interface.ArgumentsHelper):
    """Forensicstore output module CLI arguments helper."""

    NAME = 'forensicstore'
    CATEGORY = 'output'
    DESCRIPTION = 'Argument helper for the Forensicstore output module.'

    # pylint: disable=arguments-differ
    @classmethod
    def ParseOptions(cls, options, output_module):
        """Parses and validates options.

        Args:
          options (argparse.Namespace): parser options.
          output_module (OutputModule): output module to configure.

        Raises:
          BadConfigObject: when the output module object is of the wrong type.
          BadConfigOption: when the output filename was not provided.
        """
        if not isinstance(output_module, forensicstore.ForensicstoreOutputModule):
            raise errors.BadConfigObject(
                'Output module is not an instance of ForensicstoreOutputModule')

        filename = getattr(options, 'write', None)
        if not filename:
            raise errors.BadConfigOption(
                'Output filename was not provided use "-w filename" to specify.')

        output_module.SetFilename(filename)


manager.ArgumentHelperManager.RegisterHelper(ForensicstoreOutputArgumentsHelper)
