.PHONY: build
build:
	go build -mod vendor -v -buildmode=c-shared -o ../flb-out_pgsql_bin_windows/flb-out_pgsql.so .

.DEFAULT_GOAL := build