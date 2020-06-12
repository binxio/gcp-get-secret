include Makefile.mk

BINARIES=dist/gcp-get-secret-$(VERSION)-linux-amd64.zip dist/gcp-get-secret-$(VERSION)-darwin-amd64.zip
USERNAME=binxio
NAME=gcp-get-secret
GITHUB_API=https://api.github.com/repos/$(USERNAME)/$(NAME)
GITHUB_UPLOAD=https://uploads.github.com/repos/$(USERNAME)/$(NAME)


dist/gcp-get-secret-$(VERSION)-linux-amd64.zip: main.go go.mod
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/gcp-get-secret main.go
	cd dist && zip ../dist/gcp-get-secret-$(VERSION)-linux-amd64.zip gcp-get-secret  && rm gcp-get-secret

dist/gcp-get-secret-$(VERSION)-darwin-amd64.zip: main.go go.mod
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -o dist/gcp-get-secret main.go
	cd dist && zip ../dist/gcp-get-secret-$(VERSION)-darwin-amd64.zip gcp-get-secret  && rm gcp-get-secret

clean:
	rm -rf dist target



.git-release-$(VERSION): $(BINARIES)
	@set -e -o pipefail ; \
	shasum -a256 $(BINARIES) | sed -e 's^dist/^^' -e 's/-$(VERSION)-/-/' | \
	jq --raw-input --slurp \
		--arg tag $(TAG) \
		--arg release $(VERSION) \
		'{ "draft": true,  \
                   "prerelease": false,   \
                   "tag_name": $$tag,  \
                   "name": $$tag,  \
                   "body": ("release " + $$release + (split("\n") | join("\n") | ("\n```\n" + . + "```\n"))) }' | \
	curl -sS --fail \
			-d @- \
			-o .git-release-$(VERSION) \
			-H "Authorization: token $$GITHUB_API_TOKEN" \
			-H 'Content-Type: application/json' \
			-X POST \
			$(GITHUB_API)/releases

release: check-release .git-release-$(VERSION)
	@for BINARY in $(BINARIES); do \
		echo "INFO: uploading $$BINARY.." ; \
		curl --fail -sS \
			 --data-binary @$$BINARY \
			-o /dev/null \
			-X POST \
			-H "Authorization: token $$GITHUB_API_TOKEN" \
			-H 'Content-Type: application/octet-stream' \
			$(GITHUB_UPLOAD)/releases/$(shell jq -r .id .git-release-$(VERSION))/assets?name=$$(basename $${BINARY} | sed -e 's/-$(VERSION)-/-/') ; \
	done
	@curl --fail -sS \
		-d '{"draft": false}'  \
		-o /dev/null \
		-X PATCH \
		-H "Authorization: token $$GITHUB_API_TOKEN" \
		-H 'Content-Type: application/json-stream' \
		$(GITHUB_API)/releases/$(shell jq -r .id .git-release-$(VERSION))
	rm .git-release-$(VERSION)
