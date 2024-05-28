lint:
	golangci-lint run --max-same-issues=0 --max-issues-per-linter=0

lint-fmt:
	gofumpt -l -w .
	gci write --skip-generated -s standard,default .

test:
	go test ./...
proto:
	protoc --go_out=. --proto_path=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/protobuf/*.proto

docker-tabot:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -installsuffix cgo -tags timetzdata -o build/app cmd/tabot/main.go
	docker build -t tabot -f build/Dockerfile --platform linux/amd64 .

docker-prebot:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -installsuffix cgo -tags timetzdata -o build/app cmd/prebot/main.go
	docker build -t prebot -f build/Dockerfile --platform linux/amd64 .

clean:
	rm -rf build/app
	docker rmi tabot
	docker rmi prebot