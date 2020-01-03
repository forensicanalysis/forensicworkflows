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

"""Output module for the forensicstore format."""

from __future__ import unicode_literals

import forensicstore
from plaso.output import manager
from plaso.output import shared_json
from plaso.serializer import json_serializer


class ForensicstoreOutputModule(shared_json.SharedJSONOutputModule):
    """Output module for the forensicstore format."""

    NAME = 'forensicstore'
    DESCRIPTION = 'Output module that writes events into an forensicstore.'

    _JSON_SERIALIZER = json_serializer.JSONAttributeContainerSerializer

    def __init__(self, output_mediator):
        """Initializes the output module object.

        Args:
        output_mediator (OutputMediator): output mediator.

        Raises:
        ValueError: if the file handle is missing.
        """
        super(ForensicstoreOutputModule, self).__init__(output_mediator)
        self._store = None
        self._filename = None

    def WriteEventBody(self, event, event_data, event_tag):
        """Writes event values to the output.

        Args:
        event (EventObject): event.
        event_data (EventData): event data.
        event_tag (EventTag): event tag.
        """
        json_dict = self._WriteSerializedDict(event, event_data, event_tag)
        json_dict["type"] = "event"
        self._store.insert(json_dict)

    def Open(self):
        """Connects to the database and creates the required tables.

        Raises:
          IOError: if the specified output file already exists.
          OSError: if the specified output file already exists.
          ValueError: if the filename is not set.
        """
        if not self._filename:
            raise ValueError('Missing filename.')

        self._store = forensicstore.connect(self._filename)

    def Close(self):
        """Disconnects from the database.

        This method will create the necessary indices and commit outstanding
        transactions before disconnecting.
        """
        self._store.close()

    def SetFilename(self, filename):
        """Sets the filename.

        Args:
          filename (str): the filename.
        """
        self._filename = filename


manager.OutputManager.RegisterOutput(ForensicstoreOutputModule)
