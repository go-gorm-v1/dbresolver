test:
	go mod vendor
	docker build . -t gorm_v1_dbresolver
	docker run --rm --name test_run gorm_v1_dbresolver all

