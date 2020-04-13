mkdir -p test/data

rm -rf test/data/example1.forensicstore
if [ ! -f "example1.forensicstore.zip" ]; then
    curl --fail --silent --output example1.forensicstore.zip --location https://download.artifacthub.org/forensics/example1.forensicstore.zip
fi
unzip example1.forensicstore.zip
mv example1.forensicstore test/data

rm -rf test/data/example2.forensicstore
if [ ! -f "example2.forensicstore.zip" ]; then
    curl --fail --silent --output example2.forensicstore.zip --location https://download.artifacthub.org/forensics/example2.forensicstore.zip
fi
unzip example2.forensicstore.zip
mv example2.forensicstore test/data

rm -rf test/data/usb.forensicstore
if [ ! -f "usb.forensicstore.zip" ]; then
    curl --fail --silent --output usb.forensicstore.zip --location https://download.artifacthub.org/forensics/usb.forensicstore.zip
fi
unzip usb.forensicstore.zip
mv usb.forensicstore test/data

if [ ! -f "win10_mock.zip" ]; then
    curl --fail --silent --output win10_mock.zip --location https://download.artifacthub.org/windows/win10_mock.zip
fi
unzip win10_mock.zip
mv win10_mock.vhd test/data
