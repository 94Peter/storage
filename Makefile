BUILD_TIME=`date +%FT%T%z`
LDFLAGS=-ldflags "-X main.Version=${V} -X main.BuildTime=${BUILD_TIME}"
NAME=storage

gen-code:
	protoc --go_out=. --go-grpc_out=. grpc/proto/*.proto
	
build-docker-img:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(NAME):dev .
	docker rmi -f $$(docker images --filter "dangling=true" -q --no-trunc)

push-docker:
	docker tag $(NAME):dev 94peter/$(NAME):$(V)
	docker push 94peter/$(NAME):$(V)
	docker rmi $$(docker images --filter "dangling=true" -q --no-trunc)


run: build
	./bin/$(NAME)

build: clear
	go build ${LDFLAGS} -o ./bin/$(NAME) ./container/main.go
	./bin/$(NAME) -v

clear:
	rm -rf ./bin/$(SER)

clear-untag-image: 
	docker rmi -f $$(docker images --filter "dangling=true" -q --no-trunc)


# ## Push image

# gcloud auth configure-docker
# docker tag <image-name>:<tag> asia.gcr.io/muulin-universal/<image-name>:<tag>
# docker push asia.gcr.io/muulin-universal/<image-name>:<tag>

# ## get Kubernetes config
# gcloud container clusters get-credentials muulin-gcp-1 --zone=asia-east1

# ## test 
# kubectl get nodes