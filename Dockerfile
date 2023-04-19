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
RUN mkdir /tiflash && mkdir /tiflash/bin
COPY bin  /tiflash/bin/
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN cd /tiflash/ && echo $TARGETPLATFORM \
    && if [ "$TARGETPLATFORM" = "linux/arm64" ] ; then SIMPLE_ARCH=arm64 ; else  SIMPLE_ARCH=amd64 ; fi \
    && wget --no-check-certificate https://go.dev/dl/go1.19.5.linux-$SIMPLE_ARCH.tar.gz \
    && pwd && ls -lh \
    && rm -rf /usr/local/go && tar -C /usr/local -xzf /tiflash/go1.19.5.linux-$SIMPLE_ARCH.tar.gz

COPY conf/  /tiflash/conf/
COPY supervisor_proto/ /tiflash/supervisor_proto/
COPY tiflashrpc/ /tiflash/tiflashrpc/
COPY *.go go.* *.yaml *.yml *.sh /tiflash/
RUN ls -lh /tiflash/
WORKDIR /tiflash

RUN if [ "$TARGETPLATFORM" = "linux/arm64" ] ; then SIMPLE_ARCH=aarch64 ; else  SIMPLE_ARCH=x86_64 ; fi \
    && curl "https://awscli.amazonaws.com/awscli-exe-linux-$SIMPLE_ARCH.zip" -o "awscliv2.zip" && unzip awscliv2.zip && ./aws/install
# without as switch
#COPY --from=xexplorersun/tiflash:cse-3dc23cc935cfdde155df657f73dd385a27f40031 /tiflash/* /tiflash/bin/
# COPY --from=xexplorersun/tiflash:cse-bbecb57e8a3c1588792e2643a44b21609a0ec25c /tiflash/* /tiflash/bin/
COPY --from=xexplorersun/tiflash:cse-0f7f8ae4d0c59c1d2d25692641fb40d7466dd1b4 /tiflash/* /tiflash/bin/
RUN ls -lh /tiflash/bin/ && /tiflash/bin/tiflash version

ENV PATH="/usr/local/go/bin:${PATH}"


ENV LD_LIBRARY_PATH="/tiflash/bin:$LD_LIBRARY_PATH"
RUN go mod tidy
RUN go build -o rpc_server

EXPOSE 7000 8237 8126 9003 20173 20295 3933

ENTRYPOINT ["/tiflash/run.sh"]
