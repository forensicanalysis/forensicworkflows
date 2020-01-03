module github.com/forensicanalysis/forensicworkflows

go 1.13

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/containerd/containerd v1.3.2 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.4.2-0.20191108192604-36ffe9edc2b3
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/forensicanalysis/forensicstore v0.0.0-00010101000000-000000000000
	github.com/hashicorp/logutils v1.0.0
	github.com/hashicorp/terraform v0.12.17
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	gopkg.in/yaml.v2 v2.2.4
)

replace github.com/forensicanalysis/fslib => ../fslib

replace github.com/forensicanalysis/forensicstore => ../forensicstore
