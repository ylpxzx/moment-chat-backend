# 进入 backend 目录

cd backend

# 方法 1: 使用测试脚本

chmod +x run_tests.sh
./run_tests.sh

# 方法 2: 直接使用 go test

go test ./test/unit -v # 运行单元测试
go test ./test/integration # 运行集成测试
go test ./... -cover # 运行所有测试并计算覆盖率
