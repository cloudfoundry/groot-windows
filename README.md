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
