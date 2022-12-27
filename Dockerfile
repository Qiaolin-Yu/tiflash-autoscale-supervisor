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
ENV RUST_BACKTRACE=1
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Etc/UTC
RUN echo $TZ > /etc/timezone && \
    apt-get update && apt-get install -y tzdata && \
    rm /etc/localtime && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata && \
    apt-get clean
RUN apt-get update && apt-get install -y \
    git \
    bash \
    psmisc \
    curl
RUN mkdir /tiflash
COPY bin  /tiflash/bin/
COPY go1.19.2.linux-amd64.tar.gz /tiflash/
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf /tiflash/go1.19.2.linux-amd64.tar.gz
COPY conf/  /tiflash/conf/
COPY supervisor_proto/ /tiflash/supervisor_proto/
COPY *.go go.* *.yaml *.yml *.sh /tiflash/
RUN ls -lh /tiflash/
WORKDIR /tiflash

ENV PATH="/usr/local/go/bin:${PATH}"


ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
RUN go mod tidy
RUN go build -o rpc_server
# COPY --from=builder /tiflash/rpc_server /rpc_server
EXPOSE 7000 8237 8126 9003 20173 20295 3933
# CMD ["/bin/sh", "-c", "ls -lh /tiflash/rpc_server"]
# CMD ["/bin/sh", "-c", "pwd"]
# ENTRYPOINT ["/bin/sh", "-c", "go -version"]
ENTRYPOINT ["/tiflash/run.sh"]
# ENTRYPOINT ["/tiflash/rpc_server > /var/log/tiflash_supervisor.log"]