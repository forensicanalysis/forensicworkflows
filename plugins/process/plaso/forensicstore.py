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
from plaso.output import interface
from plaso.output import manager
from plaso.serializer import json_serializer


class ForensicstoreOutputModule(interface.OutputModule):
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

    def _WriteSerializedDict(self, event, event_data, event_tag):
        """Writes an event, event data and event tag to serialized form.
        Args:
          event (EventObject): event.
          event_data (EventData): event data.
          event_tag (EventTag): event tag.
        Returns:
          dict[str, object]: JSON serialized objects.
        """
        event_data_json_dict = self._JSON_SERIALIZER.WriteSerializedDict(event_data)
        del event_data_json_dict['__container_type__']
        del event_data_json_dict['__type__']

        inode = event_data_json_dict.get('inode', None)
        if inode is None:
            event_data_json_dict['inode'] = 0

        try:
            message, _ = self._output_mediator.GetFormattedMessages(event_data)
            event_data_json_dict['message'] = message
        except errors.WrongFormatter:
            pass

        event_json_dict = self._JSON_SERIALIZER.WriteSerializedDict(event)
        event_json_dict['__container_type__'] = 'event'

        event_json_dict.update(event_data_json_dict)

        if event_tag:
            event_tag_json_dict = self._JSON_SERIALIZER.WriteSerializedDict(event_tag)

            event_json_dict['tag'] = event_tag_json_dict

        return event_json_dict

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


def IsLinearOutputModule(self):
    return False


manager.OutputManager.RegisterOutput(ForensicstoreOutputModule)
manager.OutputManager.IsLinearOutputModule = IsLinearOutputModule
