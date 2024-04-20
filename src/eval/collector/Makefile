
IMAGE=flok3n/int-collector:1.0.2

.PHONY: docker
docker:
	docker build -t $(IMAGE) .
	sudo docker push $(IMAGE)

.PHONY: run
run:
	python server.py
