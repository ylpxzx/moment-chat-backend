@echo off
echo =========================================
echo 阅后即焚聊天 - 测试套件
echo =========================================

REM 检查Go是否安装
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo 错误: Go未安装
    exit /b 1
)

REM 安装测试依赖
echo 安装测试依赖...
go mod download

REM 运行单元测试
echo 运行单元测试...
go test ./test/unit -v -short

REM 运行集成测试
echo 运行集成测试...
echo 注意：集成测试需要Redis服务器

REM 运行所有测试
echo 运行所有测试...
go test ./... -v

REM 代码覆盖率
echo 生成代码覆盖率报告...
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html

echo 代码覆盖率报告已生成：coverage.html
echo =========================================
echo 所有测试完成！
echo =========================================
pause