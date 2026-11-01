# lazymark MCP server, built from this repository's code (not from a released tag), so the image always matches the source.
# Used by Glama (and handy for anyone who wants the MCP server in a container):
#   docker build -t lazymark-mcp .
#   docker run -i --rm lazymark-mcp            # speaks MCP over stdin/stdout, on the sample notes below
#   docker run -i --rm -v "$PWD/notes:/notes" lazymark-mcp   # on your own notes
# Base images are pinned by digest (the multi-arch index of the tag shown, checked 2026-10-11) so a build is reproducible and a moved tag cannot change what we ship; refresh the digests
# (and rebuild) to pick up base-image security updates.
FROM golang:1.27-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /lazymark ./cmd/lazymark

FROM alpine:3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
COPY --from=build /lazymark /usr/local/bin/lazymark
# a non-root user, and a notes folder with one sample note that has a task
RUN adduser -D -u 10001 lazymark \
 && mkdir -p /notes \
 && printf '# Welcome\n\n- [ ] Try lazymark #kb/todo\n' > /notes/welcome.md \
 && chown -R lazymark /notes
USER lazymark
ENV HOME=/home/lazymark
ENTRYPOINT ["lazymark", "mcp", "--dir", "/notes"]
