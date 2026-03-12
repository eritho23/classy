.PHONY: \
	build \
	clean \
	migrate-down \
	migrate-up \
	nginx \
	nginx-clean \
	nginx-kill \
	postgres \
	postgres-clean \
	postgres-kill \
	psql \
	dev \
	sqlc \
	templ

build: ./target/classy

./target/classy: sqlc templ
	mkdir -p ./target
	go build -o ./target/classy ./cmd/classy/main.go

clean: postgres-clean nginx-clean
	rm -rf ./internal/generated
	find . -type l -name 'result*' -delete
	rm -rf $$(readlink ./tmp)
	rm -rf ./target
	find . -name '*_templ.go' -delete
	-unlink ./tmp

templ:
	templ generate

sqlc:
	sqlc generate

migrate-up:
	migrate -path "./migrations" -database "postgresql://classy@/classy?host=$$(pwd)/tmp" up

migrate-down:
	migrate -path "./migrations" -database "postgresql://classy@/classy?host=$$(pwd)/tmp" down

./tmp:
	ln -sf $$(mktemp --directory /tmp/classy.XXXXXX) ./tmp

./tmp/.pgdata: ./tmp
	initdb \
		--username=classy \
		-D ./tmp/.pgdata \
		--auth-local=trust

postgres: ./tmp/.pgdata
	if [ ! -f ./tmp/.pgdata/postmaster.pid ]; then \
		pg_ctl -D ./tmp/.pgdata start -o "-c unix_socket_directories=$$(pwd)/tmp -c listen_addresses=''"; \
		psql -h $$(readlink ./tmp) -U classy postgres -c "create database classy;" 2>/dev/null || true; \
	fi
	while [ ! -S ./tmp/.s.PGSQL.5432 ]; do sleep 0.5; done

postgres-kill:
	if [ -f ./tmp/.pgdata/postmaster.pid ]; then \
		kill $$(head -n1 ./tmp/.pgdata/postmaster.pid); \
		while [ -f ./tmp/.pgdata/postmaster.pid ]; do sleep 0.5; done \
		fi
	-killall postgres

postgres-clean: postgres-kill
	rm -rf ./tmp/postgres ./tmp/.pgdata

psql:
	psql -U classy "postgresql://classy@/classy?host=$$(pwd)/tmp"

dev: postgres nginx sqlc
	air

nginx: ./tmp
	nginx -c $$(pwd)/config/nginx.conf -p $$(pwd) -g "pid ./tmp/nginx.pid;"
	while [ ! -f ./tmp/nginx.pid ]; do sleep 0.5; done

nginx-kill:
	if [ -f ./tmp/nginx.pid ]; then \
		kill $$(cat ./tmp/nginx.pid); \
		while [ -f ./tmp/nginx.pid ]; do sleep 0.5; done; \
		fi
	-killall nginx

nginx-clean: nginx-kill