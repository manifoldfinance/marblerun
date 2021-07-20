# syntax=docker/dockerfile:experimental

FROM alpine/git:latest AS pull
RUN git clone https://github.com/edgelesssys/marblerun.git /marblerun

FROM ghcr.io/edgelesssys/edgelessrt-deploy:latest AS release
RUN apt-key adv --fetch-keys https://packages.grapheneproject.io/graphene.asc
RUN echo 'deb [arch=amd64 signed-by=EA3C2D624681AC968521587A5EE1171912234070] https://packages.grapheneproject.io/ unstable main' | tee /etc/apt/sources.list.d/graphene-unstable.list
RUN apt-get update
# TODO: Many of this should not be required, they should be listed as dependencies in graphene's package instead
RUN apt-get install -y build-essential wget graphene-dcap python3-click python3-jinja2
# RUN apt-get install -y build-essential wget graphene-dcap libsgx-quote-ex-dev libsgx-aesm-launch-plugin python3-click python3-jinja2 python3-pip python3-protobuf libprotobuf-c-dev libsgx-aesm-launch-plugin sgx-aesm-service libsgx-aesm-ecdsa-plugin && pip3 install "toml>=0.10"

COPY --from=pull /marblerun/samples/graphene-redis /graphene-redis
WORKDIR /graphene-redis
RUN wget "https://github.com/edgelesssys/marblerun/releases/latest/download/premain-libos" && chmod u+x premain-libos
ENV BUILD_TLS yes
COPY Makefile Makefile
RUN make clean && make SGX=1
ENTRYPOINT ["graphene-sgx", "/graphene-redis/redis-server" ]
