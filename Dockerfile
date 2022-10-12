# FROM golang:1.19-alpine as builder

# # RUN apk add --no-cache \
# #     make \
# #     git \
# #     bash \
# #     curl \
# #     gcc \
# #     g++

# RUN mkdir /tiflash
# WORKDIR /tiflash
# COPY go.mod ./
# COPY go.sum ./
# COPY *.go ./
# COPY supervisor_proto ./supervisor_proto
# ENV PATH="/usr/local/go/bin:${PATH}"
# ENV RUST_BACKTRACE=1
# ENV TZ=Asia/Shanghai
# ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
# RUN go mod tidy
# RUN go build -o rpc_server
# CMD ["/tiflash/rpc_server"]

FROM ubuntu:20.04
RUN apt-get update && apt-get install -y \
    git \
    bash \
    curl
RUN mkdir /tiflash
ADD . /tiflash/
WORKDIR /tiflash
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.2.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
ENV RUST_BACKTRACE=1
ENV TZ=Asia/Shanghai
ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
RUN go mod tidy
RUN go build -o rpc_server
# COPY --from=builder /tiflash/rpc_server /rpc_server
EXPOSE 7000 8237 8126 9003 20173 20295 3933
# CMD ["/bin/sh", "-c", "ls -lh /tiflash/rpc_server"]
# CMD ["/bin/sh", "-c", "pwd"]
# ENTRYPOINT ["/bin/sh", "-c", "go -version"]
ENTRYPOINT ["/tiflash/rpc_server"]