.PHONY: windows
windows:
	go build -mod vendor -v -buildmode=c-shared -o ./bin/windows/out_postgres.so .

.PHONY: linux
linux:
	go build -mod vendor -v -buildmode=c-shared -o ./bin/linux/out_postgres.so .

