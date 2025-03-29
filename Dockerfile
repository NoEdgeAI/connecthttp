FROM m.daocloud.io/docker.io/library/golang:1.24 AS builder
ENV CGO_ENABLED=0
WORKDIR /src
COPY . .
RUN go build -o protoc-gen-connecthttp-go ./cmd/protoc-gen-connecthttp-go

FROM gcr.m.daocloud.io/distroless/static-debian12:latest
COPY --from=builder /src/protoc-gen-connecthttp-go /protoc-gen-connecthttp-go
ENTRYPOINT [ "/protoc-gen-connecthttp-go" ]