lint:
	golangci-lint run --max-same-issues=0 --max-issues-per-linter=0

lint-fmt:
	gofumpt -l -w .
	gci write --skip-generated -s standard,default .

test:
	go test ./...
proto:
	protoc --go_out=. --proto_path=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/protobuf/*.proto
