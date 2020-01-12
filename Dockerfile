FROM ubuntu:bionic
MAINTAINER Gary Hetzel <its@gary.cool>

RUN apt-get -qq update && apt-get install -qq -y libsass0 libtagc0 ca-certificates curl wget iputils-ping net-tools dnsutils ffmpeg jq python3-pip rsync nano ngrep htop
COPY contrib/rclone-1.50.2.deb /tmp/rclone.deb
RUN dpkg -i /tmp/rclone.deb
RUN apt-get clean all
RUN pip3 install --no-cache-dir -U youtube_dl
COPY bin/ /usr/bin/

WORKDIR /root
CMD ["/bin/bash"]
