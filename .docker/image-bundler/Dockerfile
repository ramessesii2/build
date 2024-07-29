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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -mod=vendor -ldflags="-s -w" -o openshift-builds-image-bundler ./cmd/bundle

# --- start build stage #2
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.4@sha256:104cf11d890aeb7dd5728b7d7732e175a0e4018f1bb00d2faebcc8f6bf29bd52

LABEL \
    com.redhat.component="openshift-builds-image-bundler" \
    name="openshift-builds/image-bundler" \
    version="v1.1.0" \
    summary="Red Hat OpenShift Builds Image Bundler" \
    maintainer="openshift-builds@redhat.com" \
    description="Red Hat OpenShift Builds Image Bundler" \
    io.k8s.description="Red Hat OpenShift Builds Image Bundler" \
    io.k8s.display-name="Red Hat OpenShift Builds Image Bundler" \
    io.openshift.tags="builds,image-bundler"

RUN \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  echo 'nonroot:x:1000:1000:nonroot:/:/sbin/nologin' > /etc/passwd && \
  echo 'nonroot:x:1000:' > /etc/group && \
  mkdir /.docker && \
  chown 1000:1000 /.docker

COPY --from=builder /opt/app-root/src/openshift-builds-image-bundler .
COPY LICENSE /licenses/

ENTRYPOINT ["./openshift-builds-image-bundler"]
