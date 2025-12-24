#!/bin/bash

# CI/CD Validation Script for NimsForest
# This script validates the CI/CD pipeline works correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track results
PASSED=0
FAILED=0
SKIPPED=0

# Helper functions
print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_test() {
    echo -e "${YELLOW}▶ Testing: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((PASSED++))
}

print_failure() {
    echo -e "${RED}✗ $1${NC}"
    ((FAILED++))
}

print_skip() {
    echo -e "${YELLOW}⊘ $1${NC}"
    ((SKIPPED++))
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Test functions
test_prerequisites() {
    print_header "Checking Prerequisites"
    
    print_test "Go installation"
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}')
        print_success "Go installed: $GO_VERSION"
    else
        print_failure "Go not installed"
        return 1
    fi
    
    print_test "Make installation"
    if command -v make &> /dev/null; then
        print_success "Make installed"
    else
        print_failure "Make not installed"
        return 1
    fi
    
    print_test "Git installation"
    if command -v git &> /dev/null; then
        print_success "Git installed"
    else
        print_failure "Git not installed"
        return 1
    fi
    
    print_test "GitHub CLI (optional)"
    if command -v gh &> /dev/null; then
        print_success "GitHub CLI installed"
    else
        print_skip "GitHub CLI not installed (optional)"
    fi
}

test_go_modules() {
    print_header "Validating Go Modules"
    
    print_test "Go module verification"
    if go mod verify &> /dev/null; then
        print_success "Go modules verified"
    else
        print_failure "Go module verification failed"
        return 1
    fi
    
    print_test "Go module download"
    if go mod download &> /dev/null; then
        print_success "Go modules downloaded"
    else
        print_failure "Go module download failed"
        return 1
    fi
}

test_make_commands() {
    print_header "Testing Make Commands"
    
    print_test "make verify"
    if make verify &> /dev/null; then
        print_success "make verify passed"
    else
        print_failure "make verify failed"
    fi
    
    print_test "make fmt"
    if make fmt &> /dev/null; then
        print_success "make fmt passed"
    else
        print_failure "make fmt failed"
    fi
    
    print_test "make vet"
    if make vet &> /dev/null; then
        print_success "make vet passed"
    else
        print_failure "make vet failed"
    fi
    
    print_test "make build"
    if make build &> /dev/null; then
        print_success "make build passed"
        if [ -f "forest" ]; then
            print_success "Binary 'forest' created"
            
            # Test binary
            if [ -x "forest" ]; then
                print_success "Binary is executable"
            else
                print_failure "Binary is not executable"
            fi
        else
            print_failure "Binary 'forest' not found"
        fi
    else
        print_failure "make build failed"
    fi
}

test_unit_tests() {
    print_header "Running Unit Tests"
    
    print_test "make test"
    if make test &> test_output.log; then
        print_success "All tests passed"
        
        # Show test summary
        if grep -q "PASS" test_output.log; then
            PASS_COUNT=$(grep -c "PASS" test_output.log || echo "0")
            print_info "Passed: $PASS_COUNT test packages"
        fi
    else
        print_failure "Tests failed"
        print_info "Check test_output.log for details"
        return 1
    fi
    
    rm -f test_output.log
}

test_linting() {
    print_header "Testing Code Quality"
    
    print_test "golangci-lint availability"
    if command -v golangci-lint &> /dev/null; then
        print_success "golangci-lint installed"
        
        print_test "Running linter"
        if make lint &> lint_output.log; then
            print_success "Linting passed"
        else
            print_failure "Linting failed"
            print_info "Check lint_output.log for details"
        fi
        rm -f lint_output.log
    else
        print_skip "golangci-lint not installed (run: make lint for installation)"
    fi
}

test_nats_integration() {
    print_header "Testing NATS Integration"
    
    print_test "NATS server availability"
    if command -v nats-server &> /dev/null; then
        print_success "NATS server installed"
        
        print_test "Starting NATS"
        if make start &> /dev/null; then
            sleep 2
            print_success "NATS started"
            
            print_test "NATS status check"
            if curl -f http://localhost:8222/varz &> /dev/null; then
                print_success "NATS is responding"
            else
                print_failure "NATS not responding"
            fi
            
            print_test "Stopping NATS"
            if make stop &> /dev/null; then
                print_success "NATS stopped"
            else
                print_failure "Failed to stop NATS"
            fi
        else
            print_failure "Failed to start NATS"
        fi
    else
        print_skip "NATS not installed (run: make install-nats)"
    fi
}

test_workflow_syntax() {
    print_header "Validating Workflow Files"
    
    print_test "Workflow files exist"
    if [ -f ".github/workflows/ci.yml" ] && \
       [ -f ".github/workflows/release.yml" ] && \
       [ -f ".github/workflows/debian-package.yml" ]; then
        print_success "All workflow files present"
    else
        print_failure "Missing workflow files"
        return 1
    fi
    
    print_test "YAML syntax validation"
    if command -v yamllint &> /dev/null; then
        ERROR_COUNT=0
        for file in .github/workflows/*.yml; do
            if yamllint "$file" &> /dev/null; then
                print_success "$(basename $file) syntax valid"
            else
                print_failure "$(basename $file) syntax invalid"
                ((ERROR_COUNT++))
            fi
        done
        
        if [ $ERROR_COUNT -eq 0 ]; then
            return 0
        else
            return 1
        fi
    else
        print_skip "yamllint not installed (pip install yamllint)"
    fi
}

test_documentation() {
    print_header "Checking Documentation"
    
    REQUIRED_DOCS=(
        "README.md"
        "DEPLOYMENT.md"
        "CI_CD.md"
        "CI_CD_SETUP.md"
        "VALIDATION_GUIDE.md"
    )
    
    for doc in "${REQUIRED_DOCS[@]}"; do
        print_test "$doc exists"
        if [ -f "$doc" ]; then
            print_success "$doc present"
        else
            print_failure "$doc missing"
        fi
    done
}

test_configuration_files() {
    print_header "Validating Configuration Files"
    
    print_test ".golangci.yml"
    if [ -f ".golangci.yml" ]; then
        print_success ".golangci.yml present"
    else
        print_failure ".golangci.yml missing"
    fi
    
    print_test ".codecov.yml"
    if [ -f ".codecov.yml" ]; then
        print_success ".codecov.yml present"
    else
        print_failure ".codecov.yml missing"
    fi
    
    print_test "Makefile"
    if [ -f "Makefile" ]; then
        print_success "Makefile present"
        
        # Check for key targets
        if grep -q "^setup:" Makefile && \
           grep -q "^test:" Makefile && \
           grep -q "^build:" Makefile; then
            print_success "Makefile has required targets"
        else
            print_failure "Makefile missing required targets"
        fi
    else
        print_failure "Makefile missing"
    fi
}

print_summary() {
    print_header "Validation Summary"
    
    TOTAL=$((PASSED + FAILED + SKIPPED))
    
    echo -e "${GREEN}Passed:  $PASSED${NC}"
    echo -e "${RED}Failed:  $FAILED${NC}"
    echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
    echo -e "Total:   $TOTAL"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║                                                                       ║${NC}"
        echo -e "${GREEN}║                 ✓ ALL VALIDATIONS PASSED! ✓                          ║${NC}"
        echo -e "${GREEN}║                                                                       ║${NC}"
        echo -e "${GREEN}║              Your CI/CD pipeline is ready to use!                    ║${NC}"
        echo -e "${GREEN}║                                                                       ║${NC}"
        echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════════╝${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Push to GitHub to trigger CI"
        echo "  2. Create a test tag to trigger release workflow"
        echo "  3. Review VALIDATION_GUIDE.md for detailed testing"
        return 0
    else
        echo -e "${RED}╔═══════════════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║                                                                       ║${NC}"
        echo -e "${RED}║                  ✗ VALIDATION FAILED ✗                               ║${NC}"
        echo -e "${RED}║                                                                       ║${NC}"
        echo -e "${RED}║              Please fix the errors above and retry                   ║${NC}"
        echo -e "${RED}║                                                                       ║${NC}"
        echo -e "${RED}╚═══════════════════════════════════════════════════════════════════════╝${NC}"
        echo ""
        echo "For help:"
        echo "  - Review VALIDATION_GUIDE.md"
        echo "  - Check CI_CD.md troubleshooting section"
        echo "  - Run: make help"
        return 1
    fi
}

# Main execution
main() {
    local MODE=${1:-all}
    
    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                                                                       ║${NC}"
    echo -e "${BLUE}║              NimsForest CI/CD Validation Script                      ║${NC}"
    echo -e "${BLUE}║                                                                       ║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════════════╝${NC}"
    
    case $MODE in
        local|all)
            test_prerequisites || true
            test_go_modules || true
            test_make_commands || true
            test_unit_tests || true
            test_linting || true
            test_workflow_syntax || true
            test_configuration_files || true
            test_documentation || true
            ;;
        nats)
            test_prerequisites || true
            test_nats_integration || true
            ;;
        quick)
            test_prerequisites || true
            test_go_modules || true
            test_unit_tests || true
            ;;
        *)
            echo "Usage: $0 [local|nats|quick|all]"
            echo ""
            echo "Modes:"
            echo "  local  - Run all local validations (default)"
            echo "  nats   - Test NATS integration only"
            echo "  quick  - Run quick checks (prereqs, modules, tests)"
            echo "  all    - Run all validations"
            exit 1
            ;;
    esac
    
    print_summary
}

# Run main with arguments
main "$@"
