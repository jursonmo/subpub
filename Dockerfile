FROM alpine:latest
ENV VERSION 2.0

# 在容器根目录 创建一个 apps/subpub 目录, 这样执行程序的subpupserver.log日志目录就放在这下面
WORKDIR /apps/subpub

# 挂载容器目录
#VOLUME ["/apps/conf"]

# 拷贝当前目录下可以执行文件
COPY bin/server /apps/subpub/bin/server

# 拷贝配置文件到容器中
COPY config/config.yaml /apps/subpub/config/config.yaml
COPY cert/* /apps/subpub/cert/

# 设置时区为上海
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' >/etc/timezone

# 设置编码
ENV LANG C.UTF-8

# 暴露端口, 0.0.0.0:8000->8000/tcp
EXPOSE 8000

#CMD, 如果用户启动容器时候指定了运行的命令，则会覆盖掉 CMD 指定的命令
#CMD ["/apps/subpub/bin/server", "-c" ,"/apps/subpub/config/config.yaml"]

# 运行程序的命令
ENTRYPOINT ["/apps/subpub/bin/server", "-c" ,"/apps/subpub/config/config.yaml"]

# docker inspect 镜像名称 ，获取镜像的元数据, docker history：查看镜像的构建历史，可以看到各个操作的大小

#进入已经运行的容器查看相关文件是否正常copy: docker exec -it ${container_name} /bin/sh