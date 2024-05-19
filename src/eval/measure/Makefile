
LATENCY_SENDER_TCP_IMAGE=flok3n/latency_sender_tcp:1.0.0
LATENCY_RECEIVER_TCP_IMAGE=flok3n/latency_receiver_tcp:1.0.0

LATENCY_SENDER_UDP_IMAGE=flok3n/latency_sender_udp:1.0.4
LATENCY_RECEIVER_UDP_IMAGE=flok3n/latency_receiver_udp:1.0.4

LATENCY_SENDER_HTTP_IMAGE=flok3n/latency_sender_http:1.0.5
LATENCY_RECEIVER_HTTP_IMAGE=flok3n/latency_receiver_http:1.0.5

.PHONY: docker_udp
docker_udp:
	docker build -f Dockerfile.sender_udp -t $(LATENCY_SENDER_UDP_IMAGE) .
	sudo docker push $(LATENCY_SENDER_UDP_IMAGE)

	docker build -f Dockerfile.receiver_udp -t $(LATENCY_RECEIVER_UDP_IMAGE) .
	sudo docker push $(LATENCY_RECEIVER_UDP_IMAGE)

.PHONY: docker_tcp
docker_tcp:
	docker build -f Dockerfile.sender_tcp -t $(LATENCY_SENDER_TCP_IMAGE) .
	sudo docker push $(LATENCY_SENDER_TCP_IMAGE)

	docker build -f Dockerfile.receiver_tcp -t $(LATENCY_RECEIVER_TCP_IMAGE) .
	sudo docker push $(LATENCY_RECEIVER_TCP_IMAGE)

.PHONY: docker_http
docker_http:
	docker build -f Dockerfile.sender_http_asyncio -t $(LATENCY_SENDER_HTTP_IMAGE) .
	sudo docker push $(LATENCY_SENDER_HTTP_IMAGE)

	docker build -f Dockerfile.receiver_http -t $(LATENCY_RECEIVER_HTTP_IMAGE) .
	sudo docker push $(LATENCY_RECEIVER_HTTP_IMAGE)
