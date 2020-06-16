FROM alpine:3.12
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apk update && apk add bash libsass taglib taglib-dev ca-certificates curl wget bind-tools apache2-utils pwgen ffmpeg jq python3 py3-pip rsync nano ngrep htop rclone
RUN pip3 install --no-cache-dir -U youtube_dl
COPY bin/ /usr/bin/

WORKDIR /root
CMD ["/bin/bash"]
