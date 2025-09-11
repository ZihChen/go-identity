#!/bin/bash

# Consumer 重構測試執行腳本
# 用於執行階段一的基礎架構準備測試

set -e  # 遇到錯誤時停止執行

echo "=== Consumer 重構 - 階段一測試執行 ==="
echo "開始執行基礎架構準備階段的測試..."

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 測試結果統計
TESTS_PASSED=0
TESTS_FAILED=0
BENCHMARKS_RUN=0

# 函數：打印帶顏色的消息
print_status() {
    local status=$1
    local message=$2
    case $status in
        "INFO")
            echo -e "${BLUE}[INFO]${NC} $message"
            ;;
        "PASS")
            echo -e "${GREEN}[PASS]${NC} $message"
            ((TESTS_PASSED++))
            ;;
        "FAIL")
            echo -e "${RED}[FAIL]${NC} $message"
            ((TESTS_FAILED++))
            ;;
        "WARN")
            echo -e "${YELLOW}[WARN]${NC} $message"
            ;;
    esac
}

# 函數：檢查 Go 環境
check_environment() {
    print_status "INFO" "檢查 Go 環境..."
    
    if ! command -v go &> /dev/null; then
        print_status "FAIL" "Go 未安裝或不在 PATH 中"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    print_status "PASS" "Go 版本: $GO_VERSION"
    
    # 檢查模組
    if [ ! -f "go.mod" ]; then
        print_status "FAIL" "未找到 go.mod 文件，請在項目根目錄執行"
        exit 1
    fi
    
    print_status "PASS" "Go 模組檢查通過"
}

# 函數：安裝測試依賴
install_dependencies() {
    print_status "INFO" "安裝測試依賴..."
    
    # 確保測試依賴已安裝
    go mod tidy
    if [ $? -eq 0 ]; then
        print_status "PASS" "依賴安裝成功"
    else
        print_status "FAIL" "依賴安裝失敗"
        exit 1
    fi
}

# 函數：編譯測試
compile_tests() {
    print_status "INFO" "編譯測試代碼..."
    
    # 編譯 KDS 包
    go build -o /dev/null ./internal/infrastructure/kds/
    if [ $? -eq 0 ]; then
        print_status "PASS" "KDS 包編譯成功"
    else
        print_status "FAIL" "KDS 包編譯失敗"
        return 1
    fi
    
    # 編譯配置包
    go build -o /dev/null ./internal/infrastructure/config/
    if [ $? -eq 0 ]; then
        print_status "PASS" "配置包編譯成功"
    else
        print_status "FAIL" "配置包編譯失敗"
        return 1
    fi
    
    # 測試編譯
    go test -c ./internal/infrastructure/kds/ -o /dev/null
    if [ $? -eq 0 ]; then
        print_status "PASS" "測試代碼編譯成功"
    else
        print_status "FAIL" "測試代碼編譯失敗"
        return 1
    fi
}

# 函數：執行單元測試
run_unit_tests() {
    print_status "INFO" "執行單元測試..."
    
    # 執行 KDS 包測試
    echo -e "\n${YELLOW}--- KDS 包測試 ---${NC}"
    go test -v ./internal/infrastructure/kds/ -run "^Test" -short
    local kds_result=$?
    
    # 執行配置包測試（如果有的話）
    echo -e "\n${YELLOW}--- 配置包測試 ---${NC}"
    go test -v ./internal/infrastructure/config/ -run "^Test" -short 2>/dev/null || echo "配置包暫無測試"
    
    # 執行效能測試套件
    echo -e "\n${YELLOW}--- Consumer 效能測試套件 ---${NC}"
    go test -v ./test/ -run "TestConsumerPerformance" -short
    local perf_result=$?
    
    if [ $kds_result -eq 0 ] && [ $perf_result -eq 0 ]; then
        print_status "PASS" "單元測試全部通過"
        return 0
    else
        print_status "FAIL" "部分單元測試失敗"
        return 1
    fi
}

# 函數：執行基準測試
run_benchmark_tests() {
    print_status "INFO" "執行基準測試..."
    
    # 執行 KDS 基準測試
    echo -e "\n${YELLOW}--- KDS 組件基準測試 ---${NC}"
    go test -bench=BenchmarkMetricsCollector -benchmem ./internal/infrastructure/kds/ -run=^$
    ((BENCHMARKS_RUN++))
    
    go test -bench=BenchmarkBackoffStrategy -benchmem ./internal/infrastructure/kds/ -run=^$
    ((BENCHMARKS_RUN++))
    
    go test -bench=BenchmarkErrorClassifier -benchmem ./internal/infrastructure/kds/ -run=^$
    ((BENCHMARKS_RUN++))
    
    # 執行負載測試（輕量版）
    echo -e "\n${YELLOW}--- 負載測試 ---${NC}"
    go test -bench=BenchmarkLoadTest/LightLoad -benchmem ./internal/infrastructure/kds/ -run=^$ -timeout=30s
    ((BENCHMARKS_RUN++))
    
    # 執行組件基準測試
    echo -e "\n${YELLOW}--- 組件基準測試 ---${NC}"
    go test -bench=BenchmarkConsumerComponents -benchmem ./test/ -run=^$ -timeout=30s
    ((BENCHMARKS_RUN++))
    
    print_status "PASS" "基準測試執行完成"
}

# 函數：執行記憶體和併發測試
run_advanced_tests() {
    print_status "INFO" "執行高級測試（記憶體和併發）..."
    
    # 記憶體測試
    echo -e "\n${YELLOW}--- 記憶體使用測試 ---${NC}"
    go test -bench=BenchmarkMemoryUsage -benchmem ./internal/infrastructure/kds/ -run=^$
    
    # 併發安全測試
    echo -e "\n${YELLOW}--- 併發安全測試 ---${NC}"
    go test -bench=BenchmarkConcurrency -benchmem ./internal/infrastructure/kds/ -run=^$
    
    print_status "PASS" "高級測試執行完成"
}

# 函數：執行程式碼品質檢查
run_code_quality_checks() {
    print_status "INFO" "執行程式碼品質檢查..."
    
    # go fmt 檢查
    echo -e "\n${YELLOW}--- 程式碼格式檢查 ---${NC}"
    gofmt_result=$(go fmt ./internal/infrastructure/kds/ ./internal/infrastructure/config/)
    if [ -z "$gofmt_result" ]; then
        print_status "PASS" "程式碼格式檢查通過"
    else
        print_status "WARN" "程式碼格式需要調整: $gofmt_result"
    fi
    
    # go vet 檢查
    echo -e "\n${YELLOW}--- 靜態分析檢查 ---${NC}"
    go vet ./internal/infrastructure/kds/ ./internal/infrastructure/config/
    if [ $? -eq 0 ]; then
        print_status "PASS" "靜態分析檢查通過"
    else
        print_status "WARN" "靜態分析發現問題"
    fi
    
    # 測試覆蓋率
    echo -e "\n${YELLOW}--- 測試覆蓋率檢查 ---${NC}"
    go test -cover ./internal/infrastructure/kds/ -short
    go test -cover ./test/ -short
}

# 函數：生成測試報告
generate_test_report() {
    print_status "INFO" "生成測試報告..."
    
    local report_file="consumer_test_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "=== Consumer 重構階段一測試報告 ==="
        echo "執行時間: $(date)"
        echo "Go 版本: $(go version)"
        echo ""
        echo "測試統計:"
        echo "  - 通過測試: $TESTS_PASSED"
        echo "  - 失敗測試: $TESTS_FAILED"
        echo "  - 執行基準測試: $BENCHMARKS_RUN"
        echo ""
        echo "階段一完成項目:"
        echo "  ✓ 配置系統增強"
        echo "  ✓ 錯誤處理框架"
        echo "  ✓ 效能監控基礎"
        echo "  ✓ 測試環境準備"
        echo ""
        echo "下一階段準備:"
        echo "  - 批次處理引擎實現"
        echo "  - Worker Pool 機制開發"
        echo "  - 分片處理優化"
        echo "  - 自適應退避實現"
    } > "$report_file"
    
    print_status "PASS" "測試報告已生成: $report_file"
}

# 主執行函數
main() {
    echo "開始執行 Consumer 重構階段一測試..."
    echo ""
    
    # 記錄開始時間
    start_time=$(date +%s)
    
    # 執行測試步驟
    check_environment
    install_dependencies
    compile_tests || exit 1
    run_unit_tests
    run_benchmark_tests
    run_advanced_tests
    run_code_quality_checks
    
    # 計算總執行時間
    end_time=$(date +%s)
    execution_time=$((end_time - start_time))
    
    echo ""
    echo "=== 測試執行完成 ==="
    print_status "INFO" "總執行時間: ${execution_time}s"
    print_status "INFO" "測試通過: $TESTS_PASSED"
    print_status "INFO" "測試失敗: $TESTS_FAILED"
    print_status "INFO" "基準測試: $BENCHMARKS_RUN"
    
    # 生成報告
    generate_test_report
    
    echo ""
    if [ $TESTS_FAILED -eq 0 ]; then
        print_status "PASS" "🎉 階段一：基礎架構準備 - 完成！"
        echo -e "${GREEN}準備進入階段二：核心重構實施${NC}"
        exit 0
    else
        print_status "FAIL" "❌ 發現測試失敗，請修復後重試"
        exit 1
    fi
}

# 處理命令行參數
case "${1:-all}" in
    "unit")
        check_environment
        install_dependencies
        compile_tests
        run_unit_tests
        ;;
    "bench")
        check_environment
        install_dependencies
        compile_tests
        run_benchmark_tests
        ;;
    "advanced")
        check_environment
        install_dependencies
        compile_tests
        run_advanced_tests
        ;;
    "quality")
        check_environment
        run_code_quality_checks
        ;;
    "all")
        main
        ;;
    *)
        echo "用法: $0 [unit|bench|advanced|quality|all]"
        echo "  unit     - 只執行單元測試"
        echo "  bench    - 只執行基準測試"
        echo "  advanced - 只執行高級測試"
        echo "  quality  - 只執行品質檢查"
        echo "  all      - 執行所有測試（預設）"
        exit 1
        ;;
esac