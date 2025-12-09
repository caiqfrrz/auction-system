#!/bin/bash

set -e

echo "Generating protobuf code..."

# Gerar código para leilao
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/leilao/leilao.proto

# Gerar código para lance
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/lance/lance.proto

# Gerar código para pagamento
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/pagamento/pagamento.proto

echo "Protobuf code generated successfully!"