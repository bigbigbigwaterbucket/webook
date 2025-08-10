# 基础镜像
FROM ubuntu:20.04
# 打包编译文件放到工作目录
COPY webook /app/webook
# 工作目录
WORKDIR /app

# 执行命令CMD  ENTRYPOINT：进来就执行
ENTRYPOINT ["/app/webook"]