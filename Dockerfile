FROM ubuntu:20.04 as builder
ENV DEBIAN_FRONTEND=noninteractive
RUN apt update  && \
    apt -y install gnupg curl

# Install nodejs
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash - && \
    apt update && \
    apt -y install nodejs && \
    node --version

# Install yarn globally with npm (installs a yarn compatible with the node version we are using)
RUN npm install -g yarn

# Install golang
RUN apt -y install golang && \
    mkdir -p /go && \
    go version

# Set GOPATH
ENV GOPATH=/go
ENV PATH=$PATH:$GOPATH/bin

# Install mage
RUN apt -y install git && \
    go get -u -d github.com/magefile/mage && \
    cd $GOPATH/src/github.com/magefile/mage && \
    go run bootstrap.go && \
    mage -version

COPY ./ /opt/grafana-plugins/mqtt-datasource
WORKDIR /opt/grafana-plugins/mqtt-datasource

# Install grafana toolikit on different RUNs so that we avoid unecessary downloads
RUN yarn add @grafana/toolkit --network-timeout 100000
RUN yarn build
RUN yarn install

FROM grafana/grafana:8.5.10-ubuntu
COPY --from=builder /opt/grafana-plugins/mqtt-datasource/dist /var/lib/grafana/plugins/mqtt-datasource/



