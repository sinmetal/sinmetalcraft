FROM k2wanko/minecraft:base
MAINTAINER sinmetal <metal.tie@gmail.com>

ONBUILD RUN download 1.12.2
ONBUILD COPY . /data