.PHONY: \
	build \
	clean \
	migrate-down \
	migrate-up \
	postgres \
	postgres-clean \
	postgres-kill \
	psql \
	sqlc

build: ./target/gopodder

./target/gopodder: sqlc
	mkdir -p ./target
	go build -o ./target/gopodder ./cmd/gopodder/main.go

clean: postgres-clean
	rm -rf ./internal/generated
	find . -type l -name 'result*' -delete
	rm -rf $$(readlink ./tmp)
	rm -rf ./target
	-unlink ./tmp

sqlc:
	sqlc generate

migrate-up:
	migrate -path "./database/migrations" -database "postgresql://gopodder@/gopodder?host=$$(pwd)/tmp" up

migrate-down:
	migrate -path "./database/migrations" -database "postgresql://gopodder@/gopodder?host=$$(pwd)/tmp" down

./tmp:
	ln -sf $$(mktemp --directory /tmp/gopodder.XXXXXX) ./tmp

./tmp/.pgdata: ./tmp
	initdb \
		--username=gopodder \
		-D ./tmp/.pgdata \
		--auth-local=trust

postgres: ./tmp/.pgdata
	if [ ! -f ./tmp/.pgdata/postmaster.pid ]; then \
		pg_ctl -D ./tmp/.pgdata start -o "-c unix_socket_directories=$$(pwd)/tmp -c listen_addresses=''"; \
		psql -h $$(readlink ./tmp) -U gopodder postgres -c "create database gopodder;" 2>/dev/null || true; \
	fi
	while [ ! -S ./tmp/.s.PGSQL.5432 ]; do sleep 0.5; done

postgres-kill:
	if [ -f ./tmp/.pgdata/postmaster.pid ]; then \
		kill $$(head -n1 ./tmp/.pgdata/postmaster.pid); \
		while [ -f ./tmp/.pgdata/postmaster.pid ]; do sleep 0.5; done \
		fi

postgres-clean: postgres-kill
	rm -rf ./tmp/postgres ./tmp/.pgdata

psql:
	psql -U gopodder "postgresql://gopodder@/gopodder?host=$$(pwd)/tmp"