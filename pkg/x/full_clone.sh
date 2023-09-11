#!/bin/bash

PWD_PATH="$(dirname $(readlink -f $0))"
# 基础 Git URL
BASE_GIT_URL="https://github.com/golang/"
# 基础包 URL
BASE_PACKAGE_URL="golang.org/x/"
# 替换 URL
BASE_REPLACE_URL="uw/pkg/x/"
# 包列表
PACKAGE_LIST="sys,sync,text,net,tools,mod,crypto,image,term"
# 自定义替换列表用 ; 分隔
REPLACE_LIST="github.com/yuin/goldmark:uw/pkg/goldmark"

cd $PWD_PATH

_CLONE_PACKAGE() {
    PACKAGE_NAME=$1
    if [ -z $PACKAGE_NAME ]; then
        echo "包名不能为空"
        exit 1
    fi

    cd $PWD_PATH
    PACKAGE_PATH=${PWD_PATH}/${PACKAGE_NAME}
    # 如果包存在则删除
    if [ -d ${PACKAGE_PATH} ]; then
        echo "删除 ${PACKAGE_PATH}"
        rm -rf ${PACKAGE_PATH}
    fi

    GIT_URL=${BASE_GIT_URL}${PACKAGE_NAME}
    PACKAGE_URL=${BASE_PACKAGE_URL}${PACKAGE_NAME}
    REPLACE_URL=${BASE_REPLACE_URL}${PACKAGE_NAME}
    # 克隆包
    echo "正在克隆 ${GIT_URL} 到 ${PACKAGE_NAME}"
    git clone ${GIT_URL} ${PACKAGE_NAME}

    echo "进入目录: ${PACKAGE_PATH}"
    cd ${PACKAGE_PATH}

    # 替换目录下面的所有文件的 URL
    echo "开始替换 URL"
    # 使用逗号分隔的包列表并遍历
    for SUB_PACKAGE_NAME in $(echo $PACKAGE_LIST | tr "," "\n"); do
        SUB_PACKAGE_URL=${BASE_PACKAGE_URL}${SUB_PACKAGE_NAME}
        SUB_REPLACE_URL=${BASE_REPLACE_URL}${SUB_PACKAGE_NAME}
        echo "正在替换 ${PACKAGE_PATH} 的 ${SUB_PACKAGE_URL} 为 ${SUB_REPLACE_URL}"
        find . -type f -exec sed -i "s#${SUB_PACKAGE_URL}#${SUB_REPLACE_URL}#g" {} \;
    done
    echo "URL 替换完成"

    # 替换自定义的 URL
    echo "开始替换自定义 URL"
    # 使用逗号分隔的替换列表并遍历
    for REPLACE_ITEM in $(echo $REPLACE_LIST | tr "," "\n"); do
        REPLACE_ITEM_ARRAY=(${REPLACE_ITEM//:/ })
        REPLACE_ITEM_URL=${REPLACE_ITEM_ARRAY[0]}
        REPLACE_ITEM_REPLACE_URL=${REPLACE_ITEM_ARRAY[1]}
        echo "正在替换 ${PACKAGE_PATH} 的 ${REPLACE_ITEM_URL} 为 ${REPLACE_ITEM_REPLACE_URL}"
        find . -type f -exec sed -i "s#${REPLACE_ITEM_URL}#${REPLACE_ITEM_REPLACE_URL}#g" {} \;
    done

    # 删除 go.mod 和 go.sum
    rm go.mod go.sum

    # 删除 .git 文件夹
    rm -rf .git
}

_DELETE_ALL_PACKAGES() {
    # 遍历删除当前目录下的所有文件夹
    for PACKAGE_NAME in $(ls); do
        if [ -d ${PACKAGE_NAME} ]; then
            echo "删除 ${PACKAGE_NAME}"
            rm -rf ${PACKAGE_NAME}
        fi
    done
}

# 判断有没有参数
if [ $# -eq 0 ]; then
    echo "重新拉取所有包"
    _DELETE_ALL_PACKAGES

    # 使用逗号分隔的包列表并遍历
    for PACKAGE_NAME in $(echo $PACKAGE_LIST | tr "," "\n"); do
        _CLONE_PACKAGE $PACKAGE_NAME
    done
else
    _CLONE_PACKAGE $1
fi

# 删除: pkg/x/tools/gopls
if [ -d "tools/gopls" ]; then
    echo "删除 tools/gopls"
    rm -r tools/gopls
fi
