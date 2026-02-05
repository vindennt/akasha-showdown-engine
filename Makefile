include .env

.PHONY: run build stop make rebuild

build:
	docker build -t $(IMAGE_NAME) .

run:
	docker run --rm --name $(CONTAINER_NAME) --env-file .env -p $(PORT):$(PORT) $(IMAGE_NAME)

stop:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true

make:
	$(MAKE) build
	$(MAKE) run

rebuild:
	$(MAKE) stop
	$(MAKE) build
	$(MAKE) run