all:
	go mod tidy
	go build ./build/arseeding ./cmd

gen-graphql:
	go get github.com/Khan/genqlient
	genqlient ./argraphql/genqlient.yaml

build-linux-bin:
	GOOS=linux GOARCH=amd64 go build -o ./build/arseeding ./cmd