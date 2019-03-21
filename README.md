# groot-windows

A [Garden](https://github.com/cloudfoundry/garden) image plugin for Windows.

## Building

Make sure `GOPATH` is set. Then run:

```
GOOS=windows go build .
```

It generates a `groot-windows.exe` in the current directory.

## Usage

```
groot-windows.exe [global options] command [command options] [arguments...]
```

#### Notes

`groot pull`: Downloads the layers from the image registry if remote, and unpacks each layer into *directories* of the same name/digest located at `<driver-store>/layers`. If `<driver-store>/layers` already contain the same unpacked layers, this is a NOOP.

`groot create`: Runs a `groot pull`, uses the relevant layers to create a virtual Hard disk file inside `<driver-store>/volumes`, mounts it as a Windows Volume path and returns a valid [runtime spec](https://github.com/opencontainers/runtime-spec/blob/master/specs-go/config.go) on stdout.


#### Examples

```
groot-windows.exe --driver-store="c:\ProgramData\groot" create "oci:///C:/hydratorOutput" container1
```

```
groot-windows.exe --driver-store="c:\ProgramData\groot" delete container1
```

Use `groot-windows.exe --help` to show detailed usage.

## Testing

#### Requirements

You must be in a windows environment to run these tests.

To run the entire suite of tests, run `ginkgo -r -race -p .`
