NPM_BIN := $(shell readlink -f "web/node_modules/.bin")

build-assets:
	$(NPM_BIN)/node-sass web/static/sass/application.scss assets/static/css/application.css \
		&& $(NPM_BIN)/postcss assets/static/css/application.css -o assets/static/css/application.css \
			--config web
	go-bindata -o assets/assets.go -pkg assets assets/...

build: build-assets
	go build

clean:
	@rm assets/assets.go
	@rm -rf assets/static

lint:
	go fmt ./...


test:
	go test ./...
