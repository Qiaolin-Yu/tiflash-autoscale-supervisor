FROM golang:latest
RUN mkdir /tiflash
ADD . /tiflash/
WORKDIR /tiflash
ENV RUST_BACKTRACE=1
ENV TZ=Asia/Shanghai
ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
RUN go mod tidy
RUN go build -o rpc_server
EXPOSE 7000 8237 8126 9003 20173 20295 3933
CMD ["/tiflash/rpc_server"]