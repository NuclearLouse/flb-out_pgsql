.PHONY: windows
windows:
	go build -mod vendor -v -buildmode=c-shared -o ./bin/windows/flb-out_pgsql.so .

.PHONY: linux
linux:
	go build -mod vendor -v -buildmode=c-shared -o ./bin/linux/flb-out_pgsql.so .

