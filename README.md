# techtask

Prepared on Ubuntu 23.04, Go 1.19.

For golang installation refer to: https://go.dev/doc/install

To install dependencies run:
```
go mod download
```

To build, call:
```
go build ./cmd/filefilter/filefilter.go
```

To run executable, call:
```
./filefilter --hot <your_hot_dir> --backup <your_backup_dir>
```