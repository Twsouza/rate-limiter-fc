
.PHONY: all build up down test clean run

all: run

build:
	docker-compose -f docker-compose.load-test.yml build

up:
	docker-compose -f docker-compose.load-test.yml up -d web redis

down:
	docker-compose -f docker-compose.load-test.yml down

test:
	@echo "Waiting for the application to start..."
	@until docker exec $$(docker-compose -f docker-compose.load-test.yml ps -q web) wget -q -O- http://localhost:8080/ > /dev/null; do \
		sleep 1; \
		echo "Waiting for application..."; \
	done
	@echo "Application is up and running!"
	docker-compose -f docker-compose.load-test.yml run --rm k6

clean:
	docker-compose -f docker-compose.load-test.yml down -v
	rm -R k6_results

run: k6_results up test

run-clean: run down

k6_results:
	mkdir -p k6_results
	chmod 777 k6_results

.env:
	cp .env.example .env
