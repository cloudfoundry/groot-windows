module code.cloudfoundry.org/groot-windows

go 1.15

require (
	code.cloudfoundry.org/archiver v0.0.0-20210513174825-6979f8d756e2 // indirect
	code.cloudfoundry.org/filelock v0.0.0-20180314203404-13cd41364639
	code.cloudfoundry.org/groot v0.0.0-20211012145205-290e4e816cb0
	code.cloudfoundry.org/hydrator v0.0.0-20180411234439-6b2757c7f6f0
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Microsoft/go-winio v0.5.0
	github.com/Microsoft/hcsshim v0.8.20
	github.com/checkpoint-restore/go-criu/v4 v4.1.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/pquerna/ffjson v0.0.0-20190813045741-dac163c6c0a9 // indirect
	github.com/urfave/cli v1.22.5
	github.com/vbauerster/mpb/v6 v6.0.3 // indirect
	github.com/willf/bitset v1.1.11 // indirect
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e
)

replace (
	github.com/Microsoft/go-winio => github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.7
)
