PROJECT = "ptun"
BUILD_DIR = "./build"

all: hub node

hub:
	mkdir -p ${BUILD_DIR}
	go build -o ${BUILD_DIR}/hub ./cmd/hub 
	cp ./conf/ptun-hub.toml ${BUILD_DIR}/

node:
	mkdir -p ${BUILD_DIR}
	go build -o ${BUILD_DIR}/node ./cmd/node
	cp ./conf/ptun-node1.toml ${BUILD_DIR}/ptun-node1.toml
	cp ./conf/ptun-node2.toml ${BUILD_DIR}/ptun-node2.toml

.PHONY: hub node
