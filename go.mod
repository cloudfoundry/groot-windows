module code.cloudfoundry.org/groot-windows

go 1.15

require (
	code.cloudfoundry.org/archiver v0.0.0-20210513174825-6979f8d756e2 // indirect
	code.cloudfoundry.org/filelock v0.0.0-20180314203404-13cd41364639
	code.cloudfoundry.org/groot v0.0.0-20210615090005-2975580e9665
	code.cloudfoundry.org/hydrator v0.0.0-20180411234439-6b2757c7f6f0
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Microsoft/go-winio v0.5.0
	github.com/Microsoft/hcsshim v0.8.17
	github.com/containers/image v3.0.2+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/urfave/cli v1.22.5
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
)

replace github.com/Microsoft/go-winio => github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
