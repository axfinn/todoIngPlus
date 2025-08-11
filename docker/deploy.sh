#!/bin/bash

# TodoIng Docker Compose 快速启动脚本 (加强版)
# 用法: ./deploy.sh <scheme> <op> [选项]
# scheme: golang | dev | prod | local | micro(实验)
# op: up | down | restart | logs | ps | status | build | pull | exec | config | help

set -eo pipefail

# 颜色 (仅在 TTY 输出时启用)
if [ -t 1 ]; then
  RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
else
  RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

print_message(){ echo -e "${1}${2}${NC}"; }

show_help(){ cat <<'EOF'
TodoIng Docker Compose 工具

用法:
  ./deploy.sh <方案> <操作> [选项]

方案:
  golang    单体 Go 后端 + 前端 (含监控 profile 可选)
  dev       全功能开发环境 (支持 golang/testing profiles)
  prod      生产部署 (可选 replica / monitoring / logging / backup profiles)
  local     极简本地运行 (Mongo + Backend + Frontend)
  micro     微服务草案 (实验/不完整, 仅参考)

操作:
  up        启动 (默认后台 -d)
  down      停止 (支持 --remove 删除卷)
  restart   重启
  logs      查看日志 (支持 --follow)
  ps|status 查看容器状态
  build     仅构建镜像
  pull      拉取镜像
  exec      进入/执行: ./deploy.sh <scheme> exec <service> <cmd>
  config    输出解析后的 compose 配置
  help      显示帮助

常用选项:
  --profile <name>   启用 profile (例如 monitoring / grpc / replica / golang / testing)
  --build            up 时强制构建
  --remove           down 时附带 -v 删除卷
  --no-detach        up 前台模式
  --follow           logs 跟随
  --port <port>      覆盖 HOST_HTTP_PORT (前端或网关对外端口)

示例:
  ./deploy.sh golang up --build --profile monitoring
  ./deploy.sh dev up --profile golang
  ./deploy.sh prod up --profile replica --port 2001
  ./deploy.sh local up --no-detach
  ./deploy.sh golang logs backend-golang --follow
EOF
}

get_compose_file(){
  case "$1" in
    golang) echo "docker-compose.golang.yml" ;;
    dev)    echo "docker-compose.dev-full.yml" ;;
    prod)   echo "docker-compose.prod.yml" ;;
    local)  echo "docker-compose.local.yml" ;;
    micro)  echo "docker-compose.microservices.yml" ;;
    help|-h|--help) show_help; exit 0 ;;
    *) print_message "$RED" "[ERROR] 未知方案: $1"; show_help; exit 1 ;;
  esac
}

check_env_file(){
  # local 与 micro 可不强制 .env (micro 为实验)
  local scheme=$1
  if [ "$scheme" = "micro" ]; then
    print_message "$YELLOW" "[WARN] micro 方案为实验/草案，缺少 services/* 源码将无法成功。"
  fi
  if [ ! -f .env ]; then
    if [ -f .env.example ]; then
      print_message "$YELLOW" ".env 缺失，基于 .env.example 生成"
      cp .env.example .env
    else
      print_message "$RED" "缺少 .env 与 .env.example"; exit 1
    fi
  fi
}

resolve_compose(){
  if command -v docker-compose >/dev/null 2>&1; then
    echo docker-compose
  elif docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  else
    print_message "$RED" "未检测到 docker compose (需安装 docker 或 compose 插件)"; exit 1
  fi
}

execute_compose(){
  local file=$1 op=$2; shift 2
  local cmd_args=("$@")
  local joined="${cmd_args[*]:-}"
  print_message "$BLUE" "执行: HOST_HTTP_PORT=${HOST_HTTP_PORT:-80} $DC_CMD -f $file $op $joined"
  HOST_HTTP_PORT=${HOST_HTTP_PORT:-80} $DC_CMD -f "$file" "$op" ${cmd_args:+"${cmd_args[@]}"}
}

validate_port(){
  local p=$1
  [[ $p =~ ^[0-9]+$ ]] || { print_message "$RED" "无效端口: $p"; exit 1; }
  (( p>=1 && p<=65535 )) || { print_message "$RED" "端口超出范围: $p"; exit 1; }
}

main(){
  [ $# -lt 2 ] && show_help && exit 1
  local scheme=$1 op=$2; shift 2
  local profile="" detach_flag="-d" build_flag="" remove_flag="" follow_flag="" host_port="" compose_extra=()
  while [ $# -gt 0 ]; do
    case $1 in
      --profile) profile=$2; shift 2;;
      --build) build_flag="--build"; shift;;
      --remove) remove_flag="-v"; shift;;
      --no-detach) detach_flag=""; shift;;
      --follow) follow_flag="-f"; shift;;
      --port) host_port=$2; shift 2;;
      -h|--help) show_help; exit 0;;
      *) compose_extra+=("$1"); shift;;
    esac
  done
  if [ -n "$host_port" ]; then validate_port "$host_port"; export HOST_HTTP_PORT=$host_port; fi
  check_env_file "$scheme"
  local file; file=$(get_compose_file "$scheme")
  local cmd_args=(); [ -n "$profile" ] && cmd_args+=(--profile "$profile")
  case $op in
    up)
      [ -n "$detach_flag" ] && cmd_args+=("$detach_flag")
      [ -n "$build_flag" ] && cmd_args+=("$build_flag")
      [ ${#compose_extra[@]} -gt 0 ] && cmd_args+=("${compose_extra[@]}")
      print_message "$GREEN" "启动 $scheme (HTTP端口: ${HOST_HTTP_PORT:-80}) ..."
      execute_compose "$file" up "${cmd_args[@]}"
      case $scheme in
        golang)
          print_message "$BLUE" "前端: http://localhost:${HOST_HTTP_PORT:-80}  API: http://localhost:5004/api  Swagger: /swagger/";;
        dev)
          print_message "$BLUE" "开发前端: http://localhost:3000  API: http://localhost:5004/api  (或通过 nginx: http://localhost:${HOST_HTTP_PORT:-80})";;
        local)
          print_message "$BLUE" "本地: 前端 http://localhost  API: http://localhost:5004/api";;
        prod)
          print_message "$BLUE" "生产: http://localhost:${HOST_HTTP_PORT:-80}  建议配置反向代理与 TLS";;
        micro)
          print_message "$YELLOW" "微服务草案已启动(可能失败) - 仅供参考, 需补齐 services/*";;
      esac
      print_message "$GREEN" "完成";;
    down)
      [ -n "$remove_flag" ] && cmd_args+=("$remove_flag")
      [ ${#compose_extra[@]} -gt 0 ] && cmd_args+=("${compose_extra[@]}")
      execute_compose "$file" down "${cmd_args[@]}" ;;
    restart)
      execute_compose "$file" restart "${compose_extra[@]:-}" ;;
    logs)
      [ -n "$follow_flag" ] && cmd_args+=("$follow_flag")
      [ ${#compose_extra[@]} -gt 0 ] && cmd_args+=("${compose_extra[@]}")
      execute_compose "$file" logs "${cmd_args[@]}" ;;
    ps|status)
      execute_compose "$file" ps "${compose_extra[@]:-}" ;;
    build)
      execute_compose "$file" build "${compose_extra[@]:-}" ;;
    pull)
      execute_compose "$file" pull "${compose_extra[@]:-}" ;;
    exec)
      [ ${#compose_extra[@]} -lt 2 ] && { print_message "$RED" "用法: $0 $scheme exec <服务> <命令>"; exit 1; }
      execute_compose "$file" exec "${compose_extra[@]}" ;;
    config) execute_compose "$file" config ;;
    help) show_help ;;
    *) print_message "$RED" "未知操作: $op"; show_help; exit 1 ;;
  esac
}

trap 'print_message "$YELLOW" "\n操作已取消"' INT
DC_CMD=$(resolve_compose)
main "$@"
