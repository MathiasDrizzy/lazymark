# lazymark MCP server, built from this repository's code (not from a released tag), so the image always matches the source.
# Used by Glama (and handy for anyone who wants the MCP server in a container):
#   docker build -t lazymark-mcp .
#   docker run -i --rm lazymark-mcp            # speaks MCP over stdin/stdout, on the sample notes below
#   docker run -i --rm -v "$PWD/notes:/notes" lazymark-mcp   # on your own notes
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /lazymark ./cmd/lazymark

FROM alpine:3
COPY --from=build /lazymark /usr/local/bin/lazymark
# a non-root user, and a notes folder with one sample note that has a task
RUN adduser -D -u 10001 lazymark \
 && mkdir -p /notes \
 && printf '# Welcome\n\n- [ ] Try lazymark #kb/todo\n' > /notes/welcome.md \
 && chown -R lazymark /notes
USER lazymark
ENV HOME=/home/lazymark
ENTRYPOINT ["lazymark", "mcp", "--dir", "/notes"]
