ARG FIREFLY_BUILDER_TAG
ARG FABRIC_BUILDER_TAG
ARG FABRIC_BUILDER_PLATFORM
ARG SOLIDITY_BUILDER_TAG
ARG BASE_TAG
ARG BUILD_VERSION
ARG GIT_REF

FROM $FIREFLY_BUILDER_TAG AS firefly-builder
ARG BUILD_VERSION
ARG GIT_REF
RUN apk add make=4.4.1-r2 \
    gcc=13.2.1_git20231014-r0 \
    build-base=0.5-r3 \
    curl=8.5.0-r0 \
    git=2.43.0-r0
WORKDIR /firefly
RUN chgrp -R 0 /firefly \
    && chmod -R g+rwX /firefly \
    && mkdir /.cache \
    && chgrp -R 0 /.cache \
    && chmod -R g+rwX /.cache
USER 1001
ADD --chown=1001:0 go.mod go.sum ./
RUN go mod download
ADD --chown=1001:0 . .
RUN make build

FROM --platform=$FABRIC_BUILDER_PLATFORM $FABRIC_BUILDER_TAG AS fabric-builder
RUN apk add gcompat=1.1.0-r4
WORKDIR /firefly/smart_contracts/fabric/firefly-go
RUN chgrp -R 0 /firefly \
    && chmod -R g+rwX /firefly \
    && mkdir /.cache \
    && chgrp -R 0 /.cache \
    && chmod -R g+rwX /.cache
USER 1001
ADD --chown=1001:0 smart_contracts/fabric/firefly-go .
RUN GO111MODULE=on go mod vendor
WORKDIR /tmp/fabric
RUN wget https://github.com/hyperledger/fabric/releases/download/v2.3.2/hyperledger-fabric-linux-amd64-2.3.2.tar.gz
RUN tar -zxf hyperledger-fabric-linux-amd64-2.3.2.tar.gz
RUN touch core.yaml
RUN ./bin/peer lifecycle chaincode package /firefly/smart_contracts/fabric/firefly-go/firefly_fabric.tar.gz --path /firefly/smart_contracts/fabric/firefly-go --lang golang --label firefly_1.0

FROM $SOLIDITY_BUILDER_TAG AS solidity-builder
WORKDIR /firefly/solidity_firefly
RUN chgrp -R 0 /firefly \ 
    && chmod -R g+rwX /firefly
ADD --chown=1001:0 smart_contracts/ethereum/solidity_firefly/ .
USER 1001
RUN mkdir -p build/contracts \
    && cd contracts \
    && solc --combined-json abi,bin,devdoc -o ../build/contracts Firefly.sol \
    && cd ../build/contracts \
    && mv combined.json Firefly.json

FROM alpine:3.19 AS SBOM
WORKDIR /
ADD . /SBOM
RUN apk add --no-cache curl 
RUN curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin v0.48.3
RUN trivy fs --format spdx-json --output /sbom.spdx.json /SBOM
RUN trivy sbom /sbom.spdx.json --severity UNKNOWN,HIGH,CRITICAL --exit-code 1

FROM $BASE_TAG
ARG UI_TAG
ARG UI_RELEASE
RUN apk add --update --no-cache \
    sqlite=3.44.2-r0 \
    postgresql16-client=16.2-r0 \
    curl=8.5.0-r0 \
    jq=1.7.1-r0
WORKDIR /firefly
RUN chgrp -R 0 /firefly \
    && chmod -R g+rwX /firefly \
    && mkdir /etc/firefly \
    && chgrp -R 0 /etc/firefly \
    && chmod -R g+rwX /etc/firefly
RUN curl -sL "https://github.com/golang-migrate/migrate/releases/download/$(curl -sL https://api.github.com/repos/golang-migrate/migrate/releases/latest | jq -r '.name')/migrate.linux-amd64.tar.gz" | tar xz \
    && chmod +x ./migrate \
    && mv ./migrate /usr/bin/migrate
COPY --from=firefly-builder --chown=1001:0 /firefly/firefly ./firefly
COPY --from=firefly-builder --chown=1001:0 /firefly/db ./db
COPY --from=solidity-builder --chown=1001:0 /firefly/solidity_firefly/build/contracts ./contracts
COPY --from=fabric-builder --chown=1001:0 /firefly/smart_contracts/fabric/firefly-go/firefly_fabric.tar.gz ./contracts/firefly_fabric.tar.gz
ENV UI_RELEASE https://github.com/hyperledger/firefly-ui/releases/download/$UI_TAG/$UI_RELEASE.tgz
RUN mkdir /firefly/frontend \
    && curl -sLo - $UI_RELEASE | tar -C /firefly/frontend -zxvf -
COPY --from=SBOM /sbom.spdx.json /sbom.spdx.json
RUN ln -s /firefly/firefly /usr/bin/firefly
USER 1001
ENTRYPOINT [ "firefly" ]
