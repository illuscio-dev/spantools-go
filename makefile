.PHONY: test
test:
	-go test -covermode=count -coverprofile=./zdevelop/tests/zreports/coverage.out -coverpkg=./... ./...
	-go tool cover -html=./zdevelop/tests/zreports/coverage.out
	open ./zdevelop/tests/zreports/coverage.html

.PHONY: test
format:
	-go fmt ./...
