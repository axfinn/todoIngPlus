#!/bin/sh
set -e

PROTO_DIR="api/proto/v1"
OUT_DIR="docs/swagger"
PLUGIN_BIN="/Users/finn/go/bin/protoc-gen-openapiv2"

PATH=$PATH:/Users/finn/go/bin

mkdir -p "$OUT_DIR"

if [ ! -x "$PLUGIN_BIN" ]; then
  echo "❌ 未找到 protoc-gen-openapiv2: $PLUGIN_BIN" >&2
  echo "运行: go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest" >&2
  exit 1
fi

echo "===> 生成 OpenAPI"
protoc \
  -I "$PROTO_DIR" \
  -I third_party \
  --plugin=protoc-gen-openapiv2="$PLUGIN_BIN" \
  --openapiv2_out "$OUT_DIR" \
  --openapiv2_opt generate_unbound_methods=true,allow_merge=true,merge_file_name=openapi \
  $(ls $PROTO_DIR/*.proto)

if [ -f "$OUT_DIR/openapi.swagger.json" ]; then
  echo "✅ 输出: $OUT_DIR/openapi.swagger.json"
else
  echo "⚠️ 未生成 JSON，查看目录: $OUT_DIR" >&2
fi
