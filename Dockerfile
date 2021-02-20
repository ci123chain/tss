FROM harbor.oneitfarm.com/zhirenyun/baseimage:bionic-1.0.0

WORKDIR /root

COPY bin/badkend .
COPY config.yaml .
COPY db ./db

RUN set -xe \
  ## disable sshd
  && rm -r /etc/service/sshd

# 镜像启动服务自动被拉起配置
COPY run /etc/service/backend/run
RUN chmod +x /etc/service/backend/run

# dockerfile 中不允许定义 CMD。镜像启动需要执行基础定义逻辑