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

FROM log2timeline/plaso:20200227

RUN apt-get update && apt-get install -y python3-pip
RUN pip3 install -v forensicstore

RUN echo "from plaso.output import forensicstore" >> /usr/lib/python3/dist-packages/plaso/output/__init__.py
RUN echo "from plaso.cli.helpers import forensicstore_output" >> /usr/lib/python3/dist-packages/plaso/cli/helpers/__init__.py

ADD forensicstore.py /usr/lib/python3/dist-packages/plaso/output/forensicstore.py
ADD forensicstore_output.py /usr/lib/python3/dist-packages/plaso/cli/helpers/forensicstore_output.py

ENTRYPOINT ["/bin/sh", "-c", "mkdir -p Plaso && log2timeline.py --status_view none --logfile test.log Plaso/events.plaso /store/WindowsDeviceSetup && psort.py --status_view none -o forensicstore -w /store/ Plaso/events.plaso"]
