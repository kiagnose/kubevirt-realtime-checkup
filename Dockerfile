FROM docker.io/library/golang:1.19.4-alpine3.17 as builder
WORKDIR /go/src/github.com/kiagnose/kubevirt-rt-checkup
COPY . .
RUN go build -v -o ./bin/kubevirt-rt-checkup ./cmd/

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.1.0-1656.1669627757

RUN microdnf install -y shadow-utils && \
    adduser --system --no-create-home -u 900 rt-checkup && \
    microdnf remove -y shadow-utils && \
    microdnf clean all

COPY --from=builder /go/src/github.com/kiagnose/kubevirt-rt-checkup/bin/kubevirt-rt-checkup /usr/local/bin

USER 900

ENTRYPOINT ["kubevirt-rt-checkup"]
