#!/bin/bash

# TodoIng Docker Compose 快速启动脚本
# 使用方法: ./deploy.sh [方案名] [操作] [选项]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 显示帮助信息
show_help() {
    cat << EOF
TodoIng Docker Compose 工具

用法:
  $0 [方案] [操作] [选项]

方案:
  golang   单体 Go 后端 + 前端
  dev      开发环境 (含数据库/工具)
  prod     生产环境 (单实例，可扩展)

操作:
  up | down | restart | logs | ps | build | pull | exec

选项:
  --profile <name>   启用额外 profile (grpc, monitoring, replica, testing)
  --build            强制重新构建
  --remove           down 时删除卷 (-v)
  --no-detach        前台运行 (默认后台)
  --follow           logs 时跟随 (-f)

示例:
  $0 golang up --build
  $0 dev up --profile golang
  $0 prod up --profile replica
  $0 golang logs backend-golang --follow
EOF
}

# 获取 docker-compose 文件路径
get_compose_file() {
  case $1 in
    golang) echo "docker-compose.golang.yml" ;;
    dev)    echo "docker-compose.dev-full.yml" ;;
    prod)   echo "docker-compose.prod.yml" ;;
    *) print_message $RED "未知方案: $1"; show_help; exit 1;;
  esac
}

# 检查环境变量文件
check_env_file() {
  if [ ! -f .env ]; then
    if [ -f .env.example ]; then
      print_message $YELLOW ".env 不存在，基于 .env.example 生成"
      cp .env.example .env
    else
      print_message $RED "缺少 .env 与 .env.example"
      exit 1
    fi
  fi
}

# 检查 Docker 和 Docker Compose
check_docker() {
  command -v docker >/dev/null 2>&1 || { print_message $RED "未安装 docker"; exit 1; }
  docker info >/dev/null 2>&1 || { print_message $RED "docker daemon 未就绪"; exit 1; }
  if ! command -v docker-compose >/dev/null 2>&1; then
    if docker compose version >/dev/null 2>&1; then
      docker_compose_wrapper(){ docker compose "$@"; }; export -f docker_compose_wrapper; alias docker-compose=docker_compose_wrapper
    else
      print_message $RED "缺少 docker compose"; exit 1
    fi
  fi
}

# 执行 docker-compose 命令
execute_compose() {
  local file=$1 op=$2; shift 2; local args=("$@")
  print_message $BLUE "执行: docker-compose -f $file $op ${args[*]}"
  docker-compose -f "$file" "$op" "${args[@]}"
}

# 主函数
main() {
  [ $# -lt 2 ] && show_help && exit 1
  local scheme=$1 op=$2; shift 2
  local profile="" detach_flag="-d" build_flag="" remove_flag="" follow_flag="" compose_extra=()
  while [[ $# -gt 0 ]]; do
    case $1 in
      --profile) profile=$2; shift 2 ;;
      --build) build_flag="--build"; shift ;;
      --remove) remove_flag="-v"; shift ;;
      --no-detach) detach_flag=""; shift ;;
      --follow) follow_flag="-f"; shift ;;
      -h|--help) show_help; exit 0 ;;
      *) compose_extra+=("$1"); shift ;;
    esac
  done
  check_docker; check_env_file
  local file; file=$(get_compose_file "$scheme")
  local cmd_args=()
  [ -n "$profile" ] && cmd_args+=(--profile "$profile")
  case $op in
    up)
      cmd_args+=("$detach_flag"); [ -n "$build_flag" ] && cmd_args+=("$build_flag"); cmd_args+=("${compose_extra[@]}")
      print_message $GREEN "启动 $scheme ..."; execute_compose "$file" up "${cmd_args[@]}"; print_message $GREEN "完成"
      case $scheme in
        golang) print_message $BLUE "API: http://localhost:5004/api  Swagger: http://localhost:5004/swagger/" ;;
        dev)    print_message $BLUE "前端: http://localhost:3000  MongoExpress: http://localhost:8081" ;;
        prod)   print_message $BLUE "前端: http://localhost/  Grafana: http://localhost:3001" ;;
      esac ;;
    down) cmd_args+=("$remove_flag" "${compose_extra[@]}"); execute_compose "$file" down "${cmd_args[@]}" ;;
    restart) execute_compose "$file" restart "${compose_extra[@]}" ;;
    logs) cmd_args+=("$follow_flag" "${compose_extra[@]}"); execute_compose "$file" logs "${cmd_args[@]}" ;;
    ps) execute_compose "$file" ps "${compose_extra[@]}" ;;
    build) cmd_args+=("${compose_extra[@]}"); execute_compose "$file" build "${cmd_args[@]}" ;;
    pull) cmd_args+=("${compose_extra[@]}"); execute_compose "$file" pull "${cmd_args[@]}" ;;
    exec) [ ${#compose_extra[@]} -lt 2 ] && { print_message $RED "用法: $0 $scheme exec <服务> <命令>"; exit 1; }; execute_compose "$file" exec "${compose_extra[@]}" ;;
    *) print_message $RED "未知操作: $op"; show_help; exit 1 ;;
  esac
}

# 捕获 Ctrl+C
trap 'print_message $YELLOW "\n操作已取消"' INT

# 运行主函数
main "$@"
