OBJECTS = dsim


## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	git diff --exit-code


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## build: build the application
.PHONY: build
build: $(OBJECTS)

dsim:
	GOARCH=amd64 GOOS=linux go build -o=./cmd/dsim-launcher/dsim-launcher-amd64 ./cmd/dsim-launcher/dsim-launcher.go ./cmd/dsim-launcher/ssh-utils.go
	GOARCH=amd64 GOOS=linux go build -o=./cmd/dsim-node/dsim-node-amd64 ./cmd/dsim-node/dsim-node.go
	GOARCH=arm64 GOOS=linux go build -o=./cmd/dsim-launcher/dsim-launcher-arm64 ./cmd/dsim-launcher/dsim-launcher.go ./cmd/dsim-launcher/ssh-utils.go
	GOARCH=arm64 GOOS=linux go build -o=./cmd/dsim-node/dsim-node-arm64 ./cmd/dsim-node/dsim-node.go

run-3subredes:
	rm -rf ~/dsim/logs/* && ./cmd/dsim-launcher/dsim-launcher-amd64 \
		-nodeFile ./data/simulation-nodes.json \
		-nodeCmd /home/mursisoy/Documents/courses/MscCS/redsisdis/CodigoSuministradoAAlumnos/distributed-petri-net-simulator/cmd/dsim-node/dsim-node-amd64 \
		-period 2 \
		./data/3subredes
		
run-6subredes:
	rm -rf ~/dsim/logs/* && ./cmd/dsim-launcher/dsim-launcher-amd64 \
		-nodeFile ./data/simulation-nodes.json \
		-nodeCmd /home/mursisoy/Documents/courses/MscCS/redsisdis/CodigoSuministradoAAlumnos/distributed-petri-net-simulator/cmd/dsim-node/dsim-node-amd64 \
		-period 200 \
		./data/6subredes

run-1subred:
	rm -rf ~/dsim/logs/* && ./cmd/dsim-launcher/dsim-launcher-amd64 \
		-nodeFile ./data/simulation-nodes.json \
		-nodeCmd /home/mursisoy/Documents/courses/MscCS/redsisdis/CodigoSuministradoAAlumnos/distributed-petri-net-simulator/cmd/dsim-node/dsim-node-amd64 \
		-period 10
		./data/2ramasDe2.rdp

shiviz-log:
	echo '(?<date>(\d{2}:){2}\d{2}.\d{6}) (?<path>\S*): \[(?<priority>(INFO|DEBUG|WARNING|ERROR))\] - (?<event>.*)\n(?<host>\S*) (?<clock>{.*})' > ~/dsim/logs/shiviz.log && \
	echo "" >> ~/dsim/logs/shiviz.log && \
	cat ~/dsim/logs/dsim-launcher.log ~/dsim/logs/sn*.log >> ~/dsim/logs/shiviz.log

## clean: clean built files
.PHONY: clean
clean:
	go clean
	rm bin/*