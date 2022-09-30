FROM golang:latest
RUN mkdir /tiflash
ADD . /tiflash/
WORKDIR /tiflash
ENV RUST_BACKTRACE=1
ENV TZ=Asia/Shanghai
ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
RUN go mod tidy
RUN go build -o rpc_server
CMD ["/tiflash/rpc_server"]