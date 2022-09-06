module code.cloudfoundry.org/groot-windows

go 1.15

require (
	code.cloudfoundry.org/archiver v0.0.0-20210513174825-6979f8d756e2 // indirect
	code.cloudfoundry.org/filelock v0.0.0-20180314203404-13cd41364639
	code.cloudfoundry.org/groot v0.0.0-20220906222815-4b69681f4d8b
	code.cloudfoundry.org/hydrator v0.0.0-20180411234439-6b2757c7f6f0
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Microsoft/go-winio v0.5.2
	github.com/Microsoft/hcsshim v0.9.3
	github.com/bits-and-blooms/bitset v1.2.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/juju/ansiterm v0.0.0-20180109212912-720a0952cc2a // indirect
	github.com/lunixbochs/vtclean v0.0.0-20180621232353-2d01aacdc34a // indirect
	github.com/mtrmac/gpgme v0.1.2 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/urfave/cli v1.22.5
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8
)

replace (
	github.com/Microsoft/go-winio => github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.7
)
