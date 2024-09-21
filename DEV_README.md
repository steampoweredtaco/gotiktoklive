## Generating Protobuf
```bash
protoc --go_out=. -I .\proto --go-grpc_out=. .\proto\enums.proto
protoc --go_out=. -I .\proto --go-grpc_out=. .\proto\data.proto
protoc --go_out=. -I .\proto  --go-grpc_out=. .\proto\webcast.proto
```
