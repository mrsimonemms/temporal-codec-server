# Copyright 2025 Simon Emms <simon@simonemms.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang AS dev
ARG APP
ARG GIT_COMMIT
ARG GIT_REPO
ARG VERSION
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
USER 1000
WORKDIR /go/root
COPY --chown=1000:1000 . .
WORKDIR /go/root/apps/$APP
COPY --from=cosmtrek/air /go/bin/air /go/bin/air
CMD [ "air", "-build.stop_on_error", "true", "-build.send_interrupt", "true", "-build.rerun", "true" ]

FROM golang AS builder
ARG APP
ARG GIT_COMMIT
ARG GIT_REPO
ARG VERSION
WORKDIR /go/root
COPY . .
WORKDIR /go/root/apps/$APP
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build \
  -ldflags \
  "-w -s -X $GIT_REPO/cmd.Version=$VERSION -X $GIT_REPO/cmd.GitCommit=$GIT_COMMIT" \
  -o /go/app
ENTRYPOINT [ "/go/app" ]

FROM scratch
ARG GIT_COMMIT
ARG VERSION
ENV GIT_COMMIT="${GIT_COMMIT}"
ENV VERSION="${VERSION}"
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/app /app
ENTRYPOINT [ "/app/app" ]
