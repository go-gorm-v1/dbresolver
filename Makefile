test:
	go mod vendor
	docker build . -t gorm_v1_dbresolver
	docker run --rm --name test_run gorm_v1_dbresolver all

build.docs:
	npm i markdown-to-standalone-html
	cp ./templates/template.html node_modules/markdown-to-standalone-html/templates/template.toc.html

render.docs: build.docs
	node_modules/markdown-to-standalone-html/dist/markdown-to-standalone-html.js README.md -hs intellij-light -o index.html

