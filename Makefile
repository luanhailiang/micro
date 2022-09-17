.PHONY: update
update:
	go mod tidy
	go mod vendor

.PHONY: proto
proto: update
	./proto/bin/protoc -I=./proto --plugin=protoc-gen-go=./proto/bin/protoc-gen-go --go_out=module=github.com/luanhailiang/micro:. proto/broker/*.proto
	./proto/bin/protoc -I=./proto --plugin=protoc-gen-go=./proto/bin/protoc-gen-go --go_out=module=github.com/luanhailiang/micro:. --plugin=protoc-gen-go-grpc=./proto/bin/protoc-gen-go-grpc --go-grpc_out=module=github.com/luanhailiang/micro:. proto/rpcmsg/*.proto
	for f in service/* ; do \
		if [ -d "$${f}/proto" ]; then \
			echo "mongo" $${f} ; \
			sed -i -r 's/json:"([^,]+),omitempty"`/json:"\1,omitempty" bson:"\1"`/g' $${f}/proto/* || true; \
			sed -i -r 's/bson:"_id"`/bson:"_id,omitempty"`/g' $${f}/proto/* || true; \
		fi \
	done

.PHONY: build
build: proto
	for f in broker/connect/* ; do \
		echo "build" $${f} ; \
		CGO_ENABLED=0 GOOS=linux go build -mod vendor -o $${f}/app $${f}/*.go ; \
	done
	# for f in service/* ; do \
	# 	echo "build" $${f} ; \
	# 	CGO_ENABLED=0 GOOS=linux go build -mod vendor -o $${f}/app $${f}/*.go ; \
	# done

.PHONY: package
package: build
	cd upmicro && docker-compose build

.PHONY: docker
docker: package
	cd upmicro && docker-compose up -d --remove-orphans && \
	docker-compose logs -f broker_grpc broker_http broker_tcp broker_web

.PHONY: down
down:
	cd upmicro && docker-compose down

.PHONY: run
run: proto
	CGO_ENABLED=0 GOOS=linux go build -mod vendor -o service/$(APP)/app service/$(APP)/*.go
	cd upmicro && docker-compose build $(APP) && \
	docker-compose up -d $(APP) && \
	docker-compose logs -f $(APP)