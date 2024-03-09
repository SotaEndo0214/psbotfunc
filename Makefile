# ビルドタスク
.PHONY: build
build: 
	go build -o bin/

# vendor 
.PHONY: vendor
vendor: 
	go mod why & go mod tidy & go mod vendor