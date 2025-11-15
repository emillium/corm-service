SERVICE_NAME="corm-service"
GIT_VERSION="v0.0.1-$(git rev-parse --short HEAD)"
TARGET_OS="linux"
TARGET_ARCH="amd64"
GO_VER="1.23"

docker build \
  --platform ${TARGET_OS}/${TARGET_ARCH} \
  --build-arg GO_VERSION=${GO_VER} \
  --build-arg VERSION=${GIT_VERSION} \
  --build-arg TARGETOS=${TARGET_OS} \
  --build-arg TARGETARCH=${TARGET_ARCH} \
  --build-arg SERVICE_NAME=${SERVICE_NAME} \
  -t my-registry/${SERVICE_NAME}:${GIT_VERSION} \
  .
