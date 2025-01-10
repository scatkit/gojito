#!/bin/bash

# Repostitory with all the proto buffers 
REPO_URL="https://github.com/jito-labs/mev-protos"
REPO_DIR="mev-protos"
OUTPUT_DIR="../pb"
IMPORT_PATH="github.com/jito-labs/mev-protos/jito_pb"
 
echo "Cloning repo..."
git clone $REPO_URL

if [ ! -d "$REPO_DIR" ]; then
  echo "Failed to clone repo"
  exit 1
fi

# pwd: scripts/mev-protos/
cd $REPO_DIR

PROTO_FILES=$(find . -name '*.proto')
MAPPING_ARGS=""

for file in $PROTO_FILES; do
  REL_PATH="${file#./}"
  # add the mapping arg with the correct Go import path
  MAPPING_ARGS+="M${REL_PATH}=${IMPORT_PATH},"
done

mkdir -p "../$OUTPUT_DIR"

echo "Generating Go code from protobuf definitions..."
protoc --proto_path=. --go_out="../$OUTPUT_DIR" --go_opt=paths=source_relative \
       --go-grpc_out="../$OUTPUT_DIR" --go-grpc_opt=paths=source_relative \
       --go_opt=${MAPPING_ARGS%,} --go-grpc_opt=${MAPPING_ARGS%,} \
       $(find . -name '*.proto')

if [ $? -ne 0 ]; then
  echo "Failed to generate Go code from protobuf definitions."
  exit 1
fi

cd ..

echo "Cleanig up..."
rm -rf $REPO_DIR

echo "Done. All files are in '$OUTPUT_DIR dir"
