FROM harbor.oneitfarm.com/zhirenyun/baseimage:bionic-1.0.0

WORKDIR /root

COPY bin/msp_backend .
COPY config.yaml .
COPY db ./db

RUN set -xe \
  ## disable cron sshd
  && rm -r /etc/service/cron /etc/service/sshd

CMD ["./msp_backend"]