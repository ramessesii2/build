# --- start build stage #1
FROM registry.access.redhat.com/ubi9/go-toolset:1.21@sha256:d128c3d36878251f039606f144ef2608746c3203708b722295e6c571df1d8613 AS builder

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# This are vendors dependencies. Copy this layer first to take advantage of cache hits.
COPY vendor/ vendor/

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY version/ version/

RUN go version

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -mod=vendor -ldflags="-s -w" -o openshift-builds-git-cloner ./cmd/git

# --- start build stage #2
# RPM dependencies can't be fetched hermetically with Cachi2
# Using UBI 9 which already includes necessary tar package
FROM registry.access.redhat.com/ubi9/ubi:9.4@sha256:763f30167f92ec2af02bf7f09e75529de66e98f05373b88bef3c631cdcc39ad8

LABEL \
    com.redhat.component="openshift-builds-git-cloner" \
    name="openshift-builds/git-cloner" \
    version="v1.1.0" \
    summary="Red Hat OpenShift Builds Git Cloner" \
    maintainer="openshift-builds@redhat.com" \
    description="Red Hat OpenShift Builds Git Cloner" \
    io.k8s.description="Red Hat OpenShift Builds Git Cloner" \
    io.k8s.display-name="Red Hat OpenShift Builds Git Cloner" \
    io.openshift.tags="builds,git-cloner"

RUN \
  echo 'nonroot:x:1000:1000:nonroot:/:/sbin/nologin' > /etc/passwd && \
  echo 'nonroot:x:1000:' > /etc/group && \
  mkdir /.docker && chown 1000:1000 /.docker && \
  mkdir /.ssh && chown 1000:1000 /.ssh

COPY --from=builder /opt/app-root/src/openshift-builds-git-cloner .
COPY LICENSE /licenses/

ENTRYPOINT ["./openshift-builds-git-cloner"]

