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
    curl \
    unzip \
    wget
RUN mkdir /tiflash
COPY bin  /tiflash/bin/
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN cd /tiflash/ && echo $TARGETPLATFORM \
    && if [ "$TARGETPLATFORM" = "linux/arm64" ] ; then SIMPLE_ARCH=arm64 ; else  SIMPLE_ARCH=amd64 ; fi \
    && wget --no-check-certificate https://go.dev/dl/go1.19.5.linux-$SIMPLE_ARCH.tar.gz \
    && pwd && ls -lh \
    && rm -rf /usr/local/go && tar -C /usr/local -xzf /tiflash/go1.19.5.linux-$SIMPLE_ARCH.tar.gz
# RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM"
# RUN echo $TARGETPLATFORM && go version
# COPY go1.19.2.linux-amd64.tar.gz /tiflash/
COPY conf/  /tiflash/conf/
COPY supervisor_proto/ /tiflash/supervisor_proto/
COPY *.go go.* *.yaml *.yml *.sh /tiflash/
RUN ls -lh /tiflash/
WORKDIR /tiflash

ENV PATH="/usr/local/go/bin:${PATH}"


ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
RUN go mod tidy
RUN go build -o rpc_server
# awscli-exe-linux-aarch64.zip
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ] ; then SIMPLE_ARCH=aarch64 ; else  SIMPLE_ARCH=x86_64 ; fi \
    && curl "https://awscli.amazonaws.com/awscli-exe-linux-$SIMPLE_ARCH.zip" -o "awscliv2.zip" && unzip awscliv2.zip && ./aws/install
COPY --from=xexplorersun/tiflash:cse-3dc23cc935cfdde155df657f73dd385a27f40031 /tiflash/* /rpc_server/bin/
RUN ls -lh /rpc_server/bin/ && /rpc_server/bin/tiflash version
EXPOSE 7000 8237 8126 9003 20173 20295 3933
# CMD ["/bin/sh", "-c", "ls -lh /tiflash/rpc_server"]
# CMD ["/bin/sh", "-c", "pwd"]
# ENTRYPOINT ["/bin/sh", "-c", "go -version"]
ENTRYPOINT ["/tiflash/run.sh"]
# ENTRYPOINT ["/tiflash/rpc_server > /var/log/tiflash_supervisor.log"]