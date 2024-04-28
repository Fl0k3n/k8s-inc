
IMAGE=flok3n/int-collector:1.0.3

.PHONY: docker
docker:
	docker build -t $(IMAGE) .
	sudo docker push $(IMAGE)

.PHONY: run
run:
	python collector.py
