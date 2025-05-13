within this folder, run protoc cmd to generate pipeline.pb.go
```
protoc --go_out=. --go_opt=paths=source_relative   --go-grpc_out=. --go-grpc_opt=paths=source_relative --proto_path=/absolute/path/tp/librarian/proto pipeline.proto
```