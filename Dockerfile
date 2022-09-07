# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY ./dist/manager .
COPY ./build/configs/ ./build/configs/

USER 65532:65532

ENTRYPOINT ["/manager"]
