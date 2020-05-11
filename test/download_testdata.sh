mkdir -p test/data

rm -rf test/data/example1.forensicstore
if [ ! -f "example1.forensicstore" ]; then
    curl --fail --silent --output example1.forensicstore --location https://download.artifacthub.org/forensics/example1.forensicstore
fi
mv example1.forensicstore test/data

rm -rf test/data/example2.forensicstore
if [ ! -f "example2.forensicstore" ]; then
    curl --fail --silent --output example2.forensicstore --location https://download.artifacthub.org/forensics/example2.forensicstore
fi
mv example2.forensicstore test/data

rm -rf test/data/usb.forensicstore
if [ ! -f "usb.forensicstore" ]; then
    curl --fail --silent --output usb.forensicstore --location https://download.artifacthub.org/forensics/usb.forensicstore
fi
mv usb.forensicstore test/data

if [ ! -f "win10_mock.zip" ]; then
    curl --fail --silent --output win10_mock.zip --location https://download.artifacthub.org/windows/win10_mock.zip
fi
unzip win10_mock.zip
mv win10_mock.vhd test/data
