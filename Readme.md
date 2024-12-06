# 创建路由

~~~
# 需要先安装gr
D:\ws_web\psp\backend\src\psp> gr

~~~



# 创建swagger.json

~~~shell
D:\ws_web\psp\backend\src\psp> swag init -d ./ -g ./PspApplication.go --ot json  -o ./bin/web/ui/swagger/json
~~~



# idea中启动

 ![image-20241028201233260](http://pic7.wtding.com/PicGo/MarkDown/202410282012306.png)



# docker打包启动

~~~dockerfile
 # Compile stage
FROM golang:1.19.7 AS build-env

ENV GO111MODULE=on \
    CGO_ENABLE=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY="https://goproxy.cn,direct"


#ADD . /go_ws/
ADD ./go.work /go_ws/
# 目录
ADD ./src/core /go_ws/src/core
ADD ./src/psp /go_ws/src/psp

WORKDIR /go_ws/src/psp

RUN go build -o psp_server PspApplication.go LoadRouter.go

# Final stage
FROM debian:buster

EXPOSE 38000

WORKDIR /go_run
COPY --from=build-env /go_ws/src/psp/psp_server /go_run/
COPY --from=build-env /go_ws/src/psp/bin/conf /go_run/conf
COPY --from=build-env /go_ws/src/psp/bin/web /go_run/web


CMD ["/go_run/psp_server","--env=docker"]
~~~



>  注意设置参数：-v /opt/psp/files:/go_run/files -v /opt/psp/cache:/go_run/cache -p 38000:38000 -e TZ=Asia/Shanghai

 ![image-20241028204524840](http://pic7.wtding.com/PicGo/MarkDown/202410282045874.png)
