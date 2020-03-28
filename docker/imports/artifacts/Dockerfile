FROM log2timeline/plaso:20200227

RUN apt-get -y update \
    && apt-get -y install curl unzip python3-distutils python3-dfvfs python3-artifacts libregf-python3 \
    && apt-get clean \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/*

RUN rm -rf data /artifacts artifacts.zip artifacts.tar.gz \
    && curl --fail --silent --output artifacts.zip --location https://github.com/forensicanalysis/artifacts/archive/v0.6.1.zip \
    && unzip artifacts.zip \
	&& mkdir /artifacts \
    && mv -fv artifacts-0.6.1/*.yaml /artifacts/

COPY artifacts.yaml /artifacts/artifacts.yaml
COPY for500.yaml /artifacts/for500.yaml

COPY requirements.txt .
RUN \
    curl --silent https://bootstrap.pypa.io/get-pip.py -o get-pip.py \
    && python3 get-pip.py \
    && rm get-pip.py \
    && pip3 install -U pip \
    && pip3 install -r requirements.txt

COPY . /app
ENTRYPOINT ["python3", "/app/main_cmd.py", "-a", "/artifacts", "-o", "/store", "-e", "DefaultEntryPoint", "-i", "/transit"]
CMD []
