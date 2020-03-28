mkdir -p test/data

curl --fail --silent --output example1.forensicstore.zip --location https://download.artifacthub.org/forensics/example1.forensicstore.zip
unzip example1.forensicstore.zip
mv example1.forensicstore test/data

curl --fail --silent --output example2.forensicstore.zip --location https://download.artifacthub.org/forensics/example2.forensicstore.zip
unzip example2.forensicstore.zip
mv example2.forensicstore test/data

curl --fail --silent --output win10_mock.zip --location https://download.artifacthub.org/windows/win10_mock.zip
unzip win10_mock.zip
mv win10_mock.vhd test/data
